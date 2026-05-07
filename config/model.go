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
	// RenderCondition gates whether the UI shows this field based on the value of
	// another field in the same config. Mirrors the destination-form renderCondition
	// shape: either ["true"|"false"] or [helmValuePath, comparison, value], where
	// comparison is one of "==", "!=", "===", "!==", ">", "<", ">=", "<=".
	RenderCondition []string `yaml:"renderCondition,omitempty"`
}
