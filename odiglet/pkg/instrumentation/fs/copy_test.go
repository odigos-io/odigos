package fs

import (
	"fmt"
	"os"
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

	filesToKeep := make(map[string]struct{})
	gotFiles, err := getFiles(tempDir+"/dir1", false, filesToKeep)
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
	filesToKeep := make(map[string]struct{})
	src := t.TempDir()
	dest := t.TempDir()
	files, err := createTestFiles(src, 10)
	if err != nil {
		t.Fatalf("createTestFiles failed: %v", err)
	}

	err = copyDirectories(src, dest, filesToKeep)
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

func BenchmarkCopyDirectories(b *testing.B) {
	filesToKeep := make(map[string]struct{})
	src := b.TempDir()
	dest := b.TempDir()
	_, err := createTestFiles(src, b.N)
	if err != nil {
		b.Fatalf("createTestFiles failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = copyDirectories(src, dest, filesToKeep)
		if err != nil {
			b.Fatalf("copyDirectories failed: %v", err)
		}
	}
}
