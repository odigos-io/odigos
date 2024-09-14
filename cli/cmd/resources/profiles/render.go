package profiles

import (
	"embed"
	"fmt"
	"reflect"

	actions "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

//go:embed *.yaml
var embeddedFiles embed.FS

// GetEmbeddedYAMLFileAsObjects is a generic function to read embedded YAML files and convert them into runtime.Object
func GetEmbeddedYAMLFileAsObjects(filename string, obj client.Object) ([]client.Object, error) {

	// Read the embedded YAML file content
	yamlBytes, err := embeddedFiles.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded file %s: %v", filename, err)
	}

	// Create a new instance of the passed object type
	objType := reflect.TypeOf(obj).Elem()
	newObj := reflect.New(objType).Interface()

	// Unmarshal the YAML content into the new object
	err = yaml.Unmarshal(yamlBytes, newObj)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %v", err)
	}

	// Return the object wrapped in a slice
	return []client.Object{newObj.(client.Object)}, nil
}

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

func GetEmbeddedYAMLRenameAttributeActionFileAsObjects(filename string) ([]client.Object, error) {

	// Read the embedded YAML file content
	yamlBytes, err := embeddedFiles.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded file %s: %v", filename, err)
	}

	var action actions.RenameAttribute
	err = yaml.Unmarshal(yamlBytes, &action)
	if err != nil {
		return nil, err
	}

	return []client.Object{&action}, nil
}

func GetEmbeddedYAMLProcessorFileAsObjects(filename string) ([]client.Object, error) {

	// Read the embedded YAML file content
	yamlBytes, err := embeddedFiles.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded file %s: %v", filename, err)
	}

	var processor odigosv1alpha1.Processor
	err = yaml.Unmarshal(yamlBytes, &processor)
	if err != nil {
		return nil, err
	}

	return []client.Object{&processor}, nil
}
