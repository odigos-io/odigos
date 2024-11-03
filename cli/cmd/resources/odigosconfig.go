package resources

import (
	"context"
	"encoding/json"

	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewOdigosConfiguration(ns string, config *common.OdigosConfiguration) (kube.Object, error) {
	data, err := json.Marshal(config)
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

type odigosConfigResourceManager struct {
	client     *kube.Client
	ns         string
	config     *common.OdigosConfiguration
	odigosTier common.OdigosTier
}

func NewOdigosConfigResourceManager(client *kube.Client, ns string, config *common.OdigosConfiguration, odigosTier common.OdigosTier) resourcemanager.ResourceManager {
	return &odigosConfigResourceManager{client: client, ns: ns, config: config, odigosTier: odigosTier}
}

func (a *odigosConfigResourceManager) Name() string { return "OdigosConfig" }

func (a *odigosConfigResourceManager) InstallFromScratch(ctx context.Context) error {

	obj, err := NewOdigosConfiguration(a.ns, a.config)
	if err != nil {
		return err
	}

	resources := []kube.Object{
		obj,
	}
	return a.client.ApplyResources(ctx, a.config.ConfigVersion, resources)
}
