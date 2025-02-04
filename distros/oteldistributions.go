package distros

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/distros/yamls"
)

func GetDefaultDistroNames(tier common.OdigosTier) map[common.ProgrammingLanguage]string {
	switch tier {
	case common.CommunityOdigosTier:
		return map[common.ProgrammingLanguage]string{
			common.JavascriptProgrammingLanguage: "nodejs-community",
			common.PythonProgrammingLanguage:     "python-community",
			common.DotNetProgrammingLanguage:     "dotnet-community",
			common.JavaProgrammingLanguage:       "java-community",
			common.GoProgrammingLanguage:         "golang-community",
		}
	case common.OnPremOdigosTier:
		return map[common.ProgrammingLanguage]string{
			common.JavascriptProgrammingLanguage: "nodejs-enterprise",
			common.PythonProgrammingLanguage:     "python-enterprise",
			common.DotNetProgrammingLanguage:     "dotnet-enterprise",
			common.JavaProgrammingLanguage:       "java-enterprise",
			common.GoProgrammingLanguage:         "golang-enterprise",
		}
	default:
		return nil
	}
}

var allDistros = []distro.OtelDistro{}

func init() {
	distroNames := yamls.GetAllDistroNames()
	for _, distroName := range distroNames {
		distro, err := yamls.ReadDistroFromYamlManifest(distroName)
		if err != nil {
			continue
		}
		allDistros = append(allDistros, *distro)
	}
}

func GetDistroByName(distroName string) *distro.OtelDistro {
	for i := range allDistros {
		if allDistros[i].Name == distroName {
			return &allDistros[i]
		}
	}
	return nil
}
