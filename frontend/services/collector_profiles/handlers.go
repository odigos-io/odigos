package collectorprofiles

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/services/common"
	"github.com/odigos-io/odigos/frontend/services/collector_profiles/flamegraph"
)

// defaultSymbolizer is used for backend symbolization when DEBUGINFOD_URLS is set.
var defaultSymbolizer *flamegraph.Symbolizer

func init() {
	if u := os.Getenv("DEBUGINFOD_URLS"); u != "" {
		defaultSymbolizer = flamegraph.NewSymbolizer(u)
	}
}

// RegisterProfilingRoutes adds routes for "enable continuous profiling" and "get profile data".
// namespace, kind, name are path params (e.g. /api/sources/:namespace/:kind/:name/profiling).
func RegisterProfilingRoutes(r *gin.RouterGroup, store ProfileStoreRef) {
	if store == nil {
		return
	}
	// Enable continuous profiling for a source (creates/refreshes slot).
	r.PUT("/sources/:namespace/:kind/:name/profiling/enable", func(c *gin.Context) {
		handleEnableProfiling(c, store)
	})
	// Get profile data for a source (snapshot of buffer).
	r.GET("/sources/:namespace/:kind/:name/profiling", func(c *gin.Context) {
		handleGetProfileData(c, store)
	})
	// Debug: list active profiling slots and which have data (for debugging empty flame graphs).
	r.GET("/debug/profiling-slots", func(c *gin.Context) {
		active, withData := store.DebugSlots()
		c.JSON(http.StatusOK, gin.H{"activeKeys": active, "keysWithData": withData})
	})
	// Debug: return raw first chunk JSON for a source (to inspect dictionary: stringTable, functionTable, locationTable).
	r.GET("/debug/sources/:namespace/:kind/:name/profiling-chunk", func(c *gin.Context) {
		handleGetProfilingChunkDebug(c, store)
	})
	// Debug: list and download raw profile dumps (for copying from pod when kubectl cp is not available).
	if dir := GetProfileDumpDir(); dir != "" {
		r.GET("/debug/profile-dumps", handleListProfileDumps)
		r.GET("/debug/profile-dumps/:filename", handleGetProfileDumpFile)
	}
}

func handleEnableProfiling(c *gin.Context, store ProfileStoreRef) {
	id, err := sourceIDFromParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	key := SourceKeyFromSourceID(id)
	store.StartViewing(key)
	activeKeys, _ := store.DebugSlots()
	maxSlots := store.MaxSlots()
	log.Printf("[profiling] enable: sourceKey=%q namespace=%q kind=%q name=%q", key, id.Namespace, id.Kind, id.Name)
	profilingDebugLog("[profiling] enable: sourceKey=%q (namespace=%q kind=%q name=%q)", key, id.Namespace, id.Kind, id.Name)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "sourceKey": key, "maxSlots": maxSlots, "activeSlots": len(activeKeys)})
}

func handleGetProfilingChunkDebug(c *gin.Context, store ProfileStoreRef) {
	id, err := sourceIDFromParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	key := SourceKeyFromSourceID(id)
	chunks := store.GetProfileData(key)
	if len(chunks) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no chunks", "sourceKey": key})
		return
	}
	c.Data(http.StatusOK, "application/json", chunks[0])
}

func handleGetProfileData(c *gin.Context, store ProfileStoreRef) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[profiling] GET panic: %v\n%s", r, debug.Stack())
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("internal error: %v", r)})
		}
	}()
	id, err := sourceIDFromParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	key := SourceKeyFromSourceID(id)
	store.StartViewing(key)
	chunks := store.GetProfileData(key)
	wantDebug := c.Query("debug") == "1"

	if chunks == nil {
		log.Printf("[profiling] get: sourceKey=%q chunks=0 (no slot or empty)", key)
		profilingDebugLog("[profiling] get: sourceKey=%q chunks=0 (no slot or empty)", key)
		payload := flamegraph.FlamebearerProfile{
			Version: 1,
			Flamebearer: flamegraph.Flamebearer{
				Names:    []string{"total"},
				Levels:   [][]int64{},
				NumTicks: 0,
				MaxSelf:  0,
			},
			Metadata: pyroscopeMetadata(0),
		}
		if wantDebug {
			c.JSON(http.StatusOK, gin.H{"profile": payload, "debug": ProfileBuildDebug{
				ChunkCount: 0,
				NumTicks:  0,
			}, "debugReason": "no_slot_or_empty"})
		} else {
			c.JSON(http.StatusOK, payload)
		}
		return
	}
	log.Printf("[profiling] get: sourceKey=%q chunks=%d", key, len(chunks))
	profilingDebugLog("[profiling] get: sourceKey=%q chunks=%d", key, len(chunks))
	profile, buildDebug := BuildPyroscopeProfileFromChunksWithDebug(chunks)
	// Always log aggregation result so we can spot dictionary/parse/flamegraph issues.
	log.Printf("[profiling] build: sourceKey=%q chunkCount=%d numTicks=%d parseErrors=%d chunksWithSamples=%d namesCount=%d",
		key, buildDebug.ChunkCount, buildDebug.NumTicks, buildDebug.ParseErrors, buildDebug.ChunksWithSamples, len(profile.Flamebearer.Names))
	if buildDebug.ParseErrors > 0 {
		log.Printf("[profiling] build: sourceKey=%q parseErrors=%d (some chunks failed to parse)", key, buildDebug.ParseErrors)
	}
	if buildDebug.ChunkCount > 0 && buildDebug.ChunksWithSamples == 0 && buildDebug.NumTicks == 0 {
		log.Printf("[profiling] build: sourceKey=%q chunks have no samples or all failed (chunkCount=%d)", key, buildDebug.ChunkCount)
	}
	if wantDebug {
		c.JSON(http.StatusOK, gin.H{"profile": profile, "debug": buildDebug})
		return
	}
	c.JSON(http.StatusOK, profile)
}

// BuildPyroscopeProfileFromChunks parses OTLP profile chunks (dump format: resourceProfiles + dictionary),
// merges samples into a tree, and returns a Pyroscope-compatible response (version, flamebearer, metadata, timeline).
// Tries Pyroscope's OTLP→pprof conversion first when the chunk has a non-empty dictionary (for proper symbols);
// otherwise falls back to ParseOTLPChunk. When dictionary names are missing, backend symbolization (DEBUGINFOD_URLS)
// resolves mapping+address to function names via debuginfod+DWARF, like Pyroscope's read path.
//
// ProfileBuildDebug holds debug info from building a profile from chunks (for ?debug=1).
type ProfileBuildDebug struct {
	ChunkCount        int   `json:"chunkCount"`
	NumTicks          int64 `json:"numTicks"`
	ParseErrors       int   `json:"parseErrors"`
	ChunksWithSamples int   `json:"chunksWithSamples"`
	ChunksViaPyroscope int  `json:"chunksViaPyroscope"`
}

// Fallback dictionary: the gateway/collector may send the top-level dictionary only in the first batch (or some batches);
// later chunks for the same source can arrive with empty dictionary. We use the first chunk that has a non-empty
// dictionary as reference to resolve names for all chunks, so symbols still appear when only some batches omit the dictionary.
func BuildPyroscopeProfileFromChunks(chunks [][]byte) flamegraph.FlamebearerProfile {
	profile, _ := BuildPyroscopeProfileFromChunksWithDebug(chunks)
	return profile
}

func BuildPyroscopeProfileFromChunksWithDebug(chunks [][]byte) (flamegraph.FlamebearerProfile, ProfileBuildDebug) {
	debug := ProfileBuildDebug{ChunkCount: len(chunks)}
	tree := flamegraph.NewTree()
	// First pass: parse all chunks and find one with non-empty dictionary to use as fallback when others have empty dict.
	var refParsed *flamegraph.ParsedChunk
	for _, b := range chunks {
		if _, ok := flamegraph.ChunksFromPyroscopeOTLP(b); ok {
			continue
		}
		parsed, err := flamegraph.ParseOTLPChunk(b)
		if err != nil {
			debug.ParseErrors++
			continue
		}
		if refParsed == nil && flamegraph.ParsedChunkHasDictionary(parsed) {
			refParsed = parsed
		}
	}
	// Second pass: merge samples into tree, resolving names from each chunk's dictionary or from refParsed.
	for _, b := range chunks {
		if samples, ok := flamegraph.ChunksFromPyroscopeOTLP(b); ok {
			debug.ChunksViaPyroscope++
			for _, s := range samples {
				tree.InsertStack(s.Value, s.Stack...)
			}
			continue
		}
		parsed, err := flamegraph.ParseOTLPChunk(b)
		if err != nil {
			debug.ParseErrors++
			continue
		}
		if len(parsed.Samples) > 0 {
			debug.ChunksWithSamples++
		}
		stackNames := resolveStackNamesWithFallback(parsed, refParsed)
		for _, s := range parsed.Samples {
			if len(s.LocIndices) == 0 {
				tree.InsertStack(s.Value, s.Stack...)
				continue
			}
			stack := make([]string, 0, len(s.LocIndices))
			for _, locIdx := range s.LocIndices {
				stack = append(stack, stackNames[locIdx])
			}
			tree.InsertStack(s.Value, stack...)
		}
	}
	fb := flamegraph.TreeToFlamebearer(tree, 1024)
	debug.NumTicks = fb.NumTicks
	startTimeSec := extractStartTimeFromChunks(chunks)
	meta := pyroscopeMetadata(fb.NumTicks)
	if allNamesArePlaceholders(fb.Names) {
		meta.SymbolsHint = "Symbols unavailable. Set DEBUGINFOD_URLS on the backend to resolve addresses to names, or ensure the collector sends symbol tables."
	}
	return flamegraph.FlamebearerProfile{
		Version:     1,
		Flamebearer: fb,
		Metadata:    meta,
		Timeline:    pyroscopeTimeline(fb.NumTicks, startTimeSec),
		Groups:      nil,
		Heatmap:     nil,
		Symbols:     nil,
	}, debug
}

// allNamesArePlaceholders returns true if every name is frame_N, 0x..., "total", or "other" (no real symbols).
func allNamesArePlaceholders(names []string) bool {
	for _, n := range names {
		if n == "" || n == "total" || n == "other" {
			continue
		}
		if len(n) > 6 && n[:6] == "frame_" {
			continue
		}
		if len(n) > 2 && n[:2] == "0x" {
			continue
		}
		return false
	}
	return true
}

// resolveStackNamesWithFallback returns location index -> display name. Uses parsed first; when a location has no name
// and ref is non-nil (another chunk with dictionary), uses ref's names and ref's location/mapping for symbolizer.
func resolveStackNamesWithFallback(parsed *flamegraph.ParsedChunk, ref *flamegraph.ParsedChunk) map[int]string {
	out := make(map[int]string)
	for idx, name := range parsed.Names {
		if name != "" {
			out[idx] = name
		}
	}
	// Prefill from reference dictionary when current chunk has no name for an index (e.g. chunk had empty dictionary).
	if ref != nil && ref != parsed {
		for idx, name := range ref.Names {
			if name != "" && out[idx] == "" {
				out[idx] = name
			}
		}
	}
	// Use symbolizer when we have location+mapping (from parsed or ref).
	locInfos := parsed.LocationInfos
	mapInfos := parsed.MappingInfos
	if ref != nil && (locInfos == nil || mapInfos == nil) {
		locInfos = ref.LocationInfos
		mapInfos = ref.MappingInfos
	}
	if defaultSymbolizer != nil && locInfos != nil && mapInfos != nil {
		for _, s := range parsed.Samples {
			for _, locIdx := range s.LocIndices {
				if out[locIdx] != "" && out[locIdx] != fmt.Sprintf("frame_%d", locIdx) {
					continue
				}
				loc, ok := locInfos[locIdx]
				if !ok {
					if out[locIdx] == "" {
						out[locIdx] = fmt.Sprintf("frame_%d", locIdx)
					}
					continue
				}
				mapping, ok := mapInfos[loc.MappingIndex]
				if !ok {
					if out[locIdx] == "" {
						out[locIdx] = fmt.Sprintf("frame_%d", locIdx)
					}
					continue
				}
				if name := defaultSymbolizer.Resolve(mapping.BuildID, loc.Address); name != "" {
					out[locIdx] = name
				} else if out[locIdx] == "" {
					out[locIdx] = fmt.Sprintf("0x%x", loc.Address)
				}
			}
		}
	}
	// Fill any still-missing with address or frame_N.
	for _, s := range parsed.Samples {
		for _, locIdx := range s.LocIndices {
			if out[locIdx] != "" {
				continue
			}
			if locInfos != nil {
				if loc, ok := locInfos[locIdx]; ok && loc.Address != 0 {
					out[locIdx] = fmt.Sprintf("0x%x", loc.Address)
					continue
				}
			}
			out[locIdx] = fmt.Sprintf("frame_%d", locIdx)
		}
	}
	return out
}

// pyroscopeMetadata returns metadata in Pyroscope API shape (format, spyName, sampleRate, units, name).
func pyroscopeMetadata(numTicks int64) flamegraph.FlamebearerMetadata {
	return flamegraph.FlamebearerMetadata{
		Format:     "single",
		SpyName:    "", // match Pyroscope (empty)
		SampleRate: 1000000000,
		Units:      "samples",
		Name:       "cpu",
	}
}

// pyroscopeTimeline returns a minimal timeline so the response matches Pyroscope (single bucket with total).
// startTimeSec is Unix seconds from earliest profile in chunks (0 if unknown).
func pyroscopeTimeline(numTicks int64, startTimeSec int64) *flamegraph.FlamebearerTimeline {
	if numTicks == 0 {
		return nil
	}
	return &flamegraph.FlamebearerTimeline{
		StartTime:     startTimeSec,
		Samples:       []int64{0, numTicks},
		DurationDelta: 15,
		Watermarks:    nil, // Pyroscope uses null
	}
}

// extractStartTimeFromChunks returns the earliest timeUnixNano from chunks as Unix seconds, or 0 if none found.
func extractStartTimeFromChunks(chunks [][]byte) int64 {
	var minNano int64
	for _, b := range chunks {
		var raw map[string]interface{}
		if json.Unmarshal(b, &raw) != nil {
			continue
		}
		rps := getKey(raw, "resourceProfiles", "ResourceProfiles")
		arr, ok := rps.([]interface{})
		if !ok {
			continue
		}
		for _, rp := range arr {
			rpm, _ := rp.(map[string]interface{})
			if rpm == nil {
				continue
			}
			scopes := getKey(rpm, "scopeProfiles", "ScopeProfiles")
			sarr, ok := scopes.([]interface{})
			if !ok {
				continue
			}
			for _, s := range sarr {
				sm, _ := s.(map[string]interface{})
				if sm == nil {
					continue
				}
				profs := getKey(sm, "profiles", "Profiles")
				parr, ok := profs.([]interface{})
				if !ok || len(parr) == 0 {
					continue
				}
				p, _ := parr[0].(map[string]interface{})
				if p == nil {
					continue
				}
				ts := getKey(p, "timeUnixNano", "TimeUnixNano")
				if ts == nil {
					continue
				}
				var nano int64
				switch v := ts.(type) {
				case string:
					for _, c := range v {
						if c >= '0' && c <= '9' {
							nano = nano*10 + int64(c-'0')
						}
					}
				case float64:
					nano = int64(v)
				}
				if nano > 0 && (minNano == 0 || nano < minNano) {
					minNano = nano
				}
			}
		}
	}
	if minNano == 0 {
		return 0
	}
	return minNano / 1e9
}

func getKey(m map[string]interface{}, keys ...string) interface{} {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return v
		}
	}
	return nil
}

var errMissingParams = errors.New("missing namespace, kind, or name")

func handleListProfileDumps(c *gin.Context) {
	dir := GetProfileDumpDir()
	if dir == "" {
		c.JSON(http.StatusOK, gin.H{"files": []string{}})
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	c.JSON(http.StatusOK, gin.H{"files": names})
}

func handleGetProfileDumpFile(c *gin.Context) {
	dir := GetProfileDumpDir()
	if dir == "" {
		c.Status(http.StatusNotFound)
		return
	}
	filename := c.Param("filename")
	if filename == "" || strings.Contains(filename, "..") || filepath.Clean(filename) != filename {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filename"})
		return
	}
	path := filepath.Join(dir, filename)
	if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(dir)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filename"})
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			c.Status(http.StatusNotFound)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/json", data)
}

// normalizeWorkloadKind returns the canonical PascalCase kind so the source key matches
// keys built from OTLP resource attributes (e.g. "deployment" -> "Deployment").
func normalizeWorkloadKind(kindStr string) k8sconsts.WorkloadKind {
	switch strings.ToLower(kindStr) {
	case "deployment":
		return k8sconsts.WorkloadKindDeployment
	case "daemonset":
		return k8sconsts.WorkloadKindDaemonSet
	case "statefulset":
		return k8sconsts.WorkloadKindStatefulSet
	case "cronjob":
		return k8sconsts.WorkloadKindCronJob
	case "job":
		return k8sconsts.WorkloadKindJob
	case "deploymentconfig":
		return k8sconsts.WorkloadKindDeploymentConfig
	case "rollout":
		return k8sconsts.WorkloadKindArgoRollout
	case "namespace":
		return k8sconsts.WorkloadKindNamespace
	case "staticpod":
		return k8sconsts.WorkloadKindStaticPod
	default:
		return k8sconsts.WorkloadKind(kindStr)
	}
}

func sourceIDFromParams(c *gin.Context) (common.SourceID, error) {
	namespace := c.Param("namespace")
	kindStr := c.Param("kind")
	name := c.Param("name")
	if namespace == "" || kindStr == "" || name == "" {
		return common.SourceID{}, errMissingParams
	}
	kind := normalizeWorkloadKind(kindStr)
	return common.SourceID{Namespace: namespace, Kind: kind, Name: name}, nil
}
