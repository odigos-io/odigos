package yamls

import (
	"embed"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/distros/distro"
	"gopkg.in/yaml.v3"
)

//go:embed *.yaml
var embeddedFiles embed.FS

func GetAllDistroNames() []string {
	files, err := embeddedFiles.ReadDir(".")
	if err != nil {
		return nil
	}

	distroNames := make([]string, 0, len(files))
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		fileName := file.Name()
		fileParts := strings.Split(fileName, ".")
		if len(fileParts) != 2 || fileParts[1] != "yaml" {
			continue
		}
		distroNames = append(distroNames, fileParts[0])
	}

	return distroNames
}

type distroResource struct {
	ApiVersion string            `json:"apiVersion"`
	Spec       distro.OtelDistro `json:"spec"`
}

func ReadDistroFromYamlManifest(distroName string) (*distro.OtelDistro, error) {
	filename := fmt.Sprintf("%s.yaml", distroName)
	yamlBytes, err := embeddedFiles.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	otelDistro := distroResource{}
	err = yaml.Unmarshal(yamlBytes, &otelDistro)
	if err != nil {
		return nil, err
	}

	return &otelDistro.Spec, nil
}
