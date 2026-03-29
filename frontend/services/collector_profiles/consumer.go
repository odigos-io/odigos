package collectorprofiles

import (
	"context"
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

func init() {
	v := strings.TrimSpace(os.Getenv("PROFILE_DEBUG_DUMP_DIR"))
	switch strings.ToLower(v) {
	case "", "off", "disabled", "false":
		dumpDir = ""
	default:
		dumpDir = v
		if err := os.MkdirAll(dumpDir, 0755); err != nil {
			bpInfof("profile dump mkdir failed dir=%q err=%v", dumpDir, err)
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
		bpInfof("otlp_receiver: batch resource_profiles=%d", n)
		profilingDebugLog("otlp_receiver: batch resource_profiles=%d", n)
		profilingOtelBatchesTotal.Inc()
		profilingResourceProfilesReceivedTotal.Add(float64(n))

		storedAny := false
		for i := 0; i < n; i++ {
			rp := rps.At(i)
			key, ok := SourceKeyFromResource(rp.Resource().Attributes())
			if !ok {
				profilingResourceProfilesDroppedNoSourceKeyTotal.Inc()
				profilingDebugLog("otlp_receiver: dropped resource (no source key); attributes=%s", attrsToDebugString(rp.Resource().Attributes()))
				continue
			}
			if !store.IsActive(key) {
				profilingResourceProfilesDroppedNoSlotTotal.Inc()
				profilingDebugLog("otlp_receiver: dropped sourceKey=%q (no profiling slot; open Profiler or call enableSourceProfiling)", key)
				continue
			}
			storedAny = true
			storeOne(store, pd, rps, i)
		}
		if !storedAny && n > 0 {
			profilingBatchesFullyDroppedTotal.Inc()
			bpInfof("otlp_receiver: dropped all %d resource_profiles (no active slot or no source key)", n)
			profilingDebugLog("otlp_receiver: dropped all %d resource_profiles (no slot for keys or no source key)", n)
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
		profilingDebugLog("otlp_receiver: dropped resource (no source key); attributes=%s", attrsToDebugString(attrs))
		return
	}
	if !store.IsActive(key) {
		profilingDebugLog("otlp_receiver: dropped sourceKey=%q (no profiling slot; open Profiler or call enableSourceProfiling)", key)
		return
	}
	newPd := pprofile.NewProfiles()
	pd.Dictionary().CopyTo(newPd.Dictionary())
	rp.CopyTo(newPd.ResourceProfiles().AppendEmpty())
	bytes, err := jsonMarshaler.MarshalProfiles(newPd)
	if err != nil {
		profilingChunkMarshalErrorsTotal.Inc()
		bpInfof("store_chunk: marshal error sourceKey=%q err=%v", key, err)
		return
	}
	profilingResourceProfilesStoredTotal.Inc()
	bpInfof("store_chunk: sourceKey=%q bytes=%d", key, len(bytes))
	store.AddProfileData(key, bytes)
	profilingDebugLog("store_chunk: buffered sourceKey=%q bytes=%d", key, len(bytes))
	if dumpDir != "" {
		writeRawProfileDump(key, bytes)
	}
}

// writeRawProfileDump writes profile JSON (post gRPC unmarshal, same as store) to dumpDir.
// Use the file with flamegraph.SamplesFromOTLPChunk(dumpBytes) locally to iterate on transforms.
// Filename: {sanitizedSourceKey}_{unixNano}_{seq}.json
func writeRawProfileDump(sourceKey string, rawJSON []byte) {
	sanitized := strings.ReplaceAll(sourceKey, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, " ", "_")
	seq := dumpSeq.Add(1)
	name := sanitized + "_" + strconv.FormatInt(time.Now().UnixNano(), 10) + "_" + strconv.FormatUint(seq, 10) + ".json"
	path := filepath.Join(dumpDir, name)
	if err := os.WriteFile(path, rawJSON, 0644); err != nil {
		bpInfof("profile_dump: write failed err=%v", err)
		return
	}
	profilingDebugLog("profile_dump: wrote path=%s bytes=%d", path, len(rawJSON))
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
