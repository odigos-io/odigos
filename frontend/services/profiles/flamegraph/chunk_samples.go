package flamegraph

// Sample is one profile stack aggregate: frame names root-first and total value (e.g. sample count).
type Sample struct {
	Stack []string
	Value int64
}

// SamplesFromOTLPChunk parses one OTLP protobuf profile chunk and returns stack samples via
// decodeOTLPChunkToSamples. Returns nil if the chunk is empty or unusable.
func SamplesFromOTLPChunk(chunk []byte) []Sample {
	if len(chunk) == 0 {
		return nil
	}
	samples, ok, _ := decodeOTLPChunkToSamples(chunk)
	if !ok || len(samples) == 0 {
		return nil
	}
	return samples
}
