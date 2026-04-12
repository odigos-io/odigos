package profiles

import (
	"github.com/odigos-io/odigos/frontend/services/profiles/flamegraph"
)

// BuildFlamegraphProfileFromChunks merges stack samples from stored OTLP profile chunks (protobuf wire) into
// one FlamebearerProfile (flame graph, metadata, timeline from earliest chunk time).
func BuildFlamegraphProfileFromChunks(chunks [][]byte) flamegraph.FlamebearerProfile {
	tree := flamegraph.NewTree()
	for _, b := range chunks {
		samples := flamegraph.SamplesFromOTLPChunk(b)
		if len(samples) == 0 {
			continue
		}
		for _, s := range samples {
			tree.InsertStack(s.Value, s.Stack...)
		}
	}
	// maxNodes 0 => flamegraph default (1024): fold small branches into "other".
	fb := flamegraph.TreeToFlamebearer(tree, 0)
	startTimeSec := earliestProfileStartTimeUnixSec(chunks)
	meta := flamebearerMetadata(fb.NumTicks)
	// Odigos-only UI hint when every frame looks unsymbolized (e.g. frame_N, 0x…).
	if allNamesArePlaceholders(fb.Names) {
		meta.SymbolsHint = "Symbols unavailable. Ensure the collector sends full OTLP profile dictionaries."
	}
	return flamegraph.FlamebearerProfile{
		Version:     flamebearerJSONVersion,
		Flamebearer: fb,
		Metadata:    meta,
		Timeline:    flamebearerTimeline(fb.NumTicks, startTimeSec),
		Groups:      nil,
		Heatmap:     nil,
		Symbols:     nil,
	}
}

func flamebearerMetadata(_ int64) flamegraph.FlamebearerMetadata {
	return flamegraph.FlamebearerMetadata{
		Format:     metadataFormatSingle,
		SpyName:    "",
		SampleRate: metadataSampleRate,
		Units:      metadataUnitsSamples,
		Name:       metadataProfileNameCPU,
	}
}

// flamebearerTimeline builds a minimal single-bar timeline when there are samples (start + duration heuristic).
func flamebearerTimeline(numTicks int64, startTimeSec int64) *flamegraph.FlamebearerTimeline {
	if numTicks == 0 {
		return nil
	}
	return &flamegraph.FlamebearerTimeline{
		StartTime:     startTimeSec,
		Samples:       []int64{0, numTicks},
		DurationDelta: timelineDurationDeltaSec,
		Watermarks:    nil,
	}
}
