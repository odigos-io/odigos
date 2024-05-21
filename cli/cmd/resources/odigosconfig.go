package resources

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OdigosConfigName = "odigos-config"
)

func otelSdkConfigCommunity() (map[common.ProgrammingLanguage]common.OtelSdk, map[common.ProgrammingLanguage][]common.OtelSdk) {

	nativeCommunity := common.OtelSdk{
		SdkType: common.NativeOtelSdkType,
		SdkTier: common.CommunityOtelSdkTier,
	}

	eBPFCommunity := common.OtelSdk{
		SdkType: common.EbpfOtelSdkType,
		SdkTier: common.CommunityOtelSdkTier,
	}

	return map[common.ProgrammingLanguage]common.OtelSdk{
			common.JavaProgrammingLanguage:       nativeCommunity,
			common.PythonProgrammingLanguage:     nativeCommunity,
			common.GoProgrammingLanguage:         eBPFCommunity,
			common.DotNetProgrammingLanguage:     nativeCommunity,
			common.JavascriptProgrammingLanguage: nativeCommunity,
		},
		map[common.ProgrammingLanguage][]common.OtelSdk{
			common.JavaProgrammingLanguage:       {nativeCommunity},
			common.PythonProgrammingLanguage:     {nativeCommunity},
			common.GoProgrammingLanguage:         {eBPFCommunity},
			common.DotNetProgrammingLanguage:     {nativeCommunity},
			common.JavascriptProgrammingLanguage: {nativeCommunity},
		}
}

func otelSdkConfigCloud() (map[common.ProgrammingLanguage]common.OtelSdk, map[common.ProgrammingLanguage][]common.OtelSdk) {

	nativeCommunity := common.OtelSdk{
		SdkType: common.NativeOtelSdkType,
		SdkTier: common.CommunityOtelSdkTier,
	}

	eBPFEnterprise := common.OtelSdk{
		SdkType: common.EbpfOtelSdkType,
		SdkTier: common.EnterpriseOtelSdkTier,
	}

	return map[common.ProgrammingLanguage]common.OtelSdk{
			common.JavaProgrammingLanguage:       nativeCommunity,
			common.PythonProgrammingLanguage:     nativeCommunity,
			common.GoProgrammingLanguage:         eBPFEnterprise,
			common.DotNetProgrammingLanguage:     nativeCommunity,
			common.JavascriptProgrammingLanguage: nativeCommunity,
		},
		map[common.ProgrammingLanguage][]common.OtelSdk{
			common.JavaProgrammingLanguage:       {nativeCommunity, eBPFEnterprise},
			common.PythonProgrammingLanguage:     {nativeCommunity, eBPFEnterprise},
			common.GoProgrammingLanguage:         {eBPFEnterprise},
			common.DotNetProgrammingLanguage:     {nativeCommunity},
			common.JavascriptProgrammingLanguage: {nativeCommunity, eBPFEnterprise},
		}
}

func otelSdkConfigOnPrem() (map[common.ProgrammingLanguage]common.OtelSdk, map[common.ProgrammingLanguage][]common.OtelSdk) {

	nativeCommunity := common.OtelSdk{
		SdkType: common.NativeOtelSdkType,
		SdkTier: common.CommunityOtelSdkTier,
	}

	eBPFEnterprise := common.OtelSdk{
		SdkType: common.EbpfOtelSdkType,
		SdkTier: common.EnterpriseOtelSdkTier,
	}

	nativeEnterprise := common.OtelSdk{
		SdkType: common.NativeOtelSdkType,
		SdkTier: common.EnterpriseOtelSdkTier,
	}

	return map[common.ProgrammingLanguage]common.OtelSdk{
			common.JavaProgrammingLanguage:       eBPFEnterprise, // Notice - for onprem, the default for java is eBPF
			common.PythonProgrammingLanguage:     eBPFEnterprise, // Also Python
			common.GoProgrammingLanguage:         eBPFEnterprise,
			common.DotNetProgrammingLanguage:     nativeCommunity,
			common.JavascriptProgrammingLanguage: eBPFEnterprise, // Also Javascript
			common.MySQLProgrammingLanguage:      eBPFEnterprise,
		},
		map[common.ProgrammingLanguage][]common.OtelSdk{
			common.JavaProgrammingLanguage:       {nativeCommunity, eBPFEnterprise, nativeEnterprise},
			common.PythonProgrammingLanguage:     {nativeCommunity, eBPFEnterprise},
			common.GoProgrammingLanguage:         {eBPFEnterprise},
			common.DotNetProgrammingLanguage:     {nativeCommunity},
			common.JavascriptProgrammingLanguage: {nativeCommunity, eBPFEnterprise},
			common.MySQLProgrammingLanguage:      {eBPFEnterprise},
		}
}

func NewOdigosConfiguration(ns string, config *odigosv1.OdigosConfigurationSpec) *odigosv1.OdigosConfiguration {
	return &odigosv1.OdigosConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "OdigosConfiguration",
			APIVersion: "odigos.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      OdigosConfigName,
			Namespace: ns,
		},
		Spec: *config,
	}
}

type odigosConfigResourceManager struct {
	client     *kube.Client
	ns         string
	config     *odigosv1.OdigosConfigurationSpec
	odigosTier common.OdigosTier
}

func NewOdigosConfigResourceManager(client *kube.Client, ns string, config *odigosv1.OdigosConfigurationSpec, odigosTier common.OdigosTier) resourcemanager.ResourceManager {
	return &odigosConfigResourceManager{client: client, ns: ns, config: config, odigosTier: odigosTier}
}

func (a *odigosConfigResourceManager) Name() string { return "OdigosConfig" }

func (a *odigosConfigResourceManager) InstallFromScratch(ctx context.Context) error {

	var defaultOtelSdkPerLanguage map[common.ProgrammingLanguage]common.OtelSdk
	var supportedOtelSdksPerLanguage map[common.ProgrammingLanguage][]common.OtelSdk
	switch a.odigosTier {
	case common.CommunityOdigosTier:
		defaultOtelSdkPerLanguage, supportedOtelSdksPerLanguage = otelSdkConfigCommunity()
	case common.CloudOdigosTier:
		defaultOtelSdkPerLanguage, supportedOtelSdksPerLanguage = otelSdkConfigCloud()
	case common.OnPremOdigosTier:
		defaultOtelSdkPerLanguage, supportedOtelSdksPerLanguage = otelSdkConfigOnPrem()
	}

	// the default SDK should be retained in the future.
	a.config.DefaultSDKs = defaultOtelSdkPerLanguage
	a.config.SupportedSDKs = supportedOtelSdksPerLanguage

	resources := []client.Object{
		NewOdigosConfiguration(a.ns, a.config),
	}
	return a.client.ApplyResources(ctx, a.config.ConfigVersion, resources)
}
