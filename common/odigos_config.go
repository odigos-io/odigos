package common

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/odigos-io/odigos/common/consts"
)

type ConfigField interface {
	ToString()
}

type ConfigBool bool

func (data ConfigBool) ToString() {
	fmt.Printf("%t\n", data)
}

type ConfigBoolPointer struct {
	Value *bool
}

func (data ConfigBoolPointer) ToString() {
	if data.Value == nil {
		fmt.Printf("not set\n")
		return
	}
	fmt.Printf("%v\n", *data.Value)
}

// looks weird but basically all variables that use this ConfigBoolPointer
// all of them are stored as objects in the yaml file
// like this
// allowConcurrentAgents:
// Value: null
func (cbp *ConfigBoolPointer) UnmarshalJSON(data []byte) error {
	// Try simple bool first
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		cbp.Value = &b
		return nil
	}

	// Try object with value
	type wrapper struct {
		Value *bool `json:"value"`
	}
	var w wrapper
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	cbp.Value = w.Value
	return nil
}

type ConfigInt int

func (data ConfigInt) ToString() {
	fmt.Printf("%d\n", data)
}

type ConfigString string

func (data ConfigString) ToString() {
	if data == "" {
		fmt.Println("not set")
		return
	}
	fmt.Printf("%s\n", data)
}

type ConfigStringList []string

func (data ConfigStringList) ToString() {
	if len(data) == 0 {
		fmt.Println("not set")
		return
	}
	fmt.Printf("\n")
	for _, value := range data {
		fmt.Printf("		-%s,\n", value)
	}

}

// no need for a new type, since most of the ones comming up are already types that are structs
// just give them this ToString() function to make it part of the interface
func (data *CollectorNodeConfiguration) ToString() {
	if data == nil {
		fmt.Println("not set")
		return
	}
	var placeholder ConfigString = ConfigString(data.K8sNodeLogsDirectory)
	placeholder.ToString()
}

func (data *MountMethod) ToString() {
	if data == nil {
		fmt.Printf("not set\n")
		return
	}
	fmt.Printf("%v\n", *data)
}

func (data UiMode) ToString() {
	fmt.Printf("%v\n", data)
}

func (data *EnvInjectionMethod) ToString() {
	if data == nil {
		fmt.Printf("not set\n")
		return
	}
	fmt.Printf("%v\n", *data)
}

type ConfigNodeSelector map[string]string

func (data ConfigNodeSelector) ToString() {
	if len(data) == 0 {
		fmt.Printf("not set\n")
		return
	}
	fmt.Printf("\n")
	for key, value := range data {
		fmt.Printf("		- key: %+v, value: %+v\n", key, value)
	}
}

func (data *UserInstrumentationEnvs) ToString() {
	if data == nil {
		fmt.Printf("not set\n")
		return
	} else if len(data.Languages) == 0 {
		fmt.Printf("not set\n")
		return
	}
	fmt.Printf("\n")
	for key, value := range data.Languages {
		fmt.Printf("		- language: %+v, mode: %+v\n", key, value)
	}
}

func (data *RolloutConfiguration) ToString() {
	if data == nil {
		fmt.Printf("not set\n")
		return
	}
	placeholder := ConfigBoolPointer{Value: data.AutomaticRolloutDisabled}
	placeholder.ToString()
}

func (data *CollectorGatewayConfiguration) ToString() {
	if data == nil {
		fmt.Printf("not set\n")
		return
	} else if data.ServiceGraphDisabled == nil {
		fmt.Printf("not set\n")
		return
	}
	placeholder := ConfigBoolPointer{Value: data.ServiceGraphDisabled}
	placeholder.ToString()
}

// i will keep the redefining here in order to differentiate between the three values
// i cannot have 3 ToString() functions for the same struct
type ConfigOidcTenant OidcConfiguration

func (data *ConfigOidcTenant) ToString() {
	if data == nil {
		fmt.Printf("not set\n")
		return
	}
	var placeholder ConfigString = ConfigString(data.TenantUrl)
	placeholder.ToString()
}

type ConfigOidcClientId OidcConfiguration

func (data *ConfigOidcClientId) ToString() {
	if data == nil {
		fmt.Printf("not set\n")
		return
	}
	var placeholder ConfigString = ConfigString(data.ClientId)
	placeholder.ToString()
}

type ConfigOidcClientSecret OidcConfiguration

func (data *ConfigOidcClientSecret) ToString() {
	if data == nil {
		fmt.Printf("not set\n")
		return
	}
	var placeholder ConfigString = ConfigString(data.ClientSecret)
	placeholder.ToString()
}

func makeAMap(config *OdigosConfiguration) map[string]ConfigField {
	displayData := map[string]ConfigField{
		consts.TelemetryEnabledProperty:           config.TelemetryEnabled,
		consts.OpenshiftEnabledProperty:           config.OpenshiftEnabled,
		consts.PspProperty:                        config.Psp,
		consts.SkipWebhookIssuerCreationProperty:  config.SkipWebhookIssuerCreation,
		consts.AllowConcurrentAgentsProperty:      config.AllowConcurrentAgents,
		consts.ImagePrefixProperty:                config.ImagePrefix,
		consts.UiModeProperty:                     config.UiMode,
		consts.UiPaginationLimitProperty:          config.UiPaginationLimit,
		consts.UiRemoteUrlProperty:                config.UiRemoteUrl,
		consts.CentralBackendURLProperty:          config.CentralBackendURL,
		consts.ClusterNameProperty:                config.ClusterName,
		consts.IgnoredNamespacesProperty:          config.IgnoredNamespaces,
		consts.IgnoredContainersProperty:          config.IgnoredContainers,
		consts.MountMethodProperty:                config.MountMethod,
		consts.CustomContainerRuntimeSocketPath:   config.CustomContainerRuntimeSocketPath,
		consts.K8sNodeLogsDirectory:               config.CollectorNode,
		consts.UserInstrumentationEnvsProperty:    config.UserInstrumentationEnvs,
		consts.AgentEnvVarsInjectionMethod:        config.AgentEnvVarsInjectionMethod,
		consts.NodeSelectorProperty:               config.NodeSelector,
		consts.KarpenterEnabledProperty:           config.KarpenterEnabled,
		consts.RollbackDisabledProperty:           config.RollbackDisabled,
		consts.RollbackGraceTimeProperty:          config.RollbackGraceTime,
		consts.RollbackStabilityWindow:            config.RollbackStabilityWindow,
		consts.AutomaticRolloutDisabledProperty:   config.Rollout,
		consts.OidcTenantUrlProperty:              (*ConfigOidcTenant)(config.Oidc),
		consts.OidcClientIdProperty:               (*ConfigOidcClientId)(config.Oidc),
		consts.OidcClientSecretProperty:           (*ConfigOidcClientSecret)(config.Oidc),
		consts.OdigletHealthProbeBindPortProperty: config.OdigletHealthProbeBindPort,
		consts.ServiceGraphDisabledProperty:       config.CollectorGateway,
		consts.GoAutoOffsetsCronProperty:          config.GoAutoOffsetsCron,
		consts.ClickhouseJsonTypeEnabledProperty:  config.ClickhouseJsonTypeEnabledProperty,
		consts.AllowedTestConnectionHostsProperty: config.AllowedTestConnectionHosts,
	}

	return displayData
}

func PrintMap(config *OdigosConfiguration) {
	display := makeAMap(config)

	var order []string
	for k := range consts.ConfigDisplay {
		order = append(order, k)
	}

	sort.Strings(order)

	for _, key := range order {
		fmt.Printf("	- %s: ", key)
		if display[key] == nil {
			fmt.Println("not set")
		} else {
			display[key].ToString()
		}
	}
}

func SpecificFeature(config *OdigosConfiguration, keyword string) {
	result := makeAMap(config)
	feature, err := result[keyword]
	if !err {
		fmt.Printf("That feature does not exist")
		os.Exit(1)
	}

	fmt.Printf("	- %s: ", keyword)
	feature.ToString()
}

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

// OdigosConfiguration defines the desired state of OdigosConfiguration
// the types are changed to the redefined types for the ToString()
type OdigosConfiguration struct {
	ConfigVersion             int                            `json:"configVersion" yaml:"configVersion"`
	TelemetryEnabled          ConfigBool                     `json:"telemetryEnabled,omitempty" yaml:"telemetryEnabled"`
	OpenshiftEnabled          ConfigBool                     `json:"openshiftEnabled,omitempty" yaml:"openshiftEnabled"`
	IgnoredNamespaces         ConfigStringList               `json:"ignoredNamespaces,omitempty" yaml:"ignoredNamespaces"`
	IgnoredContainers         ConfigStringList               `json:"ignoredContainers,omitempty" yaml:"ignoredContainers"`
	Psp                       ConfigBool                     `json:"psp,omitempty" yaml:"psp"`
	ImagePrefix               ConfigString                   `json:"imagePrefix,omitempty" yaml:"imagePrefix"`
	SkipWebhookIssuerCreation ConfigBool                     `json:"skipWebhookIssuerCreation,omitempty" yaml:"skipWebhookIssuerCreation"`
	CollectorGateway          *CollectorGatewayConfiguration `json:"collectorGateway,omitempty" yaml:"collectorGateway"`
	CollectorNode             *CollectorNodeConfiguration    `json:"collectorNode,omitempty" yaml:"collectorNode"`
	Profiles                  []ProfileName                  `json:"profiles,omitempty" yaml:"profiles"`
	AllowConcurrentAgents     ConfigBoolPointer              `json:"allowConcurrentAgents,omitempty" yaml:"allowConcurrentAgents"`
	UiMode                    UiMode                         `json:"uiMode,omitempty" yaml:"uiMode"`
	UiPaginationLimit         ConfigInt                      `json:"uiPaginationLimit,omitempty" yaml:"uiPaginationLimit"`
	UiRemoteUrl               ConfigString                   `json:"uiRemoteUrl,omitempty" yaml:"uiRemoteUrl"`
	CentralBackendURL         ConfigString                   `json:"centralBackendURL,omitempty" yaml:"centralBackendURL"`
	ClusterName               ConfigString                   `json:"clusterName,omitempty" yaml:"clusterName"`
	MountMethod               *MountMethod                   `json:"mountMethod,omitempty" yaml:"mountMethod"`
	//nolint:lll // CustomContainerRuntimeSocketPath line is long due to struct tag requirements

	CustomContainerRuntimeSocketPath  ConfigString             `json:"customContainerRuntimeSocketPath,omitempty" yaml:"customContainerRuntimeSocketPath"`
	AgentEnvVarsInjectionMethod       *EnvInjectionMethod      `json:"agentEnvVarsInjectionMethod,omitempty" yaml:"agentEnvVarsInjectionMethod"`
	UserInstrumentationEnvs           *UserInstrumentationEnvs `json:"userInstrumentationEnvs,omitempty" yaml:"userInstrumentationEnvs"`
	NodeSelector                      ConfigNodeSelector       `json:"nodeSelector,omitempty" yaml:"nodeSelector"`
<<<<<<< HEAD
<<<<<<< HEAD
	KarpenterEnabled                  ConfigBoolPointer        `json:"karpenterEnabled,omitempty" yaml:"karpenterEnabled"`
	Rollout                           *RolloutConfiguration    `json:"rollout,omitempty" yaml:"rollout"`
	RollbackDisabled                  ConfigBoolPointer        `json:"rollbackDisabled,omitempty" yaml:"rollbackDisabled"`
=======
	KarpenterEnabled                  *bool                    `json:"karpenterEnabled,omitempty" yaml:"karpenterEnabled"`
	Rollout                           *RolloutConfiguration    `json:"rollout,omitempty" yaml:"rollout"`
	RollbackDisabled                  *bool                    `json:"rollbackDisabled,omitempty" yaml:"rollbackDisabled"`
>>>>>>> 90cc9086 (fixed cli erros)
=======
	KarpenterEnabled                  ConfigBoolPointer        `json:"karpenterEnabled,omitempty" yaml:"karpenterEnabled"`
	Rollout                           *RolloutConfiguration    `json:"rollout,omitempty" yaml:"rollout"`
	RollbackDisabled                  ConfigBoolPointer        `json:"rollbackDisabled,omitempty" yaml:"rollbackDisabled"`
>>>>>>> 40279361 (fixed ConfigBoolPointer issue, added Unmarshal function, other error fixes)
	RollbackGraceTime                 ConfigString             `json:"rollbackGraceTime,omitempty" yaml:"rollbackGraceTime"`
	RollbackStabilityWindow           ConfigString             `json:"rollbackStabilityWindow,omitempty" yaml:"rollbackStabilityWindow"`
	Oidc                              *OidcConfiguration       `json:"oidc,omitempty" yaml:"oidc"`
	OdigletHealthProbeBindPort        ConfigInt                `json:"odigletHealthProbeBindPort,omitempty" yaml:"odigletHealthProbeBindPort"`
	GoAutoOffsetsCron                 ConfigString             `json:"goAutoOffsetsCron,omitempty" yaml:"goAutoOffsetsCron"`
<<<<<<< HEAD
<<<<<<< HEAD
	ClickhouseJsonTypeEnabledProperty ConfigBoolPointer        `json:"clickhouseJsonTypeEnabled,omitempty"`
<<<<<<< HEAD
	AllowedTestConnectionHosts        []string                 `json:"allowedTestConnectionHosts,omitempty" yaml:"allowedTestConnectionHosts"`
=======
	ClickhouseJsonTypeEnabledProperty *bool                    `json:"clickhouseJsonTypeEnabled,omitempty"`
>>>>>>> 90cc9086 (fixed cli erros)
=======
	ClickhouseJsonTypeEnabledProperty ConfigBoolPointer        `json:"clickhouseJsonTypeEnabled,omitempty"`
>>>>>>> 40279361 (fixed ConfigBoolPointer issue, added Unmarshal function, other error fixes)
=======
	AllowedTestConnectionHosts        ConfigStringList         `json:"allowedTestConnectionHosts,omitempty" yaml:"allowedTestConnectionHosts"`
>>>>>>> 63c22c69 (small changes)
}
