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

	// match grpc server operation (trace that starts with an incoming grpc call)
	GrpcServer *HeadSamplingGrpcServerOperationMatcher `json:"grpcServer,omitempty"`

	// match grpc client operation (trace that starts with an outgoing grpc call).
	GrpcClient *HeadSamplingGrpcClientOperationMatcher `json:"grpcClient,omitempty"`
}

// match http server operations for noisy operations matching (only attributes available at span start time)
// +kubebuilder:object:generate=true
type HeadSamplingHttpServerOperationMatcher struct {

	// match route exactly
	Route string `json:"route,omitempty"`

	// match prefix of route
	RoutePrefix string `json:"routePrefix,omitempty"`

	// match method exactly, can be empty to match any method
	Method string `json:"method,omitempty"`

	// match url query parameters.
	// ignored if empty.
	// if set, all the specified query parameters matchers must match for the operation to be matched.
	// any query param in the request url that is not specified in the matchers will not be considered for the matching.
	QueryParams []QueryParamMatcher `json:"queryParams,omitempty"`
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

// match grpc server operations for noisy operations matching (only attributes available at span start time).
// Method and Service are matched independently as users would think of them: the bare method name
// (e.g. "ListItems") and the fully-qualified service name (e.g. "my.example.com.InventoryService").
// Odigos agents normalize the OTel rpc.method / rpc.service attributes so both can be matched directly.
// +kubebuilder:object:generate=true
type HeadSamplingGrpcServerOperationMatcher struct {
	// match the bare gRPC method name exactly (e.g. "ListItems"). leave empty to match any method.
	Method string `json:"method,omitempty"`
	// match the fully-qualified gRPC service name exactly (e.g. "my.example.com.InventoryService").
	// leave empty to match any service. setting only Service is the canonical way to scope a rule
	// to every method of a single gRPC service.
	Service string `json:"service,omitempty"`
}

// match grpc client operations for noisy operations matching (only attributes available at span start time).
// can be used to filter out outgoing grpc requests for other agents calling home or exporting data.
// +kubebuilder:object:generate=true
type HeadSamplingGrpcClientOperationMatcher struct {
	// match the bare gRPC method name exactly (e.g. "ListItems"). leave empty to match any method.
	Method string `json:"method,omitempty"`
	// match the fully-qualified gRPC service name exactly (e.g. "my.example.com.InventoryService").
	// leave empty to match any service.
	Service string `json:"service,omitempty"`
	// match server address exactly (e.g. collector.my.vendor.com)
	ServerAddress string `json:"serverAddress,omitempty"`
}
