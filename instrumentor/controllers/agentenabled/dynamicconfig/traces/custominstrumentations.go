package traces

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/distros/distro"
)

func DistroSupportsCustomInstrumentations(d *distro.OtelDistro) bool {
	return d.Traces != nil && d.Traces.CustomInstrumentations != nil && d.Traces.CustomInstrumentations.Supported
}

func CalculateCustomInstrumentationsConfig(d *distro.OtelDistro, irls *[]odigosv1.InstrumentationRule) *instrumentationrules.CustomInstrumentations {

	if !DistroSupportsCustomInstrumentations(d) {
		return nil
	}

	var result *instrumentationrules.CustomInstrumentations
	for _, irl := range *irls {
		result = mergeCustomInstrumentations(result, irl.Spec.CustomInstrumentations)
	}

	return result
}

func mergeCustomInstrumentations(rule1 *instrumentationrules.CustomInstrumentations, rule2 *instrumentationrules.CustomInstrumentations) *instrumentationrules.CustomInstrumentations {
	if rule1 == nil {
		return rule2
	}
	if rule2 == nil {
		return rule1
	}

	merged := &instrumentationrules.CustomInstrumentations{}

	golangProbes := make([]instrumentationrules.GolangCustomProbe, 0, len(rule1.Golang)+len(rule2.Golang))
	golangProbes = append(golangProbes, rule1.Golang...)
	golangProbes = append(golangProbes, rule2.Golang...)
	merged.Golang = golangProbes

	javaProbes := make([]instrumentationrules.JavaCustomProbe, 0, len(rule1.Java)+len(rule2.Java))
	javaProbes = append(javaProbes, rule1.Java...)
	javaProbes = append(javaProbes, rule2.Java...)
	merged.Java = javaProbes

	return merged
}
