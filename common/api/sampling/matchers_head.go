package sampling

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
// +kubebuilder:object:generate=true
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
// +kubebuilder:object:generate=true
type HeadSamplingHttpClientOperationMatcher struct {
	// match server address exactly (e.g. collector.my.vendor.com)
	ServerAddress string `json:"serverAddress,omitempty"`
	// match templated path exactly
	TemplatedPath string `json:"templatedPath,omitempty"`
	// match preffix of templated path
	TemplatedPathPrefix string `json:"templatedPathPrefix,omitempty"`
	// match method exactly, can be empty to match any method
	Method string `json:"method,omitempty"`
}
