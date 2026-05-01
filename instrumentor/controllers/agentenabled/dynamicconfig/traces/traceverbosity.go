package traces

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros/distro"
)

func DistroSupportsTracesVerbosity(d *distro.OtelDistro) bool {
	// if the distro states trace verbosity entry, we proceed with processing it
	return d.Traces != nil && d.Traces.TraceVerbosity != nil
}

func CalculateTraceVerbosityConfig(d *distro.OtelDistro, irls *[]odigosv1.InstrumentationRule) *instrumentationrules.TraceVerbosity {

	if !DistroSupportsTracesVerbosity(d) {
		return nil
	}

	traceVerbosity := &instrumentationrules.TraceVerbosity{}

	if irls == nil {
		return traceVerbosity
	}
	for _, irl := range *irls {
		filteredPerLanguage := traceVerbosityForLanguage(irl.Spec.TraceVerbosity, d.Language)
		traceVerbosity = mergeTraceVerbosityConfigs(traceVerbosity, filteredPerLanguage)
	}

	return traceVerbosity
}

// merge 2 trace verbosity configs, filtering for the language of the distro.
// v1 is assumed to be already filtered for the language of the distro.
func mergeTraceVerbosityConfigs(v1 *instrumentationrules.TraceVerbosity, v2 *instrumentationrules.TraceVerbosity) *instrumentationrules.TraceVerbosity {
	if v1 == nil {
		return v2
	}
	if v2 == nil {
		return v1 // v1 is already filtered for the language
	}

	merged := &instrumentationrules.TraceVerbosity{}

	merged.DisabledLibraries = append(merged.DisabledLibraries, v1.DisabledLibraries...)
	merged.DisabledLibraries = append(merged.DisabledLibraries, v2.DisabledLibraries...)

	merged.EnabledLibraries = append(merged.EnabledLibraries, v1.EnabledLibraries...)
	merged.EnabledLibraries = append(merged.EnabledLibraries, v2.EnabledLibraries...)

	return merged
}

func traceVerbosityForLanguage(tv *instrumentationrules.TraceVerbosity, language common.ProgrammingLanguage) *instrumentationrules.TraceVerbosity {
	if tv == nil {
		return nil
	}

	filteredDisabled := filterLibrariesForLanguage(tv.DisabledLibraries, language)
	filteredEnabled := filterLibrariesForLanguage(tv.EnabledLibraries, language)
	if len(filteredDisabled) == 0 && len(filteredEnabled) == 0 {
		return nil
	}

	return &instrumentationrules.TraceVerbosity{
		DisabledLibraries: filteredDisabled,
		EnabledLibraries:  filteredEnabled,
	}
}

func filterLibrariesForLanguage(libraries []instrumentationrules.InstrumentationLibrary, language common.ProgrammingLanguage) []instrumentationrules.InstrumentationLibrary {
	filtered := []instrumentationrules.InstrumentationLibrary{}
	for _, library := range libraries {
		if library.Language == language {
			filtered = append(filtered, library)
		}
	}
	return filtered
}
