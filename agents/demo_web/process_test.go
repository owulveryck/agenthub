package main

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func testHub() *Hub {
	return NewHub(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
}

func TestBuildCmdSetsProcessGroup(t *testing.T) {
	pm := NewProcessManager(testHub(), slog.Default())
	pm.env = os.Environ()
	spec := ProcessSpec{
		Name:  "test",
		GoRun: []string{"./agents/demo_web"},
	}

	cmd, err := pm.buildCmd(spec, pm.env)
	if err != nil {
		t.Fatalf("buildCmd failed: %v", err)
	}

	if cmd.SysProcAttr == nil {
		t.Fatal("expected SysProcAttr to be set")
	}
	if !cmd.SysProcAttr.Setpgid {
		t.Fatal("expected Setpgid to be true")
	}
}

func TestSignalProcessGroup(t *testing.T) {
	// Start a process in its own process group
	cmd := exec.Command("sleep", "60")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start sleep: %v", err)
	}

	pid := cmd.Process.Pid

	// Signal the process group with SIGTERM
	err := signalProcessGroup(pid, syscall.SIGTERM)
	if err != nil {
		t.Fatalf("signalProcessGroup failed: %v", err)
	}

	// Wait for the process to exit
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case <-done:
		// Process exited as expected
	case <-time.After(5 * time.Second):
		cmd.Process.Kill()
		t.Fatal("process did not exit after signalProcessGroup")
	}
}

func TestManagedProcessDoneChannel(t *testing.T) {
	hub := testHub()
	go hub.Run()

	pm := NewProcessManager(hub, slog.Default())
	pm.env = os.Environ()

	spec := ProcessSpec{
		Name:  "test_sleep",
		GoRun: []string{},
	}

	// Use a direct command for testing
	cmd := exec.Command("sleep", "0.1")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	mp := &managedProcess{
		spec: spec,
		cmd:  cmd,
		done: make(chan struct{}),
	}

	go pm.monitorProcess(spec.Name, mp)

	select {
	case <-mp.done:
		// done channel was closed as expected
	case <-time.After(5 * time.Second):
		t.Fatal("done channel not closed after process exit")
	}
}

func TestStopAgentUsesProcessGroup(t *testing.T) {
	hub := testHub()
	go hub.Run()

	pm := &ProcessManager{
		hub:    hub,
		logger: slog.Default(),
		env:    os.Environ(),
	}

	// Start a long-running process
	cmd := exec.Command("sleep", "60")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	mp := &managedProcess{
		spec: ProcessSpec{Name: "test_agent"},
		cmd:  cmd,
		done: make(chan struct{}),
	}
	pm.procs = append(pm.procs, mp)

	// Start the monitor so done gets closed
	go pm.monitorProcess("test_agent", mp)

	err := pm.StopAgent("test_agent")
	if err != nil {
		t.Fatalf("StopAgent failed: %v", err)
	}

	// Verify the agent was removed from the process list
	pm.mu.Lock()
	defer pm.mu.Unlock()
	for _, p := range pm.procs {
		if p.spec.Name == "test_agent" {
			t.Fatal("agent should have been removed from procs")
		}
	}
}

func TestShutdownUsesProcessGroup(t *testing.T) {
	hub := testHub()
	go hub.Run()

	pm := &ProcessManager{
		hub:    hub,
		logger: slog.Default(),
		env:    os.Environ(),
	}

	// Start two long-running processes
	for _, name := range []string{"proc1", "proc2"} {
		cmd := exec.Command("sleep", "60")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := cmd.Start(); err != nil {
			t.Fatalf("failed to start %s: %v", name, err)
		}
		mp := &managedProcess{
			spec: ProcessSpec{Name: name},
			cmd:  cmd,
			done: make(chan struct{}),
		}
		pm.procs = append(pm.procs, mp)
		go pm.monitorProcess(name, mp)
	}

	// Shutdown should complete without hanging
	done := make(chan struct{})
	go func() {
		pm.Shutdown()
		close(done)
	}()

	select {
	case <-done:
		// Shutdown completed
	case <-time.After(10 * time.Second):
		t.Fatal("Shutdown did not complete in time")
	}
}

func TestWaitAllUsesChannel(t *testing.T) {
	hub := testHub()
	go hub.Run()

	pm := &ProcessManager{
		hub:    hub,
		logger: slog.Default(),
		env:    os.Environ(),
	}

	// Start a short-lived process
	cmd := exec.Command("sleep", "0.1")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	mp := &managedProcess{
		spec: ProcessSpec{Name: "short"},
		cmd:  cmd,
		done: make(chan struct{}),
	}
	pm.procs = append(pm.procs, mp)
	go pm.monitorProcess("short", mp)

	done := make(chan struct{})
	go func() {
		pm.WaitAll()
		close(done)
	}()

	select {
	case <-done:
		// WaitAll completed
	case <-time.After(5 * time.Second):
		t.Fatal("WaitAll did not complete in time")
	}
}

func TestStartAgent(t *testing.T) {
	hub := testHub()
	go hub.Run()

	pm := NewProcessManager(hub, slog.Default())
	pm.env = os.Environ()

	// Use a real spec that will work - just sleep
	spec := ProcessSpec{
		Name:   "test_dynamic",
		Binary: "sleep",
		GoRun:  nil,
	}

	// We can't easily test StartAgent with go run in unit tests,
	// so test the duplicate detection path
	cmd := exec.Command("sleep", "60")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	mp := &managedProcess{
		spec: spec,
		cmd:  cmd,
		done: make(chan struct{}),
	}
	pm.mu.Lock()
	pm.procs = append(pm.procs, mp)
	pm.mu.Unlock()
	go pm.monitorProcess(spec.Name, mp)

	err := pm.StartAgent(context.Background(), spec)
	if err == nil {
		t.Fatal("expected error for duplicate agent")
	}

	// Clean up
	signalProcessGroup(cmd.Process.Pid, syscall.SIGTERM)
	<-mp.done
}
