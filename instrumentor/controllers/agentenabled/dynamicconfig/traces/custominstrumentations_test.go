package traces

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/stretchr/testify/require"
)

func customRule(ci *instrumentationrules.CustomInstrumentationsWithMetadata) odigosv1.InstrumentationRule {
	return odigosv1.InstrumentationRule{Spec: odigosv1.InstrumentationRuleSpec{CustomInstrumentations: ci}}
}

func TestCalculateCustomInstrumentationsConfigPython(t *testing.T) {
	supported := &distro.Traces{CustomInstrumentations: &distro.CustomInstrumentations{Supported: true}}
	pythonDistro := &distro.OtelDistro{Language: common.PythonProgrammingLanguage, Traces: supported}
	billing := instrumentationrules.PythonCustomProbe{Module: "app.billing", Qualname: "Invoice.calculate"}
	handler := instrumentationrules.PythonCustomProbe{Module: "__main__", Qualname: "handler"}
	rules := []odigosv1.InstrumentationRule{
		customRule(&instrumentationrules.CustomInstrumentationsWithMetadata{
			Python: []instrumentationrules.PythonCustomProbeWithMetadata{{
				PythonCustomProbe:   billing,
				CustomProbeMetadata: instrumentationrules.CustomProbeMetadata{Notes: "checkout"},
			}},
			Java: []instrumentationrules.JavaCustomProbeWithMetadata{{
				JavaCustomProbe: instrumentationrules.JavaCustomProbe{ClassName: "com.acme.Pricing", MethodName: "calculate"},
			}},
		}),
		customRule(nil),
		customRule(&instrumentationrules.CustomInstrumentationsWithMetadata{
			Python: []instrumentationrules.PythonCustomProbeWithMetadata{{PythonCustomProbe: handler}},
		}),
	}

	got := CalculateCustomInstrumentationsConfig(pythonDistro, &rules)
	require.Equal(t, &instrumentationrules.CustomInstrumentations{
		Python: []instrumentationrules.PythonCustomProbe{billing, handler},
	}, got)

	javaDistro := &distro.OtelDistro{Language: common.JavaProgrammingLanguage, Traces: supported}
	require.Empty(t, CalculateCustomInstrumentationsConfig(javaDistro, &rules).Python)

	unsupported := &distro.OtelDistro{Language: common.PythonProgrammingLanguage}
	require.Nil(t, CalculateCustomInstrumentationsConfig(unsupported, &rules))
}
