package yamls

import (
	"embed"

	"github.com/odigos-io/odigos/distros/distro"
	"gopkg.in/yaml.v3"
)

//go:embed *.yaml
var embeddedFiles embed.FS

func GetFS() embed.FS {
	return embeddedFiles
}

type distroResource struct {
	ApiVersion string            `json:"apiVersion"`
	Spec       distro.OtelDistro `json:"spec"`
}

func GetDistrosMap() (map[string]*distro.OtelDistro, error) {
	files, err := embeddedFiles.ReadDir(".")
	if err != nil {
		return nil, err
	}

	distrosByName := make(map[string]*distro.OtelDistro)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		yamlBytes, err := embeddedFiles.ReadFile(filename)
		if err != nil {
			return nil, err
		}

		otelDistro := distroResource{}
		err = yaml.Unmarshal(yamlBytes, &otelDistro)
		if err != nil {
			return nil, err
		}

		distrosByName[otelDistro.Spec.Name] = &otelDistro.Spec
	}

	return distrosByName, nil
}
