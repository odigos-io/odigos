package traces

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/instrumentor/controllers/instrumentationconfig"
)

func CalculateCodeAttributesConfig(distro *distro.OtelDistro, irls *[]odigosv1.InstrumentationRule) *instrumentationrules.CodeAttributes {

	// Only support code attributes collection if the distro supports it
	if !DistroSupportsTracesCodeAttributes(distro) {
		return nil
	}

	var codeAttributes *instrumentationrules.CodeAttributes
	for _, irl := range *irls {
		codeAttributes = mergeCodeAttributesRules(codeAttributes, irl.Spec.CodeAttributes)
	}

	return codeAttributes
}

func DistroSupportsTracesCodeAttributes(distro *distro.OtelDistro) bool {
	return distro.Traces != nil && distro.Traces.CodeAttributes != nil && distro.Traces.CodeAttributes.Supported
}

func mergeCodeAttributesRules(rule1 *instrumentationrules.CodeAttributes, rule2 *instrumentationrules.CodeAttributes) *instrumentationrules.CodeAttributes {
	if rule1 == nil {
		return rule2
	} else if rule2 == nil {
		return rule1
	}

	mergedRules := instrumentationrules.CodeAttributes{}
	mergedRules.Column = instrumentationconfig.Merge2RuleBooleans(rule1.Column, rule2.Column)
	mergedRules.FilePath = instrumentationconfig.Merge2RuleBooleans(rule1.FilePath, rule2.FilePath)
	mergedRules.Function = instrumentationconfig.Merge2RuleBooleans(rule1.Function, rule2.Function)
	mergedRules.LineNumber = instrumentationconfig.Merge2RuleBooleans(rule1.LineNumber, rule2.LineNumber)
	mergedRules.Namespace = instrumentationconfig.Merge2RuleBooleans(rule1.Namespace, rule2.Namespace)
	mergedRules.Stacktrace = instrumentationconfig.Merge2RuleBooleans(rule1.Stacktrace, rule2.Stacktrace)

	return &mergedRules
}
