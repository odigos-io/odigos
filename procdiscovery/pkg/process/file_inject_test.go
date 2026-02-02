package process

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestInjectToProcessTempDir(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) (sourcePath string, cleanup func())
		wantErr bool
		errMsg  string
	}{
		{
			name: "successful injection",
			setup: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()
				sourcePath := filepath.Join(tmpDir, "test-file.txt")
				content := []byte("Hello from test!")
				if err := os.WriteFile(sourcePath, content, 0o644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return sourcePath, func() {}
			},
			wantErr: false,
		},
		{
			name: "source file does not exist",
			setup: func(t *testing.T) (string, func()) {
				return "/non/existent/file.txt", func() {}
			},
			wantErr: true,
			errMsg:  "failed to stat source path",
		},
		{
			name: "source path is a directory",
			setup: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()
				return tmpDir, func() {}
			},
			wantErr: true,
			errMsg:  "source path is not a regular file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourcePath, cleanup := tt.setup(t)
			defer cleanup()

			// Start a long-running process
			cmd := exec.Command("sleep", "10")
			if err := cmd.Start(); err != nil {
				t.Fatalf("failed to start test process: %v", err)
			}
			defer cmd.Process.Kill()

			pid := cmd.Process.Pid

			// Execute the function
			err := InjectToProcessTempDir(pid, sourcePath)

			// Check error expectations
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify the file was copied successfully
			expectedPath := filepath.Join("/proc", fmt.Sprintf("%d", pid), "root", os.TempDir(), filepath.Base(sourcePath))
			if _, err := os.Stat(expectedPath); err != nil {
				t.Errorf("file not found at expected path %s: %v", expectedPath, err)
			}

			// Read and verify content
			originalContent, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatalf("failed to read original file: %v", err)
			}

			copiedContent, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatalf("failed to read copied file: %v", err)
			}

			if string(originalContent) != string(copiedContent) {
				t.Errorf("file content mismatch: original=%q, copied=%q", originalContent, copiedContent)
			}
		})
	}
}

func TestInjectToProcessTempDir_WithTargetProcess(t *testing.T) {
	tmpDir := t.TempDir()

	// Compile the target process first
	binaryPath := filepath.Join(tmpDir, "target_process")
	compileCmd := exec.Command("go", "build", "-o", binaryPath, "./testdata/target_process.go")
	if output, err := compileCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to compile target process: %v\nOutput: %s", err, output)
	}

	sourcePath := filepath.Join(tmpDir, "injected-file.txt")
	testContent := "test content from injection"
	if err := os.WriteFile(sourcePath, []byte(testContent), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cmd := exec.Command(binaryPath, filepath.Base(sourcePath), testContent)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start target process: %v", err)
	}
	defer cmd.Process.Kill()

	pid := cmd.Process.Pid
	t.Logf("Target process PID: %d", pid)

	// Give the process a moment to start and set up signal handler
	time.Sleep(300 * time.Millisecond)

	t.Logf("Injecting file to process %d", pid)
	if err := InjectToProcessTempDir(pid, sourcePath); err != nil {
		t.Fatalf("InjectToProcessTempDir failed: %v", err)
	}

	// Send SIGUSR1 to signal the process to check the file
	t.Logf("Sending SIGUSR1 to process %d", pid)
	if err := syscall.Kill(pid, syscall.SIGUSR1); err != nil {
		t.Fatalf("failed to send SIGUSR1 to target process: %v", err)
	}

	// Wait for the process to complete its verification
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("target process verification failed: %v", err)
		} else {
			t.Log("Target process completed successfully")
		}
	case <-time.After(5 * time.Second):
		t.Error("target process verification timed out")
	}
}

func BenchmarkInjectToProcessTempDir(b *testing.B) {
	benchmarks := []struct {
		name string
		size int64
	}{
		{"1KB", 1 * 1024},
		{"1MB", 1 * 1024 * 1024},
		{"10MB", 10 * 1024 * 1024},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			tmpDir := b.TempDir()
			sourcePath := filepath.Join(tmpDir, "test-file.bin")
			
			// Generate data of the specified size
			data := make([]byte, bm.size)
			for i := range data {
				data[i] = byte(i % 256)
			}
			
			if err := os.WriteFile(sourcePath, data, 0o644); err != nil {
				b.Fatalf("failed to create test file: %v", err)
			}

			// Setup: Start a long-running target process
			cmd := exec.Command("sleep", "300")
			if err := cmd.Start(); err != nil {
				b.Fatalf("failed to start test process: %v", err)
			}
			defer cmd.Process.Kill()

			pid := cmd.Process.Pid
			time.Sleep(50 * time.Millisecond)

			destBasePath := filepath.Join("/proc", fmt.Sprintf("%d", pid), "root", os.TempDir())

			// Reset timer to exclude all setup time
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Stop timer during cleanup
				if i > 0 {
					b.StopTimer()
					destPath := filepath.Join(destBasePath, filepath.Base(sourcePath))
					os.Remove(destPath)
					b.StartTimer()
				}

				if err := InjectToProcessTempDir(pid, sourcePath); err != nil {
					b.Fatalf("InjectToProcessTempDir failed: %v", err)
				}
			}

			b.StopTimer()

			// Report throughput in MB/s
			b.SetBytes(bm.size)
		})
	}
}
