package profiles

import (
	"embed"
	"fmt"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

//go:embed *.yaml
var embeddedFiles embed.FS

// GetEmbeddedYAMLFilesAsObjects reads embedded YAML files and converts them into runtime.Object
func GetEmbeddedYAMLInstrumentationRuleFileAsObjects(filename string) ([]client.Object, error) {

	// Read the embedded YAML file content
	yamlBytes, err := embeddedFiles.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded file %s: %v", filename, err)
	}

	var instrumentationRule odigosv1alpha1.InstrumentationRule
	err = yaml.Unmarshal(yamlBytes, &instrumentationRule)
	if err != nil {
		return nil, err
	}

	return []client.Object{&instrumentationRule}, nil
}
