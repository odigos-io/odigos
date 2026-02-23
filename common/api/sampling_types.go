package api

// match operations for tail sampling with the full context of the span.
// this is used by sampling rules to limit it only to specific operations.
// if the rule matches a sapn, the behavior is determined by the rule itself.
// +kubebuilder:object:generate=true
type TailSamplingOperationMatcher struct {

	// match http server operations in a generic way.
	HttpServer *HttpServerOperationMatcher `json:"httpServer,omitempty"`

	// match kafka consumer operations (consume spans)
	KafkaConsumer *KafkaOperationMatcher `json:"kafkaConsumer,omitempty"`

	// match kafka producer operations (produce spans)
	KafkaProducer *KafkaOperationMatcher `json:"kafkaProducer,omitempty"`
}

// can match a specific operation for head sampling.
// head sampling has access only to the attributes available at span start time,
// and sampling decisions can only be based on the root span of the trace.
// +kubebuilder:object:generate=true
type HeadSamplingOperationMatcher struct {
	// match http server operation (trace that starts with an http endpoint)
	HttpServer *HeadSamplingHttpServerOperationMatcher `json:"httpServer,omitempty"`

	// match http client operation (trace that starts with an http client operation)
	// common for agents that are internally calling home over http, and exporting data.
	HttpClient *HeadSamplingHttpClientOperationMatcher `json:"httpClient,omitempty"`
}

// match http server operations for noisy operations matching (only attributes available at span start time)
type HeadSamplingHttpServerOperationMatcher struct {
	// match route exactly
	Route string `json:"route,omitempty"`
	// match preffix of route
	RoutePrefix string `json:"routePrefix,omitempty"`
	// match method exactly, can be empty to match any method
	Method string `json:"method,omitempty"`
}

// match http client operations for noisy operations matching (only attributes available at span start time)
// can be used to filter out outgoing http requests for other agents calling home or exporting data.
type HeadSamplingHttpClientOperationMatcher struct {
	// match server address exactly (e.g. collector.my.vendor.com)
	ServerAddress string `json:"serverAddress,omitempty"`
	// match url path exactly (e.g. /api/v1/metrics)
	UrlPath string `json:"urlPath,omitempty"`
	// match method exactly, can be empty to match any method
	Method string `json:"method,omitempty"`
}

// match only http server spans for a specific endpoint.
// user can specify route and method to match, and limit a sampling instruction to only this operation.
type HttpServerOperationMatcher struct {

	// a specific exact match http route
	Route string `json:"route,omitempty"`

	// any route that starts with a specific prefix
	RoutePrefix string `json:"routePrefix,omitempty"`

	// optionally limit to specific http method
	Method string `json:"method,omitempty"`
}

// match a kafka consumer or producer operation for a specific topic.
type KafkaOperationMatcher struct {

	// the topic name to match.
	// if left empty, all topics are matched.
	KafkaTopic string `json:"kafkaTopic,omitempty"`
}
