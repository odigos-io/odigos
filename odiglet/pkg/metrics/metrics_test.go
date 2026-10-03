package metrics

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/cilium/ebpf"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	k8sutilsmetrics "github.com/odigos-io/odigos/k8sutils/pkg/metrics"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"golang.org/x/sys/unix"
	controllermetric "sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	bmNodeName = "ip-10-0-1-7.ec2.internal"
	// the prometheus rendering of the k8s.node.name resource attribute the meter provider adds as
	// a constant label to every self metric.
	bmNodeLabel = "k8s_node_name"
	// the pseudo map collectSelfTotalBPFMemlock fabricates to carry the perf buffer memory.
	bmPerfBuffersName    = "PerfBuffersMemoryUsage"
	bmPerfBuffersMapType = "AllPerfBuffers"
)

// bmSmapsWithTwoPerfBuffers holds two perf_event regions of 8 kB and 16 kB, plus a heap and a stack
// region whose Rss must not be counted.
const bmSmapsWithTwoPerfBuffers = `55a0b0000000-55a0b0021000 rw-p 00000000 00:00 0                          [heap]
Size:                132 kB
Rss:                 100 kB
7f0000000000-7f0000004000 rw-s 00000000 00:0d 12345                      anon_inode:[perf_event]
Size:                 16 kB
KernelPageSize:        4 kB
Rss:                   8 kB
7f0000004000-7f0000008000 rw-s 00000000 00:0d 12346                      anon_inode:[perf_event]
Size:                 16 kB
Rss:                  16 kB
7ffd00000000-7ffd00021000 rw-p 00000000 00:00 0                          [stack]
Rss:                  64 kB
`

const bmSmapsWithoutPerfBuffers = `55a0b0000000-55a0b0021000 rw-p 00000000 00:00 0                          [heap]
Rss:                 100 kB
7ffd00000000-7ffd00021000 rw-p 00000000 00:00 0                          [stack]
Rss:                  64 kB
`

const bmTwoPerfBuffersRss = int64((8 + 16) * 1024)

func bmWriteFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

// bmUseSmaps points the collector at a fixture instead of the real /proc/self/smaps.
func bmUseSmaps(t *testing.T, content string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "smaps")
	bmWriteFile(t, path, content)

	previous := pathSmaps
	pathSmaps = path
	t.Cleanup(func() { pathSmaps = previous })
}

func TestGetPerfBuffersMemoryUsage(t *testing.T) {
	tests := []struct {
		name            string
		smaps           string
		expectedRss     int64
		expectedBuffers int64
	}{
		{
			name:            "every perf event region is counted and only their Rss is summed",
			smaps:           bmSmapsWithTwoPerfBuffers,
			expectedRss:     bmTwoPerfBuffersRss,
			expectedBuffers: 2,
		},
		{
			name:            "a process with no perf event region reports nothing",
			smaps:           bmSmapsWithoutPerfBuffers,
			expectedRss:     0,
			expectedBuffers: 0,
		},
		{
			name: "the next region header stops the accumulation",
			smaps: "7f0000000000-7f0000004000 rw-s 00000000 00:0d 1   anon_inode:[perf_event]\n" +
				"Rss:                   8 kB\n" +
				"7ffd00000000-7ffd00021000 rw-p 00000000 00:00 0   [stack]\n" +
				"Rss:                 999 kB\n",
			expectedRss:     8 * 1024,
			expectedBuffers: 1,
		},
		{
			name:            "a perf event region without an Rss line still counts as a buffer",
			smaps:           "7f0000000000-7f0000004000 rw-s 00000000 00:0d 1   anon_inode:[perf_event]\n",
			expectedRss:     0,
			expectedBuffers: 1,
		},
		{
			name: "a malformed Rss value is skipped rather than failing the scrape",
			smaps: "7f0000000000-7f0000004000 rw-s 00000000 00:0d 1   anon_inode:[perf_event]\n" +
				"Rss:                 n/a kB\n" +
				"7f0000004000-7f0000008000 rw-s 00000000 00:0d 2   anon_inode:[perf_event]\n" +
				"Rss:                  16 kB\n",
			expectedRss:     16 * 1024,
			expectedBuffers: 2,
		},
		{
			name: "an Rss line with no value is skipped",
			smaps: "7f0000000000-7f0000004000 rw-s 00000000 00:0d 1   anon_inode:[perf_event]\n" +
				"Rss:\n",
			expectedRss:     0,
			expectedBuffers: 1,
		},
		{
			name:            "an empty smaps reports nothing",
			smaps:           "",
			expectedRss:     0,
			expectedBuffers: 0,
		},
		{
			// the region pathname is matched by substring, so a mapped file whose name merely
			// contains perf_event is accounted as a perf buffer.
			name: "any region whose pathname contains perf_event is accounted",
			smaps: "7f0000000000-7f0000004000 r--p 00000000 08:01 999   /var/log/perf_event.log\n" +
				"Rss:                   4 kB\n",
			expectedRss:     4 * 1024,
			expectedBuffers: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bmUseSmaps(t, tt.smaps)

			rss, buffers, err := getPerfBuffersMemoryUsage()

			require.NoError(t, err)
			assert.Equal(t, tt.expectedRss, rss)
			assert.Equal(t, tt.expectedBuffers, buffers)
		})
	}
}

func TestGetPerfBuffersMemoryUsageReportsAnUnreadableSmaps(t *testing.T) {
	previous := pathSmaps
	pathSmaps = filepath.Join(t.TempDir(), "does-not-exist")
	t.Cleanup(func() { pathSmaps = previous })

	rss, buffers, err := getPerfBuffersMemoryUsage()

	require.Error(t, err)
	assert.True(t, os.IsNotExist(err), "expected a not-exist error, got %v", err)
	assert.Zero(t, rss)
	assert.Zero(t, buffers)
}

// isMemlockMap decides whether a map's size comes from its memlock accounting or from MaxEntries,
// so a map type landing on the wrong side silently reports a meaningless memory figure.
func TestIsMemlockMapClassifiesEveryKnownMapType(t *testing.T) {
	// ring buffers are sized from MaxEntries by design. DevMapHash is absent from the memlock
	// switch even though DevMap is in it.
	sizedFromMaxEntries := map[ebpf.MapType]bool{
		ebpf.UnspecifiedMap: true,
		ebpf.RingBuf:        true,
		ebpf.UserRingbuf:    true,
		ebpf.DevMapHash:     true,
	}

	known := 0
	for i := 0; i < 256; i++ {
		mapType := ebpf.MapType(i)
		if strings.Contains(mapType.String(), "(") {
			continue // the stringer falls back to MapType(<n>) for values this ebpf version does not know
		}
		known++
		assert.Equal(t, !sizedFromMaxEntries[mapType], isMemlockMap(mapType), "map type %s", mapType)
	}
	require.Greater(t, known, len(sizedFromMaxEntries), "the map type enumeration found nothing to classify")
}

func TestGetAllocatedBytesFromMapTypeSizesRingBuffersFromMaxEntries(t *testing.T) {
	allocated, err := getAllocatedBytesFromMapType(ebpf.RingBuf, &ebpf.MapInfo{
		Type:       ebpf.RingBuf,
		Name:       "events",
		MaxEntries: 4096,
	})

	require.NoError(t, err)
	assert.Equal(t, uint64(4096), allocated, "a ring buffer's MaxEntries is its size in bytes")
}

func TestGetAllocatedBytesFromMapTypeReportsAnUnverifiableMemlock(t *testing.T) {
	allocated, err := getAllocatedBytesFromMapType(ebpf.Hash, &ebpf.MapInfo{
		Type:       ebpf.Hash,
		Name:       "counters",
		MaxEntries: 16,
	})

	require.ErrorContains(t, err, "map counters, type: Hash")
	assert.Zero(t, allocated, "an unverifiable memlock must not fall back to MaxEntries")
}

func TestGetUniqueIDPrefersTheReservedID(t *testing.T) {
	assert.Equal(t, uint32(9999), getUniqueID(map[uint32]eBPFMapMetrics{}))
	assert.Equal(t, uint32(9999), getUniqueID(map[uint32]eBPFMapMetrics{1: {}, 2: {}}))
}

func TestGetUniqueIDFallsBackToAnUnusedIDWhenTheReservedOneIsTaken(t *testing.T) {
	maps := map[uint32]eBPFMapMetrics{9999: {}}
	for id := uint32(1000000); id < 1000100; id++ {
		maps[id] = eBPFMapMetrics{}
	}

	for range 20 {
		id := getUniqueID(maps)

		require.NotContains(t, maps, id)
		assert.GreaterOrEqual(t, id, uint32(1000000))
		assert.Less(t, id, uint32(2000000))
	}
}

func TestCreateMapMetricsFromFDRejectsADescriptorThatIsNotABpfMap(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "not-a-bpf-map")
	require.NoError(t, err)
	t.Cleanup(func() { _ = file.Close() })

	// the collector always hands over a dup, so the scan must not damage the original descriptor.
	duplicated, err := unix.Dup(int(file.Fd()))
	require.NoError(t, err)

	collected := map[uint32]eBPFMapMetrics{}
	perfArrayMaps, err := createMapMetricsFromFD(duplicated, collected, 7)

	require.ErrorContains(t, err, "failed to get map from fd")
	assert.Empty(t, collected)
	assert.Equal(t, int64(7), perfArrayMaps, "a rejected descriptor must not change the perf array count")
	_, err = file.Stat()
	assert.NoError(t, err, "the caller's descriptor was closed")
}

func TestNewEBPFMetricsCollectorAlwaysHasALogger(t *testing.T) {
	assert.NotNil(t, NewEBPFMetricsCollector(nil).logger, "a nil logger would panic on the first scrape")

	logger := commonlogger.LoggerCompat().With("subsystem", "test")
	assert.Same(t, logger, NewEBPFMetricsCollector(logger).logger)
}

func TestCollectSelfTotalBPFMemlockAlwaysReportsThePerfBufferPseudoMap(t *testing.T) {
	bmUseSmaps(t, bmSmapsWithTwoPerfBuffers)
	collector := NewEBPFMetricsCollector(nil)

	// the test process holds no bpf descriptors, so the pseudo map is the only entry collected.
	require.NoError(t, collector.collectSelfTotalBPFMemlock())

	require.Len(t, collector.eBPFMapsMetricsMap, 1)
	pseudoMap := collector.eBPFMapsMetricsMap[9999]
	assert.Equal(t, bmPerfBuffersName, pseudoMap.name)
	assert.Equal(t, bmPerfBuffersMapType, pseudoMap.mapType)
	assert.Equal(t, bmTwoPerfBuffersRss, pseudoMap.memoryUsage)
	assert.Equal(t, uint32(2), pseudoMap.refCnt, "the ref count carries the number of perf buffers")
	assert.Zero(t, collector.eBPFTotalProgCnt)
}

// A node running no eBPF at all still reports one map, because the perf buffer pseudo map is
// inserted unconditionally.
func TestCollectSelfTotalBPFMemlockReportsAPseudoMapEvenWithoutAnyPerfBuffer(t *testing.T) {
	bmUseSmaps(t, bmSmapsWithoutPerfBuffers)
	collector := NewEBPFMetricsCollector(nil)

	require.NoError(t, collector.collectSelfTotalBPFMemlock())

	require.Len(t, collector.eBPFMapsMetricsMap, 1)
	pseudoMap := collector.eBPFMapsMetricsMap[9999]
	assert.Equal(t, bmPerfBuffersName, pseudoMap.name)
	assert.Zero(t, pseudoMap.memoryUsage)
	assert.Zero(t, pseudoMap.refCnt)
}

// A missing smaps leaves the perf buffer memory at zero without failing the collection.
func TestCollectSelfTotalBPFMemlockSwallowsAnUnreadableSmaps(t *testing.T) {
	previous := pathSmaps
	pathSmaps = filepath.Join(t.TempDir(), "does-not-exist")
	t.Cleanup(func() { pathSmaps = previous })
	collector := NewEBPFMetricsCollector(nil)

	require.NoError(t, collector.collectSelfTotalBPFMemlock())

	require.Len(t, collector.eBPFMapsMetricsMap, 1)
	assert.Zero(t, collector.eBPFMapsMetricsMap[9999].memoryUsage)
}

type bmSeries struct {
	labels map[string]string
	gauge  float64
}

// bmGather reads the controller-runtime registry the odiglet meter provider exports through, which
// runs the observable gauge callbacks, and flattens the result into plain types.
func bmGather(t *testing.T) map[string][]bmSeries {
	t.Helper()
	families, err := controllermetric.Registry.Gather()
	require.NoError(t, err)

	gathered := make(map[string][]bmSeries, len(families))
	for _, family := range families {
		for _, m := range family.GetMetric() {
			labels := make(map[string]string, len(m.GetLabel()))
			for _, label := range m.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}
			gathered[family.GetName()] = append(gathered[family.GetName()], bmSeries{
				labels: labels,
				gauge:  m.GetGauge().GetValue(),
			})
		}
	}
	return gathered
}

func bmOnlySeries(t *testing.T, gathered map[string][]bmSeries, familyName string) bmSeries {
	t.Helper()
	series := gathered[familyName]
	require.Len(t, series, 1, "expected exactly one series for %s", familyName)
	return series[0]
}

// The gauges are registered on the process-wide controller-runtime registry and cannot be
// unregistered, so the odiglet wiring is reproduced at most once per test binary.
var (
	bmRegisterOnce sync.Once
	bmRegisterErr  error
)

func bmRegisterSelfMetrics(t *testing.T) {
	t.Helper()
	bmRegisterOnce.Do(func() {
		provider, err := k8sutilsmetrics.NewMeterProviderForController(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.K8SNodeName(bmNodeName),
		))
		if err != nil {
			bmRegisterErr = err
			return
		}
		otel.SetMeterProvider(provider)
		bmRegisterErr = NewEBPFMetricsCollector(nil).RegisterMetrics()
	})
	require.NoError(t, bmRegisterErr)
}

// The node name is no longer attached to the individual observations: it reaches the self metrics
// as a constant label derived from the meter provider resource. This drives the whole chain the
// odiglet wires up, from the resource down to what a prometheus scrape sees.
func TestEveryEBPFSelfMetricCarriesTheNodeNameFromTheMeterProviderResource(t *testing.T) {
	bmUseSmaps(t, bmSmapsWithTwoPerfBuffers)
	bmRegisterSelfMetrics(t)

	gathered := bmGather(t)
	selfMetrics := []string{
		"bpf_map_memory_usage_bytes",
		"bpf_maps_total_memory_usage_bytes",
		"bpf_maps_count",
		"bpf_programs_count",
		"bpf_maps_ref_count",
	}
	for _, name := range selfMetrics {
		series := gathered[name]
		require.NotEmpty(t, series, "%s was not exported", name)
		for _, s := range series {
			assert.Equal(t, bmNodeName, s.labels[bmNodeLabel], "%s lost the node name", name)
		}
	}

	perMap := bmOnlySeries(t, gathered, "bpf_map_memory_usage_bytes")
	assert.Equal(t, bmPerfBuffersName, perMap.labels["name"])
	assert.Equal(t, bmPerfBuffersMapType, perMap.labels["map_type"])
	assert.Equal(t, "9999", perMap.labels["map_id"])
	assert.Equal(t, float64(bmTwoPerfBuffersRss), perMap.gauge)

	assert.Equal(t, float64(bmTwoPerfBuffersRss), bmOnlySeries(t, gathered, "bpf_maps_total_memory_usage_bytes").gauge)
	assert.Equal(t, float64(1), bmOnlySeries(t, gathered, "bpf_maps_count").gauge)
	assert.Equal(t, float64(0), bmOnlySeries(t, gathered, "bpf_programs_count").gauge)

	refCount := bmOnlySeries(t, gathered, "bpf_maps_ref_count")
	assert.Equal(t, bmPerfBuffersName, refCount.labels["name"])
	assert.Equal(t, float64(2), refCount.gauge)
}

// The file descriptor scan reads the host /proc, which is resolved once at package initialisation
// from ODIGOS_PROC_DIR. Re-executing the test binary with that variable set is what makes the scan
// observable against a fixture instead of the real procfs.
const (
	bmScenarioEnvVar   = "ODIGLET_METRICS_TEST_PROC_SCENARIO"
	bmProcScanTestName = "TestCollectSelfTotalBPFMemlockScansTheHostProcDirectory"
)

type bmProcScenario struct {
	name string
	// fileDescriptors maps a /proc/self/fd entry name to the link target it resolves to.
	fileDescriptors     map[string]string
	smaps               string
	expectedPrograms    int64
	expectedMapCount    int
	expectedPerfBuffers uint32
	// A descriptor that classifies as a bpf map but cannot be opened as one is only observable
	// through the debug log, since collecting a real map needs CAP_BPF.
	expectedLogLines  []string
	forbiddenLogLines []string
}

// "0" is the child process' stdin: a descriptor that exists and can be duplicated, but is not a bpf
// map, which is what the bpf-map branch has to survive.
var bmProcScenarios = []bmProcScenario{
	{
		name: "programs are counted and non bpf descriptors are skipped",
		fileDescriptors: map[string]string{
			"0":            "anon_inode:bpf-map",
			"500":          "anon_inode:bpf-prog",
			"501":          "anon_inode:bpf-prog",
			"502":          "socket:[12345]",
			"503":          "/var/log/odiglet.log",
			"not-a-number": "anon_inode:bpf-prog",
		},
		smaps:               bmSmapsWithTwoPerfBuffers,
		expectedPrograms:    2,
		expectedMapCount:    1,
		expectedPerfBuffers: 2,
		expectedLogLines: []string{
			"error creating map metrics for fd",
			"Could not convert file descriptor string",
		},
	},
	{
		name: "a node with no bpf descriptors reports no programs",
		fileDescriptors: map[string]string{
			"502": "socket:[12345]",
		},
		smaps:               bmSmapsWithoutPerfBuffers,
		expectedPrograms:    0,
		expectedMapCount:    1,
		expectedPerfBuffers: 0,
		forbiddenLogLines: []string{
			"error creating map metrics for fd",
			"Could not convert file descriptor string",
		},
	},
}

func TestCollectSelfTotalBPFMemlockScansTheHostProcDirectory(t *testing.T) {
	if name, isChild := os.LookupEnv(bmScenarioEnvVar); isChild {
		bmRunProcScenario(t, name)
		return
	}

	for _, scenario := range bmProcScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			procDir := t.TempDir()
			bmWriteFile(t, filepath.Join(procDir, "self", "smaps"), scenario.smaps)
			require.NoError(t, os.MkdirAll(filepath.Join(procDir, "self", "fd"), 0o755))
			for entry, target := range scenario.fileDescriptors {
				require.NoError(t, os.Symlink(target, filepath.Join(procDir, "self", "fd", entry)))
			}

			cmd := exec.Command(os.Args[0], "-test.run=^"+bmProcScanTestName+"$", "-test.v")
			cmd.Env = append(os.Environ(),
				bmScenarioEnvVar+"="+scenario.name,
				"ODIGOS_PROC_DIR="+procDir,
			)

			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "the re-executed scan failed:\n%s", output)

			for _, line := range scenario.expectedLogLines {
				assert.Contains(t, string(output), line)
			}
			for _, line := range scenario.forbiddenLogLines {
				assert.NotContains(t, string(output), line)
			}
		})
	}
}

func bmRunProcScenario(t *testing.T, name string) {
	var scenario bmProcScenario
	for _, candidate := range bmProcScenarios {
		if candidate.name == name {
			scenario = candidate
		}
	}
	require.Equal(t, name, scenario.name, "unknown scenario")
	require.Equal(t, filepath.Join(os.Getenv("ODIGOS_PROC_DIR"), "self", "smaps"), pathSmaps,
		"the fixture proc directory was not picked up")
	require.Equal(t, os.Getenv("ODIGOS_PROC_DIR"), process.HostProcDir())

	commonlogger.Init("debug", "odiglet-metrics-test")
	collector := NewEBPFMetricsCollector(nil)
	require.NoError(t, collector.collectSelfTotalBPFMemlock())

	assert.Equal(t, scenario.expectedPrograms, collector.eBPFTotalProgCnt)
	require.Len(t, collector.eBPFMapsMetricsMap, scenario.expectedMapCount)
	pseudoMap := collector.eBPFMapsMetricsMap[9999]
	assert.Equal(t, bmPerfBuffersName, pseudoMap.name)
	assert.Equal(t, scenario.expectedPerfBuffers, pseudoMap.refCnt)

	fmt.Printf("scenario %q: programs=%d maps=%d perfBuffers=%d\n",
		name, collector.eBPFTotalProgCnt, len(collector.eBPFMapsMetricsMap), pseudoMap.refCnt)
}
