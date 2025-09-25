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
			// if true, it means that this destination will add spanmetrics connector by default
			// which will aggregate the spans for various opeartion and calculate metrics based on them.
			// some destinations are already doing this by default, and thus odigos does not need to do it again.
			// on-prem destinations, or those that only accept metrics and not traces, should set this to true.
			SpanMetricsEnabledByDefault bool `yaml:"spanMetricsEnabledByDefault"`
		}
		Logs struct {
			Supported bool `yaml:"supported"`
		}
	}
	Fields                  []Field `yaml:"fields"`
	TestConnectionSupported bool    `yaml:"testConnectionSupported"`
}

type CustomReadDataLabel struct {
	Condition string `yaml:"condition"`
	Title     string `yaml:"title"`
	Value     string `yaml:"value"`
}

type Field struct {
	Name                 string                 `yaml:"name"`
	DisplayName          string                 `yaml:"displayName"`
	ComponentType        string                 `yaml:"componentType"`
	ComponentProps       map[string]interface{} `yaml:"componentProps"`
	Secret               bool                   `yaml:"secret"`
	InitialValue         string                 `yaml:"initialValue"`
	RenderCondition      []string               `yaml:"renderCondition"`
	HideFromReadData     []string               `yaml:"hideFromReadData"`
	CustomReadDataLabels []*CustomReadDataLabel `yaml:"customReadDataLabels"`
}
