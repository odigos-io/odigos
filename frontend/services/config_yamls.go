package services

import (
	"encoding/json"

	"github.com/odigos-io/odigos/config"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

func GetConfigYamls() model.GetConfigYamls {
	var resp model.GetConfigYamls

	for _, cfg := range config.Get() {
		var fields []*model.ConfigYamlField
		for _, f := range cfg.Spec.Fields {
			field := &model.ConfigYamlField{
				DisplayName:   f.DisplayName,
				ComponentType: f.ComponentType,
				IsHelmOnly:    f.IsHelmOnly,
				Description:   f.Description,
				HelmValuePath: f.HelmValuePath,
			}

			if f.DocsLink != "" {
				field.DocsLink = &f.DocsLink
			}

			if len(f.ComponentProps) > 0 {
				propsJSON, err := json.Marshal(f.ComponentProps)
				if err == nil {
					s := string(propsJSON)
					field.ComponentProps = &s
				}
			}

			fields = append(fields, field)
		}

		resp.Configs = append(resp.Configs, &model.ConfigYaml{
			Name:        cfg.Metadata.Name,
			DisplayName: cfg.Metadata.DisplayName,
			Fields:      fields,
		})
	}

	return resp
}
