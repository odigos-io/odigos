package inspectors

import (
	"errors"
	"fmt"
	"os"

	"github.com/go-logr/logr"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/cplusplus"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/dotnet"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/golang"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/java"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/mysql"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/nginx"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/nodejs"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/php"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/postgres"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/python"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/redis"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/ruby"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/rust"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type ErrLanguageDetectionConflict struct {
	languages [2]common.ProgrammingLanguage
}

func (e ErrLanguageDetectionConflict) Error() string {
	return fmt.Sprintf("language detection conflict between %v and %v", e.languages[0], e.languages[1])
}

type InspectFunc func(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool)

// Inspector performs two types of scans (QuickScan and DeepScan), each using a
// different approach to determine the programming language of a given process.
type Inspector interface {
	QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool)
	DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool)
}

type VersionInspector interface {
	GetRuntimeVersion(pcx *process.ProcessContext, containerURL string) string
}

var inspectorsByLanguage = map[common.ProgrammingLanguage]Inspector{
	common.JavaProgrammingLanguage:       &java.JavaInspector{},
	common.DotNetProgrammingLanguage:     &dotnet.DotnetInspector{},
	common.GoProgrammingLanguage:         &golang.GolangInspector{},
	common.PythonProgrammingLanguage:     &python.PythonInspector{},
	common.JavascriptProgrammingLanguage: &nodejs.NodejsInspector{},
	common.PhpProgrammingLanguage:        &php.PhpInspector{},
	common.RubyProgrammingLanguage:       &ruby.RubyInspector{},
	common.RustProgrammingLanguage:       &rust.RustInspector{},
	common.CPlusPlusProgrammingLanguage:  &cplusplus.CPlusPlusInspector{},
	common.MySQLProgrammingLanguage:      &mysql.MySQLInspector{},
	common.NginxProgrammingLanguage:      &nginx.NginxInspector{},
	common.RedisProgrammingLanguage:      &redis.RedisInspector{},
	common.PostgresProgrammingLanguage:   &postgres.PostgresInspector{},
}

func runInspectionStage(
	procContext *process.ProcessContext,
	containerURL string,
	selectInspectionMethod func(Inspector) InspectFunc,
) (common.ProgramLanguageDetails, error) {
	detectedLanguageDetails := common.ProgramLanguageDetails{
		Language: common.UnknownProgrammingLanguage,
	}

	for _, inspector := range inspectorsByLanguage {
		inspectFunc := selectInspectionMethod(inspector)

		if languageDetected, detected := inspectFunc(procContext); detected {
			// First detection: assign language and runtime version if available
			if detectedLanguageDetails.Language == common.UnknownProgrammingLanguage {
				detectedLanguageDetails.Language = languageDetected
				if versionInspector, ok := inspector.(VersionInspector); ok {
					detectedLanguageDetails.RuntimeVersion = versionInspector.GetRuntimeVersion(procContext, containerURL)
				}
			} else if detectedLanguageDetails.Language != languageDetected {
				// Return error on language detection conflict
				return common.ProgramLanguageDetails{Language: common.UnknownProgrammingLanguage}, ErrLanguageDetectionConflict{
					languages: [2]common.ProgrammingLanguage{
						detectedLanguageDetails.Language,
						languageDetected,
					},
				}
			}
		}
	}
	return detectedLanguageDetails, nil
}

// DetectLanguage attempts to detect the programming language using QuickScan first, then DeepScan if needed.
func DetectLanguage(proc process.Details, containerURL string, logger logr.Logger) (common.ProgramLanguageDetails, error) {
	procContext := process.NewProcessContext(proc)
	defer func() {
		if err := procContext.CloseFiles(); err != nil {
			logger.Error(err, "Error closing files")
		}
	}()

	fmt.Fprintf(os.Stderr, "[DEBUG] Detecting Language for process: %+v\n", proc)

	// Try Quick Scan first
	if detectedLanguage, err := runInspectionStage(procContext, containerURL, func(inspector Inspector) InspectFunc {
		fmt.Fprintf(os.Stderr, "[DEBUG] Running Quick Scan with inspector: %+v\n", inspector)
		return inspector.QuickScan
	}); err != nil || detectedLanguage.Language != common.UnknownProgrammingLanguage {
		fmt.Fprintf(os.Stderr, "[DEBUG] Quick Scan result: %+v\n", detectedLanguage)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[DEBUG] Quick Scan error: %+v\n", err)
			var conflict ErrLanguageDetectionConflict
			if errors.As(err, &conflict) {
				// If one of the languages is C++, prefer the other language and log a warning
				if l, ok := resolveNonCpp(conflict.languages); ok {
					fmt.Fprintf(os.Stderr, "[DEBUG] Language detection conflict includes C++; preferring non-C++ language: %+v\n", l)
					logger.Info("Warning: language detection conflict includes C++; preferring non-C++ language", "languages", conflict.languages, "selected", l)
					resolved := common.ProgramLanguageDetails{Language: l}
					fmt.Fprintf(os.Stderr, "[DEBUG] Resolved language: %+v\n", resolved)
					if inspector, ok := inspectorsByLanguage[l]; ok {
						if vi, ok := inspector.(VersionInspector); ok {
							fmt.Fprintf(os.Stderr, "[DEBUG] Getting Runtime Version for language: %+v\n", l)
							resolved.RuntimeVersion = vi.GetRuntimeVersion(procContext, containerURL)
						}
					}
					return resolved, nil
				}
			}
		}
		return detectedLanguage, err
	}

	// Try Deep Scan if Quick Scan failed
	if detectedLanguage, err := runInspectionStage(procContext, containerURL, func(inspector Inspector) InspectFunc {
		return inspector.DeepScan
	}); err != nil || detectedLanguage.Language != common.UnknownProgrammingLanguage {
		if err != nil {
			var conflict ErrLanguageDetectionConflict
			if errors.As(err, &conflict) {
				if l, ok := resolveNonCpp(conflict.languages); ok {
					logger.Info("Warning: language detection conflict includes C++; preferring non-C++ language", "languages", conflict.languages, "selected", l)
					resolved := common.ProgramLanguageDetails{Language: l}
					if inspector, ok := inspectorsByLanguage[l]; ok {
						if vi, ok := inspector.(VersionInspector); ok {
							resolved.RuntimeVersion = vi.GetRuntimeVersion(procContext, containerURL)
						}
					}
					return resolved, nil
				}
			}
		}
		return detectedLanguage, err
	}
	return common.ProgramLanguageDetails{Language: common.UnknownProgrammingLanguage}, nil
}

// resolveNonCpp returns the non-C++ language from a conflict pair, if any.
// If neither or both are C++, returns false.
func resolveNonCpp(langs [2]common.ProgrammingLanguage) (common.ProgrammingLanguage, bool) {
	if langs[0] == common.CPlusPlusProgrammingLanguage && langs[1] != common.CPlusPlusProgrammingLanguage {
		return langs[1], true
	}
	if langs[1] == common.CPlusPlusProgrammingLanguage && langs[0] != common.CPlusPlusProgrammingLanguage {
		return langs[0], true
	}
	var none common.ProgrammingLanguage
	return none, false
}

func VerifyLanguage(proc process.Details, lang common.ProgrammingLanguage, logger logr.Logger) bool {
	inspector, ok := inspectorsByLanguage[lang]
	if !ok {
		return false
	}

	procContext := process.NewProcessContext(proc)
	defer func() {
		if err := procContext.CloseFiles(); err != nil {
			logger.Error(err, "Error closing files")
		}
	}()

	_, quickDetected := inspector.QuickScan(procContext)
	if quickDetected {
		return true
	}
	_, deepDetected := inspector.DeepScan(procContext)
	return deepDetected
}
