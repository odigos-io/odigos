package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The odiglet re-syncs the agents directory on every start, including in-place upgrades, while
// instrumented processes are still running with agent files mapped or lazily open. The tests in
// this file pin the bookkeeping that decides which of those files survive the sync, because
// getting it wrong breaks instrumentation in every already-running pod until it restarts.

const (
	// When this variable is set, the test binary acts as a stand-in for rsync instead of running
	// tests: it records the arguments it was called with and exits. This is the self re-exec
	// pattern used by the os/exec tests, and it lets the tests assert the exact rsync invocation
	// without requiring rsync to be installed.
	fakeRsyncArgsFileEnv = "ODIGOS_TEST_FAKE_RSYNC_ARGS_FILE"
	fakeRsyncExitCodeEnv = "ODIGOS_TEST_FAKE_RSYNC_EXIT_CODE"
)

func TestMain(m *testing.M) {
	if argsFile := os.Getenv(fakeRsyncArgsFileEnv); argsFile != "" {
		os.Exit(runFakeRsync(argsFile))
	}
	os.Exit(m.Run())
}

func runFakeRsync(argsFile string) int {
	if err := os.WriteFile(argsFile, []byte(strings.Join(os.Args[1:], "\n")), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "fake rsync: %v\n", err)
		return 2
	}
	exitCode, err := strconv.Atoi(os.Getenv(fakeRsyncExitCodeEnv))
	if err != nil {
		return 0
	}
	return exitCode
}

// useFakeRsync makes the test binary itself stand in for rsync. It returns the path to pass as the
// rsync binary and a function that reports the arguments rsync was invoked with, or false when it
// was never invoked.
func useFakeRsync(t *testing.T, exitCode int) (string, func() ([]string, bool)) {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to locate the test binary: %v", err)
	}
	argsFile := filepath.Join(t.TempDir(), "rsync-args")
	t.Setenv(fakeRsyncArgsFileEnv, argsFile)
	t.Setenv(fakeRsyncExitCodeEnv, strconv.Itoa(exitCode))

	return self, func() ([]string, bool) {
		raw, err := os.ReadFile(argsFile)
		if os.IsNotExist(err) {
			return nil, false
		}
		if err != nil {
			t.Fatalf("failed to read the recorded rsync arguments: %v", err)
		}
		return strings.Split(string(raw), "\n"), true
	}
}

func writeAgentFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

func readAgentFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}
	return string(content)
}

// keptPaths returns the keep map as a sorted slice of paths relative to dstDir, which is both
// stable to compare and readable when a test fails.
func keptPaths(t *testing.T, keep map[string]struct{}, dstDir string) []string {
	t.Helper()
	relative := make([]string, 0, len(keep))
	for path := range keep {
		rel, err := filepath.Rel(dstDir, path)
		if err != nil {
			t.Fatalf("kept path %q is not under the destination directory: %v", path, err)
		}
		relative = append(relative, rel)
	}
	slices.Sort(relative)
	return relative
}

func keeplistLines(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read keeplist %s: %v", path, err)
	}
	var lines []string
	for _, line := range strings.Split(string(raw), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	slices.Sort(lines)
	return lines
}

// criticalAgentFiles lists the agent files a running process may have mapped, dlopen'd or lazily
// open, relative to the agents directory root. Dropping an entry from getCriticalFiles breaks
// instrumentation in every already-running pod that uses that agent as soon as the file changes
// (RUN-1061), so the set is asserted exactly: changes in either direction should be deliberate.
var criticalAgentFiles = []string{
	"java-ebpf/tracing_probes.so",
	"java-ext-ebpf/end_span_usdt.so",
	"java-ext-ebpf/javaagent.jar",
	"java-ext-ebpf/otel_agent_extension.jar",
	"java/javaagent.jar",
	"loader/loader.so",
	"nodejs-ebpf/build/Release/.deps/Release/dtrace-injector-native.node.d",
	"nodejs-ebpf/build/Release/.deps/Release/obj.target/dtrace-injector-native.node.d",
	"nodejs-ebpf/build/Release/dtrace-injector-native.node",
	"nodejs-ebpf/build/Release/obj.target/dtrace-injector-native.node",
	"php/8.1/opentelemetry.so",
	"php/8.2/opentelemetry.so",
	"php/8.3/opentelemetry.so",
	"php/8.4/opentelemetry.so",
	"python-ebpf/pythonUSDT.abi3.so",
	"python/google/_upb/_message.abi3.so",
	"python/wrapt/_wrappers.cpython-311-aarch64-linux-gnu.so",
	"python/wrapt/_wrappers.cpython-311-x86_64-linux-gnu.so",
	"python3.8/google/_upb/_message.abi3.so",
	"python3.8/wrapt/_wrappers.cpython-311-aarch64-linux-gnu.so",
	"python3.8/wrapt/_wrappers.cpython-311-x86_64-linux-gnu.so",
}

func TestGetCriticalFiles(t *testing.T) {
	const root = "/var/odigos"
	got := getCriticalFiles(root)

	for _, rel := range criticalAgentFiles {
		if _, ok := got[filepath.Join(root, rel)]; !ok {
			t.Errorf("%q is not preserved on upgrade, so processes already using it break when it changes", rel)
		}
	}

	expected := make(map[string]struct{}, len(criticalAgentFiles))
	for _, rel := range criticalAgentFiles {
		expected[filepath.Join(root, rel)] = struct{}{}
	}
	for path := range got {
		if _, ok := expected[path]; !ok {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				rel = path
			}
			t.Errorf("%q is preserved on upgrade but is not listed in criticalAgentFiles; add it there if that is intended", rel)
		}
	}
}

func TestGetCriticalFilesAreRootedAtTheAgentsDirectory(t *testing.T) {
	// The source path of each critical file is derived by replacing the destination root with the
	// source root, so a path that is not under the given root would compare the wrong two files.
	root := t.TempDir()
	for path := range getCriticalFiles(root) {
		if !strings.HasPrefix(path, root+string(filepath.Separator)) {
			t.Errorf("critical file %q is not rooted at the agents directory %q", path, root)
		}
	}
}

func TestIsDirEmptyOrNotExist(t *testing.T) {
	// A true result sends the odiglet down the fresh-install path, which overwrites the
	// destination without preserving anything.
	tests := []struct {
		name  string
		setup func(t *testing.T) string
		want  bool
	}{
		{
			name: "missing directory",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "does-not-exist")
			},
			want: true,
		},
		{
			name: "empty directory",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			want: true,
		},
		{
			name: "directory holding an agent file",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeAgentFile(t, filepath.Join(dir, "java", "javaagent.jar"), "v1")
				return dir
			},
			want: false,
		},
		{
			name: "directory holding only an empty subdirectory",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.MkdirAll(filepath.Join(dir, "java"), 0o755); err != nil {
					t.Fatalf("failed to create subdirectory: %v", err)
				}
				return dir
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isDirEmptyOrNotExist(tt.setup(t))
			if err != nil {
				t.Fatalf("isDirEmptyOrNotExist failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("isDirEmptyOrNotExist = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsDirEmptyOrNotExistRejectsARegularFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "odigos")
	writeAgentFile(t, path, "not a directory")

	empty, err := isDirEmptyOrNotExist(path)
	if empty {
		t.Error("a regular file must not be reported as an empty agents directory")
	}
	// Asserted exactly: without the explicit check the function still fails, but with a readdir
	// error that does not say the host path is occupied by a file.
	if want := "not a directory: " + path; err == nil || err.Error() != want {
		t.Errorf("error = %v, want %q", err, want)
	}
}

func TestWriteKeeplist(t *testing.T) {
	const dstDir = "/var/odigos"

	t.Run("host paths become patterns relative to the destination root", func(t *testing.T) {
		// rsync matches --exclude-from patterns against paths relative to the transfer root, so an
		// absolute path matches nothing and the --delete pass removes the file it was meant to keep.
		keeplist := filepath.Join(t.TempDir(), "keeplist")
		keeps := map[string]struct{}{
			filepath.Join(dstDir, "java", "javaagent_hash_version-9166e887cfbf.jar"): {},
			filepath.Join(dstDir, "loader", "loader.so"):                             {},
			filepath.Join(dstDir, "python", "google", "_upb", "_message.abi3.so"):    {},
		}

		if err := writeKeeplist(dstDir, keeplist, keeps); err != nil {
			t.Fatalf("writeKeeplist failed: %v", err)
		}

		want := []string{
			"java/javaagent_hash_version-9166e887cfbf.jar",
			"loader/loader.so",
			"python/google/_upb/_message.abi3.so",
		}
		if got := keeplistLines(t, keeplist); !slices.Equal(got, want) {
			t.Errorf("keeplist patterns = %v, want %v", got, want)
		}
	})

	t.Run("an empty keep map still produces a readable file", func(t *testing.T) {
		// rsync is always invoked with --exclude-from, so the file has to exist even when there is
		// nothing to keep.
		keeplist := filepath.Join(t.TempDir(), "keeplist")
		if err := writeKeeplist(dstDir, keeplist, nil); err != nil {
			t.Fatalf("writeKeeplist failed: %v", err)
		}
		if lines := keeplistLines(t, keeplist); len(lines) != 0 {
			t.Errorf("expected an empty keeplist, got %v", lines)
		}
	})

	t.Run("a keeplist left over from a previous sync is replaced", func(t *testing.T) {
		keeplist := filepath.Join(t.TempDir(), "keeplist")
		stale := map[string]struct{}{
			filepath.Join(dstDir, "loader", "loader.so"):            {},
			filepath.Join(dstDir, "java-ebpf", "tracing_probes.so"): {},
			filepath.Join(dstDir, "php", "8.1", "opentelemetry.so"): {},
		}
		if err := writeKeeplist(dstDir, keeplist, stale); err != nil {
			t.Fatalf("writeKeeplist failed: %v", err)
		}

		if err := writeKeeplist(dstDir, keeplist, map[string]struct{}{
			filepath.Join(dstDir, "loader", "loader.so"): {},
		}); err != nil {
			t.Fatalf("writeKeeplist failed: %v", err)
		}

		want := []string{"loader/loader.so"}
		if got := keeplistLines(t, keeplist); !slices.Equal(got, want) {
			t.Errorf("keeplist patterns = %v, want %v", got, want)
		}
	})

	t.Run("an unwritable keeplist path is an error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing-dir", "keeplist")
		if err := writeKeeplist(dstDir, path, nil); err == nil {
			t.Fatal("expected an error when the keeplist cannot be created")
		}
	})
}

func TestRemoveChangedFilesFromKeepMap(t *testing.T) {
	// sha256("old-javaagent-v1") truncated to 12 characters, the suffix the previous jar is
	// renamed to. Hardcoded so a change to the hash function or to the suffix length is caught.
	const oldJarHashSuffix = "9166e887cfbf"
	const relJarPath = "java/javaagent.jar"

	t.Run("an unchanged file stays at its canonical path", func(t *testing.T) {
		srcDir, dstDir := t.TempDir(), t.TempDir()
		writeAgentFile(t, filepath.Join(srcDir, relJarPath), "v1")
		writeAgentFile(t, filepath.Join(dstDir, relJarPath), "v1")

		keep, err := removeChangedFilesFromKeepMap(map[string]struct{}{filepath.Join(dstDir, relJarPath): {}}, srcDir, dstDir)
		if err != nil {
			t.Fatalf("removeChangedFilesFromKeepMap failed: %v", err)
		}

		if got, want := keptPaths(t, keep, dstDir), []string{relJarPath}; !slices.Equal(got, want) {
			t.Fatalf("kept %v, want %v", got, want)
		}
		if content := readAgentFile(t, filepath.Join(dstDir, relJarPath)); content != "v1" {
			t.Errorf("destination content = %q, want %q", content, "v1")
		}
		copies, err := filepath.Glob(filepath.Join(dstDir, "java", "*_hash_version-*"))
		if err != nil {
			t.Fatalf("glob failed: %v", err)
		}
		if len(copies) != 0 {
			t.Errorf("an unchanged file must not be copied aside, found %v", copies)
		}
	})

	t.Run("a changed file is preserved under a hash suffixed name", func(t *testing.T) {
		srcDir, dstDir := t.TempDir(), t.TempDir()
		writeAgentFile(t, filepath.Join(srcDir, relJarPath), "new-javaagent-v2")
		writeAgentFile(t, filepath.Join(dstDir, relJarPath), "old-javaagent-v1")

		keep, err := removeChangedFilesFromKeepMap(map[string]struct{}{filepath.Join(dstDir, relJarPath): {}}, srcDir, dstDir)
		if err != nil {
			t.Fatalf("removeChangedFilesFromKeepMap failed: %v", err)
		}

		preservedRel := "java/javaagent_hash_version-" + oldJarHashSuffix + ".jar"
		if got, want := keptPaths(t, keep, dstDir), []string{preservedRel}; !slices.Equal(got, want) {
			t.Fatalf("kept %v, want %v", got, want)
		}
		// The rename keeps the inode the running JVM holds open, and rsync recreates the canonical
		// path for pods that start after the upgrade.
		if content := readAgentFile(t, filepath.Join(dstDir, preservedRel)); content != "old-javaagent-v1" {
			t.Errorf("preserved copy holds %q, want the pre-upgrade content %q", content, "old-javaagent-v1")
		}
		if _, err := os.Stat(filepath.Join(dstDir, relJarPath)); !os.IsNotExist(err) {
			t.Errorf("the canonical path must be free for rsync to write the new agent, stat error: %v", err)
		}
		if content := readAgentFile(t, filepath.Join(srcDir, relJarPath)); content != "new-javaagent-v2" {
			t.Errorf("source content = %q, want it untouched", content)
		}
	})

	t.Run("versions preserved by earlier upgrades are kept as well", func(t *testing.T) {
		srcDir, dstDir := t.TempDir(), t.TempDir()
		writeAgentFile(t, filepath.Join(srcDir, relJarPath), "v3")
		writeAgentFile(t, filepath.Join(dstDir, relJarPath), "v2")
		// left behind by the upgrade that replaced v1 with v2
		writeAgentFile(t, filepath.Join(dstDir, "java", "javaagent_hash_version-3bfc269594ef.jar"), "v1")

		keep, err := removeChangedFilesFromKeepMap(map[string]struct{}{filepath.Join(dstDir, relJarPath): {}}, srcDir, dstDir)
		if err != nil {
			t.Fatalf("removeChangedFilesFromKeepMap failed: %v", err)
		}

		want := []string{
			"java/javaagent_hash_version-3bfc269594ef.jar",
			"java/javaagent_hash_version-fb04dcb6970e.jar",
		}
		if got := keptPaths(t, keep, dstDir); !slices.Equal(got, want) {
			t.Fatalf("kept %v, want %v", got, want)
		}
	})

	t.Run("earlier versions are kept when the current file is unchanged", func(t *testing.T) {
		srcDir, dstDir := t.TempDir(), t.TempDir()
		writeAgentFile(t, filepath.Join(srcDir, relJarPath), "v2")
		writeAgentFile(t, filepath.Join(dstDir, relJarPath), "v2")
		writeAgentFile(t, filepath.Join(dstDir, "java", "javaagent_hash_version-3bfc269594ef.jar"), "v1")

		keep, err := removeChangedFilesFromKeepMap(map[string]struct{}{filepath.Join(dstDir, relJarPath): {}}, srcDir, dstDir)
		if err != nil {
			t.Fatalf("removeChangedFilesFromKeepMap failed: %v", err)
		}

		want := []string{relJarPath, "java/javaagent_hash_version-3bfc269594ef.jar"}
		if got := keptPaths(t, keep, dstDir); !slices.Equal(got, want) {
			t.Fatalf("kept %v, want %v", got, want)
		}
	})

	t.Run("earlier versions are kept when the new release dropped the agent", func(t *testing.T) {
		// A process started before the upgrade still has the old file mapped, so it must survive
		// even though neither the new release nor the host has the canonical file any more.
		srcDir, dstDir := t.TempDir(), t.TempDir()
		writeAgentFile(t, filepath.Join(dstDir, "java", "javaagent_hash_version-3bfc269594ef.jar"), "v1")

		keep, err := removeChangedFilesFromKeepMap(map[string]struct{}{filepath.Join(dstDir, relJarPath): {}}, srcDir, dstDir)
		if err != nil {
			t.Fatalf("removeChangedFilesFromKeepMap failed: %v", err)
		}

		want := []string{"java/javaagent_hash_version-3bfc269594ef.jar"}
		if got := keptPaths(t, keep, dstDir); !slices.Equal(got, want) {
			t.Fatalf("kept %v, want %v", got, want)
		}
	})

	t.Run("unrelated files in the same directory are not kept", func(t *testing.T) {
		srcDir, dstDir := t.TempDir(), t.TempDir()
		writeAgentFile(t, filepath.Join(srcDir, relJarPath), "v1")
		writeAgentFile(t, filepath.Join(dstDir, relJarPath), "v1")
		writeAgentFile(t, filepath.Join(dstDir, "java", "other.jar"), "other")
		writeAgentFile(t, filepath.Join(dstDir, "java", "javaagent.jar.bak"), "backup")
		writeAgentFile(t, filepath.Join(dstDir, "java", "javaagent_hash_version-3bfc269594ef.so"), "wrong extension")
		writeAgentFile(t, filepath.Join(dstDir, "java", "javaagent-hash_version-3bfc269594ef.jar"), "wrong separator")

		keep, err := removeChangedFilesFromKeepMap(map[string]struct{}{filepath.Join(dstDir, relJarPath): {}}, srcDir, dstDir)
		if err != nil {
			t.Fatalf("removeChangedFilesFromKeepMap failed: %v", err)
		}

		if got, want := keptPaths(t, keep, dstDir), []string{relJarPath}; !slices.Equal(got, want) {
			t.Fatalf("kept %v, want %v", got, want)
		}
	})

	t.Run("a file dropped by the new release is not kept", func(t *testing.T) {
		srcDir, dstDir := t.TempDir(), t.TempDir()
		writeAgentFile(t, filepath.Join(dstDir, relJarPath), "v1")

		keep, err := removeChangedFilesFromKeepMap(map[string]struct{}{filepath.Join(dstDir, relJarPath): {}}, srcDir, dstDir)
		if err != nil {
			t.Fatalf("removeChangedFilesFromKeepMap failed: %v", err)
		}

		if got := keptPaths(t, keep, dstDir); len(got) != 0 {
			t.Fatalf("kept %v, want nothing so rsync can delete the stale file", got)
		}
		if content := readAgentFile(t, filepath.Join(dstDir, relJarPath)); content != "v1" {
			t.Errorf("destination content = %q, want it left in place for rsync to delete", content)
		}
	})

	t.Run("a file added by the new release is not kept", func(t *testing.T) {
		srcDir, dstDir := t.TempDir(), t.TempDir()
		writeAgentFile(t, filepath.Join(srcDir, relJarPath), "v1")

		keep, err := removeChangedFilesFromKeepMap(map[string]struct{}{filepath.Join(dstDir, relJarPath): {}}, srcDir, dstDir)
		if err != nil {
			t.Fatalf("removeChangedFilesFromKeepMap failed: %v", err)
		}

		if got := keptPaths(t, keep, dstDir); len(got) != 0 {
			t.Fatalf("kept %v, want nothing so rsync can create the new file", got)
		}
	})

	t.Run("an empty keep map keeps nothing", func(t *testing.T) {
		srcDir, dstDir := t.TempDir(), t.TempDir()
		keep, err := removeChangedFilesFromKeepMap(map[string]struct{}{}, srcDir, dstDir)
		if err != nil {
			t.Fatalf("removeChangedFilesFromKeepMap failed: %v", err)
		}
		if len(keep) != 0 {
			t.Fatalf("kept %v, want nothing", keptPaths(t, keep, dstDir))
		}
	})

	t.Run("a destination file that cannot be hashed aborts the sync", func(t *testing.T) {
		// Continuing would hand rsync an incomplete keeplist and let it overwrite files that are
		// still in use.
		srcDir, dstDir := t.TempDir(), t.TempDir()
		writeAgentFile(t, filepath.Join(srcDir, relJarPath), "v1")
		if err := os.MkdirAll(filepath.Join(dstDir, relJarPath), 0o755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}

		keep, err := removeChangedFilesFromKeepMap(map[string]struct{}{filepath.Join(dstDir, relJarPath): {}}, srcDir, dstDir)
		if err == nil {
			t.Fatalf("expected an error, kept %v", keptPaths(t, keep, dstDir))
		}
		if keep != nil {
			t.Errorf("keep map = %v, want nil on error", keptPaths(t, keep, dstDir))
		}
		if !strings.Contains(err.Error(), "destination") {
			t.Errorf("error %q should name the destination file", err)
		}
	})

	t.Run("a source file that cannot be hashed aborts the sync", func(t *testing.T) {
		srcDir, dstDir := t.TempDir(), t.TempDir()
		if err := os.MkdirAll(filepath.Join(srcDir, relJarPath), 0o755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		writeAgentFile(t, filepath.Join(dstDir, relJarPath), "v1")

		keep, err := removeChangedFilesFromKeepMap(map[string]struct{}{filepath.Join(dstDir, relJarPath): {}}, srcDir, dstDir)
		if err == nil {
			t.Fatalf("expected an error, kept %v", keptPaths(t, keep, dstDir))
		}
		if !strings.Contains(err.Error(), "source") {
			t.Errorf("error %q should name the source file", err)
		}
	})

	t.Run("consecutive upgrades keep one copy per agent version", func(t *testing.T) {
		srcDir, dstDir := t.TempDir(), t.TempDir()
		dstJar := filepath.Join(dstDir, relJarPath)
		writeAgentFile(t, dstJar, "v1")

		// v1 -> v2 -> v3, with the hash suffix of the version being replaced on each round.
		upgrades := []struct{ version, previousSuffix string }{
			{"v2", "3bfc269594ef"},
			{"v3", "fb04dcb6970e"},
		}
		var preserved []string
		for _, upgrade := range upgrades {
			writeAgentFile(t, filepath.Join(srcDir, relJarPath), upgrade.version)

			keep, err := removeChangedFilesFromKeepMap(map[string]struct{}{dstJar: {}}, srcDir, dstDir)
			if err != nil {
				t.Fatalf("upgrade to %s failed: %v", upgrade.version, err)
			}

			preserved = append(preserved, "java/javaagent_hash_version-"+upgrade.previousSuffix+".jar")
			want := slices.Clone(preserved)
			slices.Sort(want)
			if got := keptPaths(t, keep, dstDir); !slices.Equal(got, want) {
				t.Fatalf("after upgrading to %s kept %v, want %v", upgrade.version, got, want)
			}

			// rsync writes the new agent at the canonical path once the old one is out of the way.
			writeAgentFile(t, dstJar, upgrade.version)
		}

		if content := readAgentFile(t, filepath.Join(dstDir, "java", "javaagent_hash_version-3bfc269594ef.jar")); content != "v1" {
			t.Errorf("the v1 copy holds %q, want %q", content, "v1")
		}
		if content := readAgentFile(t, filepath.Join(dstDir, "java", "javaagent_hash_version-fb04dcb6970e.jar")); content != "v2" {
			t.Errorf("the v2 copy holds %q, want %q", content, "v2")
		}
	})
}

func TestRunSingleRsyncSync(t *testing.T) {
	t.Run("syncs the directory contents while excluding the keeplist", func(t *testing.T) {
		srcDir, dstDir := t.TempDir(), t.TempDir()
		excludeFile := filepath.Join(t.TempDir(), "keeplist")
		writeAgentFile(t, excludeFile, "java/javaagent_hash_version-9166e887cfbf.jar\n")
		rsyncPath, invocation := useFakeRsync(t, 0)

		if err := runSingleRsyncSync(srcDir, dstDir, excludeFile, &rsyncPath); err != nil {
			t.Fatalf("runSingleRsyncSync failed: %v", err)
		}

		args, invoked := invocation()
		if !invoked {
			t.Fatal("rsync was not invoked")
		}
		// --delete is what makes the keeplist load bearing, --inplace is what makes preserving the
		// previous copy necessary, and the trailing slashes are what copy the contents of srcDir
		// into dstDir instead of nesting it.
		want := []string{
			"-av", "--numeric-ids", "--delete", "--whole-file", "--inplace",
			"--exclude-from=" + excludeFile,
			srcDir + "/", dstDir + "/",
		}
		if !slices.Equal(args, want) {
			t.Errorf("rsync args =\n%v\nwant\n%v", args, want)
		}
	})

	t.Run("a failing rsync is reported", func(t *testing.T) {
		srcDir, dstDir := t.TempDir(), t.TempDir()
		rsyncPath, _ := useFakeRsync(t, 1)

		if err := runSingleRsyncSync(srcDir, dstDir, filepath.Join(t.TempDir(), "keeplist"), &rsyncPath); err == nil {
			t.Fatal("expected an error when rsync exits non-zero")
		}
	})

	t.Run("without an explicit path rsync is looked up in PATH", func(t *testing.T) {
		srcDir, dstDir := t.TempDir(), t.TempDir()
		rsyncPath, invocation := useFakeRsync(t, 0)

		binDir := t.TempDir()
		if err := os.Symlink(rsyncPath, filepath.Join(binDir, "rsync")); err != nil {
			t.Fatalf("failed to link the fake rsync into PATH: %v", err)
		}
		t.Setenv("PATH", binDir)

		if err := runSingleRsyncSync(srcDir, dstDir, filepath.Join(t.TempDir(), "keeplist"), nil); err != nil {
			t.Fatalf("runSingleRsyncSync failed: %v", err)
		}
		if _, invoked := invocation(); !invoked {
			t.Error("rsync was not looked up in PATH")
		}
	})
}

func TestCopyAgentsDirectoryToHost(t *testing.T) {
	t.Run("a fresh install copies the agents directory without rsync", func(t *testing.T) {
		srcDir, dstDir := t.TempDir(), t.TempDir()
		writeAgentFile(t, filepath.Join(srcDir, "java", "javaagent.jar"), "new-javaagent-v2")
		writeAgentFile(t, filepath.Join(srcDir, "loader", "loader.so"), "loader")
		rsyncPath, invocation := useFakeRsync(t, 0)

		if err := CopyAgentsDirectoryToHost(srcDir, dstDir, &rsyncPath); err != nil {
			t.Fatalf("CopyAgentsDirectoryToHost failed: %v", err)
		}

		if _, invoked := invocation(); invoked {
			t.Error("rsync must not be used when there is nothing on the host to preserve")
		}
		if content := readAgentFile(t, filepath.Join(dstDir, "java", "javaagent.jar")); content != "new-javaagent-v2" {
			t.Errorf("copied jar holds %q, want %q", content, "new-javaagent-v2")
		}
		if content := readAgentFile(t, filepath.Join(dstDir, "loader", "loader.so")); content != "loader" {
			t.Errorf("copied loader holds %q, want %q", content, "loader")
		}
	})

	t.Run("a missing destination is a fresh install", func(t *testing.T) {
		srcDir := t.TempDir()
		dstDir := filepath.Join(t.TempDir(), "odigos")
		writeAgentFile(t, filepath.Join(srcDir, "java", "javaagent.jar"), "new-javaagent-v2")
		rsyncPath, invocation := useFakeRsync(t, 0)

		if err := CopyAgentsDirectoryToHost(srcDir, dstDir, &rsyncPath); err != nil {
			t.Fatalf("CopyAgentsDirectoryToHost failed: %v", err)
		}

		if _, invoked := invocation(); invoked {
			t.Error("rsync must not be used when the destination does not exist yet")
		}
		if content := readAgentFile(t, filepath.Join(dstDir, "java", "javaagent.jar")); content != "new-javaagent-v2" {
			t.Errorf("copied jar holds %q, want %q", content, "new-javaagent-v2")
		}
	})

	t.Run("an upgrade preserves the previous java agent jar and excludes it from the sync", func(t *testing.T) {
		srcDir, dstDir := t.TempDir(), t.TempDir()
		writeAgentFile(t, filepath.Join(srcDir, "java", "javaagent.jar"), "new-javaagent-v2")
		writeAgentFile(t, filepath.Join(dstDir, "java", "javaagent.jar"), "old-javaagent-v1")
		rsyncPath, invocation := useFakeRsync(t, 0)

		if err := CopyAgentsDirectoryToHost(srcDir, dstDir, &rsyncPath); err != nil {
			t.Fatalf("CopyAgentsDirectoryToHost failed: %v", err)
		}

		args, invoked := invocation()
		if !invoked {
			t.Fatal("rsync was not invoked for a populated destination")
		}
		if !slices.Contains(args, "--exclude-from=/tmp/keeplist") {
			t.Fatalf("rsync args %v do not exclude the keeplist", args)
		}

		want := []string{"java/javaagent_hash_version-9166e887cfbf.jar"}
		if got := keeplistLines(t, "/tmp/keeplist"); !slices.Equal(got, want) {
			t.Errorf("keeplist = %v, want %v", got, want)
		}
		preserved := filepath.Join(dstDir, "java", "javaagent_hash_version-9166e887cfbf.jar")
		if content := readAgentFile(t, preserved); content != "old-javaagent-v1" {
			t.Errorf("preserved jar holds %q, want the pre-upgrade content %q", content, "old-javaagent-v1")
		}
	})

	t.Run("a failing rsync fails the sync", func(t *testing.T) {
		srcDir, dstDir := t.TempDir(), t.TempDir()
		writeAgentFile(t, filepath.Join(srcDir, "java", "javaagent.jar"), "new-javaagent-v2")
		writeAgentFile(t, filepath.Join(dstDir, "java", "javaagent.jar"), "old-javaagent-v1")
		rsyncPath, _ := useFakeRsync(t, 1)

		if err := CopyAgentsDirectoryToHost(srcDir, dstDir, &rsyncPath); err == nil {
			t.Fatal("expected an error when rsync fails")
		}
	})

	t.Run("a destination that is a regular file is reported", func(t *testing.T) {
		srcDir := t.TempDir()
		dstDir := filepath.Join(t.TempDir(), "odigos")
		writeAgentFile(t, dstDir, "not a directory")

		err := CopyAgentsDirectoryToHost(srcDir, dstDir, nil)
		if err == nil {
			t.Fatal("expected an error when the destination is not a directory")
		}
		// Reported while inspecting the destination rather than surfacing later as a copy failure,
		// so the odiglet log points at the occupied host path.
		if !strings.Contains(err.Error(), "failed to inspect destination") {
			t.Errorf("error = %v, want it to report the destination inspection failure", err)
		}
	})
}
