package common

type OdigosInstrumentationDevice string

// This is the resource namespace of the lister in k8s device plugin manager.
// from the "github.com/kubevirt/device-plugin-manager" package source:
// GetResourceNamespace must return namespace (vendor ID) of implemented Lister. e.g. for
// resources in format "color.example.com/<color>" that would be "color.example.com".
const OdigosResourceNamespace = "instrumentation.odigos.io"

// The plugin name is also part of the device-plugin-manager.
// It is used to control which environment variables and fs mounts are available to the pod.
// Each OtelSdkType has its own plugin name, which is used to control instrumentation for that SDK.
//
// Odigos convention for plugin name is as follows:
// <runtime-name>-<sdk-type>-<sdk-tier>
//
// for example:
// the native Java SDK will be named "java-native-community".
// the ebpf Java enterprise sdk will be named "java-ebpf-enterprise".

func InstrumentationPluginName(language ProgrammingLanguage, otelSdk OtelSdk, libc *LibCType) string {
	result := string(language) + "-" + string(otelSdk.SdkType) + "-" + string(otelSdk.SdkTier)

	// If musl libc type recorded - we use different plugin name
	if libc != nil && *libc == Musl {
		result = "musl-" + result
	}

	return result
}
