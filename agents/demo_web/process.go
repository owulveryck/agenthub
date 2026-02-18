package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"time"
)

// ProcessSpec describes a child process to manage.
type ProcessSpec struct {
	Name   string        // human-readable name, also used as WSMessage.Source
	Binary string        // path to pre-built binary (e.g. "bin/broker")
	GoRun  []string      // fallback: go run package (e.g. ["./broker"])
	Delay  time.Duration // wait after start before launching the next process
}

// Mp3AgentSpec is the spec for the mp3_agent, available for dynamic start/stop.
var Mp3AgentSpec = ProcessSpec{
	Name:   "mp3_agent",
	Binary: "bin/mp3_agent",
	GoRun:  []string{"./agents/mp3_agent"},
	Delay:  0,
}

// ProcessManager starts, monitors, and stops child processes.
type ProcessManager struct {
	specs  []ProcessSpec
	mu     sync.Mutex
	procs  []*managedProcess
	env    []string
	hub    *Hub
	logger *slog.Logger
}

type managedProcess struct {
	spec ProcessSpec
	cmd  *exec.Cmd
}

// NewProcessManager creates a manager for the given process specs.
func NewProcessManager(hub *Hub, logger *slog.Logger) *ProcessManager {
	specs := []ProcessSpec{
		{
			Name:   "broker",
			Binary: "bin/broker",
			GoRun:  []string{"./broker"},
			Delay:  2 * time.Second,
		},
		{
			Name:   "cortex",
			Binary: "bin/cortex",
			GoRun:  []string{"./agents/cortex/cmd"},
			Delay:  1 * time.Second,
		},
		{
			Name:   "echo_agent",
			Binary: "bin/echo_agent",
			GoRun:  []string{"./agents/echo_agent"},
			Delay:  1 * time.Second,
		},
	}
	return &ProcessManager{
		specs:  specs,
		hub:    hub,
		logger: logger,
	}
}

// Start launches all child processes in order with staggered delays.
// It returns after all processes have been started.
func (pm *ProcessManager) Start(ctx context.Context) error {
	// Set common environment for child processes
	pm.env = os.Environ()
	pm.env = setEnv(pm.env, "AGENTHUB_GRPC_PORT", "127.0.0.1:50051")
	pm.env = setEnv(pm.env, "AGENTHUB_BROKER_ADDR", "127.0.0.1")
	pm.env = setEnv(pm.env, "LOG_LEVEL", "DEBUG")

	for _, spec := range pm.specs {
		pm.hub.Broadcast(WSMessage{
			Type:    "event",
			Source:  spec.Name,
			Icon:    "info",
			Content: "Starting...",
		})

		cmd, err := pm.buildCmd(spec, pm.env)
		if err != nil {
			return fmt.Errorf("failed to build command for %s: %w", spec.Name, err)
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("failed to get stdout pipe for %s: %w", spec.Name, err)
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("failed to get stderr pipe for %s: %w", spec.Name, err)
		}

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start %s: %w", spec.Name, err)
		}

		mp := &managedProcess{spec: spec, cmd: cmd}
		pm.procs = append(pm.procs, mp)

		// Stream stdout and stderr to the WebSocket hub
		go pm.streamOutput(spec.Name, stdout)
		go pm.streamOutput(spec.Name, stderr)

		// Monitor for unexpected exit
		go pm.monitorProcess(spec.Name, cmd)

		pm.logger.Info("started child process", "name", spec.Name, "pid", cmd.Process.Pid)
		pm.hub.Broadcast(WSMessage{
			Type:    "event",
			Source:  spec.Name,
			Icon:    "ok",
			Content: fmt.Sprintf("Started (PID %d)", cmd.Process.Pid),
		})

		// Staggered startup
		if spec.Delay > 0 {
			select {
			case <-time.After(spec.Delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return nil
}

// Shutdown stops all child processes in reverse order.
func (pm *ProcessManager) Shutdown() {
	const gracefulTimeout = 5 * time.Second

	// Reverse order
	for i := len(pm.procs) - 1; i >= 0; i-- {
		mp := pm.procs[i]
		if mp.cmd.Process == nil {
			continue
		}
		name := mp.spec.Name
		pm.logger.Info("stopping child process", "name", name, "pid", mp.cmd.Process.Pid)

		// Send SIGINT first for graceful shutdown
		if err := mp.cmd.Process.Signal(os.Interrupt); err != nil {
			pm.logger.Warn("failed to send interrupt", "name", name, "error", err)
			mp.cmd.Process.Kill()
			continue
		}

		// Wait with timeout
		done := make(chan error, 1)
		go func() { done <- mp.cmd.Wait() }()

		select {
		case <-done:
			pm.logger.Info("child process stopped", "name", name)
		case <-time.After(gracefulTimeout):
			pm.logger.Warn("child process did not stop gracefully, killing", "name", name)
			mp.cmd.Process.Kill()
		}
	}
}

// StartAgent dynamically starts a single agent process.
func (pm *ProcessManager) StartAgent(ctx context.Context, spec ProcessSpec) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for _, mp := range pm.procs {
		if mp.spec.Name == spec.Name {
			return fmt.Errorf("agent %s is already running", spec.Name)
		}
	}

	pm.hub.Broadcast(WSMessage{
		Type:    "event",
		Source:  spec.Name,
		Icon:    "info",
		Content: "Starting...",
	})

	cmd, err := pm.buildCmd(spec, pm.env)
	if err != nil {
		return fmt.Errorf("failed to build command for %s: %w", spec.Name, err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe for %s: %w", spec.Name, err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe for %s: %w", spec.Name, err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %w", spec.Name, err)
	}

	mp := &managedProcess{spec: spec, cmd: cmd}
	pm.procs = append(pm.procs, mp)

	go pm.streamOutput(spec.Name, stdout)
	go pm.streamOutput(spec.Name, stderr)
	go pm.monitorProcess(spec.Name, cmd)

	pm.logger.Info("started child process", "name", spec.Name, "pid", cmd.Process.Pid)
	pm.hub.Broadcast(WSMessage{
		Type:    "event",
		Source:  spec.Name,
		Icon:    "ok",
		Content: fmt.Sprintf("Started (PID %d)", cmd.Process.Pid),
	})

	return nil
}

// StopAgent stops a running agent by name.
func (pm *ProcessManager) StopAgent(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	const gracefulTimeout = 5 * time.Second

	for i, mp := range pm.procs {
		if mp.spec.Name != name {
			continue
		}
		if mp.cmd.Process == nil {
			pm.procs = append(pm.procs[:i], pm.procs[i+1:]...)
			return nil
		}

		pm.logger.Info("stopping child process", "name", name, "pid", mp.cmd.Process.Pid)

		if err := mp.cmd.Process.Signal(os.Interrupt); err != nil {
			pm.logger.Warn("failed to send interrupt, killing", "name", name, "error", err)
			mp.cmd.Process.Kill()
			pm.procs = append(pm.procs[:i], pm.procs[i+1:]...)
			return nil
		}

		done := make(chan error, 1)
		go func() { done <- mp.cmd.Wait() }()

		select {
		case <-done:
			pm.logger.Info("child process stopped", "name", name)
		case <-time.After(gracefulTimeout):
			pm.logger.Warn("child process did not stop gracefully, killing", "name", name)
			mp.cmd.Process.Kill()
		}

		pm.procs = append(pm.procs[:i], pm.procs[i+1:]...)

		pm.hub.Broadcast(WSMessage{
			Type:    "event",
			Source:  name,
			Icon:    "info",
			Content: "Stopped",
		})
		return nil
	}

	return fmt.Errorf("agent %s is not running", name)
}

// IsRunning checks if an agent with the given name is currently running.
func (pm *ProcessManager) IsRunning(name string) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for _, mp := range pm.procs {
		if mp.spec.Name == name {
			return true
		}
	}
	return false
}

func (pm *ProcessManager) buildCmd(spec ProcessSpec, env []string) (*exec.Cmd, error) {
	// Prefer go run (always uses fresh source) over pre-built binary
	if len(spec.GoRun) > 0 {
		args := append([]string{"run"}, spec.GoRun...)
		cmd := exec.Command("go", args...)
		cmd.Env = env
		return cmd, nil
	}

	// Fall back to pre-built binary
	if _, err := os.Stat(spec.Binary); err == nil {
		cmd := exec.Command("./" + spec.Binary)
		cmd.Env = env
		return cmd, nil
	}

	return nil, fmt.Errorf("no go run path or binary for %s", spec.Name)
}

func (pm *ProcessManager) streamOutput(source string, r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)
	for scanner.Scan() {
		if event := parseLogLine(source, scanner.Text()); event != nil {
			pm.hub.Broadcast(*event)
		}
	}
}

func (pm *ProcessManager) monitorProcess(name string, cmd *exec.Cmd) {
	err := cmd.Wait()
	if err != nil {
		pm.logger.Warn("child process exited", "name", name, "error", err)
		pm.hub.Broadcast(WSMessage{
			Type:    "event",
			Source:  name,
			Icon:    "err",
			Content: "Process exited",
			Detail:  err.Error(),
			Status:  "error",
		})
	}
}

// WaitAll blocks until all child processes have exited.
func (pm *ProcessManager) WaitAll() {
	var wg sync.WaitGroup
	for _, mp := range pm.procs {
		wg.Add(1)
		go func(cmd *exec.Cmd) {
			defer wg.Done()
			cmd.Wait()
		}(mp.cmd)
	}
	wg.Wait()
}

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if len(e) > len(prefix) && e[:len(prefix)] == prefix {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}
