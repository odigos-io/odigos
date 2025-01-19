package manifests

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/odigos-io/odigos/common"
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
