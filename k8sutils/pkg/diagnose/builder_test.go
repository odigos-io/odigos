package diagnose

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"testing/iotest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileBuilderAddFileWritesTheFileAndCountsIt(t *testing.T) {
	root := t.TempDir()
	builder := NewBuilder()

	dir := filepath.Join(root, dgNamespace, "deployment-odigos-ui")
	require.NoError(t, builder.AddFile(dir, "deployment-odigos-ui.yaml", []byte("kind: Deployment\n")))
	require.NoError(t, builder.AddFile(dir, "pod-odigos-ui-abc.yaml", []byte("kind: Pod\n")))

	onDisk, err := os.ReadFile(filepath.Join(dir, "deployment-odigos-ui.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "kind: Deployment\n", string(onDisk))

	stats := builder.GetStats()
	assert.Equal(t, 2, stats.FileCount)
	assert.Equal(t, int64(len("kind: Deployment\n")+len("kind: Pod\n")), stats.TotalSize)
}

func TestFileBuilderAddFileTruncatesAPreviousBundleFile(t *testing.T) {
	dir := t.TempDir()
	builder := NewBuilder()

	require.NoError(t, builder.AddFile(dir, "metrics", []byte("a-long-previous-body")))
	require.NoError(t, builder.AddFile(dir, "metrics", []byte("short")))

	onDisk, err := os.ReadFile(filepath.Join(dir, "metrics"))
	require.NoError(t, err)
	assert.Equal(t, "short", string(onDisk))
}

func TestFileBuilderAddFileReportsAnUnusableDirectory(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, "not-a-dir")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600))

	builder := NewBuilder()
	err := builder.AddFile(filepath.Join(blocker, "sub"), "f.yaml", []byte("x"))
	require.Error(t, err)
	assert.Equal(t, BuilderStats{}, builder.GetStats(), "a file that could not be written must not be counted")
}

func TestFileBuilderReportsAFileItCannotOpen(t *testing.T) {
	dir := t.TempDir()
	// A directory where the bundle expects a file makes the open fail rather than the
	// MkdirAll that precedes it.
	require.NoError(t, os.Mkdir(filepath.Join(dir, "taken"), 0o755))
	builder := NewBuilder()

	require.Error(t, builder.AddFile(dir, "taken", []byte("x")))
	require.Error(t, builder.AddFileGzipped(dir, "taken", strings.NewReader("x")))
	assert.Equal(t, BuilderStats{}, builder.GetStats())
}

func TestFileBuilderAddFileGzippedReportsAnUnusableDirectory(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, "not-a-dir")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600))

	err := NewBuilder().AddFileGzipped(filepath.Join(blocker, "sub"), "pod.log.gz", strings.NewReader("x"))
	require.Error(t, err)
}

func TestFileBuilderAddFileGzippedRoundTripsAcrossReadChunks(t *testing.T) {
	dir := t.TempDir()
	builder := NewBuilder()

	// Larger than logBufferSize so the read/compress loop runs more than once.
	logs := strings.Repeat("2026-09-19T10:00:00Z odiglet started\n", 80_000)
	require.Greater(t, len(logs), logBufferSize)

	require.NoError(t, builder.AddFileGzipped(dir, "pod-odiglet-x.odiglet.log.gz", strings.NewReader(logs)))

	f, err := os.Open(filepath.Join(dir, "pod-odiglet-x.odiglet.log.gz"))
	require.NoError(t, err)
	defer f.Close()
	gz, err := gzip.NewReader(f)
	require.NoError(t, err)
	decompressed, err := io.ReadAll(gz)
	require.NoError(t, err)
	assert.Equal(t, logs, string(decompressed), "the archived logs must decompress back to exactly what the pod streamed")

	compressed, err := os.Stat(filepath.Join(dir, "pod-odiglet-x.odiglet.log.gz"))
	require.NoError(t, err)
	assert.Less(t, compressed.Size(), int64(len(logs)), "the file on disk must be compressed")
	assert.Equal(t, BuilderStats{FileCount: 1, TotalSize: int64(len(logs))}, builder.GetStats(),
		"every chunk handed to the compressor counts towards the bundle size")
}

func TestFileBuilderAddFileGzippedReportsAReadFailure(t *testing.T) {
	dir := t.TempDir()
	readErr := fmt.Errorf("log stream reset by peer")

	err := NewBuilder().AddFileGzipped(dir, "pod.log.gz", iotest.ErrReader(readErr))
	require.ErrorIs(t, err, readErr)
}

func TestDryRunBuilderCountsFilesWithoutWritingAnything(t *testing.T) {
	dir := t.TempDir()
	builder := NewDryRunBuilder()

	require.NoError(t, builder.AddFile(dir, "configmap-odigos-config.yaml", []byte("data: {}\n")))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries, "a dry run estimates the bundle size and must not create files")
	assert.Equal(t, BuilderStats{FileCount: 1, TotalSize: int64(len("data: {}\n"))}, builder.GetStats())
}

func TestDryRunBuilderEstimatesGzippedFilesBelowTheirUncompressedSize(t *testing.T) {
	builder := NewDryRunBuilder()

	// Past logBufferSize so the estimate has to accumulate every chunk it read,
	// not just the first one.
	logs := strings.Repeat("x", 3*logBufferSize+17)
	require.NoError(t, builder.AddFileGzipped("dir", "pod.log.gz", strings.NewReader(logs)))

	estimated := builder.GetStats().TotalSize
	assert.Positive(t, estimated)
	assert.Less(t, estimated, int64(len(logs)), "the estimate stands in for the gzipped size")

	// The estimate must scale with the bytes actually read: half the input, half the estimate.
	half := NewDryRunBuilder()
	require.NoError(t, half.AddFileGzipped("dir", "pod.log.gz", strings.NewReader(logs[:len(logs)/2])))
	assert.InDelta(t, float64(estimated)/2, float64(half.GetStats().TotalSize), 2)
}

func TestDryRunBuilderReportsAReadFailure(t *testing.T) {
	readErr := fmt.Errorf("log stream reset by peer")
	err := NewDryRunBuilder().AddFileGzipped("dir", "pod.log.gz", iotest.ErrReader(readErr))
	require.ErrorIs(t, err, readErr)
}

// Every diagnose stage writes from its own goroutines, so a lost stats update would
// under-report the bundle size the UI shows before a download.
func TestBuildersAccountForEveryConcurrentWrite(t *testing.T) {
	dir := t.TempDir()
	const writers = 40

	for name, builder := range map[string]Builder{"file": NewBuilder(), "dry-run": NewDryRunBuilder()} {
		t.Run(name, func(t *testing.T) {
			var wg sync.WaitGroup
			for i := 0; i < writers; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					assert.NoError(t, builder.AddFile(dir, fmt.Sprintf("file-%d", i), []byte("0123456789")))
				}(i)
			}
			wg.Wait()

			assert.Equal(t, BuilderStats{FileCount: writers, TotalSize: writers * 10}, builder.GetStats())
		})
	}
}

func TestFormatBytes(t *testing.T) {
	// These strings are shown to the user by the CLI and by the UI's size estimate.
	for _, tc := range []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024*1024 - 1, "1024.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{5 * 1024 * 1024, "5.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1024 * 1024 * 1024 * 1024, "1.0 TB"},
		{1024 * 1024 * 1024 * 1024 * 1024, "1.0 PB"},
		{1024 * 1024 * 1024 * 1024 * 1024 * 1024, "1.0 EB"},
	} {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, FormatBytes(tc.bytes))
		})
	}
}

// The directory layout is the contract a support engineer navigates in the extracted
// bundle, and every getter takes the same string arguments in a different order.
func TestBundleDirectoryLayout(t *testing.T) {
	const (
		root        = "odigos_debug_190920261003"
		odigosNs    = "odigos-system"
		tenantNs    = "shop"
		workloadDir = "deployment-checkout"
	)

	assert.Equal(t, root+"/shop", GetCRDsDir(root, tenantNs))
	assert.Equal(t, root+"/odigos-system/Profile", GetProfileDir(root, odigosNs))
	assert.Equal(t, root+"/odigos-system/Metrics", GetMetricsDir(root, odigosNs))
	assert.Equal(t, root+"/odigos-system/ConfigMaps", GetConfigMapsDir(root, odigosNs))
	assert.Equal(t, root+"/shop/deployment-checkout", GetWorkloadDir(root, tenantNs, workloadDir))
}

func TestGetRootDirMatchesTheDocumentedBundleName(t *testing.T) {
	before := time.Now().Add(-time.Second)

	rootDir := GetRootDir()

	// `odigos diagnose --help` promises odigos_debug_ddmmyyyyhhmmss.tar.gz, and the UI
	// derives the download filename from this directory name.
	stamp, ok := strings.CutPrefix(rootDir, "odigos_debug_")
	require.True(t, ok, "got %q", rootDir)
	parsed, err := time.ParseInLocation("02012006150405", stamp, time.Local)
	require.NoError(t, err, "timestamp %q is not ddmmyyyyhhmmss", stamp)
	assert.WithinRange(t, parsed, before.Truncate(time.Second), time.Now().Add(time.Second))
}

// DefaultOptions is what the UI starts from, so a collection toggle that is added to
// Options and forgotten here silently stops being collected for every UI user.
func TestDefaultOptionsClassifiesEveryCollectionToggle(t *testing.T) {
	onByDefault := map[string]string{
		"IncludeProfiles":   "pprof for the odigos components",
		"IncludeMetrics":    "component own-telemetry",
		"IncludeLogs":       "component pod logs",
		"IncludeCRDs":       "odigos CRDs",
		"IncludeConfigMaps": "odigos configmaps",
	}
	offByDefault := map[string]string{
		// Collecting every instrumented application's manifests is opt-in because it
		// scales with the size of the cluster, not with odigos.
		"IncludeSourceWorkloads": "opt-in, scales with the cluster",
	}

	defaults := DefaultOptions()
	value := reflect.ValueOf(defaults)
	boolFields := 0
	for i := 0; i < value.NumField(); i++ {
		field := value.Type().Field(i)
		if field.Type.Kind() != reflect.Bool {
			continue
		}
		boolFields++
		_, on := onByDefault[field.Name]
		_, off := offByDefault[field.Name]
		require.True(t, on != off, "Options.%s is not classified as on or off by default", field.Name)
		assert.Equal(t, on, value.Field(i).Bool(), "DefaultOptions().%s", field.Name)
	}

	assert.Equal(t, len(onByDefault)+len(offByDefault), boolFields, "a collection toggle was added to Options without being classified here")
	assert.Positive(t, len(onByDefault))
	assert.Positive(t, len(offByDefault))
	assert.Empty(t, defaults.OdigosNamespace, "the namespace has no safe default; RunDiagnose rejects an empty one")
}
