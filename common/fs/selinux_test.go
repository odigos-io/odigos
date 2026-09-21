package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

// The literals below are pinned here instead of being read from the production
// constants, so that renaming one of them fails a test rather than silently
// leaving the agent files unreadable by instrumented containers:
//   - the attribute name is the kernel's, SELinux stores a file's context in it
//   - var_t is what RHEL policy gives files under /var
//   - container_ro_file_t is what container-selinux lets any container read
const (
	selinuxKernelAttr = "security.selinux"
	selinuxVarType    = "var_t"
	selinuxAgentsType = "container_ro_file_t"
)

// selinuxSkipWithoutWritableLabels skips a test on a host where the fixtures
// cannot express "this file has exactly this context": either the host labels
// new files itself (a real SELinux node, where the expected contexts below are
// not what the policy would produce) or it rejects the attribute outright.
func selinuxSkipWithoutWritableLabels(t *testing.T) {
	t.Helper()

	probe := filepath.Join(t.TempDir(), "probe")
	require.NoError(t, os.WriteFile(probe, []byte("probe"), 0o600))

	if current := selinuxRawContext(t, probe); current != "" {
		t.Skipf("host labels new files itself (%q); the fixtures cannot control contexts here", current)
	}

	want := "system_u:object_r:" + selinuxVarType + ":s0"
	if err := unix.Lsetxattr(probe, selinuxKernelAttr, []byte(want), 0); err != nil {
		t.Skipf("host rejects writes to %s: %v", selinuxKernelAttr, err)
	}
	require.Equal(t, want, selinuxRawContext(t, probe), "the fixture must be able to stamp an exact context")
}

func selinuxSetContext(t *testing.T, path, context string) {
	t.Helper()
	require.NoError(t, unix.Lsetxattr(path, selinuxKernelAttr, []byte(context), 0))
}

// selinuxRawContext reads the attribute back through the kernel without going
// through fileContext, so an assertion cannot be fooled by a change to the
// production reader. NUL bytes are returned as stored.
func selinuxRawContext(t *testing.T, path string) string {
	t.Helper()

	buf := make([]byte, 4096)
	n, err := unix.Lgetxattr(path, selinuxKernelAttr, buf)
	if err == unix.ENODATA || err == unix.EOPNOTSUPP {
		return ""
	}
	require.NoError(t, err)
	return string(buf[:n])
}

// selinuxEntry is one entry of a fixture agents directory: a path relative to
// the directory root, whether it is a directory, and the raw context to stamp on
// it. An empty context leaves the entry unlabeled.
type selinuxEntry struct {
	path    string
	dir     bool
	symlink bool
	context string
}

// selinuxTree builds an agents directory whose root carries rootContext and
// whose entries carry theirs, and returns the root path.
func selinuxTree(t *testing.T, rootContext string, entries ...selinuxEntry) string {
	t.Helper()

	root := filepath.Join(t.TempDir(), "odigos")
	require.NoError(t, os.MkdirAll(root, 0o755))

	for _, e := range entries {
		full := filepath.Join(root, e.path)
		switch {
		case e.dir:
			require.NoError(t, os.MkdirAll(full, 0o755))
		case e.symlink:
			require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
			require.NoError(t, os.Symlink("does-not-exist", full))
		default:
			require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
			require.NoError(t, os.WriteFile(full, []byte(e.path), 0o644))
		}
	}

	if rootContext != "" {
		selinuxSetContext(t, root, rootContext)
	}
	for _, e := range entries {
		if e.context != "" {
			selinuxSetContext(t, filepath.Join(root, e.path), e.context)
		}
	}
	return root
}

// selinuxContextsUnder maps every path under root, relative to root, to the
// context the kernel currently holds for it.
func selinuxContextsUnder(t *testing.T, root string) map[string]string {
	t.Helper()

	contexts := map[string]string{}
	require.NoError(t, filepath.WalkDir(root, func(path string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		contexts[rel] = selinuxRawContext(t, path)
		return nil
	}))
	return contexts
}

func TestContextType(t *testing.T) {
	for _, tc := range []struct {
		name    string
		context string
		want    string
	}{
		{name: "full context", context: "system_u:object_r:var_t:s0", want: "var_t"},
		{name: "no level", context: "system_u:object_r:var_t", want: "var_t"},
		{name: "level with a category range", context: "unconfined_u:object_r:var_t:s0:c0.c1023", want: "var_t"},
		{name: "user and role only", context: "system_u:object_r", want: ""},
		{name: "single field", context: "garbage", want: ""},
		{name: "unlabeled", context: "", want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, contextType(tc.context))
		})
	}
}

func TestFileContextReadsTheStoredContext(t *testing.T) {
	selinuxSkipWithoutWritableLabels(t)

	want := "unconfined_u:object_r:" + selinuxVarType + ":s0:c0.c1023"
	dir := selinuxTree(t, "", selinuxEntry{path: "loader.so", context: want})

	got, err := fileContext(filepath.Join(dir, "loader.so"))
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestFileContextTrimsTheTrailingNulTheKernelStores(t *testing.T) {
	selinuxSkipWithoutWritableLabels(t)

	// the kernel hands back a NUL-terminated context on most filesystems, and the
	// relabel writes whatever was read straight back, so a NUL left in place ends
	// up embedded in the context of every agent file
	dir := selinuxTree(t, "", selinuxEntry{path: "loader.so", context: "system_u:object_r:" + selinuxVarType + ":s0\x00"})

	got, err := fileContext(filepath.Join(dir, "loader.so"))
	require.NoError(t, err)
	assert.Equal(t, "system_u:object_r:"+selinuxVarType+":s0", got)
}

// selinuxLongLevel is a category list that pushes a context past the 128 byte
// buffer fileContext starts with.
func selinuxLongLevel() string {
	categories := make([]string, 0, 40)
	for i := range 40 {
		categories = append(categories, fmt.Sprintf("c%d", i))
	}
	return "s0:" + strings.Join(categories, ",")
}

func TestFileContextReadsAContextLongerThanTheFixedBuffer(t *testing.T) {
	selinuxSkipWithoutWritableLabels(t)

	want := "unconfined_u:object_r:" + selinuxVarType + ":" + selinuxLongLevel()
	require.Greater(t, len(want), 128, "the fixture must not fit the initial buffer")
	dir := selinuxTree(t, "", selinuxEntry{path: "loader.so", context: want})

	got, err := fileContext(filepath.Join(dir, "loader.so"))
	require.NoError(t, err)
	assert.Equal(t, want, got, "a truncated context would be written back as the file's new label")
}

func TestFileContextAFileWithoutAContextIsNotAnError(t *testing.T) {
	selinuxSkipWithoutWritableLabels(t)

	dir := selinuxTree(t, "", selinuxEntry{path: "loader.so"})

	got, err := fileContext(filepath.Join(dir, "loader.so"))
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestFileContextAFilesystemWithoutSELinuxLabelsIsNotAnError(t *testing.T) {
	// procfs supports no extended attributes at all, which the kernel reports as
	// ENOTSUP rather than "this file has no context"
	const unsupported = "/proc/self/status"
	if _, err := unix.Lgetxattr(unsupported, selinuxKernelAttr, make([]byte, 128)); err != unix.EOPNOTSUPP {
		t.Skipf("%s does not report ENOTSUP on this host: %v", unsupported, err)
	}

	got, err := fileContext(unsupported)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestFileContextAMissingPathIsAnError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent")

	got, err := fileContext(missing)
	require.Error(t, err)
	assert.Empty(t, got)
	assert.ErrorIs(t, err, os.ErrNotExist)
	assert.Contains(t, err.Error(), missing, "the operator needs to know which path could not be read")
}

func TestApplyOpenShiftSELinuxSettingsRelabelsOnlyTheTypeField(t *testing.T) {
	selinuxSkipWithoutWritableLabels(t)

	// a context is "user:role:type:level" and only the type decides who may read
	// the file, so replacing any other field produces a label the node rejects
	dir := selinuxTree(t,
		"system_u:object_r:"+selinuxVarType+":s0",
		selinuxEntry{path: "loader.so", context: "unconfined_u:object_r:" + selinuxVarType + ":s0:c0.c1023"},
	)

	relabeled, rootType, err := ApplyOpenShiftSELinuxSettings(dir)
	require.NoError(t, err)
	assert.Equal(t, selinuxVarType, rootType)
	assert.Equal(t, 2, relabeled)
	assert.Equal(t, "unconfined_u:object_r:"+selinuxAgentsType+":s0:c0.c1023",
		selinuxRawContext(t, filepath.Join(dir, "loader.so")))
}

func TestApplyOpenShiftSELinuxSettingsRelabelsEveryEntryAtEveryDepth(t *testing.T) {
	selinuxSkipWithoutWritableLabels(t)

	varContext := "system_u:object_r:" + selinuxVarType + ":s0"
	agentsContext := "system_u:object_r:" + selinuxAgentsType + ":s0"
	dir := selinuxTree(t, varContext,
		selinuxEntry{path: "loader", dir: true, context: varContext},
		selinuxEntry{path: "loader/loader.so", context: varContext},
		selinuxEntry{path: "java-ebpf", dir: true, context: varContext},
		selinuxEntry{path: "java-ebpf/build", dir: true, context: varContext},
		selinuxEntry{path: "java-ebpf/build/tracing_probes.so", context: varContext},
	)

	relabeled, rootType, err := ApplyOpenShiftSELinuxSettings(dir)
	require.NoError(t, err)
	assert.Equal(t, selinuxVarType, rootType)
	assert.Equal(t, 6, relabeled)
	assert.Equal(t, map[string]string{
		".":                                 agentsContext,
		"loader":                            agentsContext,
		"loader/loader.so":                  agentsContext,
		"java-ebpf":                         agentsContext,
		"java-ebpf/build":                   agentsContext,
		"java-ebpf/build/tracing_probes.so": agentsContext,
	}, selinuxContextsUnder(t, dir))
}

func TestApplyOpenShiftSELinuxSettingsRelabelsATransitionedSubtree(t *testing.T) {
	selinuxSkipWithoutWritableLabels(t)

	// a named type transition gives a directory such as "debug" a type of its own
	// that its contents inherit, so a subtree under the agents directory can hold
	// a type that is neither var_t nor the one we are about to write
	varContext := "system_u:object_r:" + selinuxVarType + ":s0"
	dir := selinuxTree(t, varContext,
		selinuxEntry{path: "debug", dir: true, context: "system_u:object_r:var_log_t:s0"},
		selinuxEntry{path: "debug/trace.log", context: "system_u:object_r:var_log_t:s0"},
	)

	relabeled, rootType, err := ApplyOpenShiftSELinuxSettings(dir)
	require.NoError(t, err)
	assert.Equal(t, selinuxVarType, rootType)
	assert.Equal(t, 3, relabeled)
	agentsContext := "system_u:object_r:" + selinuxAgentsType + ":s0"
	assert.Equal(t, agentsContext, selinuxRawContext(t, filepath.Join(dir, "debug")))
	assert.Equal(t, agentsContext, selinuxRawContext(t, filepath.Join(dir, "debug/trace.log")))
}

func TestApplyOpenShiftSELinuxSettingsRelabelsASymlinkWithoutFollowingIt(t *testing.T) {
	selinuxSkipWithoutWritableLabels(t)

	varContext := "system_u:object_r:" + selinuxVarType + ":s0"
	dir := selinuxTree(t, varContext,
		selinuxEntry{path: "libodigos.so", symlink: true, context: varContext},
	)

	relabeled, rootType, err := ApplyOpenShiftSELinuxSettings(dir)
	require.NoError(t, err)
	assert.Equal(t, selinuxVarType, rootType)
	assert.Equal(t, 2, relabeled, "a symlink to a missing target must not abort the relabel")
	assert.Equal(t, "system_u:object_r:"+selinuxAgentsType+":s0",
		selinuxRawContext(t, filepath.Join(dir, "libodigos.so")))
}

func TestApplyOpenShiftSELinuxSettingsPreservesALevelLongerThanTheFixedReadBuffer(t *testing.T) {
	selinuxSkipWithoutWritableLabels(t)

	level := selinuxLongLevel()
	dir := selinuxTree(t,
		"system_u:object_r:"+selinuxVarType+":s0",
		selinuxEntry{path: "loader.so", context: "unconfined_u:object_r:" + selinuxVarType + ":" + level},
	)

	relabeled, _, err := ApplyOpenShiftSELinuxSettings(dir)
	require.NoError(t, err)
	assert.Equal(t, 2, relabeled)
	assert.Equal(t, "unconfined_u:object_r:"+selinuxAgentsType+":"+level,
		selinuxRawContext(t, filepath.Join(dir, "loader.so")))
}

func TestApplyOpenShiftSELinuxSettingsOnANodeWithoutSELinuxTouchesNothing(t *testing.T) {
	selinuxSkipWithoutWritableLabels(t)

	// odiglet's init container exits the pod when this returns an error, so a node
	// with no SELinux at all has to be a silent no-op
	varContext := "system_u:object_r:" + selinuxVarType + ":s0"
	dir := selinuxTree(t, "", selinuxEntry{path: "loader.so", context: varContext})

	relabeled, rootType, err := ApplyOpenShiftSELinuxSettings(dir)
	require.NoError(t, err)
	assert.Zero(t, relabeled)
	assert.Empty(t, rootType)
	assert.Equal(t, varContext, selinuxRawContext(t, filepath.Join(dir, "loader.so")))
}

func TestApplyOpenShiftSELinuxSettingsLeavesAnUnexpectedlyLabeledDirectoryAlone(t *testing.T) {
	selinuxSkipWithoutWritableLabels(t)

	varContext := "system_u:object_r:" + selinuxVarType + ":s0"
	for _, rootType := range []string{"container_file_t", "unlabeled_t", "usr_t"} {
		t.Run(rootType, func(t *testing.T) {
			dir := selinuxTree(t,
				"system_u:object_r:"+rootType+":s0",
				selinuxEntry{path: "loader.so", context: varContext},
			)

			relabeled, gotRootType, err := ApplyOpenShiftSELinuxSettings(dir)
			require.NoError(t, err)
			assert.Zero(t, relabeled)
			assert.Equal(t, rootType, gotRootType, "the reported type is what explains a zero count")
			assert.Equal(t, varContext, selinuxRawContext(t, filepath.Join(dir, "loader.so")))
		})
	}
}

func TestApplyOpenShiftSELinuxSettingsFixesStaleFilesUnderAnAlreadyLabeledDirectory(t *testing.T) {
	selinuxSkipWithoutWritableLabels(t)

	// an upgrade finds the directory already labeled by a previous run, and only
	// the files that predate it still need fixing
	agentsContext := "system_u:object_r:" + selinuxAgentsType + ":s0"
	dir := selinuxTree(t, agentsContext,
		selinuxEntry{path: "loader.so", context: agentsContext},
		selinuxEntry{path: "stale.so", context: "system_u:object_r:" + selinuxVarType + ":s0"},
	)

	relabeled, rootType, err := ApplyOpenShiftSELinuxSettings(dir)
	require.NoError(t, err)
	assert.Equal(t, selinuxAgentsType, rootType)
	assert.Equal(t, 1, relabeled, "only the file that predates the previous run is relabeled")
	assert.Equal(t, agentsContext, selinuxRawContext(t, filepath.Join(dir, "stale.so")))
}

func TestApplyOpenShiftSELinuxSettingsIsIdempotent(t *testing.T) {
	selinuxSkipWithoutWritableLabels(t)

	varContext := "system_u:object_r:" + selinuxVarType + ":s0"
	dir := selinuxTree(t, varContext,
		selinuxEntry{path: "loader", dir: true, context: varContext},
		selinuxEntry{path: "loader/loader.so", context: varContext},
	)

	relabeled, rootType, err := ApplyOpenShiftSELinuxSettings(dir)
	require.NoError(t, err)
	assert.Equal(t, 3, relabeled)
	assert.Equal(t, selinuxVarType, rootType)
	before := selinuxContextsUnder(t, dir)

	relabeled, rootType, err = ApplyOpenShiftSELinuxSettings(dir)
	require.NoError(t, err)
	assert.Zero(t, relabeled)
	assert.Equal(t, selinuxAgentsType, rootType)
	assert.Equal(t, before, selinuxContextsUnder(t, dir))
}

func TestApplyOpenShiftSELinuxSettingsSkipsEntriesWithoutAContext(t *testing.T) {
	selinuxSkipWithoutWritableLabels(t)

	varContext := "system_u:object_r:" + selinuxVarType + ":s0"
	dir := selinuxTree(t, varContext,
		selinuxEntry{path: "unlabeled.so"},
		selinuxEntry{path: "loader.so", context: varContext},
	)

	relabeled, _, err := ApplyOpenShiftSELinuxSettings(dir)
	require.NoError(t, err)
	assert.Equal(t, 2, relabeled)
	assert.Empty(t, selinuxRawContext(t, filepath.Join(dir, "unlabeled.so")))
	assert.Equal(t, "system_u:object_r:"+selinuxAgentsType+":s0",
		selinuxRawContext(t, filepath.Join(dir, "loader.so")))
}

func TestApplyOpenShiftSELinuxSettingsSkipsAMalformedContextInsteadOfPanicking(t *testing.T) {
	selinuxSkipWithoutWritableLabels(t)

	varContext := "system_u:object_r:" + selinuxVarType + ":s0"
	// a context with no type field has nothing to replace; splicing one in anyway
	// would index past the end of the fields and take down odiglet's init container
	for _, malformed := range []string{"garbage", "system_u:object_r"} {
		t.Run(malformed, func(t *testing.T) {
			dir := selinuxTree(t, varContext,
				selinuxEntry{path: "malformed.so", context: malformed},
				selinuxEntry{path: "loader.so", context: varContext},
			)

			relabeled, _, err := ApplyOpenShiftSELinuxSettings(dir)
			require.NoError(t, err)
			assert.Equal(t, 2, relabeled)
			assert.Equal(t, malformed, selinuxRawContext(t, filepath.Join(dir, "malformed.so")))
		})
	}
}

// selinuxFillExtendedAttributes writes padding attributes until the filesystem
// refuses more, leaving no room to grow the file's existing context.
func selinuxFillExtendedAttributes(t *testing.T, path string) {
	t.Helper()

	for _, size := range []int{200, 100, 50, 20, 10, 4, 1, 0} {
		for i := range 200 {
			name := fmt.Sprintf("user.odigospad%d_%03d", size, i)
			if err := unix.Lsetxattr(path, name, []byte(strings.Repeat("a", size)), 0); err != nil {
				break
			}
		}
	}
}

func TestApplyOpenShiftSELinuxSettingsReportsAFileItCouldNotLabel(t *testing.T) {
	selinuxSkipWithoutWritableLabels(t)

	varContext := "system_u:object_r:" + selinuxVarType + ":s0"
	dir := selinuxTree(t, varContext,
		selinuxEntry{path: "aaa.so", context: varContext},
		selinuxEntry{path: "zfull.so", context: varContext},
	)
	full := filepath.Join(dir, "zfull.so")
	selinuxFillExtendedAttributes(t, full)

	// the type this writes is longer than the one it replaces, so a file with no
	// room left for attributes cannot be labeled at all
	grown := "system_u:object_r:" + selinuxAgentsType + ":s0"
	if err := unix.Lsetxattr(full, selinuxKernelAttr, []byte(grown), 0); err == nil {
		t.Skip("this filesystem still has room for a longer context after the attributes were filled")
	}

	relabeled, rootType, err := ApplyOpenShiftSELinuxSettings(dir)
	require.Error(t, err)
	assert.ErrorIs(t, err, unix.ENOSPC)
	assert.Contains(t, err.Error(), full, "the operator needs to know which file could not be labeled")
	assert.NotContains(t, err.Error(), "SELinux policy does not define",
		"only a rejected type may be reported as a missing policy definition")
	assert.Equal(t, selinuxVarType, rootType)
	assert.Equal(t, 2, relabeled)
	assert.Equal(t, grown, selinuxRawContext(t, filepath.Join(dir, "aaa.so")))
	assert.Equal(t, varContext, selinuxRawContext(t, full))
}

func TestApplyOpenShiftSELinuxSettingsAMissingDirectoryIsAnError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "odigos")

	relabeled, rootType, err := ApplyOpenShiftSELinuxSettings(missing)
	require.Error(t, err)
	assert.Zero(t, relabeled)
	assert.Empty(t, rootType)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestApplyOpenShiftSELinuxSettingsAnUnreadableSubdirectoryFailsTheRun(t *testing.T) {
	selinuxSkipWithoutWritableLabels(t)
	if os.Geteuid() == 0 {
		t.Skip("root reads a directory regardless of its mode")
	}

	varContext := "system_u:object_r:" + selinuxVarType + ":s0"
	dir := selinuxTree(t, varContext,
		selinuxEntry{path: "aaa.so", context: varContext},
		selinuxEntry{path: "locked", dir: true, context: varContext},
		selinuxEntry{path: "locked/inner.so", context: varContext},
	)
	locked := filepath.Join(dir, "locked")
	require.NoError(t, os.Chmod(locked, 0o000))
	t.Cleanup(func() {
		require.NoError(t, os.Chmod(locked, 0o755))
	})

	relabeled, rootType, err := ApplyOpenShiftSELinuxSettings(dir)
	require.Error(t, err)
	assert.ErrorIs(t, err, os.ErrPermission)
	assert.Equal(t, selinuxVarType, rootType)
	// the root, the file and the directory itself were relabeled before the read of
	// its contents failed, and odiglet logs that count next to the error
	assert.Equal(t, 3, relabeled)
}
