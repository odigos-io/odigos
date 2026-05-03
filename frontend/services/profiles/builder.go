package profiles

import (
	"context"

	pyrometadata "github.com/grafana/pyroscope/pkg/og/storage/metadata"
	pyrofb "github.com/grafana/pyroscope/pkg/og/structs/flamebearer"
	"github.com/odigos-io/odigos/frontend/services/profiles/flamegraph"
)

// buildPyroscopeProfileFromChunks builds a Pyroscope-shaped profile from OTLP chunks stored in the ProfileStore buffer.
func buildPyroscopeProfileFromChunks(ctx context.Context, chunks [][]byte) flamegraph.FlamebearerProfile {
	const maxNodes = 2048
	// flamebearerProfile: Grafana Pyroscope flame-tree JSON (levels, names, ticks) from merged OTLP chunks.
	// functionNameTree: parallel structure used only for Odigos symbol statistics on top of the same merge.
	flamebearerProfile, functionNameTree, err := flamegraph.BuildFlamebearerViaPyroscopeSymdb(ctx, chunks, maxNodes)
	if err != nil {
		flamebearerProfile = nil
	}
	startTimeSec := earliestProfileStartTimeUnixSec(chunks)
	numTicks := int64(0)
	if flamebearerProfile != nil {
		numTicks = int64(flamebearerProfile.Flamebearer.NumTicks)
	}
	timeline := pyroscopeTimeline(numTicks, startTimeSec)
	symbols := flamegraph.SymbolStatsFromFunctionNameTree(functionNameTree)
	adapted := flamegraph.AdaptPyroscopeFlamebearerProfile(flamebearerProfile, timeline, symbols)
	if adapted.FlamebearerProfile != nil && adapted.FlamebearerProfile.Metadata.Format == "" {
		adapted.FlamebearerProfile.Metadata = pyroscopeMetadata()
	}
	return adapted
}

// Pyroscope web UI expects this metadata shape for single CPU-style profiles (historical JSON contract).
const (
	pyroscopeFlamebearerJSONVersion = 1
	pyroscopeMetadataFormatSingle   = "single"
	pyroscopeMetadataUnitsSamples   = "samples"
	pyroscopeMetadataProfileNameCPU = "cpu"
	// pyroscopeMetadataSampleRate matches Grafana ExportToFlamebearer for CPU (nanoseconds period hint).
	pyroscopeMetadataSampleRate = 1_000_000_000
)

func pyroscopeMetadata() pyrofb.FlamebearerMetadataV1 {
	return pyrofb.FlamebearerMetadataV1{
		Format:     pyroscopeMetadataFormatSingle,
		SpyName:    "",
		SampleRate: pyroscopeMetadataSampleRate,
		Units:      pyrometadata.Units(pyroscopeMetadataUnitsSamples),
		Name:       pyroscopeMetadataProfileNameCPU,
	}
}

// pyroscopeTimeline builds a minimal single-bar timeline when there are samples (start + duration heuristic).
func pyroscopeTimeline(numTicks int64, startTimeSec int64) *pyrofb.FlamebearerTimelineV1 {
	if numTicks == 0 {
		return nil
	}
	return &pyrofb.FlamebearerTimelineV1{
		StartTime:     startTimeSec,
		Samples:       []uint64{0, uint64(numTicks)},
		DurationDelta: 15,
		Watermarks:    nil,
	}
}
