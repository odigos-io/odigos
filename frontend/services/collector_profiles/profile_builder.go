package collectorprofiles

import (
	"github.com/odigos-io/odigos/frontend/services/collector_profiles/flamegraph"
)

// BuildPyroscopeProfileFromChunks builds a Pyroscope-shaped profile from OTLP JSON chunks.
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

// Pyroscope web UI expects this metadata shape for single CPU-style profiles (historical JSON contract).
const (
	pyroscopeFlamebearerJSONVersion = 1
	pyroscopeMetadataFormatSingle   = "single"
	pyroscopeMetadataUnitsSamples   = "samples"
	pyroscopeMetadataProfileNameCPU = "cpu"
	// pyroscopeMetadataSampleRate is a display hint for the UI, not OTLP sample timing.
	pyroscopeMetadataSampleRate = 100
)

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
		DurationDelta: 15,
		Watermarks:    nil,
	}
}
