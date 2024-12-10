package inspectors

import (
	"fmt"

	"github.com/hashicorp/go-version"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/dotnet"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/golang"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/java"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/mysql"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/nginx"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/nodejs"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/python"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type ErrLanguageDetectionConflict struct {
	languages [2]common.ProgrammingLanguage
}

func (e ErrLanguageDetectionConflict) Error() string {
	return fmt.Sprintf("language detection conflict between %v and %v", e.languages[0], e.languages[1])
}

type LanguageInspector interface {
	Inspect(process *process.Details) (common.ProgrammingLanguage, bool)
}

type VersionInspector interface {
	GetRuntimeVersion(process *process.Details, containerURL string) *version.Version
}

var inspectorsList = []LanguageInspector{
	&golang.GolangInspector{},
	&java.JavaInspector{},
	&dotnet.DotnetInspector{},
	&nodejs.NodejsInspector{},
	&python.PythonInspector{},
	&mysql.MySQLInspector{},
	&nginx.NginxInspector{},
}

// DetectLanguage returns the detected language for the process or
// common.UnknownProgrammingLanguage if the language could not be detected, in which case error == nil
// if error or language detectors disagree common.UnknownProgrammingLanguage is also returned
func DetectLanguage(process process.Details, containerURL string) (common.ProgramLanguageDetails, error) {
	detectedProgramLanguageDetails := common.ProgramLanguageDetails{
		Language: common.UnknownProgrammingLanguage,
	}

	for _, i := range inspectorsList {
		languageDetected, detected := i.Inspect(&process)
		if detected {
			if detectedProgramLanguageDetails.Language == common.UnknownProgrammingLanguage {
				detectedProgramLanguageDetails.Language = languageDetected
				if v, ok := i.(VersionInspector); ok {
					detectedProgramLanguageDetails.RuntimeVersion = v.GetRuntimeVersion(&process, containerURL)
				}
				continue
			}
			return common.ProgramLanguageDetails{
					Language: common.UnknownProgrammingLanguage,
				}, ErrLanguageDetectionConflict{
					languages: [2]common.ProgrammingLanguage{
						detectedProgramLanguageDetails.Language,
						languageDetected,
					},
				}
		}
	}

	return detectedProgramLanguageDetails, nil
}
