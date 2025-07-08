package process

import (
	"os/exec"
	"runtime"
	"testing"
	"time"
)

func TestIsSafeExecutionMode_NormalProcess(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping test on non-Linux OS")
	}

	// Start a background sleep process
	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start sleep process: %v", err)
	}
	defer cmd.Process.Kill()

	pid := cmd.Process.Pid

	// Give the process a moment to start
	time.Sleep(100 * time.Millisecond)

	secure, err := isSecureExecutionMode(pid)
	if err != nil {
		t.Fatalf("Error reading auxv: %v", err)
	}

	if secure {
		t.Errorf("Expected AT_SECURE=0 for normal process, got 1")
	}
}
