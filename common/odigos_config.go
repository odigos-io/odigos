package common

type ProfileName string

type CollectorNodeConfiguration struct {
	// The port to use for exposing the collector's own metrics as a prometheus endpoint.
	// This can be used to resolve conflicting ports when a collector is using the host network.
	CollectorOwnMetricsPort int32 `json:"collectorOwnMetricsPort,omitempty"`
}

type CollectorGatewayConfiguration struct {
	// RequestMemoryMiB is the memory request for the cluster gateway collector deployment.
	// it will be embedded in the deployment as a resource request of the form "memory: <value>Mi"
	// default value is 500Mi
	RequestMemoryMiB int `json:"requestMemoryMiB,omitempty"`

	// this parameter sets the "limit_mib" parameter in the memory limiter configuration for the collector gateway.
	// it is the hard limit after which a force garbage collection will be performed.
	// if not set, it will be 50Mi below the memory request.
	MemoryLimiterLimitMiB int `json:"memoryLimiterLimitMiB,omitempty"`

	// this parameter sets the "spike_limit_mib" parameter in the memory limiter configuration for the collector gateway.
	// note that this is not the processor soft limit, but the diff in Mib between the hard limit and the soft limit.
	// if not set, this will be set to 20% of the hard limit (so the soft limit will be 80% of the hard limit).
	MemoryLimiterSpikeLimitMiB int `json:"memoryLimiterSpikeLimitMiB,omitempty"`

	// the GOMEMLIMIT environment variable value for the collector gateway deployment.
	// this is when go runtime will start garbage collection.
	// if not specified, it will be set to 80% of the hard limit of the memory limiter.
	GoMemLimitMib int `json:"goMemLimitMiB,omitempty"`
}

// OdigosConfiguration defines the desired state of OdigosConfiguration
type OdigosConfiguration struct {
	ConfigVersion     int                             `json:"configVersion"`
	TelemetryEnabled  bool                            `json:"telemetryEnabled,omitempty"`
	OpenshiftEnabled  bool                            `json:"openshiftEnabled,omitempty"`
	IgnoredNamespaces []string                        `json:"ignoredNamespaces,omitempty"`
	IgnoredContainers []string                        `json:"ignoredContainers,omitempty"`
	Psp               bool                            `json:"psp,omitempty"`
	ImagePrefix       string                          `json:"imagePrefix,omitempty"`
	OdigletImage      string                          `json:"odigletImage,omitempty"`
	InstrumentorImage string                          `json:"instrumentorImage,omitempty"`
	AutoscalerImage   string                          `json:"autoscalerImage,omitempty"`
	CollectorGateway  *CollectorGatewayConfiguration  `json:"collectorGateway,omitempty"`
	CollectorNode     *CollectorNodeConfiguration     `json:"collectorNode,omitempty"`
	Profiles          []ProfileName                   `json:"profiles,omitempty"`

	// this is internal currently, and is not exposed on the CLI / helm
	// used for odigos enterprise
	GoAutoIncludeCodeAttributes bool `json:"goAutoIncludeCodeAttributes,omitempty"`
}
