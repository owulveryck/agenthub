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
	"syscall"
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

// SummaryAgentSpec is the spec for the summary_agent, available for dynamic start/stop.
var SummaryAgentSpec = ProcessSpec{
	Name:   "summary_agent",
	Binary: "bin/summary_agent",
	GoRun:  []string{"./agents/summary_agent"},
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
	spec    ProcessSpec
	cmd     *exec.Cmd
	done    chan struct{}
	waitErr error
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

		mp := &managedProcess{spec: spec, cmd: cmd, done: make(chan struct{})}
		pm.procs = append(pm.procs, mp)

		// Stream stdout and stderr to the WebSocket hub
		go pm.streamOutput(spec.Name, stdout)
		go pm.streamOutput(spec.Name, stderr)

		// Monitor for unexpected exit
		go pm.monitorProcess(spec.Name, mp)

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
		pid := mp.cmd.Process.Pid
		pm.logger.Info("stopping child process", "name", name, "pid", pid)

		// Send SIGINT to the entire process group for graceful shutdown
		if err := signalProcessGroup(pid, syscall.SIGINT); err != nil {
			pm.logger.Warn("failed to send interrupt to process group", "name", name, "error", err)
			signalProcessGroup(pid, syscall.SIGKILL)
			continue
		}

		// Wait with timeout using the done channel (single Wait owner)
		select {
		case <-mp.done:
			pm.logger.Info("child process stopped", "name", name)
		case <-time.After(gracefulTimeout):
			pm.logger.Warn("child process did not stop gracefully, killing", "name", name)
			signalProcessGroup(pid, syscall.SIGKILL)
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

	mp := &managedProcess{spec: spec, cmd: cmd, done: make(chan struct{})}
	pm.procs = append(pm.procs, mp)

	go pm.streamOutput(spec.Name, stdout)
	go pm.streamOutput(spec.Name, stderr)
	go pm.monitorProcess(spec.Name, mp)

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

		pid := mp.cmd.Process.Pid
		pm.logger.Info("stopping child process", "name", name, "pid", pid)

		if err := signalProcessGroup(pid, syscall.SIGINT); err != nil {
			pm.logger.Warn("failed to send interrupt to process group, killing", "name", name, "error", err)
			signalProcessGroup(pid, syscall.SIGKILL)
			pm.procs = append(pm.procs[:i], pm.procs[i+1:]...)
			return nil
		}

		select {
		case <-mp.done:
			pm.logger.Info("child process stopped", "name", name)
		case <-time.After(gracefulTimeout):
			pm.logger.Warn("child process did not stop gracefully, killing", "name", name)
			signalProcessGroup(pid, syscall.SIGKILL)
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
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		return cmd, nil
	}

	// Fall back to pre-built binary
	if _, err := os.Stat(spec.Binary); err == nil {
		cmd := exec.Command("./" + spec.Binary)
		cmd.Env = env
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		return cmd, nil
	}

	return nil, fmt.Errorf("no go run path or binary for %s", spec.Name)
}

// signalProcessGroup sends a signal to the entire process group identified by
// the given PID. Using a negative PID targets the group, ensuring both the
// parent (e.g. "go run") and its children receive the signal.
func signalProcessGroup(pid int, sig syscall.Signal) error {
	return syscall.Kill(-pid, sig)
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

func (pm *ProcessManager) monitorProcess(name string, mp *managedProcess) {
	mp.waitErr = mp.cmd.Wait()
	close(mp.done)
	if mp.waitErr != nil {
		pm.logger.Warn("child process exited", "name", name, "error", mp.waitErr)
		pm.hub.Broadcast(WSMessage{
			Type:    "event",
			Source:  name,
			Icon:    "err",
			Content: "Process exited",
			Detail:  mp.waitErr.Error(),
			Status:  "error",
		})
	}
}

// WaitAll blocks until all child processes have exited.
func (pm *ProcessManager) WaitAll() {
	for _, mp := range pm.procs {
		<-mp.done
	}
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
