package profiles

import (
	"github.com/odigos-io/odigos/frontend/services/profiles/flamegraph"
)

// BuildPyroscopeProfileFromChunks merges stack samples from stored OTLP profile chunks (protobuf wire) into
// one Pyroscope-shaped FlamebearerProfile (flame graph, metadata, timeline from earliest chunk time).
func BuildPyroscopeProfileFromChunks(chunks [][]byte) flamegraph.FlamebearerProfile {
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
	// maxNodes 0 => flamegraph default (1024): fold small branches into "other" like Grafana Pyroscope.
	fb := flamegraph.TreeToFlamebearer(tree, 0)
	startTimeSec := earliestProfileStartTimeUnixSec(chunks)
	meta := pyroscopeMetadata(fb.NumTicks)
	// Pyroscope does not expose a helper for this: it is Odigos-only UI text when every frame looks
	// unsymbolized (e.g. frame_N, 0x…) so the UI can explain missing dictionaries vs a real profile.
	if allNamesArePlaceholders(fb.Names) {
		meta.SymbolsHint = "Symbols unavailable. Ensure the collector sends full OTLP profile dictionaries (Pyroscope-shaped path)."
	}
	return flamegraph.FlamebearerProfile{
		Version:     pyroscopeFlamebearerJSONVersion,
		Flamebearer: fb,
		Metadata:    meta,
		Timeline:    pyroscopeTimeline(fb.NumTicks, startTimeSec),
		Groups:      nil,
		Heatmap:     nil,
		Symbols:     nil,
	}
}

func pyroscopeMetadata(_ int64) flamegraph.FlamebearerMetadata {
	return flamegraph.FlamebearerMetadata{
		Format:     pyroscopeMetadataFormatSingle,
		SpyName:    "",
		SampleRate: pyroscopeMetadataSampleRate,
		Units:      pyroscopeMetadataUnitsSamples,
		Name:       pyroscopeMetadataProfileNameCPU,
	}
}

// pyroscopeTimeline builds a minimal single-bar timeline when there are samples (start + duration heuristic).
func pyroscopeTimeline(numTicks int64, startTimeSec int64) *flamegraph.FlamebearerTimeline {
	if numTicks == 0 {
		return nil
	}
	return &flamegraph.FlamebearerTimeline{
		StartTime:     startTimeSec,
		Samples:       []int64{0, numTicks},
		DurationDelta: pyroscopeTimelineDurationDeltaSec,
		Watermarks:    nil,
	}
}
