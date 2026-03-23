package collectorprofiles

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

var jsonMarshaler pprofile.JSONMarshaler

// dumpDir is set from PROFILE_DEBUG_DUMP_DIR at first use; when non-empty, raw profile chunks are written there.
var dumpDir string
var dumpSeq atomic.Uint64

const defaultDumpDir = "profile-dumps"

func init() {
	dumpDir = os.Getenv("PROFILE_DEBUG_DUMP_DIR")
	switch strings.ToLower(dumpDir) {
	case "off", "disabled", "false":
		dumpDir = ""
	case "":
		dumpDir = defaultDumpDir
	}
	if dumpDir != "" {
		if err := os.MkdirAll(dumpDir, 0755); err != nil {
			log.Printf("[profiling] profile dump mkdir %q: %v", dumpDir, err)
			dumpDir = ""
		}
	}
}

// GetProfileDumpDir returns the directory where raw profile chunks are written (empty if disabled).
func GetProfileDumpDir() string {
	return dumpDir
}

// NewProfilesConsumer returns an xconsumer.Profiles that routes incoming profile data
// to the store only for sources that are in the active set (have a slot).
//
// Simplified model: one batch can contain multiple ResourceProfiles (e.g. different services).
// For each ResourceProfile we derive a source key (namespace/kind/name). If the key is active,
// we append one chunk per resource: dictionary (shared in batch) + that single ResourceProfile.
// No merge: each chunk is self-contained with its own dictionary copy, so symbols resolve correctly.
// Chunks are appended in time-series order; the store's buffer is a rolling window (trimmed by size).
func NewProfilesConsumer(store *ProfileStore) (xconsumer.Profiles, error) {
	return xconsumer.NewProfiles(func(ctx context.Context, pd pprofile.Profiles) error {
		rps := pd.ResourceProfiles()
		n := rps.Len()
		if n == 0 {
			return nil
		}
		log.Printf("[profiling] receiver: batch with %d resource profile(s)", n)
		profilingDebugLog("[profiling] receiver: batch with %d resource profile(s)", n)

		storedAny := false
		for i := 0; i < n; i++ {
			rp := rps.At(i)
			key, ok := SourceKeyFromResource(rp.Resource().Attributes())
			if !ok {
				profilingDebugLog("[profiling] receiver: dropped resource (no source key); have attributes: %s", attrsToDebugString(rp.Resource().Attributes()))
				continue
			}
			if !store.IsActive(key) {
				profilingDebugLog("[profiling] receiver: dropped sourceKey=%q (not active/viewing)", key)
				continue
			}
			storedAny = true
			storeOne(store, pd, rps, i)
		}
		if !storedAny && n > 0 {
			log.Printf("[profiling] dropped all %d resource profile(s) (no active slot or key extraction failed)", n)
			profilingDebugLog("[profiling] receiver: dropped all %d resource profile(s) (no matching active slot or key extraction failed)", n)
		}
		return nil
	}, consumer.WithCapabilities(consumer.Capabilities{MutatesData: false}))
}

// storeOne stores a single resource profile (index i in rps) as its own chunk, if the source is active.
func storeOne(store *ProfileStore, pd pprofile.Profiles, rps pprofile.ResourceProfilesSlice, i int) {
	rp := rps.At(i)
	attrs := rp.Resource().Attributes()
	key, ok := SourceKeyFromResource(attrs)
	if !ok {
		profilingDebugLog("[profiling] receiver: dropped resource (no source key); have attributes: %s", attrsToDebugString(attrs))
		return
	}
	if !store.IsActive(key) {
		profilingDebugLog("[profiling] receiver: dropped sourceKey=%q (not active/viewing)", key)
		return
	}
	newPd := pprofile.NewProfiles()
	pd.Dictionary().CopyTo(newPd.Dictionary())
	rp.CopyTo(newPd.ResourceProfiles().AppendEmpty())
	bytes, err := jsonMarshaler.MarshalProfiles(newPd)
	if err != nil {
		log.Printf("[profiling] marshal error for source %q: %v", key, err)
		return
	}
	hasDict := len(bytes) > 0 && (strings.Contains(string(bytes), "stringTable") || strings.Contains(string(bytes), "functionTable") || strings.Contains(string(bytes), "locationTable"))
	dictStats := dictionaryStatsFromChunkJSON(bytes)
	log.Printf("[profiling] stored single chunk key=%q size=%d dictionary=%v %s", key, len(bytes), hasDict, dictStats)
	if !hasDict {
		profilingDebugLog("[profiling] receiver: chunk sourceKey=%q has no dictionary (symbols will show as frame_N); add backend symbolization or have exporter fill dictionary", key)
	}
	store.AddProfileData(key, bytes)
	profilingDebugLog("[profiling] receiver: stored chunk sourceKey=%q size=%d", key, len(bytes))
	if dumpDir != "" {
		writeRawProfileDump(key, bytes)
	}
}

// writeRawProfileDump writes profile JSON (post gRPC unmarshal, same as store) to dumpDir.
// Use the file with ParseOTLPChunk(dumpBytes) locally to iterate on parser and Pyroscope-format output.
// Filename: {sanitizedSourceKey}_{unixNano}_{seq}.json
func writeRawProfileDump(sourceKey string, rawJSON []byte) {
	sanitized := strings.ReplaceAll(sourceKey, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, " ", "_")
	seq := dumpSeq.Add(1)
	name := sanitized + "_" + strconv.FormatInt(time.Now().UnixNano(), 10) + "_" + strconv.FormatUint(seq, 10) + ".json"
	path := filepath.Join(dumpDir, name)
	if err := os.WriteFile(path, rawJSON, 0644); err != nil {
		log.Printf("[profiling] dump write failed: %v", err)
		return
	}
	profilingDebugLog("[profiling] dump wrote %s (%d bytes)", path, len(rawJSON))
}

// attrsToDebugString returns a short string of attribute keys for debug logs (e.g. "k8s.namespace.name,k8s.pod.name").
func attrsToDebugString(attrs pcommon.Map) string {
	var keys []string
	attrs.Range(func(k string, _ pcommon.Value) bool {
		keys = append(keys, k)
		return len(keys) <= 15
	})
	return strings.Join(keys, ",")
}

// dictionaryStatsFromChunkJSON parses the stored chunk JSON and returns a one-line summary of
// dictionary table lengths so we can see in UI logs whether we received symbols (stringTable,
// locationTable, mappingTable). Example: "stringTable=0 locationTable=0 mappingTable=0" or
// "stringTable=42 locationTable=100 mappingTable=2".
func dictionaryStatsFromChunkJSON(chunkJSON []byte) string {
	var root map[string]interface{}
	if err := json.Unmarshal(chunkJSON, &root); err != nil {
		return "dictionary=parse_error"
	}
	dict, _ := root["dictionary"].(map[string]interface{})
	if dict == nil {
		if d, _ := root["Dictionary"].(map[string]interface{}); d != nil {
			dict = d
		}
	}
	if dict == nil {
		return "dictionary=empty"
	}
	length := func(keys ...string) int {
		for _, k := range keys {
			if v, ok := dict[k].([]interface{}); ok {
				return len(v)
			}
		}
		return 0
	}
	st := length("stringTable", "StringTable")
	lt := length("locationTable", "LocationTable")
	mt := length("mappingTable", "MappingTable")
	return "stringTable=" + strconv.Itoa(st) + " locationTable=" + strconv.Itoa(lt) + " mappingTable=" + strconv.Itoa(mt)
}
