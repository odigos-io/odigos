package status

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// OdigosStatus is the root object for an Odigos status definition.
type OdigosStatus struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

type Metadata struct {
	Name string `yaml:"name"`
}

type Spec struct {
	Type    string   `yaml:"type"`
	Reasons []Reason `yaml:"reasons"`
}

type Reason struct {
	Name               string                 `yaml:"name"`
	Message            string                 `yaml:"message"`
	K8sConditionStatus metav1.ConditionStatus `yaml:"k8sConditionStatus,omitempty"`
	OdigosSeverity     OdigosSeverity         `yaml:"odigosSeverity"`
}
