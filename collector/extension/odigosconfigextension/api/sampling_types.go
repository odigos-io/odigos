// Package api holds mirror types for Odigos CRD specs, used to parse InstrumentationConfig
// without depending on the full api module.
//
// Types in this file are copied from:
//   github.com/odigos-io/odigos/api/odigos/v1alpha1/sampling_types.go
// Keep in sync when the upstream API changes.

package api

// SourcesScope limits a rule to specific sources (workload, namespace, container, language).
// WorkloadKind and WorkloadLanguage are strings matching k8sconsts.WorkloadKind and common.ProgrammingLanguage.
type SourcesScope struct {
	WorkloadName      string `json:"workloadName,omitempty"`
	WorkloadKind      string `json:"workloadKind,omitempty"`
	WorkloadNamespace string `json:"workloadNamespace,omitempty"`
	ContainerName     string `json:"containerName,omitempty"`
	WorkloadLanguage  string `json:"workloadLanguage,omitempty"`
}

// NoisyOperationHttpServerMatcher matches http server operations for noisy-operation rules.
type NoisyOperationHttpServerMatcher struct {
	Route       string `json:"route,omitempty"`
	RoutePrefix string `json:"routePrefix,omitempty"`
	Method      string `json:"method,omitempty"`
}

// NoisyOperationHttpClientMatcher matches http client operations for noisy-operation rules.
type NoisyOperationHttpClientMatcher struct {
	ServerAddress string `json:"serverAddress,omitempty"`
	UrlPath       string `json:"urlPath,omitempty"`
	Method        string `json:"method,omitempty"`
}

// NoisyOperations defines endpoints considered "noise" (e.g. health checks, /metrics) to drop or sample aggressively.
type NoisyOperations struct {
	SourceScopes     []SourcesScope                   `json:"sourceScopes,omitempty"`
	HttpServer       *NoisyOperationHttpServerMatcher `json:"httpServer,omitempty"`
	HttpClient       *NoisyOperationHttpClientMatcher `json:"httpClient,omitempty"`
	PercentageAtMost *float64                         `json:"percentageAtMost,omitempty"`
	Notes            string                           `json:"notes,omitempty"`
}

// HttpServerOperationMatcher matches http server spans by route and method.
type HttpServerOperationMatcher struct {
	Route       string `json:"route,omitempty"`
	RoutePrefix string `json:"routePrefix,omitempty"`
	Method      string `json:"method,omitempty"`
}

// KafkaOperationMatcher matches Kafka consumer or producer operations by topic.
type KafkaOperationMatcher struct {
	KafkaTopic string `json:"kafkaTopic,omitempty"`
}

// OperationMatcher limits a rule to specific operations (http server, kafka, etc.).
type OperationMatcher struct {
	HttpServer    *HttpServerOperationMatcher `json:"httpServer,omitempty"`
	KafkaConsumer *KafkaOperationMatcher      `json:"kafkaConsumer,omitempty"`
	KafkaProducer *KafkaOperationMatcher      `json:"kafkaProducer,omitempty"`
}

// HighlyRelevantOperation defines operations with high observability value; matching traces are kept.
type HighlyRelevantOperation struct {
	SourceScopes      []SourcesScope    `json:"sourceScopes,omitempty"`
	Error             bool              `json:"error,omitempty"`
	DurationAtLeastMs *int              `json:"durationAtLeastMs,omitempty"`
	Operation         *OperationMatcher `json:"operation,omitempty"`
	PercentageAtLeast *float64          `json:"percentageAtLeast,omitempty"`
	Notes             string            `json:"notes,omitempty"`
}

// CostReductionRule limits sampling for cost reduction (at most the given percentage).
type CostReductionRule struct {
	SourceScopes     []SourcesScope    `json:"sourceScopes,omitempty"`
	Operation        *OperationMatcher `json:"operation,omitempty"`
	PercentageAtMost float64           `json:"percentageAtMost"`
	Notes            string            `json:"notes,omitempty"`
}
