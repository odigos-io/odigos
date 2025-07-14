package resources

import (
	"context"

	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func NewOdigosConfiguration(ns string, config *common.OdigosConfiguration) (kube.Object, error) {
	data, err := yaml.Marshal(config)
	if err != nil {
		return nil, err
	}
	return &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosConfigurationName,
			Namespace: ns,
		},
		Data: map[string]string{
			consts.OdigosConfigurationFileName: string(data),
		},
	}, nil
}

type odigosConfigurationResourceManager struct {
	client      *kube.Client
	ns          string
	config      *common.OdigosConfiguration
	odigosTier  common.OdigosTier
	managerOpts resourcemanager.ManagerOpts
}

func NewOdigosConfigurationResourceManager(client *kube.Client, ns string, config *common.OdigosConfiguration, odigosTier common.OdigosTier, managerOpts resourcemanager.ManagerOpts) resourcemanager.ResourceManager {
	return &odigosConfigurationResourceManager{client: client, ns: ns, config: config, odigosTier: odigosTier, managerOpts: managerOpts}
}

func (a *odigosConfigurationResourceManager) Name() string { return "OdigosConfiguration" }

func (a *odigosConfigurationResourceManager) InstallFromScratch(ctx context.Context) error {

	obj, err := NewOdigosConfiguration(a.ns, a.config)
	if err != nil {
		return err
	}

	resources := []kube.Object{
		obj,
	}
	return a.client.ApplyResources(ctx, a.config.ConfigVersion, resources, a.managerOpts)
}
