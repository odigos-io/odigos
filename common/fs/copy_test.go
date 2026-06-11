package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func createTestFiles(tempDir string, num int) ([]string, error) {
	dirs := []string{"dir1", "dir1/dir2", "dir1/dir2/dir3"}
	var files []string
	for i := 0; i < num; i++ {
		file := fmt.Sprintf("dir1/dir2/dir3/file%d", i)
		files = append(files, file)
	}

	for _, dir := range dirs {
		path := filepath.Join(tempDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}
	for _, file := range files {
		path := filepath.Join(tempDir, file)
		if err := os.WriteFile(path, []byte("content-"+file), 0644); err != nil {
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

	if err := CopyDirectories(src, dest, nil); err != nil {
		t.Fatalf("CopyDirectories failed: %v", err)
	}

	for _, file := range files {
		srcFile := filepath.Join(src, file)
		destFile := filepath.Join(dest, file)
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

func TestCopyDirectories_DeletesStaleFiles(t *testing.T) {
	src := t.TempDir()
	dest := t.TempDir()

	if _, err := createTestFiles(src, 3); err != nil {
		t.Fatalf("createTestFiles failed: %v", err)
	}

	staleFile := filepath.Join(dest, "dir1", "dir2", "dir3", "stale_file")
	if err := os.MkdirAll(filepath.Dir(staleFile), 0755); err != nil {
		t.Fatalf("failed to create dest dirs: %v", err)
	}
	if err := os.WriteFile(staleFile, []byte("old"), 0644); err != nil {
		t.Fatalf("failed to create stale file: %v", err)
	}

	if err := CopyDirectories(src, dest, nil); err != nil {
		t.Fatalf("CopyDirectories failed: %v", err)
	}

	if _, err := os.Stat(staleFile); !os.IsNotExist(err) {
		t.Fatalf("stale file should have been deleted, but still exists")
	}
}

func TestCopyDirectories_RespectsExcludes(t *testing.T) {
	src := t.TempDir()
	dest := t.TempDir()

	if _, err := createTestFiles(src, 3); err != nil {
		t.Fatalf("createTestFiles failed: %v", err)
	}

	excludedRel := "dir1/dir2/dir3/file0"
	excludes := map[string]bool{excludedRel: true}

	// Pre-create the excluded file in dest with different content
	excludedDest := filepath.Join(dest, excludedRel)
	if err := os.MkdirAll(filepath.Dir(excludedDest), 0755); err != nil {
		t.Fatalf("failed to create dest dirs: %v", err)
	}
	if err := os.WriteFile(excludedDest, []byte("original"), 0644); err != nil {
		t.Fatalf("failed to create excluded file: %v", err)
	}

	if err := CopyDirectories(src, dest, excludes); err != nil {
		t.Fatalf("CopyDirectories failed: %v", err)
	}

	content, err := os.ReadFile(excludedDest)
	if err != nil {
		t.Fatalf("failed to read excluded file: %v", err)
	}
	if string(content) != "original" {
		t.Fatalf("excluded file was overwritten: got %q, want %q", string(content), "original")
	}
}

func TestRemoveChangedFilesFromKeepMap_RenamesChangedFile(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	relPath := "loader/loader.so"
	srcFile := filepath.Join(srcDir, relPath)
	dstFile := filepath.Join(dstDir, relPath)

	if err := os.MkdirAll(filepath.Dir(srcFile), 0755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(dstFile), 0755); err != nil {
		t.Fatalf("mkdir destination: %v", err)
	}
	if err := os.WriteFile(srcFile, []byte("new-version"), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(dstFile, []byte("old-version"), 0644); err != nil {
		t.Fatalf("write destination: %v", err)
	}

	filesToKeepMap := map[string]struct{}{dstFile: {}}

	keep, err := removeChangedFilesFromKeepMap(filesToKeepMap, srcDir, dstDir)
	if err != nil {
		t.Fatalf("removeChangedFilesFromKeepMap failed: %v", err)
	}

	// The original destination file should be renamed (no longer at original path)
	if _, err := os.Stat(dstFile); !os.IsNotExist(err) {
		t.Fatalf("original destination file should have been renamed")
	}

	// The renamed file should be in the keep map
	if len(keep) == 0 {
		t.Fatalf("expected keep map to contain renamed file")
	}

	// Verify the renamed file exists on disk
	for dstPath := range keep {
		if _, err := os.Stat(dstPath); err != nil {
			t.Fatalf("renamed file %s does not exist: %v", dstPath, err)
		}
	}
}

func TestCopyDirectories_SkipsUnchangedFiles(t *testing.T) {
	src := t.TempDir()
	dest := t.TempDir()

	files, err := createTestFiles(src, 3)
	if err != nil {
		t.Fatalf("createTestFiles failed: %v", err)
	}

	// First sync — copies everything.
	if err := CopyDirectories(src, dest, nil); err != nil {
		t.Fatalf("first CopyDirectories failed: %v", err)
	}

	// Record dest mtimes after first sync.
	mtimes := make(map[string]int64)
	for _, file := range files {
		info, err := os.Stat(filepath.Join(dest, file))
		if err != nil {
			t.Fatalf("stat after first sync: %v", err)
		}
		mtimes[file] = info.ModTime().UnixNano()
	}

	// Second sync with identical source — files should be skipped and dest
	// mtimes must remain unchanged.
	if err := CopyDirectories(src, dest, nil); err != nil {
		t.Fatalf("second CopyDirectories failed: %v", err)
	}

	for _, file := range files {
		info, err := os.Stat(filepath.Join(dest, file))
		if err != nil {
			t.Fatalf("stat after second sync: %v", err)
		}
		if info.ModTime().UnixNano() != mtimes[file] {
			t.Fatalf("file %s was re-copied even though it was unchanged", file)
		}
	}
}

func TestCopyDirectories_CopiesChangedFile(t *testing.T) {
	src := t.TempDir()
	dest := t.TempDir()

	files, err := createTestFiles(src, 3)
	if err != nil {
		t.Fatalf("createTestFiles failed: %v", err)
	}

	if err := CopyDirectories(src, dest, nil); err != nil {
		t.Fatalf("first CopyDirectories failed: %v", err)
	}

	// Modify one source file (different content and size).
	changed := files[0]
	if err := os.WriteFile(filepath.Join(src, changed), []byte("updated-content-that-is-longer"), 0644); err != nil {
		t.Fatalf("write updated file: %v", err)
	}

	if err := CopyDirectories(src, dest, nil); err != nil {
		t.Fatalf("second CopyDirectories failed: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dest, changed))
	if err != nil {
		t.Fatalf("read dest file: %v", err)
	}
	if string(got) != "updated-content-that-is-longer" {
		t.Fatalf("changed file not updated in dest: got %q", string(got))
	}
}

func TestCopyDirectories_PreservesMtime(t *testing.T) {
	src := t.TempDir()
	dest := t.TempDir()

	files, err := createTestFiles(src, 3)
	if err != nil {
		t.Fatalf("createTestFiles failed: %v", err)
	}

	if err := CopyDirectories(src, dest, nil); err != nil {
		t.Fatalf("CopyDirectories failed: %v", err)
	}

	for _, file := range files {
		srcInfo, err := os.Stat(filepath.Join(src, file))
		if err != nil {
			t.Fatalf("stat src: %v", err)
		}
		dstInfo, err := os.Stat(filepath.Join(dest, file))
		if err != nil {
			t.Fatalf("stat dst: %v", err)
		}
		if !srcInfo.ModTime().Equal(dstInfo.ModTime()) {
			t.Fatalf("mtime not preserved for %s: src=%v dst=%v", file, srcInfo.ModTime(), dstInfo.ModTime())
		}
	}
}

func BenchmarkCopyDirectories(b *testing.B) {
	src := b.TempDir()
	dest := b.TempDir()
	if _, err := createTestFiles(src, 100); err != nil {
		b.Fatalf("createTestFiles failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := CopyDirectories(src, dest, nil); err != nil {
			b.Fatalf("CopyDirectories failed: %v", err)
		}
	}
}
