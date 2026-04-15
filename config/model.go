package config

type Configuration struct {
	ApiVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   ConfigurationMeta `yaml:"metadata"`
	Spec       ConfigurationSpec `yaml:"spec"`
}

type ConfigurationMeta struct {
	Name        string `yaml:"name"`
	DisplayName string `yaml:"displayName"`
}

type ConfigurationSpec struct {
	Fields []ConfigurationField `yaml:"fields"`
}

type ConfigurationField struct {
	DisplayName      string                 `yaml:"displayName"`
	ComponentType    string                 `yaml:"componentType"`
	IsHelmOnly       bool                   `yaml:"isHelmOnly"`
	IsEnterpriseOnly bool                   `yaml:"isEnterpriseOnly"`
	Description      string                 `yaml:"description"`
	HelmValuePath    string                 `yaml:"helmValuePath"`
	DocsLink         string                 `yaml:"docsLink,omitempty"`
	ComponentProps   map[string]interface{} `yaml:"componentProps,omitempty"`
}
