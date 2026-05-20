package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestGetFiles(t *testing.T) {
	// Make nested directories with files in them
	// Test if getFiles returns all files
	tempDir := t.TempDir()
	files, err := createTestFiles(tempDir, 10)
	if err != nil {
		t.Fatalf("createTestFiles failed: %v", err)
	}

	gotFiles, err := getFiles(tempDir + "/dir1")
	if err != nil {
		t.Fatalf("getFiles failed: %v", err)
	}

	if len(gotFiles) != len(files) {
		t.Fatalf("Expected %d files, got %d", len(files), len(gotFiles))
	}

	for _, file := range files {
		filePath := tempDir + "/" + file
		found := false
		for _, gotFile := range gotFiles {
			if gotFile == filePath {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Expected file %s not found", file)
		}
	}

}

func createTestFiles(tempDir string, num int) ([]string, error) {
	dirs := []string{"dir1", "dir1/dir2", "dir1/dir2/dir3"}
	var files []string
	for i := 0; i < num; i++ {
		file := fmt.Sprintf("dir1/dir2/dir3/file%d", i)
		files = append(files, file)
	}

	for _, dir := range dirs {
		path := tempDir + "/" + dir
		err := os.MkdirAll(path, 0755)
		if err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}
	for _, file := range files {
		path := tempDir + "/" + file
		_, err := os.Create(path)
		if err != nil {
			return nil, fmt.Errorf("failed to create file %s: %v", file, err)
		}
	}
	return files, nil
}

func TestCopyDirectories(t *testing.T) {
	src := t.TempDir()
	dest := t.TempDir()
	files, err := createTestFiles(src, 10)
	if err != nil {
		t.Fatalf("createTestFiles failed: %v", err)
	}

	err = copyDirectories(src, dest)
	if err != nil {
		t.Fatalf("copyDirectories failed: %v", err)
	}

	for _, file := range files {
		srcFile := src + "/" + file
		destFile := dest + "/" + file
		srcStat, err := os.Stat(srcFile)
		if err != nil {
			t.Fatalf("failed to stat source file %s: %v", srcFile, err)
		}

		destStat, err := os.Stat(destFile)
		if err != nil {
			t.Fatalf("failed to stat destination file %s: %v", destFile, err)
		}

		if srcStat.Size() != destStat.Size() {
			t.Fatalf("file sizes do not match: %s (%d) != %s (%d)", srcFile, srcStat.Size(), destFile, destStat.Size())
		}
	}
}

func TestEnsureDirectoryTraversalPermissions(t *testing.T) {
	root := t.TempDir()
	nestedDir := filepath.Join(root, "java", "nested")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "javaagent.jar"), []byte("agent"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	restrictedJavaDir := filepath.Join(root, "java")
	defer os.Chmod(restrictedJavaDir, 0755)
	defer os.Chmod(nestedDir, 0755)

	if err := os.Chmod(nestedDir, 0644); err != nil {
		t.Fatalf("failed to restrict nested directory: %v", err)
	}
	if err := os.Chmod(restrictedJavaDir, 0644); err != nil {
		t.Fatalf("failed to restrict java directory: %v", err)
	}

	if err := ensureDirectoryTraversalPermissions(root); err != nil {
		t.Fatalf("ensureDirectoryTraversalPermissions failed: %v", err)
	}

	for _, dir := range []string{restrictedJavaDir, nestedDir} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("failed to stat directory %s: %v", dir, err)
		}
		if got := info.Mode().Perm(); got&0111 != 0111 {
			t.Fatalf("directory %s is not traversable, mode: %v", dir, got)
		}
	}
}

func BenchmarkCopyDirectories(b *testing.B) {
	src := b.TempDir()
	dest := b.TempDir()
	_, err := createTestFiles(src, b.N)
	if err != nil {
		b.Fatalf("createTestFiles failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = copyDirectories(src, dest)
		if err != nil {
			b.Fatalf("copyDirectories failed: %v", err)
		}
	}
}
