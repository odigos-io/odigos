package flamegraph

// ChunkTransformRoute describes how SamplesFromOTLPChunk produced stack samples.
type ChunkTransformRoute string

const (
	// RoutePyroscopeOTLP uses Grafana Pyroscope ingest logic: protojson → ConvertOtelToGoogle → pprof stacks.
	RoutePyroscopeOTLP ChunkTransformRoute = "pyroscope-otlp"
	// RouteError means the chunk could not be converted (same constraints as Pyroscope OTLP ingest).
	RouteError ChunkTransformRoute = "error"
)

// ChunkTransformStats is per-chunk diagnostics for logging and debug JSON.
type ChunkTransformStats struct {
	Route               ChunkTransformRoute
	ByteLen             int
	PyroscopeFailReason string
	SampleCount         int
}

// SamplesFromOTLPChunk is the single entry point for turning one stored OTLP JSON blob into stack samples.
// It uses the same conversion as Grafana Pyroscope ingest (github.com/grafana/pyroscope/pkg/ingester/otlp).
// There is no alternate JSON parser: invalid OTLP profile JSON or conversion failures yield RouteError.
func SamplesFromOTLPChunk(chunk []byte) ([]Sample, ChunkTransformStats) {
	st := ChunkTransformStats{ByteLen: len(chunk)}
	if len(chunk) == 0 {
		st.Route = RouteError
		st.PyroscopeFailReason = "empty_chunk"
		bpFlamef("chunk→samples: empty chunk")
		return nil, st
	}

	samples, ok, reason := tryPyroscopeOTLP(chunk)
	if ok && len(samples) > 0 {
		st.Route = RoutePyroscopeOTLP
		st.SampleCount = len(samples)
		bpFlamef("chunk→samples: route=%s bytes=%d samples=%d (ConvertOtelToGoogle)", RoutePyroscopeOTLP, len(chunk), len(samples))
		return samples, st
	}
	if reason == "" {
		reason = "unknown"
	}
	st.Route = RouteError
	st.PyroscopeFailReason = reason
	bpFlamef("chunk→samples: route=%s bytes=%d reason=%s", RouteError, len(chunk), reason)
	return nil, st
}
