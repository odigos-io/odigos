package metrics

import (
	"bufio"
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"sync"

	"github.com/cilium/ebpf"
	"github.com/go-logr/logr"
	"golang.org/x/sys/unix"
)

const (
	perfEvent = "perf_event"
	rss       = "Rss:"
)

var (
	pathSmaps = process.HostProcDir() + "/self/smaps"
	// headerRegex matches smaps memory region header lines from /proc/[pid]/smaps.
	//Matches: addr_start-addr_end perms offset dev inode [pathname]
	headerRegex = regexp.MustCompile(`^[0-9a-f]+-[0-9a-f]+\s+\S+\s+\S+\s+\S+\s+\S+\s*(.*)$`)
)

type eBPFMapMetrics struct {
	name        string
	mapType     string
	memoryUsage int64
	refCnt      uint32
}
type EBPFMetricsCollector struct {
	eBPFMapsMetricsMap map[uint32]eBPFMapMetrics
	eBPFTotalProgCnt   int64
	mu                 sync.RWMutex
	nodeName           string
	logger             logr.Logger
}

// Returns true of the map allocation size should be fetched from the map's memlock
func isMemlockMap(mapType ebpf.MapType) bool {
	switch mapType {
	case ebpf.Hash, ebpf.Array,
		ebpf.ProgramArray, ebpf.StackTrace, ebpf.CGroupArray,
		ebpf.LRUHash, ebpf.Queue, ebpf.Stack, ebpf.BloomFilter,
		ebpf.StructOpsMap, ebpf.Arena, ebpf.DevMap,
		ebpf.SockMap, ebpf.CPUMap, ebpf.XSKMap,
		ebpf.SockHash, ebpf.ReusePortSockArray, ebpf.ArrayOfMaps,
		ebpf.HashOfMaps, ebpf.LPMTrie, ebpf.SkStorage, ebpf.InodeStorage, ebpf.TaskStorage, ebpf.CgroupStorage, ebpf.CGroupStorage, ebpf.PerfEventArray,
		ebpf.PerCPUArray, ebpf.PerCPUHash, ebpf.PerCPUCGroupStorage, ebpf.LRUCPUHash:
		return true
	default:
		return false
	}
}

// Initialize a new ebpf metrics collector struct
func NewEBPFMetricsCollector(nodeName string, logger logr.Logger) *EBPFMetricsCollector {
	return &EBPFMetricsCollector{
		eBPFMapsMetricsMap: make(map[uint32]eBPFMapMetrics),
		eBPFTotalProgCnt:   0,
		nodeName:           nodeName,
		logger:             logger,
	}

}

// Reads the Smaps file under the /proc/<odigletPID>/smaps and parses perf buffers actual memory usage (Rss)
func getPerfBuffersMemoryUsage() (int64, int64, error) {
	file, err := os.Open(pathSmaps)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	var totalRss, totalBuffers int64
	var inPerfEvent bool

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if matches := headerRegex.FindStringSubmatch(line); matches != nil {
			inPerfEvent = strings.Contains(matches[1], perfEvent)
			if inPerfEvent {
				totalBuffers++
			}
			continue
		}

		if inPerfEvent && strings.HasPrefix(line, rss) {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if rssKb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
					totalRss += rssKb * 1024
				}
			}
		}
	}

	return totalRss, totalBuffers, scanner.Err()
}

// Get number of bytes allocated in memory by the map (memlock could be a upper bond)
func getAllocatedBytesFromMapType(mapType ebpf.MapType, mapInfo *ebpf.MapInfo) (uint64, error) {
	if !isMemlockMap(mapType) {
		// Currently only Ring buffers maps memory size is represented by the MaxEntries
		return uint64(mapInfo.MaxEntries), nil
	}
	memlockBytes, verified := mapInfo.Memlock()
	if verified {
		return memlockBytes, nil
	}
	return 0, fmt.Errorf("Could not get memlock'd bytes for map %s, type: %s", mapInfo.Name, mapInfo.Type)

}

// Generate a unique id for the fabricated perf buffers map - try a specific id first
func getUniqueID(BPFMapsMetrics map[uint32]eBPFMapMetrics) uint32 {
	if _, exists := BPFMapsMetrics[9999]; !exists {
		return 9999
	}
	for {
		id := rand.Uint32N(1000000) + 1000000

		if _, exists := BPFMapsMetrics[id]; !exists {
			return id
		}
	}
}

// Creates a map metric struct and populates it with current map data - if an existing map is found, just increase refCnt
func createMapMetricsFromFD(fd int, BPFMapsMetrics map[uint32]eBPFMapMetrics, numPerfArrayMaps int64) (int64, error) {
	var allocatedBytes uint64
	eBPFMap, err := ebpf.NewMapFromFD(fd)
	if err != nil {
		return numPerfArrayMaps, fmt.Errorf("failed to get map from fd: %w", err)
	}
	defer eBPFMap.Close()
	mapInfo, err := eBPFMap.Info()
	if err != nil {
		return numPerfArrayMaps, err
	}
	mapID, ver := mapInfo.ID()
	if !ver {
		return numPerfArrayMaps, fmt.Errorf("Could not retrieve map id for map: %s", mapInfo.Name)
	}

	// Meaning the map this file descriptor points to was already processed, increase ref count
	if exisitingMapMetrics, exists := BPFMapsMetrics[uint32(mapID)]; exists {
		exisitingMapMetrics.refCnt++
		BPFMapsMetrics[uint32(mapID)] = exisitingMapMetrics
		return numPerfArrayMaps, nil
	}
	if mapInfo.Type == ebpf.PerfEventArray {
		// In order to track number of Perf array maps
		numPerfArrayMaps++
	}

	// different map types have a more accurate memory allocation at different places
	allocatedBytes, err = getAllocatedBytesFromMapType(mapInfo.Type, mapInfo)
	BPFMapsMetrics[uint32(mapID)] = eBPFMapMetrics{
		name:        mapInfo.Name,
		mapType:     mapInfo.Type.String(),
		memoryUsage: int64(allocatedBytes),
		refCnt:      1,
	}
	if err != nil {
		return numPerfArrayMaps, err
	}
	return numPerfArrayMaps, nil
}

// Collect the total BPF maps memory usage metric
func (mc *EBPFMetricsCollector) collectSelfTotalBPFMemlock() error {
	var (
		totalBPFProgs, numPerfArrayMaps int64
		BPFMapsMetrics                  = make(map[uint32]eBPFMapMetrics)
		cpuSet                          unix.CPUSet
		numCPUs                         int8
	)

	mc.logger.V(2).Info("Scanning for eBPF file descriptors")
	dirEntries, err := os.ReadDir(process.HostProcDir() + "/self/fd")
	if err != nil {
		return err
	}
	for _, entry := range dirEntries {
		fdString := entry.Name()
		fdInt, err := strconv.Atoi(fdString)
		if err != nil {
			mc.logger.V(2).Info("Could not convert file descriptor string", "fd", fdString)
			continue
		}
		// read the soft link of entry
		linkedFD, err := os.Readlink(process.HostProcDir() + "/self/fd/" + fdString)
		if err != nil {
			mc.logger.V(2).Info("Could not read fd, Skipping", "fd", fdString, "err", err)
			if os.IsNotExist(err) {
				continue
			}

		}
		// Classify the file descriptor by its link
		switch {
		case strings.Contains(linkedFD, "bpf-map"):
			// Move on to code
		case strings.Contains(linkedFD, "bpf-prog"):
			totalBPFProgs++
			continue
		default:
			continue // not a BPF FD
		}

		// Possible to have multiple FDs to the same BPF object, no need to calculate twice
		dupFD, err := unix.Dup(fdInt)
		if err != nil {
			mc.logger.V(2).Info("Could not dup FD", "fd", fdInt)
			continue
		}

		numPerfArrayMaps, err = createMapMetricsFromFD(dupFD, BPFMapsMetrics, numPerfArrayMaps)
		if err != nil {
			mc.logger.V(2).Info("error creating map metrics for fd:", "fd", fdInt, "error", err)
		}
	}

	err = unix.SchedGetaffinity(0, &cpuSet)
	if err != nil {
		mc.logger.V(2).Info("Error fetching sched affinity mask", err)
		numCPUs = 0

	} else {
		numCPUs = int8(cpuSet.Count())
	}
	totalPerfBuffersMemUsage, totalPerfBuffers, err := getPerfBuffersMemoryUsage()
	if totalPerfBuffers != int64(numPerfArrayMaps)*int64(numCPUs) {
		mc.logger.V(2).Info("Number of perf buffers does not match expected!", "total perf buffers", totalPerfBuffers, "expected number of buffers", uint64(numCPUs)*uint64(numPerfArrayMaps))
	}
	// Insert a fabricated eBPFMapMetrics in order to track all perf buffer memory usage
	uniqueID := getUniqueID(BPFMapsMetrics)
	BPFMapsMetrics[uniqueID] = eBPFMapMetrics{
		name:        "PerfBuffersMemoryUsage",
		mapType:     "AllPerfBuffers",
		memoryUsage: totalPerfBuffersMemUsage,
		refCnt:      uint32(totalPerfBuffers),
	}

	mc.mu.Lock()
	mc.eBPFMapsMetricsMap = BPFMapsMetrics
	mc.eBPFTotalProgCnt = totalBPFProgs
	mc.mu.Unlock()

	return nil
}

// Register the metrics with Observable gauges that can be used by prometheus
func (mc *EBPFMetricsCollector) RegisterMetrics() error {
	meter := otel.GetMeterProvider().Meter("github.com/odigos-io/odigos-enterprise/odiglet/pkg/metrics")
	eBPFMapMemoryGauge, err := meter.Int64ObservableGauge(
		"bpf_map_memory_usage",
		metric.WithDescription("Memory usage of each BPF map"),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		return err
	}
	totalMapsMemGauge, err := meter.Int64ObservableGauge(
		"bpf_maps_total_memory_usage",
		metric.WithDescription("Total memory usage of all BPF objects"),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		return err
	}
	mapCountGauge, err := meter.Int64ObservableGauge(
		"bpf_maps_count",
		metric.WithDescription("Number of BPF maps"),
		metric.WithUnit("{maps}"),
	)
	if err != nil {
		return err
	}

	programCountGauge, err := meter.Int64ObservableGauge(
		"bpf_programs_count",
		metric.WithDescription("Number of BPF programs"),
		metric.WithUnit("{programs}"),
	)
	if err != nil {
		return err
	}
	refCountGauge, err := meter.Int64ObservableGauge(
		"bpf_maps_ref_count",
		metric.WithDescription("Reference count of each bpf map"),
		metric.WithUnit("{ref}"),
	)
	if err != nil {
		return err
	}
	// The callback that runs everytime /metrics is scraped from odiglet, collects data and refreshes gauges
	_, err = meter.RegisterCallback(
		func(ctx context.Context, observer metric.Observer) error {

			if err := mc.collectSelfTotalBPFMemlock(); err != nil {
				mc.logger.Error(err, "failed to collect eBPF metrics during scrape")
			}

			var totalMapsMemory int64
			mapCount := len(mc.eBPFMapsMetricsMap)

			for _, eBPFMapMetrics := range mc.eBPFMapsMetricsMap {
				totalMapsMemory += eBPFMapMetrics.memoryUsage

				observer.ObserveInt64(eBPFMapMemoryGauge, eBPFMapMetrics.memoryUsage,
					metric.WithAttributes(
						attribute.String("name", eBPFMapMetrics.name),
						attribute.String("map_type", eBPFMapMetrics.mapType),
						semconv.K8SNodeName(mc.nodeName),
					),
				)
				observer.ObserveInt64(refCountGauge, int64(eBPFMapMetrics.refCnt),
					metric.WithAttributes(
						attribute.String("name", eBPFMapMetrics.name),
					),
				)
			}

			nodeAttrs := metric.WithAttributes(
				semconv.K8SNodeName(mc.nodeName),
			)
			observer.ObserveInt64(totalMapsMemGauge, totalMapsMemory, nodeAttrs)
			observer.ObserveInt64(mapCountGauge, int64(mapCount), nodeAttrs)
			observer.ObserveInt64(programCountGauge, mc.eBPFTotalProgCnt, nodeAttrs)

			return nil
		},
		eBPFMapMemoryGauge,
		totalMapsMemGauge,
		mapCountGauge,
		programCountGauge,
		refCountGauge,
	)

	return err
}
