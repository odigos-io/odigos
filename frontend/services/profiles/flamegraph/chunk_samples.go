package flamegraph

// Sample is one profile stack aggregate: frame names root-first and total value (e.g. sample count).
type Sample struct {
	Stack []string
	Value int64
}

// SamplesFromOTLPChunk turns one stored OTLP JSON blob into stack samples using the same conversion
// as Grafana Pyroscope ingest (github.com/grafana/pyroscope/pkg/ingester/otlp).
// Invalid JSON or conversion failures yield nil.
func SamplesFromOTLPChunk(chunk []byte) []Sample {
	if len(chunk) == 0 {
		return nil
	}
	samples, ok, _ := tryPyroscopeOTLP(chunk)
	if !ok || len(samples) == 0 {
		return nil
	}
	return samples
}
