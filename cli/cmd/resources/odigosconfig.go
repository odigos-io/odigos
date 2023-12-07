package resources

import (
	"context"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/cli/pkg/kube"
	"github.com/keyval-dev/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OdigosConfigName = "odigos-config"
)

func otelSdkConfigCommunity() (odigosv1.DefaultOtelSdkPerLanguage, odigosv1.SupportedOtelSdksPerLanguage) {

	nativeCommunity := odigosv1.SupportedOtelSdk{
		SdkType: common.NativeOtelSdkType,
		SdkTier: common.CommunityOtelSdkTier,
	}

	eBPFCommunity := odigosv1.SupportedOtelSdk{
		SdkType: common.EbpfOtelSdkType,
		SdkTier: common.CommunityOtelSdkTier,
	}

	return odigosv1.DefaultOtelSdkPerLanguage{
			common.JavaProgrammingLanguage:       nativeCommunity,
			common.PythonProgrammingLanguage:     nativeCommunity,
			common.GoProgrammingLanguage:         eBPFCommunity,
			common.DotNetProgrammingLanguage:     nativeCommunity,
			common.JavascriptProgrammingLanguage: nativeCommunity,
		},
		odigosv1.SupportedOtelSdksPerLanguage{
			common.JavaProgrammingLanguage:       []odigosv1.SupportedOtelSdk{nativeCommunity},
			common.PythonProgrammingLanguage:     []odigosv1.SupportedOtelSdk{nativeCommunity},
			common.GoProgrammingLanguage:         []odigosv1.SupportedOtelSdk{eBPFCommunity},
			common.DotNetProgrammingLanguage:     []odigosv1.SupportedOtelSdk{nativeCommunity},
			common.JavascriptProgrammingLanguage: []odigosv1.SupportedOtelSdk{nativeCommunity},
		}
}

func otelSdkConfigEnterprise() (odigosv1.DefaultOtelSdkPerLanguage, odigosv1.SupportedOtelSdksPerLanguage) {

	nativeCommunity := odigosv1.SupportedOtelSdk{
		SdkType: common.NativeOtelSdkType,
		SdkTier: common.CommunityOtelSdkTier,
	}

	eBPFEnterprise := odigosv1.SupportedOtelSdk{
		SdkType: common.EbpfOtelSdkType,
		SdkTier: common.EnterpriseOtelSdkTier,
	}

	return odigosv1.DefaultOtelSdkPerLanguage{
			common.JavaProgrammingLanguage:       nativeCommunity,
			common.PythonProgrammingLanguage:     nativeCommunity,
			common.GoProgrammingLanguage:         eBPFEnterprise,
			common.DotNetProgrammingLanguage:     nativeCommunity,
			common.JavascriptProgrammingLanguage: nativeCommunity,
		},
		odigosv1.SupportedOtelSdksPerLanguage{
			common.JavaProgrammingLanguage:       []odigosv1.SupportedOtelSdk{nativeCommunity},
			common.PythonProgrammingLanguage:     []odigosv1.SupportedOtelSdk{nativeCommunity},
			common.GoProgrammingLanguage:         []odigosv1.SupportedOtelSdk{eBPFEnterprise},
			common.DotNetProgrammingLanguage:     []odigosv1.SupportedOtelSdk{nativeCommunity},
			common.JavascriptProgrammingLanguage: []odigosv1.SupportedOtelSdk{nativeCommunity},
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
	client        *kube.Client
	ns            string
	config        *odigosv1.OdigosConfigurationSpec
	isOdigosCloud bool
}

func NewOdigosConfigResourceManager(client *kube.Client, ns string, config *odigosv1.OdigosConfigurationSpec, isOdigosCloud bool) ResourceManager {
	return &odigosConfigResourceManager{client: client, ns: ns, config: config, isOdigosCloud: isOdigosCloud}
}

func (a *odigosConfigResourceManager) Name() string { return "OdigosConfig" }

func (a *odigosConfigResourceManager) InstallFromScratch(ctx context.Context) error {

	var defaultOtelSdkPerLanguage odigosv1.DefaultOtelSdkPerLanguage
	var supportedOtelSdksPerLanguage odigosv1.SupportedOtelSdksPerLanguage
	if a.isOdigosCloud {
		defaultOtelSdkPerLanguage, supportedOtelSdksPerLanguage = otelSdkConfigEnterprise()
	} else {
		defaultOtelSdkPerLanguage, supportedOtelSdksPerLanguage = otelSdkConfigCommunity()
	}

	// the default SDK should be retained in the future.
	a.config.DefaultSDKs = defaultOtelSdkPerLanguage
	a.config.SupportedSDKs = supportedOtelSdksPerLanguage

	resources := []client.Object{
		NewOdigosConfiguration(a.ns, a.config),
	}
	return a.client.ApplyResources(ctx, a.config.ConfigVersion, resources)
}
