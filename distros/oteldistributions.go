package distros

import (
	"embed"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros/distro"

	"gopkg.in/yaml.v3"
)

type Defaulter interface {
	GetDefaultDistroNames() map[common.ProgrammingLanguage]string
}

type communityDefaulter struct{}

var _ Defaulter = &communityDefaulter{}

// TODO: remove this once we have an enterprise instrumentor
func NewCommunityDefaulter() *communityDefaulter {
	return &communityDefaulter{}
}

// TODO: remove this once we have an enterprise instrumentor
func NewOnPremDefaulter() *onPremDefaulter {
	return &onPremDefaulter{}
}

func (c *communityDefaulter) GetDefaultDistroNames() map[common.ProgrammingLanguage]string {
	return map[common.ProgrammingLanguage]string{
		common.JavascriptProgrammingLanguage: "nodejs-community",
		common.PythonProgrammingLanguage:     "python-community",
		common.DotNetProgrammingLanguage:     "dotnet-community",
		common.JavaProgrammingLanguage:       "java-community",
		common.GoProgrammingLanguage:         "golang-community",
	}
}

// TODO: remove this once we have an enterprise instrumentor
type onPremDefaulter struct{}

var _ Defaulter = &onPremDefaulter{}

func (o *onPremDefaulter) GetDefaultDistroNames() map[common.ProgrammingLanguage]string {
	return map[common.ProgrammingLanguage]string{
		common.JavascriptProgrammingLanguage: "nodejs-enterprise",
		common.PythonProgrammingLanguage:     "python-enterprise",
		common.DotNetProgrammingLanguage:     "dotnet-community",
		common.JavaProgrammingLanguage:       "java-enterprise",
		common.GoProgrammingLanguage:         "golang-enterprise",
		common.MySQLProgrammingLanguage:      "mysql-enterprise",
	}
}

type Getter struct {
	distrosByName map[string]*distro.OtelDistro
}

func (g *Getter) GetDistroByName(distroName string) *distro.OtelDistro {
	return g.distrosByName[distroName]
}

type Provider struct {
	Defaulter
	*Getter
}

func NewProvider(defaulter Defaulter, fs ...embed.FS) (*Provider, error) {
	distros := make(map[string]*distro.OtelDistro)
	for _, f := range fs {
		currentDistros, err := getDistrosMap(f)
		if err != nil {
			return nil, err
		}
		for k, v := range currentDistros {
			distros[k] = v
		}
	}

	return &Provider{
		Defaulter: defaulter,
		Getter: &Getter{
			distrosByName: distros,
		},
	}, nil
}

type distroResource struct {
	ApiVersion string            `json:"apiVersion"`
	Spec       distro.OtelDistro `json:"spec"`
}

func getDistrosMap(fs embed.FS) (map[string]*distro.OtelDistro, error) {
	files, err := fs.ReadDir(".")
	if err != nil {
		return nil, err
	}

	distrosByName := make(map[string]*distro.OtelDistro)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		yamlBytes, err := fs.ReadFile(file.Name())
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
