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
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func getOdigosConfigFromConfigMap(ctx context.Context, c client.Client, configMapName string) (*common.OdigosConfiguration, error) {
	ns := env.GetCurrentNamespace()

	var cm v1.ConfigMap
	err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: configMapName}, &cm)
	if err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	if cm.Data == nil || cm.Data[consts.OdigosConfigurationFileName] == "" {
		return nil, nil
	}

	var odigosConfig common.OdigosConfiguration
	if err := yaml.Unmarshal([]byte(cm.Data[consts.OdigosConfigurationFileName]), &odigosConfig); err != nil {
		return nil, fmt.Errorf("failed to parse odigos config: %w", err)
	}

	return &odigosConfig, nil
}

// GetEffectiveConfig retrieves the current effective configuration from the effective-config ConfigMap.
func GetEffectiveConfig(ctx context.Context, c client.Client) (*common.OdigosConfiguration, error) {
	config, _, _, err := GetEffectiveConfigWithRawYAML(ctx, c)
	return config, err
}

// GetHelmDeploymentConfig retrieves the current helm deployment configuration from the odigos-helm-deployment-config ConfigMap.
func GetHelmDeploymentConfig(ctx context.Context, c client.Client) (*common.OdigosConfiguration, error) {
	return getOdigosConfigFromConfigMap(ctx, c, consts.OdigosConfigurationName)
}

// GetRemoteConfig retrieves the current remote configuration from the odigos-remote-config ConfigMap.
func GetRemoteConfig(ctx context.Context, c client.Client) (*common.OdigosConfiguration, error) {
	return getOdigosConfigFromConfigMap(ctx, c, consts.OdigosRemoteConfigName)
}

// GetLocalUIConfig retrieves the current local UI configuration from the odigos-local-ui-config ConfigMap.
func GetLocalUIConfig(ctx context.Context, c client.Client) (*common.OdigosConfiguration, error) {
	return getOdigosConfigFromConfigMap(ctx, c, consts.OdigosLocalUiConfigName)
}

func GetConfigYamls() ([]*model.ConfigYaml, error) {
	var resp []*model.ConfigYaml

	for _, cfg := range config.Get() {
		var fields []*model.ConfigYamlField
		for _, f := range cfg.Spec.Fields {
			field := &model.ConfigYamlField{
				DisplayName:   f.DisplayName,
				ComponentType: model.FieldType(f.ComponentType),
				IsHelmOnly:    f.IsHelmOnly,
				Description:   f.Description,
				HelmValuePath: f.HelmValuePath,
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

// GetEffectiveConfigWithRawYAML retrieves the effective config along with its raw YAML representation
// and the provenance map that records which ConfigMap each field originated from.
func GetEffectiveConfigWithRawYAML(ctx context.Context, c client.Client) (*common.OdigosConfiguration, string, map[string]string, error) {
	ns := env.GetCurrentNamespace()

	var cm v1.ConfigMap
	err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: consts.OdigosEffectiveConfigName}, &cm)
	if err != nil {
		return nil, "", nil, client.IgnoreNotFound(err)
	}

	if cm.Data == nil || cm.Data[consts.OdigosConfigurationFileName] == "" {
		return nil, "", nil, nil
	}

	rawYAML := cm.Data[consts.OdigosConfigurationFileName]

	var odigosConfig common.OdigosConfiguration
	if err := yaml.Unmarshal([]byte(rawYAML), &odigosConfig); err != nil {
		return nil, "", nil, fmt.Errorf("failed to parse odigos config: %w", err)
	}

	provenance := make(map[string]string)
	if provenanceYAML, ok := cm.Data[consts.OdigosConfigurationProvenanceFileName]; ok && provenanceYAML != "" {
		if err := yaml.Unmarshal([]byte(provenanceYAML), &provenance); err != nil {
			return nil, "", nil, fmt.Errorf("failed to parse provenance data: %w", err)
		}
	}

	return &odigosConfig, rawYAML, provenance, nil
}

func PersistUiLocalSamplingConfig(ctx context.Context, c client.Client, samplingConfig *common.SamplingConfiguration) error {
	ns := env.GetCurrentNamespace()

	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var cm v1.ConfigMap
		err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: consts.OdigosLocalUiConfigName}, &cm)
		if err != nil {
			return err
		}
		config := common.OdigosConfiguration{}
		if err := yaml.Unmarshal([]byte(cm.Data[consts.OdigosConfigurationFileName]), &config); err != nil {
			return err
		}
		config.Sampling = samplingConfig
		yamlText, err := yaml.Marshal(config)
		if err != nil {
			return err
		}
		cm.Data[consts.OdigosConfigurationFileName] = string(yamlText)
		return c.Update(ctx, &cm)
	})
}
