package resources

import (
	"context"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/cli/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OdigosConfigName = "odigos-config"
)

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
	client *kube.Client
	ns     string
	config *odigosv1.OdigosConfigurationSpec
}

func NewOdigosConfigResourceManager(client *kube.Client, ns string, config *odigosv1.OdigosConfigurationSpec) ResourceManager {
	return &odigosConfigResourceManager{client: client, ns: ns, config: config}
}

func (a *odigosConfigResourceManager) Name() string { return "OdigosConfig" }

func (a *odigosConfigResourceManager) InstallFromScratch(ctx context.Context) error {
	resources := []client.Object{
		NewOdigosConfiguration(a.ns, a.config),
	}
	return a.client.ApplyResources(ctx, a.config.OdigosVersion, resources)
}
