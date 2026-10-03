package fs

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// mappedAgentFiles are agent files that a running application process keeps mapped for its whole
// lifetime, because the runtime loads them at startup: via dlopen (the PHP and .NET native
// extensions), -javaagent, or an interpreter importing a native module. rsync runs with
// --inplace, so rewriting one of these on an upgrade replaces the bytes of the inode the live
// process is executing from. Every one of them must therefore land on a NEW inode, with the old
// inode preserved under a hash-suffixed name for the processes still using it.
var mappedAgentFiles = []string{
	// dlopen'ed by CoreCLR through CORECLR_PROFILER_PATH, see distros/yamls/dotnet-community.yaml
	"dotnet/linux-glibc/OpenTelemetry.AutoInstrumentation.Native.so",
	"dotnet/linux-musl/OpenTelemetry.AutoInstrumentation.Native.so",
	// dlopen'ed by the PHP runtime
	"php/8.1/opentelemetry.so",
	"php/8.4/opentelemetry.so",
	// read lazily by the JVM through -javaagent
	"java/javaagent.jar",
	// imported as native modules by the Python interpreter
	"python/google/_upb/_message.abi3.so",
	// LD_PRELOAD'ed loader
	"loader/loader.so",
}

func TestCopyAgentsDirectoryToHost_MappedAgentFilesGetNewInodeOnUpgrade(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// A previous odigos version already populated the host directory.
	for _, rel := range mappedAgentFiles {
		writeAgentFile(t, filepath.Join(srcDir, rel), "v1")
		writeAgentFile(t, filepath.Join(dstDir, rel), "v1")
	}

	oldInodes := make(map[string]uint64, len(mappedAgentFiles))
	for _, rel := range mappedAgentFiles {
		oldInodes[rel] = inodeOf(t, filepath.Join(dstDir, rel))
	}

	// The upgrade ships new content for every agent.
	for _, rel := range mappedAgentFiles {
		writeAgentFile(t, filepath.Join(srcDir, rel), "v2-which-is-longer")
	}

	if err := CopyAgentsDirectoryToHost(srcDir, dstDir, nil); err != nil {
		t.Fatalf("CopyAgentsDirectoryToHost failed: %v", err)
	}

	for _, rel := range mappedAgentFiles {
		t.Run(rel, func(t *testing.T) {
			dstPath := filepath.Join(dstDir, rel)

			if got := readAgentFile(t, dstPath); got != "v2-which-is-longer" {
				t.Fatalf("new pods would get stale content at %s: got %q", rel, got)
			}

			if inodeOf(t, dstPath) == oldInodes[rel] {
				t.Fatalf("%s was rewritten in place (inode %d unchanged): a process that has it mapped now executes the new bytes", rel, oldInodes[rel])
			}

			preserved := findPreservedCopy(t, dstPath, oldInodes[rel])
			if preserved == "" {
				t.Fatalf("the previous version of %s was not preserved under a hash-suffixed name", rel)
			}
			if got := readAgentFile(t, preserved); got != "v1" {
				t.Fatalf("preserved copy %s has content %q, want the old version", preserved, got)
			}
		})
	}
}

func writeAgentFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readAgentFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func inodeOf(t *testing.T, path string) uint64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("stat %s: no inode information", path)
	}
	return uint64(stat.Ino)
}

// findPreservedCopy looks for the hash-suffixed sibling that holds the given inode.
func findPreservedCopy(t *testing.T, dstPath string, inode uint64) string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(dstPath))
	if err != nil {
		t.Fatalf("read dir %s: %v", filepath.Dir(dstPath), err)
	}
	for _, entry := range entries {
		if !strings.Contains(entry.Name(), "_hash_version-") {
			continue
		}
		candidate := filepath.Join(filepath.Dir(dstPath), entry.Name())
		if inodeOf(t, candidate) == inode {
			return candidate
		}
	}
	return ""
}
