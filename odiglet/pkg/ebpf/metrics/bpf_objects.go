package metrics

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// BPF syscall commands
const (
	BPF_PROG_GET_NEXT_ID = 11
	BPF_MAP_GET_NEXT_ID  = 12
	BPF_PROG_GET_FD_BY_ID = 13
	BPF_MAP_GET_FD_BY_ID  = 14
	BPF_OBJ_GET_INFO_BY_FD = 15
	BPF_LINK_GET_NEXT_ID = 28
	BPF_LINK_GET_FD_BY_ID = 30
)

// BPF map types
const (
	BPF_MAP_TYPE_UNSPEC = iota
	BPF_MAP_TYPE_HASH
	BPF_MAP_TYPE_ARRAY
	BPF_MAP_TYPE_PROG_ARRAY
	BPF_MAP_TYPE_PERF_EVENT_ARRAY
	BPF_MAP_TYPE_PERCPU_HASH
	BPF_MAP_TYPE_PERCPU_ARRAY
	BPF_MAP_TYPE_STACK_TRACE
	BPF_MAP_TYPE_CGROUP_ARRAY
	BPF_MAP_TYPE_LRU_HASH
	BPF_MAP_TYPE_LRU_PERCPU_HASH
	BPF_MAP_TYPE_LPM_TRIE
	BPF_MAP_TYPE_ARRAY_OF_MAPS
	BPF_MAP_TYPE_HASH_OF_MAPS
	BPF_MAP_TYPE_DEVMAP
	BPF_MAP_TYPE_SOCKMAP
	BPF_MAP_TYPE_CPUMAP
	BPF_MAP_TYPE_XSKMAP
	BPF_MAP_TYPE_SOCKHASH
	BPF_MAP_TYPE_CGROUP_STORAGE
	BPF_MAP_TYPE_REUSEPORT_SOCKARRAY
	BPF_MAP_TYPE_PERCPU_CGROUP_STORAGE
	BPF_MAP_TYPE_QUEUE
	BPF_MAP_TYPE_STACK
	BPF_MAP_TYPE_SK_STORAGE
	BPF_MAP_TYPE_DEVMAP_HASH
	BPF_MAP_TYPE_STRUCT_OPS
	BPF_MAP_TYPE_RINGBUF
	BPF_MAP_TYPE_INODE_STORAGE
	BPF_MAP_TYPE_TASK_STORAGE
	BPF_MAP_TYPE_BLOOM_FILTER
)

// BPF program types
const (
	BPF_PROG_TYPE_UNSPEC = iota
	BPF_PROG_TYPE_SOCKET_FILTER
	BPF_PROG_TYPE_KPROBE
	BPF_PROG_TYPE_SCHED_CLS
	BPF_PROG_TYPE_SCHED_ACT
	BPF_PROG_TYPE_TRACEPOINT
	BPF_PROG_TYPE_XDP
	BPF_PROG_TYPE_PERF_EVENT
	BPF_PROG_TYPE_CGROUP_SKB
	BPF_PROG_TYPE_CGROUP_SOCK
	BPF_PROG_TYPE_LWT_IN
	BPF_PROG_TYPE_LWT_OUT
	BPF_PROG_TYPE_LWT_XMIT
	BPF_PROG_TYPE_SOCK_OPS
	BPF_PROG_TYPE_SK_SKB
	BPF_PROG_TYPE_CGROUP_DEVICE
	BPF_PROG_TYPE_SK_MSG
	BPF_PROG_TYPE_RAW_TRACEPOINT
	BPF_PROG_TYPE_CGROUP_SOCK_ADDR
	BPF_PROG_TYPE_LWT_SEG6LOCAL
	BPF_PROG_TYPE_LIRC_MODE2
	BPF_PROG_TYPE_SK_REUSEPORT
	BPF_PROG_TYPE_FLOW_DISSECTOR
	BPF_PROG_TYPE_CGROUP_SYSCTL
	BPF_PROG_TYPE_RAW_TRACEPOINT_WRITABLE
	BPF_PROG_TYPE_CGROUP_SOCKOPT
	BPF_PROG_TYPE_TRACING
	BPF_PROG_TYPE_STRUCT_OPS
	BPF_PROG_TYPE_EXT
	BPF_PROG_TYPE_LSM
	BPF_PROG_TYPE_SK_LOOKUP
	BPF_PROG_TYPE_SYSCALL
)

// BPF link types
const (
	BPF_LINK_TYPE_UNSPEC = iota
	BPF_LINK_TYPE_RAW_TRACEPOINT
	BPF_LINK_TYPE_TRACING
	BPF_LINK_TYPE_CGROUP
	BPF_LINK_TYPE_ITER
	BPF_LINK_TYPE_NETNS
	BPF_LINK_TYPE_XDP
	BPF_LINK_TYPE_PERF_EVENT
	BPF_LINK_TYPE_KPROBE_MULTI
	BPF_LINK_TYPE_STRUCT_OPS
)

// BPF syscall structures
type bpfAttr struct {
	// Union for different BPF commands
	// We'll use the first 64 bytes as a generic buffer
	data [64]byte
}

type bpfMapInfo struct {
	Type                  uint32
	ID                    uint32
	KeySize              uint32
	ValueSize            uint32
	MaxEntries           uint32
	MapFlags             uint32
	NameLen              uint32
	Name                 [16]int8
	IfIndex              uint32
	NetNSDev             uint64
	NetNSIno             uint64
	BTFId                uint32
	BTFKeyTypeId         uint32
	BTFValueTypeId       uint32
	Pad                  uint32
	MapExtra             uint64
}

type bpfProgInfo struct {
	Type                 uint32
	ID                   uint32
	Tag                  [8]uint8
	JitedProgLen         uint32
	XlatedProgLen        uint32
	JitedProgInsns       uint64
	XlatedProgInsns      uint64
	LoadTime             uint64
	CreatedByUID         uint32
	NrMapIDs             uint32
	MapIDs               uint64
	NameLen              uint32
	Name                 [16]int8
	IfIndex              uint32
	GplCompatible        uint32
	NetNSDev             uint64
	NetNSIno             uint64
	NrJitedKSyms         uint32
	NrJitedFuncLens      uint32
	JitedKSyms           uint64
	JitedFuncLens        uint64
	BTFId                uint32
	FuncInfoRecSize      uint32
	FuncInfo             uint64
	NrFuncInfo           uint32
	LineInfoRecSize      uint32
	LineInfo             uint64
	NrLineInfo           uint32
	JitedLineInfo        uint64
	NrJitedLineInfo      uint32
	LineInfoRecSize2     uint32
	JitedLineInfoRecSize uint32
	NrProgTags           uint32
	ProgTags             uint64
	RunTimeNs            uint64
	RunCnt               uint64
	RecursionMisses      uint64
	VerifiedInsnCnt      uint32
	AttachBtfId          uint32
	AttachBtfObjId       uint32
}

type bpfLinkInfo struct {
	Type    uint32
	ID      uint32
	ProgID  uint32
	// Additional fields would be type-specific
}

// Map type names
var mapTypeNames = map[uint32]string{
	BPF_MAP_TYPE_UNSPEC:                   "unspec",
	BPF_MAP_TYPE_HASH:                     "hash",
	BPF_MAP_TYPE_ARRAY:                    "array",
	BPF_MAP_TYPE_PROG_ARRAY:               "prog_array",
	BPF_MAP_TYPE_PERF_EVENT_ARRAY:         "perf_event_array",
	BPF_MAP_TYPE_PERCPU_HASH:              "percpu_hash",
	BPF_MAP_TYPE_PERCPU_ARRAY:             "percpu_array",
	BPF_MAP_TYPE_STACK_TRACE:              "stack_trace",
	BPF_MAP_TYPE_CGROUP_ARRAY:             "cgroup_array",
	BPF_MAP_TYPE_LRU_HASH:                 "lru_hash",
	BPF_MAP_TYPE_LRU_PERCPU_HASH:          "lru_percpu_hash",
	BPF_MAP_TYPE_LPM_TRIE:                 "lpm_trie",
	BPF_MAP_TYPE_ARRAY_OF_MAPS:            "array_of_maps",
	BPF_MAP_TYPE_HASH_OF_MAPS:             "hash_of_maps",
	BPF_MAP_TYPE_DEVMAP:                   "devmap",
	BPF_MAP_TYPE_SOCKMAP:                  "sockmap",
	BPF_MAP_TYPE_CPUMAP:                   "cpumap",
	BPF_MAP_TYPE_XSKMAP:                   "xskmap",
	BPF_MAP_TYPE_SOCKHASH:                 "sockhash",
	BPF_MAP_TYPE_CGROUP_STORAGE:           "cgroup_storage",
	BPF_MAP_TYPE_REUSEPORT_SOCKARRAY:      "reuseport_sockarray",
	BPF_MAP_TYPE_PERCPU_CGROUP_STORAGE:    "percpu_cgroup_storage",
	BPF_MAP_TYPE_QUEUE:                    "queue",
	BPF_MAP_TYPE_STACK:                    "stack",
	BPF_MAP_TYPE_SK_STORAGE:               "sk_storage",
	BPF_MAP_TYPE_DEVMAP_HASH:              "devmap_hash",
	BPF_MAP_TYPE_STRUCT_OPS:               "struct_ops",
	BPF_MAP_TYPE_RINGBUF:                  "ringbuf",
	BPF_MAP_TYPE_INODE_STORAGE:            "inode_storage",
	BPF_MAP_TYPE_TASK_STORAGE:             "task_storage",
	BPF_MAP_TYPE_BLOOM_FILTER:             "bloom_filter",
}

// Program type names
var progTypeNames = map[uint32]string{
	BPF_PROG_TYPE_UNSPEC:                  "unspec",
	BPF_PROG_TYPE_SOCKET_FILTER:           "socket_filter",
	BPF_PROG_TYPE_KPROBE:                  "kprobe",
	BPF_PROG_TYPE_SCHED_CLS:               "sched_cls",
	BPF_PROG_TYPE_SCHED_ACT:               "sched_act",
	BPF_PROG_TYPE_TRACEPOINT:              "tracepoint",
	BPF_PROG_TYPE_XDP:                     "xdp",
	BPF_PROG_TYPE_PERF_EVENT:              "perf_event",
	BPF_PROG_TYPE_CGROUP_SKB:              "cgroup_skb",
	BPF_PROG_TYPE_CGROUP_SOCK:             "cgroup_sock",
	BPF_PROG_TYPE_LWT_IN:                  "lwt_in",
	BPF_PROG_TYPE_LWT_OUT:                 "lwt_out",
	BPF_PROG_TYPE_LWT_XMIT:                "lwt_xmit",
	BPF_PROG_TYPE_SOCK_OPS:                "sock_ops",
	BPF_PROG_TYPE_SK_SKB:                  "sk_skb",
	BPF_PROG_TYPE_CGROUP_DEVICE:           "cgroup_device",
	BPF_PROG_TYPE_SK_MSG:                  "sk_msg",
	BPF_PROG_TYPE_RAW_TRACEPOINT:          "raw_tracepoint",
	BPF_PROG_TYPE_CGROUP_SOCK_ADDR:        "cgroup_sock_addr",
	BPF_PROG_TYPE_LWT_SEG6LOCAL:           "lwt_seg6local",
	BPF_PROG_TYPE_LIRC_MODE2:              "lirc_mode2",
	BPF_PROG_TYPE_SK_REUSEPORT:            "sk_reuseport",
	BPF_PROG_TYPE_FLOW_DISSECTOR:          "flow_dissector",
	BPF_PROG_TYPE_CGROUP_SYSCTL:           "cgroup_sysctl",
	BPF_PROG_TYPE_RAW_TRACEPOINT_WRITABLE: "raw_tracepoint_writable",
	BPF_PROG_TYPE_CGROUP_SOCKOPT:          "cgroup_sockopt",
	BPF_PROG_TYPE_TRACING:                 "tracing",
	BPF_PROG_TYPE_STRUCT_OPS:              "struct_ops",
	BPF_PROG_TYPE_EXT:                     "ext",
	BPF_PROG_TYPE_LSM:                     "lsm",
	BPF_PROG_TYPE_SK_LOOKUP:               "sk_lookup",
	BPF_PROG_TYPE_SYSCALL:                 "syscall",
}

// Link type names
var linkTypeNames = map[uint32]string{
	BPF_LINK_TYPE_UNSPEC:        "unspec",
	BPF_LINK_TYPE_RAW_TRACEPOINT: "raw_tracepoint",
	BPF_LINK_TYPE_TRACING:       "tracing",
	BPF_LINK_TYPE_CGROUP:        "cgroup",
	BPF_LINK_TYPE_ITER:          "iter",
	BPF_LINK_TYPE_NETNS:         "netns",
	BPF_LINK_TYPE_XDP:           "xdp",
	BPF_LINK_TYPE_PERF_EVENT:    "perf_event",
	BPF_LINK_TYPE_KPROBE_MULTI:  "kprobe_multi",
	BPF_LINK_TYPE_STRUCT_OPS:    "struct_ops",
}

// bpfSyscall makes a BPF syscall
func bpfSyscall(cmd int, attr *bpfAttr) (int, error) {
	r1, _, errno := unix.Syscall(unix.SYS_BPF, uintptr(cmd), uintptr(unsafe.Pointer(attr)), unsafe.Sizeof(*attr))
	if errno != 0 {
		return -1, errno
	}
	return int(r1), nil
}

// getNextMapID gets the next map ID
func getNextMapID(id uint32) (uint32, error) {
	attr := &bpfAttr{}
	// Set start_id in the attr structure
	*(*uint32)(unsafe.Pointer(&attr.data[0])) = id

	_, err := bpfSyscall(BPF_MAP_GET_NEXT_ID, attr)
	if err != nil {
		return 0, err
	}

	// Get next_id from the attr structure
	nextID := *(*uint32)(unsafe.Pointer(&attr.data[4]))
	return nextID, nil
}

// getNextProgID gets the next program ID
func getNextProgID(id uint32) (uint32, error) {
	attr := &bpfAttr{}
	// Set start_id in the attr structure
	*(*uint32)(unsafe.Pointer(&attr.data[0])) = id

	_, err := bpfSyscall(BPF_PROG_GET_NEXT_ID, attr)
	if err != nil {
		return 0, err
	}

	// Get next_id from the attr structure
	nextID := *(*uint32)(unsafe.Pointer(&attr.data[4]))
	return nextID, nil
}

// getNextLinkID gets the next link ID
func getNextLinkID(id uint32) (uint32, error) {
	attr := &bpfAttr{}
	// Set start_id in the attr structure
	*(*uint32)(unsafe.Pointer(&attr.data[0])) = id

	_, err := bpfSyscall(BPF_LINK_GET_NEXT_ID, attr)
	if err != nil {
		return 0, err
	}

	// Get next_id from the attr structure
	nextID := *(*uint32)(unsafe.Pointer(&attr.data[4]))
	return nextID, nil
}

// getMapFD gets a file descriptor for a map by ID
func getMapFD(id uint32) (int, error) {
	attr := &bpfAttr{}
	// Set map_id in the attr structure
	*(*uint32)(unsafe.Pointer(&attr.data[0])) = id

	fd, err := bpfSyscall(BPF_MAP_GET_FD_BY_ID, attr)
	if err != nil {
		return -1, err
	}
	return fd, nil
}

// getProgFD gets a file descriptor for a program by ID
func getProgFD(id uint32) (int, error) {
	attr := &bpfAttr{}
	// Set prog_id in the attr structure
	*(*uint32)(unsafe.Pointer(&attr.data[0])) = id

	fd, err := bpfSyscall(BPF_PROG_GET_FD_BY_ID, attr)
	if err != nil {
		return -1, err
	}
	return fd, nil
}

// getLinkFD gets a file descriptor for a link by ID
func getLinkFD(id uint32) (int, error) {
	attr := &bpfAttr{}
	// Set link_id in the attr structure
	*(*uint32)(unsafe.Pointer(&attr.data[0])) = id

	fd, err := bpfSyscall(BPF_LINK_GET_FD_BY_ID, attr)
	if err != nil {
		return -1, err
	}
	return fd, nil
}

// getObjInfo gets information about a BPF object by file descriptor
func getObjInfo(fd int, info unsafe.Pointer, infoLen uint32) error {
	attr := &bpfAttr{}
	// Set bpf_fd, info_len, and info pointer in the attr structure
	*(*uint32)(unsafe.Pointer(&attr.data[0])) = uint32(fd)
	*(*uint32)(unsafe.Pointer(&attr.data[4])) = infoLen
	*(*uint64)(unsafe.Pointer(&attr.data[8])) = uint64(uintptr(info))

	_, err := bpfSyscall(BPF_OBJ_GET_INFO_BY_FD, attr)
	return err
}

// Implementation of the parsing functions from collector.go

func (c *EBPFMetricsCollector) parseMapInfo() ([]*EBPFMapInfo, error) {
	// Check memory limits first
	if c.memoryPool.IsMemoryLimitExceeded() {
		atomic.AddInt64(c.memoryLimitHit, 1)
		return nil, fmt.Errorf("memory limit exceeded, skipping map collection")
	}

	// Step 1: Efficiently enumerate all map IDs first
	mapIDs, err := c.enumerateMapIDs()
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate map IDs: %w", err)
	}

	// Step 2: Process in batches to reduce syscall overhead
	var maps []*EBPFMapInfo
	
	err = c.batchProcessor.ProcessInBatches(mapIDs, func(mapID uint32) error {
		// Get pre-allocated object from pool
		mapInfo := c.memoryPool.GetMapInfo()
		if mapInfo == nil {
			// Pool exhausted, stop processing
			atomic.AddInt64(c.memoryLimitHit, 1)
			return fmt.Errorf("map pool exhausted")
		}

		if err := c.populateMapInfo(mapID, mapInfo); err != nil {
			c.logger.V(1).Info("Failed to get map info", "map_id", mapID, "error", err)
			return nil // Continue with next map
		}

		maps = append(maps, mapInfo)
		return nil
	})

	return maps, err
}

// enumerateMapIDs efficiently gets all map IDs in one pass
func (c *EBPFMetricsCollector) enumerateMapIDs() ([]uint32, error) {
	var mapIDs []uint32
	var id uint32 = 0
	
	for len(mapIDs) < MaxTrackedMaps { // Limit enumeration
		nextID, err := getNextMapID(id)
		if err != nil {
			break // End of maps
		}
		mapIDs = append(mapIDs, nextID)
		id = nextID
	}
	
	return mapIDs, nil
}

// populateMapInfo efficiently populates a single map info object
func (c *EBPFMetricsCollector) populateMapInfo(mapID uint32, mapInfo *EBPFMapInfo) error {
	fd, err := getMapFD(mapID)
	if err != nil {
		return err
	}
	defer unix.Close(fd)

	var info bpfMapInfo
	err = getObjInfo(fd, unsafe.Pointer(&info), uint32(unsafe.Sizeof(info)))
	if err != nil {
		return err
	}

	// Convert C string to Go string efficiently
	nameBytes := (*[16]byte)(unsafe.Pointer(&info.Name[0]))
	name := string(nameBytes[:clen(nameBytes[:])])

	// Calculate memory usage estimate
	memoryUsage := uint64(info.KeySize+info.ValueSize) * uint64(info.MaxEntries)

	// Get map type name
	mapTypeName := mapTypeNames[info.Type]
	if mapTypeName == "" {
		mapTypeName = fmt.Sprintf("unknown_%d", info.Type)
	}

	// Populate the pre-allocated object
	mapInfo.ID = info.ID
	mapInfo.Type = mapTypeName
	mapInfo.Name = name
	mapInfo.KeySize = info.KeySize
	mapInfo.ValueSize = info.ValueSize
	mapInfo.MaxEntries = info.MaxEntries
	mapInfo.MapFlags = info.MapFlags
	mapInfo.MemoryUsage = memoryUsage
	mapInfo.FrozenFlag = false // Default
	mapInfo.PinnedPath = "" // Skip pinned path lookup for performance

	return nil
}

func (c *EBPFMetricsCollector) parseProgInfo() ([]*EBPFProgInfo, error) {
	// Check memory limits first
	if c.memoryPool.IsMemoryLimitExceeded() {
		atomic.AddInt64(c.memoryLimitHit, 1)
		return nil, fmt.Errorf("memory limit exceeded, skipping program collection")
	}

	// Step 1: Efficiently enumerate all program IDs first
	progIDs, err := c.enumerateProgIDs()
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate program IDs: %w", err)
	}

	// Step 2: Process in batches to reduce syscall overhead
	var progs []*EBPFProgInfo
	
	err = c.batchProcessor.ProcessInBatches(progIDs, func(progID uint32) error {
		// Get pre-allocated object from pool
		progInfo := c.memoryPool.GetProgInfo()
		if progInfo == nil {
			// Pool exhausted, stop processing
			atomic.AddInt64(c.memoryLimitHit, 1)
			return fmt.Errorf("program pool exhausted")
		}

		if err := c.populateProgInfo(progID, progInfo); err != nil {
			c.logger.V(1).Info("Failed to get program info", "prog_id", progID, "error", err)
			return nil // Continue with next program
		}

		progs = append(progs, progInfo)
		return nil
	})

	return progs, err
}

// enumerateProgIDs efficiently gets all program IDs in one pass
func (c *EBPFMetricsCollector) enumerateProgIDs() ([]uint32, error) {
	var progIDs []uint32
	var id uint32 = 0
	
	for len(progIDs) < MaxTrackedProgs { // Limit enumeration
		nextID, err := getNextProgID(id)
		if err != nil {
			break // End of programs
		}
		progIDs = append(progIDs, nextID)
		id = nextID
	}
	
	return progIDs, nil
}

// populateProgInfo efficiently populates a single program info object
func (c *EBPFMetricsCollector) populateProgInfo(progID uint32, progInfo *EBPFProgInfo) error {
	fd, err := getProgFD(progID)
	if err != nil {
		return err
	}
	defer unix.Close(fd)

	var info bpfProgInfo
	err = getObjInfo(fd, unsafe.Pointer(&info), uint32(unsafe.Sizeof(info)))
	if err != nil {
		return err
	}

	// Convert C string to Go string efficiently
	nameBytes := (*[16]byte)(unsafe.Pointer(&info.Name[0]))
	name := string(nameBytes[:clen(nameBytes[:])])

	// Get program type name
	progTypeName := progTypeNames[info.Type]
	if progTypeName == "" {
		progTypeName = fmt.Sprintf("unknown_%d", info.Type)
	}

	// Calculate load time
	loadTime := time.Unix(0, int64(info.LoadTime))

	// Populate the pre-allocated object
	progInfo.ID = info.ID
	progInfo.Type = progTypeName
	progInfo.Name = name
	progInfo.LoadTime = loadTime
	progInfo.CreatedByUID = info.CreatedByUID
	progInfo.InsnCnt = info.XlatedProgLen / 8 // Assuming 8 bytes per instruction
	progInfo.JitedProgLen = info.JitedProgLen
	progInfo.XlatedProgLen = info.XlatedProgLen
	progInfo.MemoryUsage = uint64(info.JitedProgLen + info.XlatedProgLen)
	progInfo.NrMapIDs = info.NrMapIDs
	progInfo.VerifiedInsnCnt = info.VerifiedInsnCnt
	progInfo.RunTimeBs = info.RunTimeNs
	progInfo.RunCnt = info.RunCnt

	return nil
}

func (c *EBPFMetricsCollector) parseLinkInfo() ([]*EBPFLinkInfo, error) {
	// Check memory limits first
	if c.memoryPool.IsMemoryLimitExceeded() {
		atomic.AddInt64(c.memoryLimitHit, 1)
		return nil, fmt.Errorf("memory limit exceeded, skipping link collection")
	}

	// Step 1: Efficiently enumerate all link IDs first
	linkIDs, err := c.enumerateLinkIDs()
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate link IDs: %w", err)
	}

	// Step 2: Process in batches to reduce syscall overhead
	var links []*EBPFLinkInfo
	
	err = c.batchProcessor.ProcessInBatches(linkIDs, func(linkID uint32) error {
		// Get pre-allocated object from pool
		linkInfo := c.memoryPool.GetLinkInfo()
		if linkInfo == nil {
			// Pool exhausted, stop processing
			atomic.AddInt64(c.memoryLimitHit, 1)
			return fmt.Errorf("link pool exhausted")
		}

		if err := c.populateLinkInfo(linkID, linkInfo); err != nil {
			c.logger.V(1).Info("Failed to get link info", "link_id", linkID, "error", err)
			return nil // Continue with next link
		}

		links = append(links, linkInfo)
		return nil
	})

	return links, err
}

// enumerateLinkIDs efficiently gets all link IDs in one pass
func (c *EBPFMetricsCollector) enumerateLinkIDs() ([]uint32, error) {
	var linkIDs []uint32
	var id uint32 = 0
	
	for len(linkIDs) < MaxTrackedLinks { // Limit enumeration
		nextID, err := getNextLinkID(id)
		if err != nil {
			break // End of links
		}
		linkIDs = append(linkIDs, nextID)
		id = nextID
	}
	
	return linkIDs, nil
}

// populateLinkInfo efficiently populates a single link info object
func (c *EBPFMetricsCollector) populateLinkInfo(linkID uint32, linkInfo *EBPFLinkInfo) error {
	fd, err := getLinkFD(linkID)
	if err != nil {
		return err
	}
	defer unix.Close(fd)

	var info bpfLinkInfo
	err = getObjInfo(fd, unsafe.Pointer(&info), uint32(unsafe.Sizeof(info)))
	if err != nil {
		return err
	}

	// Get link type name
	linkTypeName := linkTypeNames[info.Type]
	if linkTypeName == "" {
		linkTypeName = fmt.Sprintf("unknown_%d", info.Type)
	}

	// Populate the pre-allocated object
	linkInfo.ID = info.ID
	linkInfo.Type = linkTypeName
	linkInfo.ProgID = info.ProgID

	return nil
}

// Helper function to find pinned path for a BPF object
func (c *EBPFMetricsCollector) findPinnedPath(id uint32, objType string) string {
	// Check common pinned locations
	pinnedPaths := []string{
		"/sys/fs/bpf",
		"/sys/fs/bpf/tc",
		"/sys/fs/bpf/xdp",
	}

	for _, basePath := range pinnedPaths {
		if _, err := os.Stat(basePath); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Continue walking even if there's an error
			}

			// Check if this might be our object by trying to read its ID
			// This is a simplified check - in practice you'd need to open
			// the pinned file and check its metadata
			if strings.Contains(path, objType) {
				return filepath.SkipDir
			}
			return nil
		})

		if err != nil {
			c.logger.V(1).Info("Error walking pinned path", "path", basePath, "error", err)
		}
	}

	return ""
}

// Helper function to calculate the length of a null-terminated C string
func clen(b []byte) int {
	for i := 0; i < len(b); i++ {
		if b[i] == 0 {
			return i
		}
	}
	return len(b)
}

// Additional system-level eBPF metrics collection

func (c *EBPFMetricsCollector) getEBPFSystemMemoryUsage() (int64, error) {
	// Use the object tracker to get aggregated memory usage efficiently
	_, _, _, totalMapMemory, totalProgMemory, _, _ := 
		c.objectTracker.GetAggregatedStats()
	
	totalMemory := int64(totalMapMemory + totalProgMemory)
	return totalMemory, nil
}

func (c *EBPFMetricsCollector) getEBPFResourceUsage() (float64, error) {
	// Get eBPF resource usage percentage using object tracker
	mapCount, progCount, linkCount, _, _, _, _ := 
		c.objectTracker.GetAggregatedStats()
	
	totalObjects := mapCount + progCount + linkCount
	
	// Use configuration limits for more accurate resource usage calculation
	const maxReasonableObjects = 1000 // Could be made configurable
	
	usage := float64(totalObjects) / float64(maxReasonableObjects) * 100.0
	if usage > 100.0 {
		usage = 100.0
	}

	return usage, nil
}