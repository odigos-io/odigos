package traces

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/common"
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
		result = mergeCustomInstrumentations(result, irl.Spec.CustomInstrumentations, d.Language)
	}

	return result
}

func mergeCustomInstrumentations(existing *instrumentationrules.CustomInstrumentations, incoming *instrumentationrules.CustomInstrumentations, lang common.ProgrammingLanguage) *instrumentationrules.CustomInstrumentations {
	if incoming == nil {
		return existing
	}

	if existing == nil {
		existing = &instrumentationrules.CustomInstrumentations{}
	}

	switch lang {
	case common.GoProgrammingLanguage:
		existing.Golang = append(existing.Golang, incoming.Golang...)
	case common.JavaProgrammingLanguage:
		existing.Java = append(existing.Java, incoming.Java...)
	}

	return existing
}
