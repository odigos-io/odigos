package traces

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros/distro"
)

func CalculateTraceVerbosityConfig(d *distro.OtelDistro, irls *[]odigosv1.InstrumentationRule) *instrumentationrules.TraceVerbosity {
	traceVerbosity := &instrumentationrules.TraceVerbosity{}

	if irls == nil {
		return traceVerbosity
	}
	for _, irl := range *irls {
		traceVerbosity = mergeTraceVerbosityConfigs(d, traceVerbosity, irl.Spec.TraceVerbosity)
	}

	return traceVerbosity
}

// merge 2 trace verbosity configs, filtering for the language of the distro.
// v1 is assumed to be already filtered for the language of the distro.
func mergeTraceVerbosityConfigs(d *distro.OtelDistro, v1 *instrumentationrules.TraceVerbosity, v2 *instrumentationrules.TraceVerbosity) *instrumentationrules.TraceVerbosity {
	if v1 == nil {
		return traceVerbosityForLanguage(v2, d.Language)
	}
	if v2 == nil {
		return v1 // v1 is already filtered for the language
	}

	v2LanguageFiltered := traceVerbosityForLanguage(v2, d.Language)

	merged := &instrumentationrules.TraceVerbosity{}
	merged.DisabledInstrumentationLibraries = append(merged.DisabledInstrumentationLibraries, v1.DisabledInstrumentationLibraries...)
	merged.DisabledInstrumentationLibraries = append(merged.DisabledInstrumentationLibraries, v2LanguageFiltered.DisabledInstrumentationLibraries...)

	return merged
}

func traceVerbosityForLanguage(tv *instrumentationrules.TraceVerbosity, language common.ProgrammingLanguage) *instrumentationrules.TraceVerbosity {
	if tv == nil {
		return nil
	}

	filtered := filterLibrariesForLanguage(tv.DisabledInstrumentationLibraries, language)
	if len(filtered) == 0 {
		return nil
	}

	return &instrumentationrules.TraceVerbosity{
		DisabledInstrumentationLibraries: filtered,
	}
}

func filterLibrariesForLanguage(libraries []instrumentationrules.InstrumentationLibrary, language common.ProgrammingLanguage) []instrumentationrules.InstrumentationLibrary {
	filtered := []instrumentationrules.InstrumentationLibrary{}
	for _, library := range libraries {
		if library.ProgrammingLanguage == language {
			filtered = append(filtered, library)
		}
	}
	return filtered
}
