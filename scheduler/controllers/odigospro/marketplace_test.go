package odigospro

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	corev1 "k8s.io/api/core/v1"
)

func TestMarketplaceClearsTokenMetadata(t *testing.T) {
	config := &corev1.ConfigMap{Data: map[string]string{
		k8sconsts.OdigosDeploymentConfigMapOnPremTokenAudKey:       "previous-customer",
		k8sconsts.OdigosDeploymentConfigMapOnPremTokenExpKey:       "previous-expiry",
		k8sconsts.OdigosDeploymentConfigMapOnPremClientProfilesKey: "previous-profile",
		k8sconsts.OdigosDeploymentConfigMapTierKey:                 "onprem",
	}}
	secret := &corev1.Secret{Data: map[string][]byte{"aws-marketplace": []byte("true")}}
	if err := updateProInfoInConfigMap(config, secret); err != nil {
		t.Fatal(err)
	}
	if len(config.Data) != 1 || config.Data[k8sconsts.OdigosDeploymentConfigMapTierKey] != "onprem" {
		t.Fatalf("stale token metadata retained or tier changed: %v", config.Data)
	}
}
