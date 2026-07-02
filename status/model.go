package status

type Status struct {
	ApiVersion string     `yaml:"apiVersion"`
	Kind       string     `yaml:"kind"`
	Metadata   StatusMeta `yaml:"metadata"`
	Spec       StatusSpec `yaml:"spec"`
}

type StatusMeta struct {
	Name string `yaml:"name"`
}

type StatusSpec struct {
	Type    string         `yaml:"type"`
	Codegen *StatusCodegen `yaml:"codegen,omitempty"`
	Reasons []StatusReason `yaml:"reasons"`
}

// StatusCodegen configures Go reason enum generation for this manifest.
type StatusCodegen struct {
	// StatusName is the generated const for spec.type (e.g. "RollbackStatus").
	StatusName string `yaml:"statusName"`
	TypeName   string `yaml:"typeName"`
	ConstPrefix string `yaml:"constPrefix,omitempty"`
	Package    string `yaml:"package"`
	OutputFile string `yaml:"outputFile"`
}

type StatusReason struct {
	Name               string `yaml:"name"`
	Message            string `yaml:"message"`
	K8sConditionStatus string `yaml:"k8sConditionStatus,omitempty"`
	OdigosSeverity     string `yaml:"odigosSeverity"`
}
