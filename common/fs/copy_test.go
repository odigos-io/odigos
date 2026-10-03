package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
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

func TestRenameWithHashSuffix(t *testing.T) {
	// sha256("old-javaagent-v1"), the hash removeChangedFilesFromKeepMap passes in.
	const oldJarHash = "9166e887cfbf29e9f515c2c2d23ca381f21f8931c84cf14d9f2d28cb53f44046"

	tests := []struct {
		name     string
		filename string
		hash     string
		want     string
	}{
		{
			name:     "the suffix is the first twelve characters of the hash",
			filename: "javaagent.jar",
			hash:     oldJarHash,
			want:     "javaagent_hash_version-9166e887cfbf.jar",
		},
		{
			name:     "only the last extension is separated from the base name",
			filename: "pythonUSDT.abi3.so",
			hash:     oldJarHash,
			want:     "pythonUSDT.abi3_hash_version-9166e887cfbf.so",
		},
		{
			name:     "a file without an extension keeps its whole name",
			filename: "loader",
			hash:     oldJarHash,
			want:     "loader_hash_version-9166e887cfbf",
		},
		{
			name:     "a hash shorter than the suffix length is used as is",
			filename: "javaagent.jar",
			hash:     "abc",
			want:     "javaagent_hash_version-abc.jar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			original := filepath.Join(dir, tt.filename)
			if err := os.WriteFile(original, []byte("old-javaagent-v1"), 0644); err != nil {
				t.Fatalf("write file: %v", err)
			}

			got, err := renameWithHashSuffix(original, tt.hash)
			if err != nil {
				t.Fatalf("renameWithHashSuffix failed: %v", err)
			}

			if want := filepath.Join(dir, tt.want); got != want {
				t.Fatalf("renamed to %q, want %q", got, want)
			}
			// The rename must move the inode a running process holds open, not copy it, and it must
			// free the canonical path for the incoming version.
			content, err := os.ReadFile(got)
			if err != nil {
				t.Fatalf("read renamed file: %v", err)
			}
			if string(content) != "old-javaagent-v1" {
				t.Errorf("renamed file holds %q, want %q", string(content), "old-javaagent-v1")
			}
			if _, err := os.Stat(original); !os.IsNotExist(err) {
				t.Errorf("original path still exists, stat error: %v", err)
			}
		})
	}

	t.Run("a missing file is not an error", func(t *testing.T) {
		path, err := renameWithHashSuffix(filepath.Join(t.TempDir(), "javaagent.jar"), oldJarHash)
		if err != nil {
			t.Fatalf("renameWithHashSuffix failed: %v", err)
		}
		if path != "" {
			t.Errorf("returned path %q, want none", path)
		}
	})
}

func TestFindHashVersionFiles(t *testing.T) {
	t.Run("returns every version preserved by an earlier upgrade", func(t *testing.T) {
		dir := t.TempDir()
		base := filepath.Join(dir, "javaagent.jar")
		want := []string{
			filepath.Join(dir, "javaagent_hash_version-3bfc269594ef.jar"),
			filepath.Join(dir, "javaagent_hash_version-fb04dcb6970e.jar"),
		}
		for _, path := range append([]string{base}, want...) {
			if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
				t.Fatalf("write file: %v", err)
			}
		}

		got, err := findHashVersionFiles(base)
		if err != nil {
			t.Fatalf("findHashVersionFiles failed: %v", err)
		}
		if !slices.Equal(got, want) {
			t.Errorf("found %v, want %v", got, want)
		}
	})

	t.Run("ignores files that are not versions of the same agent file", func(t *testing.T) {
		dir := t.TempDir()
		base := filepath.Join(dir, "javaagent.jar")
		for _, name := range []string{
			"javaagent.jar",
			"javaagent.jar.bak",
			"other_hash_version-3bfc269594ef.jar",
			"javaagent_hash_version-3bfc269594ef.so",
			"javaagent-hash_version-3bfc269594ef.jar",
		} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("content"), 0644); err != nil {
				t.Fatalf("write file: %v", err)
			}
		}

		got, err := findHashVersionFiles(base)
		if err != nil {
			t.Fatalf("findHashVersionFiles failed: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("found %v, want nothing", got)
		}
	})

	t.Run("a missing directory yields no versions", func(t *testing.T) {
		got, err := findHashVersionFiles(filepath.Join(t.TempDir(), "java", "javaagent.jar"))
		if err != nil {
			t.Fatalf("findHashVersionFiles failed: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("found %v, want nothing", got)
		}
	})
}

func TestFileHash(t *testing.T) {
	// Whether an agent file is considered changed, and therefore whether the copy still in use is
	// preserved, is decided entirely by these hashes.
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "hashes the content with sha256",
			content: "odigos",
			want:    "9b8407c4a1718d0abc03bcb3ef56f4dee61ef75e1b34851496e5270cea752168",
		},
		{
			name:    "an empty file has the empty sha256",
			content: "",
			want:    "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "javaagent.jar")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatalf("write file: %v", err)
			}

			got, err := fileHash(path)
			if err != nil {
				t.Fatalf("fileHash failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("fileHash = %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("the same content in two files hashes the same", func(t *testing.T) {
		dir := t.TempDir()
		src, dst := filepath.Join(dir, "src.jar"), filepath.Join(dir, "dst.jar")
		for _, path := range []string{src, dst} {
			if err := os.WriteFile(path, []byte("same-agent-version"), 0644); err != nil {
				t.Fatalf("write file: %v", err)
			}
		}

		srcHash, err := fileHash(src)
		if err != nil {
			t.Fatalf("fileHash failed: %v", err)
		}
		dstHash, err := fileHash(dst)
		if err != nil {
			t.Fatalf("fileHash failed: %v", err)
		}
		if srcHash != dstHash {
			t.Errorf("identical files hashed differently: %q != %q", srcHash, dstHash)
		}
	})

	t.Run("a missing file is an error", func(t *testing.T) {
		if _, err := fileHash(filepath.Join(t.TempDir(), "javaagent.jar")); err == nil {
			t.Fatal("expected an error for a missing file")
		}
	})
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
