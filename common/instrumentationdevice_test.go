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
	pluginName := InstrumentationPluginName(language, otelSdk, nil)
	want := "dotnet-native-community"
	if pluginName != want {
		t.Errorf("InstrumentationPluginName() = %v, want %v", pluginName, want)
	}
}

func TestInstrumentationPluginNameMusl(t *testing.T) {
	language := DotNetProgrammingLanguage
	otelSdk := OtelSdk{
		SdkType: NativeOtelSdkType,
		SdkTier: CommunityOtelSdkTier,
	}
	musl := Musl
	pluginName := InstrumentationPluginName(language, otelSdk, &musl)
	want := "musl-dotnet-native-community"
	if pluginName != want {
		t.Errorf("InstrumentationPluginName() = %v, want %v", pluginName, want)
	}
}

func TestInstrumentationPluginNameGlib(t *testing.T) {
	language := DotNetProgrammingLanguage
	otelSdk := OtelSdk{
		SdkType: NativeOtelSdkType,
		SdkTier: CommunityOtelSdkTier,
	}
	glib := Glibc
	pluginName := InstrumentationPluginName(language, otelSdk, &glib)
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
	deviceName := InstrumentationDeviceName(language, otelSdk, nil)
	want := "instrumentation.odigos.io/java-ebpf-enterprise"
	if string(deviceName) != want {
		t.Errorf("InstrumentationDeviceName() = %v, want %v", deviceName, want)
	}
}

func TestInstrumentationDeviceNameGlib(t *testing.T) {
	language := JavaProgrammingLanguage
	otelSdk := OtelSdk{
		SdkType: EbpfOtelSdkType,
		SdkTier: EnterpriseOtelSdkTier,
	}
	glib := Glibc
	deviceName := InstrumentationDeviceName(language, otelSdk, &glib)
	want := "instrumentation.odigos.io/java-ebpf-enterprise"
	if string(deviceName) != want {
		t.Errorf("InstrumentationDeviceName() = %v, want %v", deviceName, want)
	}
}

func TestInstrumentationDeviceNameToComponents(t *testing.T) {
	language := GoProgrammingLanguage
	otelSdkType := EbpfOtelSdkType
	otelSdkTier := CommunityOtelSdkTier
	sdk := OtelSdk{SdkType: otelSdkType, SdkTier: otelSdkTier}

	deviceName := InstrumentationDeviceName(language, sdk, nil)
	gotLanguage, gotSdk := InstrumentationDeviceNameToComponents(string(deviceName))

	if gotLanguage != language {
		t.Errorf("InstrumentationDeviceNameToComponents() gotLanguage = %v, want %v", gotLanguage, language)
	}
	if gotSdk.SdkType != otelSdkType {
		t.Errorf("InstrumentationDeviceNameToComponents() gotOtelSdkType = %v, want %v", gotSdk.SdkType, otelSdkType)
	}
	if gotSdk.SdkTier != otelSdkTier {
		t.Errorf("InstrumentationDeviceNameToComponents() gotOtelSdkTier = %v, want %v", gotSdk.SdkTier, otelSdkTier)
	}
}

func TestInstrumentationDeviceNameToComponentsMusl(t *testing.T) {

	language := DotNetProgrammingLanguage
	otelSdkType := NativeOtelSdkType
	otelSdkTier := CommunityOtelSdkTier
	sdk := OtelSdk{SdkType: otelSdkType, SdkTier: otelSdkTier}

	musl := Musl
	deviceName := InstrumentationDeviceName(language, sdk, &musl)
	gotLanguage, gotSdk := InstrumentationDeviceNameToComponents(string(deviceName))

	if gotLanguage != language {
		t.Errorf("InstrumentationDeviceNameToComponents() gotLanguage = %v, want %v", gotLanguage, language)
	}
	if gotSdk.SdkType != otelSdkType {
		t.Errorf("InstrumentationDeviceNameToComponents() gotOtelSdkType = %v, want %v", gotSdk.SdkType, otelSdkType)
	}
	if gotSdk.SdkTier != otelSdkTier {
		t.Errorf("InstrumentationDeviceNameToComponents() gotOtelSdkTier = %v, want %v", gotSdk.SdkTier, otelSdkTier)
	}
}
