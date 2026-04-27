package traces

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/distros/distro"
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
	mergedRules.Column = merge2RuleBooleans(rule1.Column, rule2.Column)
	mergedRules.FilePath = merge2RuleBooleans(rule1.FilePath, rule2.FilePath)
	mergedRules.Function = merge2RuleBooleans(rule1.Function, rule2.Function)
	mergedRules.LineNumber = merge2RuleBooleans(rule1.LineNumber, rule2.LineNumber)
	mergedRules.Namespace = merge2RuleBooleans(rule1.Namespace, rule2.Namespace)
	mergedRules.Stacktrace = merge2RuleBooleans(rule1.Stacktrace, rule2.Stacktrace)

	return &mergedRules
}

func merge2RuleBooleans(value1 *bool, value2 *bool) *bool {
	if value1 == nil {
		return value2
	} else if value2 == nil {
		return value1
	}
	return boolPtr(*value1 || *value2)
}

func boolPtr(b bool) *bool {
	return &b
}
