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

type InspectFunc func(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool)

// Inspector holds two kinds of checks as well as an optional runtime version getter.
type Inspector interface {
	QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool)
	DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool)
}

type LanguageInspector interface {
	Inspect(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool)
}
type VersionInspector interface {
	GetRuntimeVersion(pcx *process.ProcessContext, containerURL string) *version.Version
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
	detectedLanguageDetailes *common.ProgramLanguageDetails,
	inspector Inspector,
	inspectFunc InspectFunc,
) (common.ProgramLanguageDetails, error) {
	if languageDetected, detected := inspectFunc(procContext); detected {
		// First detection: assign language and runtime version if available.
		if detectedLanguageDetailes.Language == common.UnknownProgrammingLanguage {
			detectedLanguageDetailes.Language = languageDetected
			if v, ok := inspector.(VersionInspector); ok {
				detectedLanguageDetailes.RuntimeVersion = v.GetRuntimeVersion(&process.ProcessContext{}, containerURL)

			}
		}
		// If a conflict is found, return an error.
		if detectedLanguageDetailes.Language != languageDetected {
			return common.ProgramLanguageDetails{
					Language: common.UnknownProgrammingLanguage,
				}, ErrLanguageDetectionConflict{
					languages: [2]common.ProgrammingLanguage{
						detectedLanguageDetailes.Language,
						languageDetected,
					},
				}
		}
	}

	return *detectedLanguageDetailes, nil
}

// DetectLanguage returns the detected language for the process or
// common.UnknownProgrammingLanguage if the language could not be detected, in which case error == nil
// if error or language detectors has conflict common.UnknownProgrammingLanguage is also returned.
// DetectLanguage creates a process context, runs the light checks first,
// and if no language is detected, falls back to the expensive checks.
func DetectLanguage(proc process.Details, containerURL string) (common.ProgramLanguageDetails, error) {
	procContext := process.NewProcessContext(proc)
	defer procContext.CloseFiles()
	// TODO: add error handling for CloseFiles function
	detectedLanguageDetails := common.ProgramLanguageDetails{
		Language: common.UnknownProgrammingLanguage,
	}

	// Helper function to run inspection for each process
	runInspection := func(scanFunc func(Inspector) InspectFunc) (common.ProgramLanguageDetails, error) {
		for _, inspector := range inspectorsByLanguage {
			if detectedLanguage, err := runInspectionStage(procContext, containerURL, &detectedLanguageDetails, inspector, scanFunc(inspector)); err != nil || detectedLanguage.Language != common.UnknownProgrammingLanguage {
				return detectedLanguage, err
			}
		}
		return detectedLanguageDetails, nil
	}

	// Stage 1: Quick Scan
	if detectedLanguage, err := runInspection(func(inspector Inspector) InspectFunc {
		return inspector.QuickScan
	}); err != nil || detectedLanguage.Language != common.UnknownProgrammingLanguage {
		return detectedLanguage, err
	}

	// Stage 2: Deep Scan (only if no language was detected in Stage 1)
	if detectedLanguage, err := runInspection(func(inspector Inspector) InspectFunc {
		return inspector.DeepScan
	}); err != nil || detectedLanguage.Language != common.UnknownProgrammingLanguage {
		return detectedLanguage, err
	}

	return detectedLanguageDetails, nil
}

func VerifyLanguage(proc process.Details, lang common.ProgrammingLanguage) bool {
	inspector, ok := inspectorsByLanguage[lang]
	if !ok {
		return false
	}
	procContext := process.NewProcessContext(proc)

	_, lightDetected := inspector.QuickScan(procContext)
	if lightDetected {
		return true
	}
	_, expensiveDetected := inspector.DeepScan(procContext)
	return expensiveDetected
}
