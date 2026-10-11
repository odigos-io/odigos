package sampling

// +kubebuilder:object:generate=true
type HeadSamplingConfig struct {
	// If true, the sampling decision will be made in dry-run mode.
	// When dry-run is enabled, the sampling decision will be made but the trace will not be dropped.
	// This is useful to evaluate the sampling decision before actually committing to it.
	// 2 additional attributes will be set on the spans:
	// - odigos.sampling.dry_run: true
	// - odigos.sampling.dry_run.kept: true if the trace would have been kept, false if it would have been dropped.
	DryRun bool `json:"dryRun,omitempty"`

	// Controls the tradeoff between metric accuracy and resource usage.
	// Determines how sampling affects which spans are used to compute metrics.
	// Possible values:
	//   - "sampled-spans-only" (default): metrics are computed only from sampled spans.
	//     Unsampled spans are dropped early, resulting in lower resource usage but reduced accuracy.
	//   - "all-spans": metrics are computed from all spans, regardless of sampling.
	//     Unsampled spans are forwarded for metric computation and dropped later in the pipeline,
	//     resulting in higher accuracy at the cost of increased resource usage.
	SpanMetricsMode SpanMetricsMode `json:"spanMetricsMode,omitempty"`

	// Noisy operations are categories of matchers that are used on the root span.
	// If match, the fraction is used to determine the sampling decision for the entire trace.
	// If multiple noisy operations match, the lowest fraction is used.
	NoisyOperations []NoisyOperation `json:"noisyOperations,omitempty"`

	// +kubebuilder:default:=1
	//
	// Deprecated: do not use. Will be removed once python and node migration is complete.
	// Use NoisyOperations instead.
	FallbackFraction float64 `json:"fallbackFraction,omitempty"`
}
