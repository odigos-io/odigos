//go:build linux

// symbolize.go is the entry point and engine: the Symbolizer holds the symbol
// and maps caches, parses binaries on background workers, evicts on a budget,
// sweeps exited processes, and turns one (pid, mapping, address) into a named
// Frame. See doc.go for the package overview.
package symbolize

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ianlancetaylor/demangle"
	"go.uber.org/zap"
)

// Defaults. Caches evict by both entry count and total bytes so a long-running
// collector on a busy node cannot leak memory — one Oracle binary alone can hold
// tens of MB of symbols, so the byte budget is the real guard at scale.
const (
	defaultMaxSymbolCache = 1024
	defaultMaxSymbolBytes = 256 << 20 // 256 MiB of parsed symbols across all binaries
	defaultMaxMapsCache   = 4096
	defaultMapsTTL        = 30 * time.Second
	defaultBackoff        = 60 * time.Second // wait before retrying a failed/oversized binary
	defaultParseWorkers   = 2
	defaultParseQueue     = 1024
	defaultSweepEvery     = 30 * time.Second // how often to drop caches for exited processes
	defaultMaxFileBytes   = 8 << 30          // 8 GiB — corrupt/absurd ELF guard
	defaultMaxSymbols     = 5_000_000        // pathological-binary guard
	defaultMaxSymtabBytes = 512 << 20        // 512 MiB — cap the transient symbol-table decode
)

// Mapping describes one loaded module as an OTLP profile carries it.
type Mapping struct {
	Name        string // module basename (e.g. "libCXOPSX00.so")
	MemoryStart uint64 // OTLP Mapping.MemoryStart (0 when the profiler normalized it)
	FileOffset  uint64 // OTLP Mapping.FileOffset
	BuildID     string // GNU build-id (hex), or "" to skip verification
}

// Frame is the resolved result for one address.
type Frame struct {
	Name    string
	Module  string
	BuildID string
	Offset  uint64
	Size    uint64
	Source  string // "symtab" | "dynsym"
}

// Symbolizer resolves native instruction addresses to function names from the
// on-disk ELF symbols of a process's loaded modules. Safe for concurrent use:
// the hot path (cache hit) takes only a read lock, and ELF symbol tables are
// parsed on background workers, never inline in Resolve. See the package doc.
type Symbolizer struct {
	log *zap.Logger

	mu          sync.RWMutex
	symbolCache map[string]*cachedSymbols // path -> parsed symbols
	mapsCache   map[int]*cachedMaps       // pid -> parsed /proc/<pid>/maps
	backoff     map[string]int64          // path -> unix-nano until; failed/oversized binaries
	parsing     map[string]struct{}       // paths currently queued or being parsed
	cachedBytes int64                     // total estimated bytes held by symbolCache

	clock atomic.Uint64 // monotonic counter driving LRU "last used"

	maxSymbols      int
	maxSymbolBytes  int64
	maxMaps         int
	mapsTTL         time.Duration
	backoffDuration time.Duration
	sweepEvery      time.Duration
	parseWorkers    int
	limits          parseLimits

	parseQueue chan string
	stop       chan struct{}
	wg         sync.WaitGroup
}

// cachedSymbols is one binary's parsed symbols plus the file identity (mtime,
// size) used to detect a rebuilt binary, and an LRU timestamp.
type cachedSymbols struct {
	symbols   *elfSymbols
	modTime   int64
	size      int64
	heapBytes int64
	lastUsed  atomic.Uint64
}

// cachedMaps is one process's parsed maps plus when it was cached (for TTL) and
// an LRU timestamp.
type cachedMaps struct {
	maps     *procMaps
	cachedAt int64 // unix-nano
	lastUsed atomic.Uint64
}

// Option configures a Symbolizer.
type Option func(*Symbolizer)

// WithLogger sets the zap logger used for debug diagnostics. Defaults to a no-op.
func WithLogger(l *zap.Logger) Option {
	return func(s *Symbolizer) {
		if l != nil {
			s.log = l
		}
	}
}

// WithMaxSymbolCache caps cached parsed binaries by count (LRU). n<=0 keeps the default.
func WithMaxSymbolCache(n int) Option {
	return func(s *Symbolizer) {
		if n > 0 {
			s.maxSymbols = n
		}
	}
}

// WithMaxSymbolBytes caps the total bytes of parsed symbols held across all cached
// binaries (LRU eviction by memory — the real guard at scale). n<=0 keeps the default.
func WithMaxSymbolBytes(n int64) Option {
	return func(s *Symbolizer) {
		if n > 0 {
			s.maxSymbolBytes = n
		}
	}
}

// WithMaxMapsCache caps cached per-pid maps (LRU). n<=0 keeps the default.
func WithMaxMapsCache(n int) Option {
	return func(s *Symbolizer) {
		if n > 0 {
			s.maxMaps = n
		}
	}
}

// WithMapsTTL sets how long a cached /proc/<pid>/maps is reused. d<=0 keeps the default.
func WithMapsTTL(d time.Duration) Option {
	return func(s *Symbolizer) {
		if d > 0 {
			s.mapsTTL = d
		}
	}
}

// WithParseWorkers sets the number of background ELF-parse workers. n<=0 keeps the default.
func WithParseWorkers(n int) Option {
	return func(s *Symbolizer) {
		if n > 0 {
			s.parseWorkers = n
		}
	}
}

// New returns a started Symbolizer. Call Close to stop the background workers.
func New(opts ...Option) *Symbolizer {
	s := &Symbolizer{
		log:             zap.NewNop(),
		symbolCache:     make(map[string]*cachedSymbols),
		mapsCache:       make(map[int]*cachedMaps),
		backoff:         make(map[string]int64),
		parsing:         make(map[string]struct{}),
		maxSymbols:      defaultMaxSymbolCache,
		maxSymbolBytes:  defaultMaxSymbolBytes,
		maxMaps:         defaultMaxMapsCache,
		mapsTTL:         defaultMapsTTL,
		backoffDuration: defaultBackoff,
		sweepEvery:      defaultSweepEvery,
		limits:          parseLimits{maxFileBytes: defaultMaxFileBytes, maxSymbols: defaultMaxSymbols, maxSymtabBytes: defaultMaxSymtabBytes},
		stop:            make(chan struct{}),
	}
	for _, o := range opts {
		o(s)
	}
	if s.parseWorkers <= 0 {
		s.parseWorkers = defaultParseWorkers
	}
	s.parseQueue = make(chan string, defaultParseQueue)
	for i := 0; i < s.parseWorkers; i++ {
		s.wg.Add(1)
		go s.runParseWorker()
	}
	s.wg.Add(1)
	go s.runSweeper()
	s.log.Debug("symbolize: started",
		zap.Int("parse_workers", s.parseWorkers),
		zap.Int("max_symbol_cache", s.maxSymbols),
		zap.Int64("max_symbol_bytes", s.maxSymbolBytes),
		zap.Int("max_maps_cache", s.maxMaps))
	return s
}

// Close stops the background parse and sweep workers.
func (s *Symbolizer) Close() {
	close(s.stop)
	s.wg.Wait()
}

// Forget drops the cached maps for a pid (called by the sweeper on process exit;
// caches are also TTL- and LRU-bounded).
func (s *Symbolizer) Forget(pid int) {
	s.mu.Lock()
	delete(s.mapsCache, pid)
	s.mu.Unlock()
}

// PreWarm asynchronously parses the symbol tables of the binaries a process has
// mapped, so a later Resolve hits a warm cache (no hot-path parse). Cheap and
// non-blocking; safe to call on every observed process start.
func (s *Symbolizer) PreWarm(pid int) {
	maps, err := s.mapsFor(pid)
	if err != nil {
		return
	}
	for _, path := range maps.executablePaths() {
		s.scheduleParse(path)
	}
}

// Resolve maps a frame address within mapping m of process pid to a named Frame.
// ok is false when the module can't be located, its symbols aren't parsed yet
// (a parse is scheduled in the background), the build-id mismatches, or no symbol
// contains the address — the caller then keeps module+offset, and a not-yet-parsed
// binary resolves on a later batch.
func (s *Symbolizer) Resolve(pid int, m Mapping, addr uint64) (Frame, bool) {
	path, ok := s.hostPathForModule(pid, m.Name)
	if !ok {
		return Frame{Module: filepath.Base(m.Name)}, false
	}

	es := s.symbolsFor(path)
	if es == nil {
		s.scheduleParse(path) // background; never parse inline
		return Frame{Module: filepath.Base(path)}, false
	}
	if m.BuildID != "" && es.buildID != m.BuildID {
		return Frame{Module: filepath.Base(path), BuildID: es.buildID}, false // wrong/redeployed binary
	}
	if len(es.functions) == 0 {
		return Frame{Module: filepath.Base(path), BuildID: es.buildID}, false
	}
	vaddr, ok := es.toVirtualAddr(m.MemoryStart, m.FileOffset, addr)
	if !ok {
		return Frame{Module: filepath.Base(path), BuildID: es.buildID}, false
	}
	fn, ok := es.functionAt(vaddr)
	if !ok {
		return Frame{Module: filepath.Base(path), BuildID: es.buildID}, false
	}
	offset := fn.addr
	if off, ok := es.toFileOffset(fn.addr); ok {
		offset = off
	}
	return Frame{
		Name:    demangle.Filter(fn.name, demangle.NoClones),
		Module:  filepath.Base(path),
		BuildID: es.buildID,
		Offset:  offset,
		Size:    fn.size,
		Source:  es.source,
	}, true
}

// hostPathForModule resolves a module basename to an openable host path via the
// process's (cached) /proc/<pid>/maps.
func (s *Symbolizer) hostPathForModule(pid int, nameOrPath string) (string, bool) {
	maps, err := s.mapsFor(pid)
	if err != nil {
		return "", false
	}
	return maps.hostPath(nameOrPath)
}

// mapsFor returns the process's parsed maps, reading /proc/<pid>/maps on a cache
// miss or once the TTL expires. The read-locked fast path lets cache hits run
// concurrently.
func (s *Symbolizer) mapsFor(pid int) (*procMaps, error) {
	now := time.Now().UnixNano()
	s.mu.RLock()
	if e, ok := s.mapsCache[pid]; ok && now-e.cachedAt < int64(s.mapsTTL) {
		e.lastUsed.Store(s.nextClock())
		maps := e.maps
		s.mu.RUnlock()
		return maps, nil
	}
	s.mu.RUnlock()

	maps, err := parseProcMaps(pid) // a small read, fine to do inline
	if err != nil {
		return nil, err
	}
	e := &cachedMaps{maps: maps, cachedAt: now}
	e.lastUsed.Store(s.nextClock())
	s.mu.Lock()
	s.mapsCache[pid] = e
	s.evictMapsOverBudget()
	s.mu.Unlock()
	return maps, nil
}

// symbolsFor returns a binary's cached symbols, or nil if it isn't parsed yet
// (or the on-disk file changed). Read-locked so cache hits run concurrently.
func (s *Symbolizer) symbolsFor(path string) *elfSymbols {
	fi, err := os.Stat(path)
	if err != nil {
		return nil
	}
	modTime, size := fi.ModTime().UnixNano(), fi.Size()
	s.mu.RLock()
	defer s.mu.RUnlock()
	if e, ok := s.symbolCache[path]; ok && e.modTime == modTime && e.size == size {
		e.lastUsed.Store(s.nextClock())
		return e.symbols
	}
	return nil
}

// scheduleParse queues a binary for background parsing (deduped and back-off
// aware). Non-blocking: if the queue is full the request is dropped (debug-logged)
// and retried on the next Resolve.
func (s *Symbolizer) scheduleParse(path string) {
	now := time.Now().UnixNano()
	s.mu.Lock()
	if until, ok := s.backoff[path]; ok && now < until {
		s.mu.Unlock()
		return // still backing off after a recent failure
	}
	if _, busy := s.parsing[path]; busy {
		s.mu.Unlock()
		return
	}
	s.parsing[path] = struct{}{}
	s.mu.Unlock()

	select {
	case s.parseQueue <- path:
	default:
		s.mu.Lock()
		delete(s.parsing, path)
		queued := len(s.parseQueue)
		s.mu.Unlock()
		s.log.Debug("symbolize: parse queue full, dropping (will retry next batch)",
			zap.String("path", path), zap.Int("queue_len", queued))
	}
}

// runParseWorker pulls paths off the queue and parses them until Close.
func (s *Symbolizer) runParseWorker() {
	defer s.wg.Done()
	for {
		select {
		case <-s.stop:
			return
		case path := <-s.parseQueue:
			s.parseAndCache(path)
		}
	}
}

// parseAndCache reads a binary's ELF symbols (the expensive step, off the hot
// path) and stores them, evicting if the cache is over budget. On failure it
// records a back-off so the binary isn't retried every batch.
func (s *Symbolizer) parseAndCache(path string) {
	defer func() {
		s.mu.Lock()
		delete(s.parsing, path)
		s.mu.Unlock()
	}()
	fi, err := os.Stat(path)
	if err != nil {
		s.recordParseFailure(path, "stat", err)
		return
	}
	es, err := loadELFSymbols(path, s.limits)
	if err != nil {
		s.recordParseFailure(path, "parse", err)
		return
	}
	e := &cachedSymbols{symbols: es, modTime: fi.ModTime().UnixNano(), size: fi.Size(), heapBytes: es.heapBytes}
	e.lastUsed.Store(s.nextClock())
	s.mu.Lock()
	if old, ok := s.symbolCache[path]; ok {
		s.cachedBytes -= old.heapBytes
	}
	s.symbolCache[path] = e
	s.cachedBytes += e.heapBytes
	s.evictSymbolsOverBudget()
	s.mu.Unlock()
	s.log.Debug("symbolize: parsed binary",
		zap.String("path", path), zap.String("source", es.source),
		zap.Int("funcs", len(es.functions)), zap.Int64("bytes", es.heapBytes),
		zap.String("build_id", es.buildID))
}

// recordParseFailure marks a binary as failed for backoffDuration so process
// churn doesn't make us re-parse it every batch; the sweeper expires the entry.
func (s *Symbolizer) recordParseFailure(path, stage string, err error) {
	s.mu.Lock()
	s.backoff[path] = time.Now().Add(s.backoffDuration).UnixNano()
	s.mu.Unlock()
	s.log.Debug("symbolize: skipping binary (backing off)",
		zap.String("path", path), zap.String("stage", stage),
		zap.Duration("backoff", s.backoffDuration), zap.Error(err))
}

// runSweeper periodically evicts caches for processes that have exited.
func (s *Symbolizer) runSweeper() {
	defer s.wg.Done()
	t := time.NewTicker(s.sweepEvery)
	defer t.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-t.C:
			s.evictExitedProcesses()
		}
	}
}

// evictExitedProcesses drops cached maps for processes that no longer exist and
// expires stale back-off entries. We can only change odigos/vm-agent (not the
// eBPF runtime-detector that would deliver precise exit events), so we stat
// /proc/<pid> — cheap and reliable. This bounds memory and stops a reused pid
// from resolving against stale maps within the TTL window.
func (s *Symbolizer) evictExitedProcesses() {
	s.mu.RLock()
	pids := make([]int, 0, len(s.mapsCache))
	for pid := range s.mapsCache {
		pids = append(pids, pid)
	}
	s.mu.RUnlock()

	dead := 0
	for _, pid := range pids {
		if !processExists(pid) {
			s.Forget(pid)
			dead++
		}
	}
	now := time.Now().UnixNano()
	s.mu.Lock()
	for path, until := range s.backoff {
		if now >= until {
			delete(s.backoff, path)
		}
	}
	s.mu.Unlock()
	if dead > 0 {
		s.log.Debug("symbolize: swept exited processes", zap.Int("forgotten_pids", dead))
	}
}

// nextClock returns the next monotonic tick used as an entry's LRU timestamp.
func (s *Symbolizer) nextClock() uint64 { return s.clock.Add(1) }

// evictSymbolsOverBudget drops least-recently-used binaries until the symbol
// cache is within both its entry-count and byte budgets. Caller holds s.mu.
func (s *Symbolizer) evictSymbolsOverBudget() {
	for len(s.symbolCache) > s.maxSymbols || s.cachedBytes > s.maxSymbolBytes {
		victim, found := oldestKey(s.symbolCache)
		if !found {
			return
		}
		s.cachedBytes -= s.symbolCache[victim].heapBytes
		delete(s.symbolCache, victim)
	}
}

// evictMapsOverBudget drops least-recently-used pid maps until within the count
// budget. Caller holds s.mu.
func (s *Symbolizer) evictMapsOverBudget() {
	for len(s.mapsCache) > s.maxMaps {
		victim, found := oldestKey(s.mapsCache)
		if !found {
			return
		}
		delete(s.mapsCache, victim)
	}
}

// lruEntry is the LRU timestamp accessor shared by both caches.
type lruEntry interface{ lru() uint64 }

func (e *cachedSymbols) lru() uint64 { return e.lastUsed.Load() }
func (e *cachedMaps) lru() uint64    { return e.lastUsed.Load() }

// oldestKey returns the key of the least-recently-used entry in a cache.
func oldestKey[K comparable, V lruEntry](m map[K]V) (K, bool) {
	var oldest K
	var oldestLRU uint64
	found := false
	for k, e := range m {
		if !found || e.lru() < oldestLRU {
			oldest, oldestLRU, found = k, e.lru(), true
		}
	}
	return oldest, found
}
