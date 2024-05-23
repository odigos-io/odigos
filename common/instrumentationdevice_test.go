package common

import (
	"testing"
)

func TestInstrumentationPluginName(t *testing.T) {
	language := DotNetProgrammingLanguage
	otelSdk := OtelSdk{
		sdkType: NativeOtelSdkType,
		sdkTier: CommunityOtelSdkTier,
	}
	pluginName := InstrumentationPluginName(language, otelSdk)
	want := "dotnet-native-community"
	if pluginName != want {
		t.Errorf("InstrumentationPluginName() = %v, want %v", pluginName, want)
	}
}

func TestInstrumentationDeviceName(t *testing.T) {
	language := JavaProgrammingLanguage
	otelSdk := OtelSdk{
		sdkType: EbpfOtelSdkType,
		sdkTier: EnterpriseOtelSdkTier,
	}
	deviceName := InstrumentationDeviceName(language, otelSdk)
	want := "instrumentation.odigos.io/java-ebpf-enterprise"
	if string(deviceName) != want {
		t.Errorf("InstrumentationDeviceName() = %v, want %v", deviceName, want)
	}
}

func TestInstrumentationDeviceNameToComponents(t *testing.T) {

	language := GoProgrammingLanguage
	otelSdkType := EbpfOtelSdkType
	otelSdkTier := CommunityOtelSdkTier
	sdk := OtelSdk{sdkType: otelSdkType, sdkTier: otelSdkTier}

	deviceName := InstrumentationDeviceName(language, sdk)
	gotLanguage, gotSdk := InstrumentationDeviceNameToComponents(string(deviceName))

	if gotLanguage != language {
		t.Errorf("InstrumentationDeviceNameToComponents() gotLanguage = %v, want %v", gotLanguage, language)
	}
	if gotSdk.GetSdkType() != otelSdkType {
		t.Errorf("InstrumentationDeviceNameToComponents() gotOtelSdkType = %v, want %v", gotSdk.GetSdkType(), otelSdkType)
	}
	if gotSdk.GetSdkTier() != otelSdkTier {
		t.Errorf("InstrumentationDeviceNameToComponents() gotOtelSdkTier = %v, want %v", gotSdk.GetSdkTier(), otelSdkTier)
	}
}
