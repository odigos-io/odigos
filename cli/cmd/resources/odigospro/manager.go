package odigospro

import (
	"context"

	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type odigosCloudResourceManager struct {
	client       *kube.Client
	ns           string
	config       *common.OdigosConfiguration
	odigosTier   common.OdigosTier
	proTierToken *string
}

// the odigos pro resource manager supports the following flows:
// 1. odigos tier is community - the resource manager should not be installed.
// 2. User has provided a cloud api key or onprem token - the resource manager should be initialized with the pro tier token.
// 3. User wishes to update resources but leave the token as is - proTierToken should be nil.
func NewOdigosProResourceManager(client *kube.Client, ns string, config *common.OdigosConfiguration, odigosTier common.OdigosTier, proTierToken *string) resourcemanager.ResourceManager {
	return &odigosCloudResourceManager{client: client, ns: ns, config: config, odigosTier: odigosTier, proTierToken: proTierToken}
}

func (a *odigosCloudResourceManager) Name() string { return "Odigos Pro" }

func (a *odigosCloudResourceManager) InstallFromScratch(ctx context.Context) error {

	var secret *corev1.Secret

	if a.proTierToken == nil {

		// no-op - just apply the resources again to make sure the labels are up to date.
		existingSecret, err := getCurrentOdigosProSecret(ctx, a.client, a.ns)
		if err != nil || existingSecret == nil {
			return err
		}

		// Without the following line, I get an error:
		// ERROR metadata.managedFields must be nil
		// But not sure if this is the right way to fix it.
		existingSecret.ManagedFields = nil

		secret = existingSecret
	} else {

		var cloudApiKey = ""
		if a.odigosTier == common.CloudOdigosTier {
			cloudApiKey = *a.proTierToken
		}

		var onpremToken = ""
		if a.odigosTier == common.OnPremOdigosTier {
			onpremToken = *a.proTierToken
		}

		secret = newOdigosProSecret(a.ns, cloudApiKey, onpremToken)
	}

	resources := []client.Object{secret}
	return a.client.ApplyResources(ctx, a.config.ConfigVersion, resources)
}
