package sdks

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

var defaultOtelSdkPerLanguage = map[common.ProgrammingLanguage]common.OtelSdk{}

func otelSdkConfigCommunity() map[common.ProgrammingLanguage]common.OtelSdk {
	return map[common.ProgrammingLanguage]common.OtelSdk{
		common.JavaProgrammingLanguage:       common.OtelSdkNativeCommunity,
		common.PythonProgrammingLanguage:     common.OtelSdkNativeCommunity,
		common.GoProgrammingLanguage:         common.OtelSdkEbpfCommunity,
		common.DotNetProgrammingLanguage:     common.OtelSdkNativeCommunity,
		common.JavascriptProgrammingLanguage: common.OtelSdkNativeCommunity,
		common.PhpProgrammingLanguage:        common.OtelSdkNativeCommunity,
		common.RubyProgrammingLanguage:       common.OtelSdkNativeCommunity,
		common.RustProgrammingLanguage:       common.OtelSdkEbpfCommunity,
	}
}

func otelSdkConfigCloud() map[common.ProgrammingLanguage]common.OtelSdk {
	return map[common.ProgrammingLanguage]common.OtelSdk{
		common.JavaProgrammingLanguage:       common.OtelSdkNativeCommunity,
		common.PythonProgrammingLanguage:     common.OtelSdkNativeCommunity,
		common.GoProgrammingLanguage:         common.OtelSdkEbpfEnterprise,
		common.DotNetProgrammingLanguage:     common.OtelSdkNativeCommunity,
		common.JavascriptProgrammingLanguage: common.OtelSdkNativeCommunity,
		common.PhpProgrammingLanguage:        common.OtelSdkNativeCommunity,
		common.RubyProgrammingLanguage:       common.OtelSdkNativeCommunity,
		common.RustProgrammingLanguage:       common.OtelSdkEbpfEnterprise,
	}
}

func otelSdkConfigOnPrem() map[common.ProgrammingLanguage]common.OtelSdk {
	return map[common.ProgrammingLanguage]common.OtelSdk{
		common.JavaProgrammingLanguage:       common.OtelSdkNativeEnterprise,
		common.PythonProgrammingLanguage:     common.OtelSdkEbpfEnterprise,
		common.GoProgrammingLanguage:         common.OtelSdkEbpfEnterprise,
		common.DotNetProgrammingLanguage:     common.OtelSdkNativeCommunity,
		common.JavascriptProgrammingLanguage: common.OtelSdkEbpfEnterprise,
		common.PhpProgrammingLanguage:        common.OtelSdkNativeCommunity,
		common.RubyProgrammingLanguage:       common.OtelSdkNativeCommunity,
		common.RustProgrammingLanguage:       common.OtelSdkEbpfEnterprise,
		common.MySQLProgrammingLanguage:      common.OtelSdkEbpfEnterprise,
		common.NginxProgrammingLanguage:      common.OtelSdkNativeCommunity,
	}
}

func SetDefaultSDKs() {
	odigosTier := env.GetOdigosTierFromEnv()

	switch odigosTier {
	case common.CommunityOdigosTier:
		defaultOtelSdkPerLanguage = otelSdkConfigCommunity()
	case common.CloudOdigosTier:
		defaultOtelSdkPerLanguage = otelSdkConfigCloud()
	case common.OnPremOdigosTier:
		defaultOtelSdkPerLanguage = otelSdkConfigOnPrem()
	}
}

func copyOtelSdkMap(m map[common.ProgrammingLanguage]common.OtelSdk) map[common.ProgrammingLanguage]common.OtelSdk {
	// Create a new map with the same capacity as the original
	newMap := make(map[common.ProgrammingLanguage]common.OtelSdk, len(m))

	// Copy each key-value pair to the new map
	for key, value := range m {
		newMap[key] = common.OtelSdk{
			SdkType: value.SdkType,
			SdkTier: value.SdkTier,
		}
	}

	return newMap
}

func GetDefaultSDKs() map[common.ProgrammingLanguage]common.OtelSdk {
	// return a copy of the map, so it can be modified without affecting the original
	// and if used by multiple goroutines, it will be safe to write to the instance returned
	return copyOtelSdkMap(defaultOtelSdkPerLanguage)
}
