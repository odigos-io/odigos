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

// OdigosConfiguration defines the desired state of OdigosConfiguration
type OdigosConfiguration struct {
	ConfigVersion                    int                            `json:"configVersion"`
	TelemetryEnabled                 bool                           `json:"telemetryEnabled,omitempty"`
	OpenshiftEnabled                 bool                           `json:"openshiftEnabled,omitempty"`
	IgnoredNamespaces                []string                       `json:"ignoredNamespaces,omitempty"`
	IgnoredContainers                []string                       `json:"ignoredContainers,omitempty"`
	Psp                              bool                           `json:"psp,omitempty"`
	ImagePrefix                      string                         `json:"imagePrefix,omitempty"`
	SkipWebhookIssuerCreation        bool                           `json:"skipWebhookIssuerCreation,omitempty"`
	CollectorGateway                 *CollectorGatewayConfiguration `json:"collectorGateway,omitempty"`
	CollectorNode                    *CollectorNodeConfiguration    `json:"collectorNode,omitempty"`
	Profiles                         []ProfileName                  `json:"profiles,omitempty"`
	AllowConcurrentAgents            *bool                          `json:"allowConcurrentAgents,omitempty"`
	UiMode                           UiMode                         `json:"uiMode,omitempty"`
	UiPaginationLimit                int                            `json:"uiPaginationLimit,omitempty"`
	UiRemoteUrl                      string                         `json:"uiRemoteUrl,omitempty"`
	CentralBackendURL                string                         `json:"centralBackendURL,omitempty"`
	ClusterName                      string                         `json:"clusterName,omitempty"`
	MountMethod                      *MountMethod                   `json:"mountMethod,omitempty"`
	CustomContainerRuntimeSocketPath string                         `json:"customContainerRuntimeSocketPath,omitempty"`
	AgentEnvVarsInjectionMethod      *EnvInjectionMethod            `json:"agentEnvVarsInjectionMethod,omitempty"`
	UserInstrumentationEnvs          *UserInstrumentationEnvs       `json:"UserInstrumentationEnvs,omitempty"`
	NodeSelector                     map[string]string              `json:"nodeSelector,omitempty"`
	KarpenterEnabled                 *bool                          `json:"karpenterEnabled,omitempty"`
	Rollout                          *RolloutConfiguration          `json:"rollout,omitempty"`
	RollbackDisabled                 *bool                          `json:"rollbackDisabled,omitempty"`
	RollbackGraceTime                string                         `json:"rollbackGraceTime,omitempty"`
	RollbackStabilityWindow          string                         `json:"rollbackStabilityWindow,omitempty"`
	Oidc                             *OidcConfiguration             `json:"oidc,omitempty"`
	OdigletHealthProbeBindPort       int                            `json:"odigletHealthProbeBindPort,omitempty"`
}
