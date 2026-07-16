package agentsignalconfig

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/common/api/sampling"
)

// random id generator is the default, and most common.
// it creates span ids and trace ids using random bytes.
// It has no configuration.
type IdGeneratorRandomConfig struct{}

// trace id includes timestamp, source id byte, and random number bytes.
// this id generator can be leveraged by databases to do efficient indexing.
type IdGeneratorTimedWallConfig struct {
	// sourceId is a number between 0-255 (8 bits) written into the 8th byte of the trace id.
	// if timedWall is specified, the sourceId is required.
	SourceId uint8 `json:"sourceId"`
}

// id generator configuration for the traces
// +kubebuilder:object:generate=true
type IdGeneratorConfig struct {
	Random    *IdGeneratorRandomConfig    `json:"random,omitempty"`
	TimedWall *IdGeneratorTimedWallConfig `json:"timedWall,omitempty"`
}

// +kubebuilder:object:generate=true
type AgentSpanMetricsConfig struct {
	// additional dimensions to add for the span metrics.
	// for example, if you add `http.method` to the dimensions,
	// then the span metrics data points will include the `http.method` in the attributes,
	// and different values of `http.method` will be aggregated into different time series.
	Dimensions []string `json:"dimensions,omitempty"`

	// time interval in miliseconds for flushing the span metrics.
	// defaults: 60000 (60 seconds, 1 minute)
	IntervalMs int `json:"intervalMs,omitempty"`

	// explicit buckets list for the histogram metrics in ms
	HistogramBucketsMs []int `json:"histogramBucketsMs,omitempty"`
}

// all "traces" related configuration for an agent running on any process in a specific container.
// The presence of this struct (as opposed to nil) means that trace collection is enabled for this container.
// +kubebuilder:object:generate=true
type AgentTracesConfig struct {
	// id generator configuration for the traces.
	// if not specified, the default random id generator will be used.
	IdGenerator *IdGeneratorConfig `json:"idGenerator,omitempty"`

	// A list of URL templatization configurations to be applied to the traces.
	UrlTemplatization *actions.UrlTemplatizationConfig `json:"urlTemplatization,omitempty"`

	// Configuration for headers collection. If not specified, no headers will be collected.
	HeadersCollection *instrumentationrules.HttpHeadersCollection `json:"headersCollection,omitempty"`

	// HeadSamplingConfig is a set sampling rules.
	// This config currently only applies to root spans.
	// In the Future we might add another level of configuration base on the parent span (ParentBased Sampling)
	HeadSampling *sampling.HeadSamplingConfig `json:"headSampling,omitempty"`

	// Configuration for span renamer.
	SpanRenamer *actions.SpanRenamerConfig `json:"spanRenamer,omitempty"`

	// configuration for payload collection for this container.
	PayloadCollection *instrumentationrules.PayloadCollection `json:"payloadCollection,omitempty"`

	// configuration for code attributes collection for this container.
	CodeAttributes *instrumentationrules.CodeAttributes `json:"codeAttributes,omitempty"`

	// configuration for how verbose the trace should be - e.g. which spans should be included / excluded.
	TraceVerbosity *instrumentationrules.TraceVerbosity `json:"traceVerbosity,omitempty"`

	// custom instrumentation probes for this container.
	CustomInstrumentations *instrumentationrules.CustomInstrumentations `json:"customInstrumentations,omitempty"`
}

// all "metrics" related configuration for an agent running on any process in a specific container.
// The presence of this struct (as opposed to nil) means that metrics collection is enabled for this container.
// +kubebuilder:object:generate=true
type AgentMetricsConfig struct {
	// if not nil, it means agent should report span metrics,
	// calculated directly in the agent.
	// this is most accurate as it includes any sampled spans,
	// and is not affected if spans are dropped anywhere in the pipeline.
	SpanMetrics *AgentSpanMetricsConfig `json:"spanMetrics,omitempty"`

	// if not nil, it means agent should report runtime metrics,
	// such as JVM metrics for Java applications.
	// these metrics provide insights into the runtime environment performance.
	RuntimeMetrics *common.MetricsSourceAgentRuntimeMetricsConfiguration `json:"runtimeMetrics,omitempty"`

	// NetworkMetrics enables network flow and TCP stats metrics for this container.
	NetworkMetrics *instrumentationrules.NetworkMetricsConfig `json:"networkMetrics,omitempty"`
}

// all "logs" related configuration for an agent running on any process in a specific container.
// The presence of this struct (as opposed to nil) means that logs collection is enabled for this container.
// +kubebuilder:object:generate=true
type AgentLogsConfig struct {
	// if set, switches the logs pipeline to use the eBPF receiver instead of filelog.
	EbpfLogCapture *instrumentationrules.EbpfLogCapture `json:"ebpfLogCapture,omitempty"`
}
