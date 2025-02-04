package yamls

import (
	"embed"
	"fmt"

	"github.com/odigos-io/odigos/distros"
	"gopkg.in/yaml.v3"
)

//go:embed *.yaml
var embeddedFiles embed.FS

func ReadDistroFromYamlManifest(distroName string) (*distros.OtelDistro, error) {
	// TODO: allow multiple files per profiles with any name (not just profileName.yaml)
	filename := fmt.Sprintf("%s.yaml", distroName)
	yamlBytes, err := embeddedFiles.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	otelDistro := distros.OtelDistro{}
	err = yaml.Unmarshal(yamlBytes, &otelDistro)
	if err != nil {
		return nil, err
	}

	return &otelDistro, nil
}
