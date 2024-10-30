// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

package v1alpha1

import (
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Deprecated: Use common.OdigosConfiguration instead
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

// OdigosConfigurationSpec defines the desired state of OdigosConfiguration
//
// Deprecated: Use common.OdigosConfiguration instead
type OdigosConfigurationSpec struct {
	OdigosVersion     string                                          `json:"odigosVersion"`
	ConfigVersion     int                                             `json:"configVersion"`
	TelemetryEnabled  bool                                            `json:"telemetryEnabled,omitempty"`
	OpenshiftEnabled  bool                                            `json:"openshiftEnabled,omitempty"`
	IgnoredNamespaces []string                                        `json:"ignoredNamespaces,omitempty"`
	IgnoredContainers []string                                        `json:"ignoredContainers,omitempty"`
	Psp               bool                                            `json:"psp,omitempty"`
	ImagePrefix       string                                          `json:"imagePrefix,omitempty"`
	OdigletImage      string                                          `json:"odigletImage,omitempty"`
	InstrumentorImage string                                          `json:"instrumentorImage,omitempty"`
	AutoscalerImage   string                                          `json:"autoscalerImage,omitempty"`
	SupportedSDKs     map[common.ProgrammingLanguage][]common.OtelSdk `json:"supportedSDKs,omitempty"`
	DefaultSDKs       map[common.ProgrammingLanguage]common.OtelSdk   `json:"defaultSDKs,omitempty"`
	CollectorGateway  *CollectorGatewayConfiguration                  `json:"collectorGateway,omitempty"`

	// this is internal currently, and is not exposed on the CLI / helm
	// used for odigos enterprise
	GoAutoIncludeCodeAttributes bool `json:"goAutoIncludeCodeAttributes,omitempty"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:metadata:labels=odigos.io/config=1
//+kubebuilder:metadata:labels=odigos.io/system-object=true

// OdigosConfiguration is the Schema for the odigos configuration
//
// Deprecated: Use common.OdigosConfiguration instead
type OdigosConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec OdigosConfigurationSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// OdigosConfigurationList contains a list of OdigosConfiguration
type OdigosConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OdigosConfiguration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OdigosConfiguration{}, &OdigosConfigurationList{})
}

func (odigosConfig *OdigosConfiguration) ToCommonConfig() *common.OdigosConfiguration {
	var collectorGateway common.CollectorGatewayConfiguration
	if odigosConfig.Spec.CollectorGateway != nil {
		collectorGateway = common.CollectorGatewayConfiguration{
			RequestMemoryMiB:           odigosConfig.Spec.CollectorGateway.RequestMemoryMiB,
			MemoryLimiterLimitMiB:      odigosConfig.Spec.CollectorGateway.MemoryLimiterLimitMiB,
			MemoryLimiterSpikeLimitMiB: odigosConfig.Spec.CollectorGateway.MemoryLimiterSpikeLimitMiB,
			GoMemLimitMib:              odigosConfig.Spec.CollectorGateway.GoMemLimitMib,
		}
	}
	return &common.OdigosConfiguration{
		ConfigVersion:               odigosConfig.Spec.ConfigVersion,
		TelemetryEnabled:            odigosConfig.Spec.TelemetryEnabled,
		OpenshiftEnabled:            odigosConfig.Spec.OpenshiftEnabled,
		IgnoredNamespaces:           odigosConfig.Spec.IgnoredNamespaces,
		IgnoredContainers:           odigosConfig.Spec.IgnoredContainers,
		Psp:                         odigosConfig.Spec.Psp,
		ImagePrefix:                 odigosConfig.Spec.ImagePrefix,
		OdigletImage:                odigosConfig.Spec.OdigletImage,
		InstrumentorImage:           odigosConfig.Spec.InstrumentorImage,
		AutoscalerImage:             odigosConfig.Spec.AutoscalerImage,
		DefaultSDKs:                 odigosConfig.Spec.DefaultSDKs,
		CollectorGateway:            &collectorGateway,
		GoAutoIncludeCodeAttributes: odigosConfig.Spec.GoAutoIncludeCodeAttributes,
	}
}
