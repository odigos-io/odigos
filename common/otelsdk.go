package common

// Odigos supports two types of OpenTelemetry SDKs: native and ebpf.
type OtelSdkType string

const (
	// The native SDK is implemented in the language of the application and
	// is integrated into the application code via runtime support (e.g. Java agent).
	NativeOtelSdkType OtelSdkType = "native"

	// EbpfOtelSdkType SDK can record telemetry data from the application with eBPF
	// code injected into the application process.
	EbpfOtelSdkType OtelSdkType = "ebpf"
)

type OtelSdkTier string

const (
	CommunityOtelSdkTier  OtelSdkTier = "community"
	EnterpriseOtelSdkTier OtelSdkTier = "enterprise"
)

type OtelSdk struct {
	SdkType OtelSdkType `json:"sdkType"`
	SdkTier OtelSdkTier `json:"sdkTier"`
}

var (
	OtelSdkNativeCommunity  = OtelSdk{SdkType: NativeOtelSdkType, SdkTier: CommunityOtelSdkTier}
	OtelSdkEbpfCommunity    = OtelSdk{SdkType: EbpfOtelSdkType, SdkTier: CommunityOtelSdkTier}
	OtelSdkNativeEnterprise = OtelSdk{SdkType: NativeOtelSdkType, SdkTier: EnterpriseOtelSdkTier}
	OtelSdkEbpfEnterprise   = OtelSdk{SdkType: EbpfOtelSdkType, SdkTier: EnterpriseOtelSdkTier}
)
