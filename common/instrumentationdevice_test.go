package common

import (
	"testing"
)

func TestInstrumentationPluginName(t *testing.T) {
	language := DotNetProgrammingLanguage
	otelSdk := OtelSdk{
		SdkType: NativeOtelSdkType,
		SdkTier: CommunityOtelSdkTier,
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
		SdkType: EbpfOtelSdkType,
		SdkTier: EnterpriseOtelSdkTier,
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

	deviceName := InstrumentationDeviceName(language, OtelSdk{SdkType: otelSdkType, SdkTier: otelSdkTier})
	gotLanguage, gotOtelSdkType, gotOtelSdkTier := InstrumentationDeviceNameToComponents(string(deviceName))

	if gotLanguage != language {
		t.Errorf("InstrumentationDeviceNameToComponents() gotLanguage = %v, want %v", gotLanguage, language)
	}
	if gotOtelSdkType != otelSdkType {
		t.Errorf("InstrumentationDeviceNameToComponents() gotOtelSdkType = %v, want %v", gotOtelSdkType, otelSdkType)
	}
	if gotOtelSdkTier != otelSdkTier {
		t.Errorf("InstrumentationDeviceNameToComponents() gotOtelSdkTier = %v, want %v", gotOtelSdkTier, otelSdkTier)
	}
}
