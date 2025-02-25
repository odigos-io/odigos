package distros

import (
	"embed"
	"errors"
	"fmt"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/distros/yamls"

	"gopkg.in/yaml.v3"
)

type Defaulter interface {
	GetDefaultDistroNames() map[common.ProgrammingLanguage]string
}

type communityDefaulter struct{}

var _ Defaulter = &communityDefaulter{}

func NewCommunityDefaulter() Defaulter {
	return &communityDefaulter{}
}

func NewCommunityGetter() (*Getter, error) {
	return NewGetterFromFS(yamls.GetFS())
}

func NewGetterFromFS(fs embed.FS) (*Getter, error) {
	g := Getter{}

	distrosByName, err := GetDistrosMap(fs)
	if err != nil {
		return nil, err
	}

	g.distrosByName = distrosByName

	return &g, nil
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

type Getter struct {
	distrosByName map[string]*distro.OtelDistro
}

func (g *Getter) GetDistroByName(distroName string) *distro.OtelDistro {
	return g.distrosByName[distroName]
}

// GetAllDistros returns all the distributions available in the getter.
// used in the enterprise repo
func (g *Getter) GetAllDistros() []*distro.OtelDistro {
	distros := make([]*distro.OtelDistro, 0, len(g.distrosByName))
	for _, d := range g.distrosByName {
		distros = append(distros, d)
	}
	return distros
}

type Provider struct {
	Defaulter
	*Getter
}

// NewProvider creates a new distributions provider.
// A provider is a combination of a defaulter and a getter.
// The defaulter is used to get the default distro names for each programming language.
// The getter is used to get the distro object itself from the available distros.
//
// A provider is constructed from a single defaulter and one or more getters.
// The getters are unioned together to create a single getter for the provider.
//
// Each default distribution must be provided by at least one of the getters.
func NewProvider(defaulter Defaulter, getters ...*Getter) (*Provider, error) {
	if len(getters) == 0 {
		return nil, errors.New("at least one getter must be provided")
	}

	distros := make(map[string]*distro.OtelDistro)
	for _, g := range getters {
		for k, v := range g.distrosByName {
			distros[k] = v
		}
	}

	// make sure the default distributions are provided by at least one of the getters
	defaultDistroNames := defaulter.GetDefaultDistroNames()
	for _, distroName := range defaultDistroNames {
		if _, ok := distros[distroName]; !ok {
			return nil, fmt.Errorf("default distribution %s not found in any getter", distroName)
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

func GetDistrosMap(fs embed.FS) (map[string]*distro.OtelDistro, error) {
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
