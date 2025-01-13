package manifests

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"reflect"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
	"sigs.k8s.io/yaml"
)

//go:embed *.yaml
var embeddedFiles embed.FS

func ReadProfileYamlManifests(profileName common.ProfileName) ([][]byte, error) {

	// TODO: allow multiple files per profiles with any name (not just profileName.yaml)
	filename := fmt.Sprintf("%s.yaml", profileName)
	yamlBytes, err := embeddedFiles.ReadFile(filename)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return [][]byte{}, nil
		}
	}

	return [][]byte{yamlBytes}, nil
}

// GetEmbeddedResourceManifestsAsObjects is a generic function to read embedded YAML files and convert them into runtime.Object
func GetEmbeddedResourceManifestsAsObjects(filename string, obj profile.K8sObject) ([]profile.K8sObject, error) {

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

	k8sObj, ok := newObj.(profile.K8sObject)
	if !ok {
		return nil, fmt.Errorf("unmarshaled object is not a k8s object")
	}

	// Return the object wrapped in a slice
	return []profile.K8sObject{k8sObj}, nil
}
