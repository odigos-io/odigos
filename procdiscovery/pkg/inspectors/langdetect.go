package inspectors

import (
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/procdiscovery/pkg/inspectors/dotnet"
	"github.com/keyval-dev/odigos/procdiscovery/pkg/inspectors/golang"
	"github.com/keyval-dev/odigos/procdiscovery/pkg/inspectors/java"
	"github.com/keyval-dev/odigos/procdiscovery/pkg/inspectors/nodejs"
	"github.com/keyval-dev/odigos/procdiscovery/pkg/inspectors/python"
	"github.com/keyval-dev/odigos/procdiscovery/pkg/process"
)

type inspector interface {
	Inspect(process *process.Details) (common.ProgrammingLanguage, bool)
}

var inspectorsList = []inspector{
	&golang.GolangInspector{},
	&java.JavaInspector{},
	&python.PythonInspector{},
	&dotnet.DotnetInspector{},
	&nodejs.NodejsInspector{},
}

type LanguageDetectionResult struct {
	Language common.ProgrammingLanguage
}

// DetectLanguage returns a list of all the detected languages in the process list
func DetectLanguage(processes []process.Details) []LanguageDetectionResult {
	var result []LanguageDetectionResult
	for _, p := range processes {
		for _, i := range inspectorsList {
			language, detected := i.Inspect(&p)
			if detected {
				detectionResult := LanguageDetectionResult{
					Language: language,
				}
				result = append(result, detectionResult)
				break
			}
		}
	}

	return result
}
