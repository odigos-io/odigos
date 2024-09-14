package profiles

import (
	"embed"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

//go:embed *.yaml
var embeddedFiles embed.FS

// GetEmbeddedYAMLFilesAsObjects reads embedded YAML files and converts them into runtime.Object
func GetEmbeddedYAMLFilesAsObjects() ([]runtime.Object, error) {
	// List all the embedded YAML files
	files, err := embeddedFiles.ReadDir("config")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded files: %v", err)
	}

	var objects []runtime.Object

	for _, file := range files {
		// Skip directories if any
		if file.IsDir() {
			continue
		}

		// Read the embedded YAML file content
		filePath := "config/" + file.Name()
		yamlBytes, err := embeddedFiles.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read embedded file %s: %v", file.Name(), err)
		}

		// Unmarshal the YAML into an unstructured object
		var unstructuredObj unstructured.Unstructured
		err = yaml.Unmarshal(yamlBytes, &unstructuredObj)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal YAML file %s: %v", file.Name(), err)
		}

		// Convert to runtime.Object
		obj := &unstructuredObj
		objects = append(objects, obj)
	}

	return objects, nil
}
