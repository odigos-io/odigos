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

type InspectFunc func(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool)

// Inspector holds two kinds of checks as well as an optional runtime version getter.
type Inspector interface {
	LightCheck(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool)
	ExpensiveCheck(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool)
}

type LanguageInspector interface {
	Inspect(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool)
}
type VersionInspector interface {
	GetRuntimeVersion(ctx *process.ProcessContext, containerURL string) *version.Version
}

var inspectorsByLanguage = map[common.ProgrammingLanguage]Inspector{
	common.JavaProgrammingLanguage:       &java.JavaInspector{},
	common.DotNetProgrammingLanguage:     &dotnet.DotnetInspector{},
	common.GoProgrammingLanguage:         &golang.GolangInspector{},
	common.PythonProgrammingLanguage:     &python.PythonInspector{},
	common.JavascriptProgrammingLanguage: &nodejs.NodejsInspector{},
	common.MySQLProgrammingLanguage:      &mysql.MySQLInspector{},
	common.NginxProgrammingLanguage:      &nginx.NginxInspector{},
}

// runInspectionStage iterates over the inspectors using the check provided by checkSelector.
// It returns a ProgramLanguageDetails with the detected language (and runtime version, if available).
// If multiple inspectors return different languages, it returns an error.
func runInspectionStage(
	procContext *process.ProcessContext,
	containerURL string,
	inspector Inspector,
	inspectFunc InspectFunc,
) (common.ProgramLanguageDetails, error) {
	detectedProgramLanguageDetails := common.ProgramLanguageDetails{
		Language: common.UnknownProgrammingLanguage,
	}
	if languageDetected, detected := inspectFunc(procContext); detected {
		// First detection: assign language and runtime version if available.
		if detectedProgramLanguageDetails.Language == common.UnknownProgrammingLanguage {
			detectedProgramLanguageDetails.Language = languageDetected
			if v, ok := inspector.(VersionInspector); ok {
				detectedProgramLanguageDetails.RuntimeVersion = v.GetRuntimeVersion(&process.ProcessContext{}, containerURL)

			}
		}
		// If a conflict is found, return an error.
		if detectedProgramLanguageDetails.Language != languageDetected {
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

// DetectLanguage returns the detected language for the process or
// common.UnknownProgrammingLanguage if the language could not be detected, in which case error == nil
// if error or language detectors disagree common.UnknownProgrammingLanguage is also returned.
// DetectLanguage creates a process context, runs the light checks first,
// and if no language is detected, falls back to the expensive checks.
func DetectLanguage(proc process.Details, containerURL string) (common.ProgramLanguageDetails, error) {
	procContext := process.NewProcessContext(proc)
	defer procContext.CloseFiles()

	detectedLanguageDetailes := common.ProgramLanguageDetails{
		Language: common.UnknownProgrammingLanguage,
	}

	for _, inspector := range inspectorsByLanguage {
		// Stage 1: Low-Cost (light) Checks
		detectedLanguage, err := runInspectionStage(procContext, containerURL, inspector, func(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
			return inspector.LightCheck(ctx)
		})
		if detectedLanguage.Language != common.UnknownProgrammingLanguage {
			detectedLanguageDetailes = detectedLanguage
		}
		// if we found double match, return common.UnknownProgrammingLanguage and the error
		if err != nil {
			return detectedLanguage, err
		}
	}
	if detectedLanguageDetailes.Language != common.UnknownProgrammingLanguage {
		return detectedLanguageDetailes, nil
	} else {
		// if no language was detected in stage 1, run stage 2
		for _, inspector := range inspectorsByLanguage {
			// Stage 2: Expensive Checks (only if no language was detected in stage 1)
			detectedLanguage, err := runInspectionStage(procContext, containerURL, inspector, func(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
				return inspector.ExpensiveCheck(ctx)
			})
			if detectedLanguage.Language != common.UnknownProgrammingLanguage {
				detectedLanguageDetailes = detectedLanguage
			}
			// if we found double match, return common.UnknownProgrammingLanguage and the error
			if err != nil {
				return detectedLanguage, err
			}
		}
		if detectedLanguageDetailes.Language != common.UnknownProgrammingLanguage {
			return detectedLanguageDetailes, nil
		}
	}
	return common.ProgramLanguageDetails{
		Language: common.UnknownProgrammingLanguage,
	}, nil
}

func VerifyLanguage(proc process.Details, lang common.ProgrammingLanguage) bool {
	inspector, ok := inspectorsByLanguage[lang]
	if !ok {
		return false
	}
	procContext := process.NewProcessContext(proc)

	_, lightDetected := inspector.LightCheck(procContext)
	if lightDetected {
		return true
	}
	_, expensiveDetected := inspector.ExpensiveCheck(procContext)
	return expensiveDetected
}
