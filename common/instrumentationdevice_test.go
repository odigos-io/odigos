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
