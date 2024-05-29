//go:build embed_manifests

package api

import (
	"embed"
	"strings"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"
)

//go:embed config/*
var DefaultFS embed.FS

const crdBasePath = "config/crd/bases"

// GetCRDs returns a list of CustomResourceDefinitions from the embedded manifests.
// excludeFiles is a list of files to exclude from the returned list.
func GetCRDs(excludeFiles []string) ([]*v1.CustomResourceDefinition, error) {
	files, err := DefaultFS.ReadDir(crdBasePath)
	if err != nil {
		return nil, err
	}

	var crds []*v1.CustomResourceDefinition

FilesLoop:
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		for _, excludeFile := range excludeFiles {
			if strings.Contains(file.Name(), excludeFile) {
				continue FilesLoop
			}
		}

		crdBytes, err := DefaultFS.ReadFile(crdBasePath + "/" + file.Name())
		if err != nil {
			return nil, err
		}

		var crd v1.CustomResourceDefinition
		err = yaml.Unmarshal(crdBytes, &crd)
		if err != nil {
			return nil, err
		}
		crds = append(crds, &crd)
	}

	return crds, nil
}