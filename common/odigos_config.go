package common

type ProfileName string

// "normal" is deprecated. Kept here in the enum for backwards compatibility with operator CRD.
// +kubebuilder:validation:Enum=default;readonly;normal
type UiMode string

const (
	UiModeDefault  UiMode = "default"
	UiModeReadonly UiMode = "readonly"
)

type CollectorNodeConfiguration struct {
	// The port to use for exposing the collector's own metrics as a prometheus endpoint.
	// This can be used to resolve conflicting ports when a collector is using the host network.
	CollectorOwnMetricsPort int32 `json:"collectorOwnMetricsPort,omitempty"`

	// RequestMemoryMiB is the memory request for the node collector daemonset.
	// it will be embedded in the daemonset as a resource request of the form "memory: <value>Mi"
	// default value is 250Mi
	RequestMemoryMiB int `json:"requestMemoryMiB,omitempty"`

	// LimitMemoryMiB is the memory limit for the node collector daemonset.
	// it will be embedded in the daemonset as a resource limit of the form "memory: <value>Mi"
	// default value is 2x the memory request.
	LimitMemoryMiB int `json:"limitMemoryMiB,omitempty"`

	// RequestCPUm is the CPU request for the node collector daemonset.
	// it will be embedded in the daemonset as a resource request of the form "cpu: <value>m"
	// default value is 250m
	RequestCPUm int `json:"requestCPUm,omitempty"`

	// LimitCPUm is the CPU limit for the node collector daemonset.
	// it will be embedded in the daemonset as a resource limit of the form "cpu: <value>m"
	// default value is 500m
	LimitCPUm int `json:"limitCPUm,omitempty"`

	// this parameter sets the "limit_mib" parameter in the memory limiter configuration for the node collector.
	// it is the hard limit after which a force garbage collection will be performed.
	// if not set, it will be 50Mi below the memory request.
	MemoryLimiterLimitMiB int `json:"memoryLimiterLimitMiB,omitempty"`

	// this parameter sets the "spike_limit_mib" parameter in the memory limiter configuration for the node collector.
	// note that this is not the processor soft limit, but the diff in Mib between the hard limit and the soft limit.
	// if not set, this will be set to 20% of the hard limit (so the soft limit will be 80% of the hard limit).
	MemoryLimiterSpikeLimitMiB int `json:"memoryLimiterSpikeLimitMiB,omitempty"`

	// the GOMEMLIMIT environment variable value for the node collector daemonset.
	// this is when go runtime will start garbage collection.
	// if not specified, it will be set to 80% of the hard limit of the memory limiter.
	GoMemLimitMib int `json:"goMemLimitMiB,omitempty"`

	// Odigos will by default attempt to collect logs from '/var/log' on each k8s node.
	// Sometimes, this directory is actually a symlink to another directory.
	// In this case, for logs collection to work, we need to add a mount to the target directory.
	// This field is used to specify this target directory in these cases.
	// A common target directory is '/mnt/var/log'.
	K8sNodeLogsDirectory string `json:"k8sNodeLogsDirectory,omitempty"`

	// EnableDataCompression is a feature that allows you to enable data compression before sending data to the Gateway collector.
	// It is disabled by default and can be enabled by setting the enabled flag to true.
	EnableDataCompression *bool `json:"enableDataCompression,omitempty"`
}

type CollectorGatewayConfiguration struct {
	// MinReplicas is the number of replicas for the cluster gateway collector deployment.
	// Also set the minReplicas for the HPA to this value.
	MinReplicas int `json:"minReplicas,omitempty"`

	// MaxReplicas set the maxReplicas for the HPA to this value.
	MaxReplicas int `json:"maxReplicas,omitempty"`

	// RequestMemoryMiB is the memory request for the cluster gateway collector deployment.
	// it will be embedded in the deployment as a resource request of the form "memory: <value>Mi"
	// default value is 500Mi
	RequestMemoryMiB int `json:"requestMemoryMiB,omitempty"`

	// LimitMemoryMiB is the memory limit for the cluster gateway collector deployment.
	// it will be embedded in the deployment as a resource limit of the form "memory: <value>Mi"
	// default value is 1.25 the memory request.
	LimitMemoryMiB int `json:"limitMemoryMiB,omitempty"`

	// RequestCPUm is the CPU request for the cluster gateway collector deployment.
	// it will be embedded in the deployment as a resource request of the form "cpu: <value>m"
	// default value is 500m
	RequestCPUm int `json:"requestCPUm,omitempty"`

	// LimitCPUm is the CPU limit for the cluster gateway collector deployment.
	// it will be embedded in the deployment as a resource limit of the form "cpu: <value>m"
	// default value is 1000m
	LimitCPUm int `json:"limitCPUm,omitempty"`

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

	// ServiceGraphDisabled is a feature that allows you to visualize the service graph of your application.
	// It is enabled by default and can be disabled by setting the disabled flag to true.
	ServiceGraphDisabled *bool `json:"serviceGraphDisabled,omitempty"`

	// ClusterMetricsEnabled is a feature that allows you to enable the cluster metrics.
	// https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/k8sclusterreceiver
	// It is disabled by default and can be enabled by setting the enabled flag to true.
	// This feature is only available when metrics destination is configured.
	ClusterMetricsEnabled *bool `json:"clusterMetricsEnabled,omitempty"`

	// for destinations that uses https for exporting data, this value can be used to set the value for the https proxy.
	HttpsProxyAddress *string `json:"httpsProxyAddress,omitempty"`
}
type UserInstrumentationEnvs struct {
	Languages map[ProgrammingLanguage]LanguageConfig `json:"languages,omitempty"`
}

// Struct to represent configuration for each language
type LanguageConfig struct {
	Enabled bool              `json:"enabled"`
	EnvVars map[string]string `json:"env,omitempty"`
}

type RolloutConfiguration struct {

	// When set to true, Odigos will never trigger a rollout for workloads when instrumenting or uninstrumenting.
	// It is expected that users will manually trigger a rollout to apply the changes when needed,
	// but it gives them the option to control the process.
	// Any new pods that are created after agent is enabled or disabled (via manual rollout or auto scaling)
	// will be have agent injection regardless of this setting.
	// This setting does not control manual rollouts executed from the UI or via the API.
	// Any additional configuration regarding rollouts and rollbacks are ignored when this is set to true.
	AutomaticRolloutDisabled *bool `json:"automaticRolloutDisabled"`
}

type OidcConfiguration struct {
	// The URL of the OIDC tenant (e.g. "https://abc-123.okta.com").
	TenantUrl string `json:"tenantUrl,omitempty"`

	// The client ID of the OIDC application.
	ClientId string `json:"clientId,omitempty"`

	// The client secret of the OIDC application.
	ClientSecret string `json:"clientSecret,omitempty"`
}

// +kubebuilder:object:generate=true
type MetricsSourceSpanMetricsConfiguration struct {

	// control beahvior for when to collect span metrics.
	// - true - span metrics will never be collected, even if destinations require it.
	// - false / nil - span metrics will be collected if destinations require it (can be opt-in and out in destination level).
	Disabled *bool `json:"disabled,omitempty"`

	// time interval for flusing metrics (format: 15s, 1m etc). defaults: 60s (one minute).
	Interval string `json:"interval,omitempty"`

	// additional dimensions to add to the span metrics.
	// these are span attributes that you want to convert to metrics attributes during collection.
	// the values of the attributes must have low cardinality.
	// it can increase the number of series in the destination.
	// some dimensions are already added by default, regardless of this setting.
	AdditionalDimensions []string `json:"additionalDimensions,omitempty"`

	// if true, histogram metrics will not be collected.
	HistogramDisabled bool `json:"histogramDisabled,omitempty"`

	// explicit buckets list for the histogram metrics.
	// format is duration string (`1us`, `2ms`, `3s`, `4m`, `5h`, `6d` etc).
	// if not set, the default buckets list will be used.
	// the buckets must be in ascending order.
	// example: ["100us", "1ms", "2ms", "6ms", "10ms", "100ms", "250ms"]
	// Default value when unset: ["2ms", "4ms", "6ms", "8ms", "10ms", "50ms", "100ms", "200ms", "400ms", "800ms", "1s", "1400ms", "2s", "5s", "10s", "15s"]
	// notice that more granular buckets are recommended for better precision but costs more since more metric series are produced.
	ExplicitHistogramBuckets []string `json:"histogramBuckets,omitempty"`
}

// +kubebuilder:object:generate=true
type MetricsSourceHostMetricsConfiguration struct {

	// control beahvior for when to collect host metrics.
	// - true - host metrics will never be collected, even if destinations require it.
	// - false / nil - host metrics will be collected if destinations require it (can be opt-in and out in destination level).
	Disabled *bool `json:"disabled,omitempty"`

	// time interval for scraping metrics (format: 15s, 1m etc). defaults: 10s.
	Interval string `json:"interval,omitempty"`
}

// +kubebuilder:object:generate=true
type MetricsSourceKubeletStatsConfiguration struct {

	// control beahvior for when to collect kubelet stats.
	// - true - kubelet stats will never be collected, even if destinations require it.
	// - false / nil - kubelet stats will be collected if destinations require it (can be opt-in and out in destination level).
	Disabled *bool `json:"disabled,omitempty"`

	// time interval for scraping metrics (format: 15s, 1m etc). defaults: 10s.
	Interval string `json:"interval,omitempty"`
}

// +kubebuilder:object:generate=true
type MetricsSourceConfiguration struct {

	// configuration for span metrics.
	SpanMetrics *MetricsSourceSpanMetricsConfiguration `json:"spanMetrics,omitempty"`

	// configuration for host metrics.
	HostMetrics *MetricsSourceHostMetricsConfiguration `json:"hostMetrics,omitempty"`

	// configuration for kubelet stats.
	KubeletStats *MetricsSourceKubeletStatsConfiguration `json:"kubeletStats,omitempty"`
}

// OdigosConfiguration defines the desired state of OdigosConfiguration
type OdigosConfiguration struct {
	ConfigVersion             int                            `json:"configVersion" yaml:"configVersion"`
	TelemetryEnabled          bool                           `json:"telemetryEnabled,omitempty" yaml:"telemetryEnabled"`
	OpenshiftEnabled          bool                           `json:"openshiftEnabled,omitempty" yaml:"openshiftEnabled"`
	IgnoredNamespaces         []string                       `json:"ignoredNamespaces,omitempty" yaml:"ignoredNamespaces"`
	IgnoredContainers         []string                       `json:"ignoredContainers,omitempty" yaml:"ignoredContainers"`
	Psp                       bool                           `json:"psp,omitempty" yaml:"psp"`
	ImagePrefix               string                         `json:"imagePrefix,omitempty" yaml:"imagePrefix"`
	SkipWebhookIssuerCreation bool                           `json:"skipWebhookIssuerCreation,omitempty" yaml:"skipWebhookIssuerCreation"`
	CollectorGateway          *CollectorGatewayConfiguration `json:"collectorGateway,omitempty" yaml:"collectorGateway"`
	CollectorNode             *CollectorNodeConfiguration    `json:"collectorNode,omitempty" yaml:"collectorNode"`
	Profiles                  []ProfileName                  `json:"profiles,omitempty" yaml:"profiles"`
	AllowConcurrentAgents     *bool                          `json:"allowConcurrentAgents,omitempty" yaml:"allowConcurrentAgents"`
	UiMode                    UiMode                         `json:"uiMode,omitempty" yaml:"uiMode"`
	UiPaginationLimit         int                            `json:"uiPaginationLimit,omitempty" yaml:"uiPaginationLimit"`
	UiRemoteUrl               string                         `json:"uiRemoteUrl,omitempty" yaml:"uiRemoteUrl"`
	CentralBackendURL         string                         `json:"centralBackendURL,omitempty" yaml:"centralBackendURL"`
	ClusterName               string                         `json:"clusterName,omitempty" yaml:"clusterName"`
	MountMethod               *MountMethod                   `json:"mountMethod,omitempty" yaml:"mountMethod"`
	//nolint:lll // CustomContainerRuntimeSocketPath line is long due to struct tag requirements
	CustomContainerRuntimeSocketPath  string                      `json:"customContainerRuntimeSocketPath,omitempty" yaml:"customContainerRuntimeSocketPath"`
	AgentEnvVarsInjectionMethod       *EnvInjectionMethod         `json:"agentEnvVarsInjectionMethod,omitempty" yaml:"agentEnvVarsInjectionMethod"`
	UserInstrumentationEnvs           *UserInstrumentationEnvs    `json:"userInstrumentationEnvs,omitempty" yaml:"userInstrumentationEnvs"`
	NodeSelector                      map[string]string           `json:"nodeSelector,omitempty" yaml:"nodeSelector"`
	KarpenterEnabled                  *bool                       `json:"karpenterEnabled,omitempty" yaml:"karpenterEnabled"`
	Rollout                           *RolloutConfiguration       `json:"rollout,omitempty" yaml:"rollout"`
	RollbackDisabled                  *bool                       `json:"rollbackDisabled,omitempty" yaml:"rollbackDisabled"`
	RollbackGraceTime                 string                      `json:"rollbackGraceTime,omitempty" yaml:"rollbackGraceTime"`
	RollbackStabilityWindow           string                      `json:"rollbackStabilityWindow,omitempty" yaml:"rollbackStabilityWindow"`
	Oidc                              *OidcConfiguration          `json:"oidc,omitempty" yaml:"oidc"`
	OdigletHealthProbeBindPort        int                         `json:"odigletHealthProbeBindPort,omitempty" yaml:"odigletHealthProbeBindPort"`
	GoAutoOffsetsCron                 string                      `json:"goAutoOffsetsCron,omitempty" yaml:"goAutoOffsetsCron"`
	GoAutoOffsetsMode                 string                      `json:"goAutoOffsetsMode,omitempty" yaml:"goAutoOffsetsMode"`
	ClickhouseJsonTypeEnabledProperty *bool                       `json:"clickhouseJsonTypeEnabled,omitempty"`
	CheckDeviceHealthBeforeInjection  *bool                       `json:"checkDeviceHealthBeforeInjection,omitempty"`
	ResourceSizePreset                string                      `json:"resourceSizePreset,omitempty" yaml:"resourceSizePreset"`
	WaspEnabled                       *bool                       `json:"waspEnabled,omitempty" yaml:"waspEnabled"`
	MetricsSources                    *MetricsSourceConfiguration `json:"metricsSources,omitempty" yaml:"metricsSources"`

	AllowedTestConnectionHosts []string `json:"allowedTestConnectionHosts,omitempty" yaml:"allowedTestConnectionHosts"`
}
