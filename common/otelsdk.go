package common

import (
	"encoding/json"
)

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
	sdkType OtelSdkType
	sdkTier OtelSdkTier
}

var (
	OtelSdkNativeCommunity  = OtelSdk{sdkType: NativeOtelSdkType, sdkTier: CommunityOtelSdkTier}
	OtelSdkEbpfCommunity    = OtelSdk{sdkType: EbpfOtelSdkType, sdkTier: CommunityOtelSdkTier}
	OtelSdkNativeEnterprise = OtelSdk{sdkType: NativeOtelSdkType, sdkTier: EnterpriseOtelSdkTier}
	OtelSdkEbpfEnterprise   = OtelSdk{sdkType: EbpfOtelSdkType, sdkTier: EnterpriseOtelSdkTier}
)

func NewOtelSdk(sdkType OtelSdkType, sdkTier OtelSdkTier) OtelSdk {
	return OtelSdk{sdkType: sdkType, sdkTier: sdkTier}
}

func (s OtelSdk) GetSdkType() OtelSdkType {
	return s.sdkType
}

func (s OtelSdk) GetSdkTier() OtelSdkTier {
	return s.sdkTier
}

func (s OtelSdk) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		SdkType OtelSdkType `json:"sdkType"`
		SdkTier OtelSdkTier `json:"sdkTier"`
	}{
		SdkType: s.sdkType,	
		SdkTier: s.sdkTier,
	})
}

func (s *OtelSdk) UnmarshalJSON(data []byte) error {
	var v struct {
		SdkType OtelSdkType `json:"sdkType"`
		SdkTier OtelSdkTier `json:"sdkTier"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	s.sdkType = v.SdkType
	s.sdkTier = v.SdkTier
	return nil
}
