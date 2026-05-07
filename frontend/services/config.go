package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/config"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func GetConfigYamls() ([]*model.ConfigYaml, error) {
	var resp []*model.ConfigYaml

	for _, cfg := range config.Get() {
		var fields []*model.ConfigYamlField
		for _, f := range cfg.Spec.Fields {
			field := &model.ConfigYamlField{
				DisplayName:      f.DisplayName,
				ComponentType:    model.FieldType(f.ComponentType),
				IsHelmOnly:       f.IsHelmOnly,
				IsEnterpriseOnly: f.IsEnterpriseOnly,
				Description:      f.Description,
				HelmValuePath:    f.HelmValuePath,
			}

			if f.DocsLink != "" {
				field.DocsLink = &f.DocsLink
			}

			if len(f.ComponentProps) > 0 {
				propsJSON, err := json.Marshal(f.ComponentProps)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal component props: %w", err)
				}
				s := string(propsJSON)
				field.ComponentProps = &s
			}

			fields = append(fields, field)
		}

		resp = append(resp, &model.ConfigYaml{
			Name:        cfg.Metadata.Name,
			DisplayName: cfg.Metadata.DisplayName,
			Fields:      fields,
		})
	}

	return resp, nil
}

// GetEffectiveConfigWithRawYAML retrieves the effective config along with its raw YAML representation.
func GetEffectiveConfigWithRawYAML(ctx context.Context, c client.Client) (*common.OdigosConfiguration, string, error) {
	ns := env.GetCurrentNamespace()

	var cm v1.ConfigMap
	err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: consts.OdigosEffectiveConfigName}, &cm)
	if err != nil {
		return nil, "", client.IgnoreNotFound(err)
	}

	if cm.Data == nil || cm.Data[consts.OdigosConfigurationFileName] == "" {
		return nil, "", nil
	}

	rawYAML := cm.Data[consts.OdigosConfigurationFileName]

	var odigosConfig common.OdigosConfiguration
	if err := yaml.Unmarshal([]byte(rawYAML), &odigosConfig); err != nil {
		return nil, "", fmt.Errorf("failed to parse odigos config: %w", err)
	}

	return &odigosConfig, rawYAML, nil
}
