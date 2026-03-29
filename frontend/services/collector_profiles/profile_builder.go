package collectorprofiles

import (
	"encoding/json"
	"github.com/odigos-io/odigos/frontend/services/collector_profiles/flamegraph"
	pprofileotlp "go.opentelemetry.io/proto/otlp/collector/profiles/v1development"
	"google.golang.org/protobuf/encoding/protojson"
	"os"
)

// ProfileBuildDebug holds debug info from building a profile from chunks (for ?debug=1).
type ProfileBuildDebug struct {
	ChunkCount         int   `json:"chunkCount"`
	NumTicks           int64 `json:"numTicks"`
	ParseErrors        int   `json:"parseErrors"`
	ChunksWithSamples  int   `json:"chunksWithSamples"`
	ChunksViaPyroscope int   `json:"chunksViaPyroscope"`
}

// BuildPyroscopeProfileFromChunks parses OTLP profile chunks and returns a Pyroscope-compatible response.
func BuildPyroscopeProfileFromChunks(chunks [][]byte) flamegraph.FlamebearerProfile {
	profile, _ := BuildPyroscopeProfileFromChunksWithDebug(chunks)
	return profile
}

// BuildPyroscopeProfileFromChunksWithDebug is like BuildPyroscopeProfileFromChunks but also returns debug info.
func BuildPyroscopeProfileFromChunksWithDebug(chunks [][]byte) (flamegraph.FlamebearerProfile, ProfileBuildDebug) {
	dbg := ProfileBuildDebug{ChunkCount: len(chunks)}
	tree := flamegraph.NewTree()
	bpInfof("build_profile: start chunk_count=%d", len(chunks))
	for i, b := range chunks {
		samples, st := flamegraph.SamplesFromOTLPChunk(b)
		switch st.Route {
		case flamegraph.RoutePyroscopeOTLP:
			dbg.ChunksViaPyroscope++
		case flamegraph.RouteError:
			dbg.ParseErrors++
			bpInfof("build_profile: chunk[%d] transform_error bytes=%d pyroscope_reason=%q",
				i, st.ByteLen, st.PyroscopeFailReason)
			continue
		}
		if len(samples) > 0 {
			dbg.ChunksWithSamples++
		}
		if len(samples) == 0 {
			bpInfof("build_profile: chunk[%d] no samples after transform route=%s bytes=%d", i, st.Route, st.ByteLen)
			continue
		}
		for _, s := range samples {
			tree.InsertStack(s.Value, s.Stack...)
		}
	}
	fb := flamegraph.TreeToFlamebearer(tree, 1024)
	dbg.NumTicks = fb.NumTicks
	startTimeSec := extractStartTimeFromChunks(chunks)
	meta := pyroscopeMetadata(fb.NumTicks)
	if allNamesArePlaceholders(fb.Names) {
		meta.SymbolsHint = "Symbols unavailable. Ensure the collector sends full OTLP profile dictionaries (Pyroscope-shaped path)."
	}
	if os.Getenv("PROFILE_BUILD_SUMMARY") != "" {
		b, _ := json.Marshal(dbg)
		bpInfof("build_profile: summary_json=%s", string(b))
	}
	bpInfof("build_profile: done num_ticks=%d names=%d levels=%d pyroscope_chunks=%d parse_errors=%d",
		fb.NumTicks, len(fb.Names), len(fb.Levels), dbg.ChunksViaPyroscope, dbg.ParseErrors)
	return flamegraph.FlamebearerProfile{
		Version:     1,
		Flamebearer: fb,
		Metadata:    meta,
		Timeline:    pyroscopeTimeline(fb.NumTicks, startTimeSec),
		Groups:      nil,
		Heatmap:     nil,
		Symbols:     nil,
	}, dbg
}

// pyroscopeMetadata returns metadata in Pyroscope API shape (format, spyName, sampleRate, units, name).
func pyroscopeMetadata(_ int64) flamegraph.FlamebearerMetadata {
	return flamegraph.FlamebearerMetadata{
		Format:     "single",
		SpyName:    "",
		SampleRate: 100,
		Units:      "samples",
		Name:       "cpu",
	}
}

// pyroscopeTimeline returns a minimal timeline so the response matches Pyroscope (single bucket with total).
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

// extractStartTimeFromChunks returns the earliest timeUnixNano from chunks as Unix seconds, or 0 if none found.
func extractStartTimeFromChunks(chunks [][]byte) int64 {
	var unmarshal = protojson.UnmarshalOptions{DiscardUnknown: true}
	var minNano int64
	for _, b := range chunks {
		req := &pprofileotlp.ExportProfilesServiceRequest{}
		if unmarshal.Unmarshal(b, req) != nil {
			continue
		}
		for _, rp := range req.ResourceProfiles {
			for _, sp := range rp.ScopeProfiles {
				for _, p := range sp.Profiles {
					nano := int64(p.TimeUnixNano)
					if nano > 0 && (minNano == 0 || nano < minNano) {
						minNano = nano
					}
				}
			}
		}
	}
	if minNano == 0 {
		return 0
	}
	return minNano / 1e9
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
