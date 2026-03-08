package sampling

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

// match only http server spans for a specific endpoint.
// user can specify route and method to match, and limit a sampling instruction to only this operation.
// +kubebuilder:object:generate=true
type HttpServerOperationMatcher struct {

	// a specific exact match http route
	Route string `json:"route,omitempty"`

	// any route that starts with a specific prefix
	RoutePrefix string `json:"routePrefix,omitempty"`

	// optionally limit to specific http method
	Method string `json:"method,omitempty"`
}

// match a kafka consumer or producer operation for a specific topic.
// +kubebuilder:object:generate=true
type KafkaOperationMatcher struct {

	// the topic name to match.
	// if left empty, all topics are matched.
	KafkaTopic string `json:"kafkaTopic,omitempty"`
}
