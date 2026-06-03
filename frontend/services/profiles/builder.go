package profiles

import (
	"context"

	pyrometadata "github.com/grafana/pyroscope/pkg/og/storage/metadata"
	pyrofb "github.com/grafana/pyroscope/pkg/og/structs/flamebearer"
	"github.com/odigos-io/odigos/frontend/services/profiles/flamegraph"
)

// buildPyroscopeProfileFromChunks builds a Pyroscope-shaped profile from OTLP chunks stored in the ProfileStore buffer.
// profileType selects the sample type to render ("cpu" or "alloc_space"; empty defaults to "cpu").
func buildPyroscopeProfileFromChunks(ctx context.Context, chunks [][]byte, profileType string) flamegraph.FlamebearerProfile {
	const maxNodes = 2048
	profileType = flamegraph.NormalizeProfileType(profileType)
	// flamebearerProfile: Grafana Pyroscope flame-tree JSON (levels, names, ticks) from merged OTLP chunks.
	// functionNameTree: parallel structure used only for Odigos symbol statistics on top of the same merge.
	flamebearerProfile, functionNameTree, err := flamegraph.BuildFlamebearerViaPyroscopeSymdb(ctx, chunks, maxNodes, profileType)
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
	if adapted.FlamebearerProfile != nil {
		if adapted.FlamebearerProfile.Metadata.Format == "" {
			// Empty/aggregate-only profile: ExportToFlamebearer was not used, so fill metadata ourselves.
			adapted.FlamebearerProfile.Metadata = pyroscopeMetadataFor(profileType)
		} else if profileType == flamegraph.SampleTypeAllocSpace {
			// ExportToFlamebearer already set Format/levels; normalize the memory metadata to the
			// Odigos contract (name "memory", units "bytes"). CPU output is left byte-identical to today.
			md := pyroscopeMetadataFor(profileType)
			adapted.FlamebearerProfile.Metadata.Units = md.Units
			adapted.FlamebearerProfile.Metadata.Name = md.Name
		}
	}
	return adapted
}

// Pyroscope web UI expects this metadata shape for single profiles (historical JSON contract).
const (
	pyroscopeFlamebearerJSONVersion = 1
	pyroscopeMetadataFormatSingle   = "single"
	pyroscopeMetadataUnitsSamples   = "samples"
	pyroscopeMetadataProfileNameCPU = "cpu"
	// pyroscopeMetadataSampleRate matches Grafana ExportToFlamebearer for CPU (nanoseconds period hint).
	pyroscopeMetadataSampleRate = 1_000_000_000

	// Memory (alloc_space) reports byte weights; Pyroscope's bytes unit drives byte-formatted ticks.
	pyroscopeMetadataUnitsBytes        = "bytes"
	pyroscopeMetadataProfileNameMemory = "memory"
	// Memory profiles have no per-second rate; ExportToFlamebearer uses 100 for non-CPU types.
	pyroscopeMetadataSampleRateMemory = 100
)

// pyroscopeMetadataFor returns the flamebearer metadata for a given profile type. CPU keeps the exact
// samples/nanoseconds shape used historically; alloc_space reports bytes under the "memory" name.
func pyroscopeMetadataFor(profileType string) pyrofb.FlamebearerMetadataV1 {
	if flamegraph.NormalizeProfileType(profileType) == flamegraph.SampleTypeAllocSpace {
		return pyrofb.FlamebearerMetadataV1{
			Format:     pyroscopeMetadataFormatSingle,
			SpyName:    "",
			SampleRate: pyroscopeMetadataSampleRateMemory,
			Units:      pyrometadata.Units(pyroscopeMetadataUnitsBytes),
			Name:       pyroscopeMetadataProfileNameMemory,
		}
	}
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
