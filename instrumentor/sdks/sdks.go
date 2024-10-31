package sdks

import (
	"context"

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
	}
}

func otelSdkConfigCloud() map[common.ProgrammingLanguage]common.OtelSdk {
	return map[common.ProgrammingLanguage]common.OtelSdk{
		common.JavaProgrammingLanguage:       common.OtelSdkNativeCommunity,
		common.PythonProgrammingLanguage:     common.OtelSdkNativeCommunity,
		common.GoProgrammingLanguage:         common.OtelSdkEbpfEnterprise,
		common.DotNetProgrammingLanguage:     common.OtelSdkNativeCommunity,
		common.JavascriptProgrammingLanguage: common.OtelSdkNativeCommunity,
	}
}

func otelSdkConfigOnPrem() map[common.ProgrammingLanguage]common.OtelSdk {
	return map[common.ProgrammingLanguage]common.OtelSdk{
		common.JavaProgrammingLanguage:       common.OtelSdkEbpfEnterprise, // Notice - for onprem, the default for java is eBPF
		common.PythonProgrammingLanguage:     common.OtelSdkEbpfEnterprise,
		common.GoProgrammingLanguage:         common.OtelSdkEbpfEnterprise,
		common.DotNetProgrammingLanguage:     common.OtelSdkNativeCommunity,
		common.JavascriptProgrammingLanguage: common.OtelSdkEbpfEnterprise,
		common.MySQLProgrammingLanguage:      common.OtelSdkEbpfEnterprise,
		common.NginxProgrammingLanguage:      common.OtelSdkNativeCommunity,
	}
}

func SetDefaultSDKs(ctx context.Context) error {
	odigosTier := env.GetOdigosTierFromEnv()

	switch odigosTier {
	case common.CommunityOdigosTier:
		defaultOtelSdkPerLanguage = otelSdkConfigCommunity()
	case common.CloudOdigosTier:
		defaultOtelSdkPerLanguage = otelSdkConfigCloud()
	case common.OnPremOdigosTier:
		defaultOtelSdkPerLanguage = otelSdkConfigOnPrem()
	}

	return nil
}

func GetDefaultSDKs() map[common.ProgrammingLanguage]common.OtelSdk {
	return defaultOtelSdkPerLanguage
}
