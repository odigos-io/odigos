package destinations

import "github.com/odigos-io/odigos/common"

type Destination struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

type Metadata struct {
	Type        common.DestinationType `yaml:"type"`
	DisplayName string                 `yaml:"displayName"`
	Category    string                 `yaml:"category"`
}

type Spec struct {
	Image   string `yaml:"image"`
	Signals struct {
		Traces struct {
			Supported bool `yaml:"supported"`
		}
		Metrics struct {
			Supported bool `yaml:"supported"`
		}
		Logs struct {
			Supported bool `yaml:"supported"`
		}
	}
	Fields                  []Field `yaml:"fields"`
	TestConnectionSupported bool    `yaml:"testConnectionSupported"`
}

type Field struct {
	Name            string                 `yaml:"name"`
	DisplayName     string                 `yaml:"displayName"`
	ComponentType   string                 `yaml:"componentType"`
	ComponentProps  map[string]interface{} `yaml:"componentProps"`
	Secret          bool                   `yaml:"secret"`
	InitialValue    string                 `yaml:"initialValue"`
	RenderCondition []string               `yaml:"renderCondition"`
}
