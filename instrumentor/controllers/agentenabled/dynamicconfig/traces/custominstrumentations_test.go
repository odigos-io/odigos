package traces

import (
	"testing"

	"github.com/stretchr/testify/require"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/distros/distro"
)

func TestCalculateCustomInstrumentationsConfigIncludesCppForCppDistro(t *testing.T) {
	rules := []odigosv1.InstrumentationRule{
		{
			Spec: odigosv1.InstrumentationRuleSpec{
				CustomInstrumentations: &instrumentationrules.CustomInstrumentations{
					Cpp: []instrumentationrules.CppCustomProbe{
						{Signature: "std::vector::push_back"},
					},
					Java: []instrumentationrules.JavaCustomProbe{
						{ClassName: "com.example.Service", MethodName: "handle"},
					},
				},
			},
		},
		{
			Spec: odigosv1.InstrumentationRuleSpec{
				CustomInstrumentations: &instrumentationrules.CustomInstrumentations{
					Cpp: []instrumentationrules.CppCustomProbe{
						{Signature: "SSL_write"},
					},
				},
			},
		},
	}
	d := &distro.OtelDistro{
		Language: common.CPlusPlusProgrammingLanguage,
		Traces: &distro.Traces{
			CustomInstrumentations: &distro.CustomInstrumentations{
				Supported: true,
			},
		},
	}

	result := CalculateCustomInstrumentationsConfig(d, &rules)

	require.NotNil(t, result)
	require.Equal(t, []instrumentationrules.CppCustomProbe{
		{Signature: "std::vector::push_back"},
		{Signature: "SSL_write"},
	}, result.Cpp)
	require.Empty(t, result.Java)
	require.Empty(t, result.Golang)
}
