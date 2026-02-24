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
			err := InjectFileToProcessTempDir(pid, sourcePath)

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
	if err := InjectFileToProcessTempDir(pid, sourcePath); err != nil {
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

func TestInjectDirToProcessTempDir(t *testing.T) {
	sourceDir := setupTestDirectory(t)
	pid := startTestProcess(t)

	if err := InjectDirToProcessTempDir(pid, sourceDir, false); err != nil {
		t.Fatalf("InjectDirToProcessTempDir failed: %v", err)
	}

	verifyDirectoryCopied(t, pid, sourceDir)
}

func setupTestDirectory(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "test-dir")

	structure := map[string]fileSpec{
		"file1.txt":               {content: "content of file1", perm: 0o644},
		"subdir/file2.txt":        {content: "content of file2", perm: 0o600},
		"subdir/nested/file3.txt": {content: "content of file3", perm: 0o644},
	}

	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	for path, spec := range structure {
		fullPath := filepath.Join(sourceDir, path)

		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}

		if err := os.WriteFile(fullPath, []byte(spec.content), spec.perm); err != nil {
			t.Fatalf("failed to create file %s: %v", path, err)
		}
	}

	return sourceDir
}

func startTestProcess(t *testing.T) int {
	t.Helper()

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start test process: %v", err)
	}
	t.Cleanup(func() { cmd.Process.Kill() })

	return cmd.Process.Pid
}

func verifyDirectoryCopied(t *testing.T, pid int, sourceDir string) {
	t.Helper()

	procTmpDir := filepath.Join("/proc", fmt.Sprintf("%d", pid), "root", os.TempDir())
	destDir := filepath.Join(procTmpDir, filepath.Base(sourceDir))

	checks := []struct {
		path        string
		isDir       bool
		content     string
		permissions os.FileMode
	}{
		{path: ".", isDir: true},
		{path: "file1.txt", content: "content of file1", permissions: 0o644},
		{path: "subdir", isDir: true},
		{path: "subdir/file2.txt", content: "content of file2", permissions: 0o600},
		{path: "subdir/nested", isDir: true},
		{path: "subdir/nested/file3.txt", content: "content of file3", permissions: 0o644},
	}

	for _, check := range checks {
		fullPath := filepath.Join(destDir, check.path)
		info, err := os.Stat(fullPath)
		if err != nil {
			t.Errorf("path %s: not found: %v", check.path, err)
			continue
		}

		if check.isDir {
			if !info.IsDir() {
				t.Errorf("path %s: expected directory, got file", check.path)
			}
		} else {
			if info.IsDir() {
				t.Errorf("path %s: expected file, got directory", check.path)
				continue
			}

			content, err := os.ReadFile(fullPath)
			if err != nil {
				t.Errorf("path %s: failed to read: %v", check.path, err)
				continue
			}

			if string(content) != check.content {
				t.Errorf("path %s: content mismatch: got %q, want %q",
					check.path, content, check.content)
			}

			if info.Mode().Perm() != check.permissions {
				t.Errorf("path %s: permission mismatch: got %o, want %o",
					check.path, info.Mode().Perm(), check.permissions)
			}
		}
	}
}

type fileSpec struct {
	content string
	perm    os.FileMode
}

func TestInjectDirToProcessTempDir_SourceNotDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "not-a-dir.txt")
	if err := os.WriteFile(sourceFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	pid := startTestProcess(t)

	err := InjectDirToProcessTempDir(pid, sourceFile, false)
	if err == nil {
		t.Error("expected error for non-directory source, got nil")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("expected 'not a directory' error, got: %v", err)
	}
}

func TestInjectDirToProcessTempDir_SourceNotExist(t *testing.T) {
	pid := startTestProcess(t)

	err := InjectDirToProcessTempDir(pid, "/non/existent/directory", false)
	if err == nil {
		t.Error("expected error for non-existent source, got nil")
	}
	if !strings.Contains(err.Error(), "failed to stat source directory") {
		t.Errorf("expected stat error, got: %v", err)
	}
}

func TestInjectDirToProcessTempDir_NoOverrideSkipsExisting(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "test-dir-no-override")

	// Create initial directory structure
	initialStructure := map[string]fileSpec{
		"file1.txt":        {content: "original content", perm: 0o644},
		"subdir/file2.txt": {content: "original nested", perm: 0o644},
	}

	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	for path, spec := range initialStructure {
		fullPath := filepath.Join(sourceDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(spec.content), spec.perm); err != nil {
			t.Fatalf("failed to create file %s: %v", path, err)
		}
	}

	pid := startTestProcess(t)

	if err := InjectDirToProcessTempDir(pid, sourceDir, false); err != nil {
		t.Fatalf("first injection failed: %v", err)
	}

	procTmpDir := filepath.Join("/proc", fmt.Sprintf("%d", pid), "root", os.TempDir())
	destDir := filepath.Join(procTmpDir, filepath.Base(sourceDir))

	// Verify initial content was copied
	content, err := os.ReadFile(filepath.Join(destDir, "file1.txt"))
	if err != nil {
		t.Fatalf("failed to read file1.txt after first injection: %v", err)
	}
	if string(content) != "original content" {
		t.Fatalf("unexpected content after first injection: got %q, want %q", content, "original content")
	}

	// Modify source files
	if err := os.WriteFile(filepath.Join(sourceDir, "file1.txt"), []byte("modified content"), 0o644); err != nil {
		t.Fatalf("failed to modify file1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "file3.txt"), []byte("new file"), 0o644); err != nil {
		t.Fatalf("failed to create file3: %v", err)
	}

	// Second injection with override=false - should be a no-op since directory exists
	if err := InjectDirToProcessTempDir(pid, sourceDir, false); err != nil {
		t.Fatalf("second injection failed: %v", err)
	}

	// Verify destination still has ORIGINAL content (not modified)
	checks := []struct {
		path            string
		shouldExist     bool
		expectedContent string
	}{
		{path: "file1.txt", shouldExist: true, expectedContent: "original content"},      // Should NOT be modified
		{path: "subdir/file2.txt", shouldExist: true, expectedContent: "original nested"}, // Should remain
		{path: "file3.txt", shouldExist: false, expectedContent: ""},                      // Should NOT exist (new file not copied)
	}

	for _, check := range checks {
		fullPath := filepath.Join(destDir, check.path)
		content, err := os.ReadFile(fullPath)

		if check.shouldExist {
			if err != nil {
				t.Errorf("path %s: expected to exist but not found: %v", check.path, err)
				continue
			}
			if string(content) != check.expectedContent {
				t.Errorf("path %s: content mismatch: got %q, want %q (override=false should preserve original)",
					check.path, content, check.expectedContent)
			}
		} else {
			if err == nil {
				t.Errorf("path %s: expected not to exist but found with content %q (override=false should not copy new files)",
					check.path, content)
			}
		}
	}
}

func TestInjectDirToProcessTempDir_DirectoryAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "test-dir-override-test")

	// Create initial directory structure
	initialStructure := map[string]fileSpec{
		"file1.txt":        {content: "original file1", perm: 0o644},
		"file2.txt":        {content: "original file2", perm: 0o644},
		"subdir/file3.txt": {content: "original file3", perm: 0o644},
	}

	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	for path, spec := range initialStructure {
		fullPath := filepath.Join(sourceDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(spec.content), spec.perm); err != nil {
			t.Fatalf("failed to create file %s: %v", path, err)
		}
	}

	pid := startTestProcess(t)

	// First injection
	if err := InjectDirToProcessTempDir(pid, sourceDir, true); err != nil {
		t.Fatalf("first injection failed: %v", err)
	}

	procTmpDir := filepath.Join("/proc", fmt.Sprintf("%d", pid), "root", os.TempDir())
	destDir := filepath.Join(procTmpDir, filepath.Base(sourceDir))

	// Add a file directly to the destination (simulating existing content)
	extraFilePath := filepath.Join(destDir, "extra-file.txt")
	if err := os.WriteFile(extraFilePath, []byte("extra content"), 0o644); err != nil {
		t.Fatalf("failed to create extra file: %v", err)
	}

	// Modify source: change file1, remove file2, add file4
	if err := os.WriteFile(filepath.Join(sourceDir, "file1.txt"), []byte("modified file1"), 0o644); err != nil {
		t.Fatalf("failed to modify file1: %v", err)
	}
	if err := os.Remove(filepath.Join(sourceDir, "file2.txt")); err != nil {
		t.Fatalf("failed to remove file2: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "file4.txt"), []byte("new file4"), 0o644); err != nil {
		t.Fatalf("failed to create file4: %v", err)
	}

	// Second injection, test overwrite behavior
	if err := InjectDirToProcessTempDir(pid, sourceDir, true); err != nil {
		t.Fatalf("second injection failed: %v", err)
	}

	// Verify the results
	checks := []struct {
		path            string
		shouldExist     bool
		expectedContent string
		expectedPerm    os.FileMode
	}{
		{path: "file1.txt", shouldExist: true, expectedContent: "modified file1", expectedPerm: 0o644},
		{path: "file2.txt", shouldExist: true, expectedContent: "original file2", expectedPerm: 0o644}, // Still exists (merge behavior)
		{path: "file4.txt", shouldExist: true, expectedContent: "new file4", expectedPerm: 0o644},
		{path: "extra-file.txt", shouldExist: true, expectedContent: "extra content", expectedPerm: 0o644}, // Preserved
		{path: "subdir/file3.txt", shouldExist: true, expectedContent: "original file3", expectedPerm: 0o644},
	}

	for _, check := range checks {
		fullPath := filepath.Join(destDir, check.path)
		info, err := os.Stat(fullPath)

		if check.shouldExist {
			if err != nil {
				t.Errorf("path %s: expected to exist but not found: %v", check.path, err)
				continue
			}

			content, err := os.ReadFile(fullPath)
			if err != nil {
				t.Errorf("path %s: failed to read: %v", check.path, err)
				continue
			}

			if string(content) != check.expectedContent {
				t.Errorf("path %s: content mismatch: got %q, want %q",
					check.path, content, check.expectedContent)
			}

			if info.Mode().Perm() != check.expectedPerm {
				t.Errorf("path %s: permission mismatch: got %o, want %o",
					check.path, info.Mode().Perm(), check.expectedPerm)
			}
		} else {
			if err == nil {
				t.Errorf("path %s: expected not to exist but found", check.path)
			}
		}
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

				if err := InjectFileToProcessTempDir(pid, sourcePath); err != nil {
					b.Fatalf("InjectToProcessTempDir failed: %v", err)
				}
			}

			b.StopTimer()

			// Report throughput in MB/s
			b.SetBytes(bm.size)
		})
	}
}
