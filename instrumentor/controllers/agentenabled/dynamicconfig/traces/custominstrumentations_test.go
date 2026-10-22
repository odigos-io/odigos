package traces

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Probes reach an agent only if the distro opts in and the merge step has a case for the distro's
// language. Both are silent when they decline, so each has to be driven on its own.

func cicDistro(language common.ProgrammingLanguage, traces *distro.Traces) *distro.OtelDistro {
	return &distro.OtelDistro{Name: "test-distro", Language: language, Traces: traces}
}

func cicSupportingDistro(language common.ProgrammingLanguage) *distro.OtelDistro {
	return cicDistro(language, &distro.Traces{CustomInstrumentations: &distro.CustomInstrumentations{Supported: true}})
}

func cicRule(probes *instrumentationrules.CustomInstrumentationsWithMetadata) odigosv1.InstrumentationRule {
	return odigosv1.InstrumentationRule{Spec: odigosv1.InstrumentationRuleSpec{CustomInstrumentations: probes}}
}

// cicMetadata is the rule-only bookkeeping #6145 added. It must never reach an agent.
var cicMetadata = instrumentationrules.CustomProbeMetadata{
	Notes:    "added while chasing a checkout latency regression",
	Category: instrumentationrules.CustomProbeCategoryPerformance,
	AddedAt:  "2026-10-19T09:00:00Z",
}

var (
	cicGoProbe   = instrumentationrules.GolangCustomProbe{PackageName: "net/http", FunctionName: "ListenAndServe"}
	cicJavaProbe = instrumentationrules.JavaCustomProbe{ClassName: "com.acme.Pricing", MethodName: "calculate"}
	cicPhpProbe  = instrumentationrules.PhpCustomProbe{ClassName: `App\Service\OrderService`, FunctionName: "processOrder"}
	cicCppProbe  = instrumentationrules.CppCustomProbe{Signature: "std::vector::push_back"}
)

// cicEveryLanguageRule carries one probe for each of the four languages, so calculating it against
// a single-language distro shows both that the right probe is selected and that the other three
// are left behind.
func cicEveryLanguageRule() *instrumentationrules.CustomInstrumentationsWithMetadata {
	return &instrumentationrules.CustomInstrumentationsWithMetadata{
		Golang: []instrumentationrules.GolangCustomProbeWithMetadata{{GolangCustomProbe: cicGoProbe, CustomProbeMetadata: cicMetadata}},
		Java:   []instrumentationrules.JavaCustomProbeWithMetadata{{JavaCustomProbe: cicJavaProbe, CustomProbeMetadata: cicMetadata}},
		Php:    []instrumentationrules.PhpCustomProbeWithMetadata{{PhpCustomProbe: cicPhpProbe, CustomProbeMetadata: cicMetadata}},
		Cpp:    []instrumentationrules.CppCustomProbeWithMetadata{{CppCustomProbe: cicCppProbe, CustomProbeMetadata: cicMetadata}},
	}
}

// Three conditions are ANDed together, so each needs a fixture where it is the only one failing.
func TestDistroSupportsCustomInstrumentations(t *testing.T) {
	tests := []struct {
		name   string
		distro *distro.OtelDistro
		want   bool
	}{
		{
			name:   "distro declares no traces support at all",
			distro: cicDistro(common.JavaProgrammingLanguage, nil),
		},
		{
			name: "distro supports traces but not custom instrumentations",
			distro: cicDistro(common.JavaProgrammingLanguage, &distro.Traces{
				CodeAttributes: &distro.CodeAttributes{Supported: true},
			}),
		},
		{
			name: "custom instrumentations are declared but switched off",
			distro: cicDistro(common.JavaProgrammingLanguage, &distro.Traces{
				CustomInstrumentations: &distro.CustomInstrumentations{Supported: false},
			}),
		},
		{
			name:   "custom instrumentations are supported",
			distro: cicSupportingDistro(common.JavaProgrammingLanguage),
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, DistroSupportsCustomInstrumentations(tt.distro))
		})
	}
}

// The support check is what keeps probes away from an agent that cannot run them, so it has to be
// shown refusing a rule that would otherwise have produced a config.
func TestCalculateCustomInstrumentationsConfig_unsupportedDistroIgnoresRules(t *testing.T) {
	rules := []odigosv1.InstrumentationRule{cicRule(cicEveryLanguageRule())}

	assert.Nil(t, CalculateCustomInstrumentationsConfig(cicDistro(common.JavaProgrammingLanguage, nil), &rules))

	// The same rules against a supporting distro must produce something, or the assertion above
	// would pass for a fixture that was never going to yield probes in the first place.
	require.NotNil(t, CalculateCustomInstrumentationsConfig(cicSupportingDistro(common.JavaProgrammingLanguage), &rules))
}

// The merge step switches on the distro language and writes one field per case, so the only
// fixture that can see a case being dropped or pointed at the wrong field is one that offers every
// language at once and expects exactly one of them back.
func TestCalculateCustomInstrumentationsConfig_selectsProbesForTheDistroLanguage(t *testing.T) {
	rules := []odigosv1.InstrumentationRule{cicRule(cicEveryLanguageRule())}

	tests := []struct {
		language common.ProgrammingLanguage
		want     *instrumentationrules.CustomInstrumentations
	}{
		{
			language: common.GoProgrammingLanguage,
			want:     &instrumentationrules.CustomInstrumentations{Golang: []instrumentationrules.GolangCustomProbe{cicGoProbe}},
		},
		{
			language: common.JavaProgrammingLanguage,
			want:     &instrumentationrules.CustomInstrumentations{Java: []instrumentationrules.JavaCustomProbe{cicJavaProbe}},
		},
		{
			language: common.PhpProgrammingLanguage,
			want:     &instrumentationrules.CustomInstrumentations{Php: []instrumentationrules.PhpCustomProbe{cicPhpProbe}},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.language), func(t *testing.T) {
			// Comparing the whole struct is what asserts the other three languages stay empty.
			require.Equal(t, tt.want, CalculateCustomInstrumentationsConfig(cicSupportingDistro(tt.language), &rules))
		})
	}
}

// mergeCustomInstrumentations has cases for go, java and php only, so a C++ distro that opts in
// receives an allocated-but-empty config and the user's C++ probes are dropped without a word.
// #6145 made C++ probes settable through the GraphQL rule API, which is what makes this reachable.
// Fixing it should make this test fail; replace it with the go/java/php row shape above.
func TestCalculateCustomInstrumentationsConfig_cppProbesAreNeverMerged(t *testing.T) {
	rules := []odigosv1.InstrumentationRule{cicRule(cicEveryLanguageRule())}

	got := CalculateCustomInstrumentationsConfig(cicSupportingDistro(common.CPlusPlusProgrammingLanguage), &rules)

	require.NotNil(t, got, "the merge still allocates a config before failing to find a case for the language")
	require.Equal(t, &instrumentationrules.CustomInstrumentations{}, got,
		"C++ probes now reach the agent config; update this test to assert they are selected")

	// The rule really did carry a C++ probe, so the empty result above is the merge dropping it
	// rather than the fixture never having one.
	require.Len(t, cicEveryLanguageRule().Cpp, 1)
}

// Rules are merged by appending, so the order the probes arrive in is the order the rules are
// listed in. Two rules with distinguishable probes are the minimum that can show either one being
// dropped or the pair being reversed.
func TestCalculateCustomInstrumentationsConfig_appendsProbesInRuleOrder(t *testing.T) {
	second := instrumentationrules.GolangCustomProbe{PackageName: "database/sql", ReceiverName: "DB", ReceiverMethodName: "QueryContext"}
	third := instrumentationrules.GolangCustomProbe{PackageName: "os", FunctionName: "ReadFile"}

	rules := []odigosv1.InstrumentationRule{
		cicRule(&instrumentationrules.CustomInstrumentationsWithMetadata{
			Golang: []instrumentationrules.GolangCustomProbeWithMetadata{{GolangCustomProbe: cicGoProbe, CustomProbeMetadata: cicMetadata}},
		}),
		cicRule(&instrumentationrules.CustomInstrumentationsWithMetadata{
			Golang: []instrumentationrules.GolangCustomProbeWithMetadata{
				{GolangCustomProbe: second},
				{GolangCustomProbe: third},
			},
		}),
	}

	got := CalculateCustomInstrumentationsConfig(cicSupportingDistro(common.GoProgrammingLanguage), &rules)

	require.Equal(t, &instrumentationrules.CustomInstrumentations{
		Golang: []instrumentationrules.GolangCustomProbe{cicGoProbe, second, third},
	}, got)
}

// A rule that configures something other than custom instrumentations leaves the field nil, which
// the merge has to treat as "contributes nothing" rather than as an empty probe set.
func TestCalculateCustomInstrumentationsConfig_skipsRulesWithoutProbes(t *testing.T) {
	supporting := cicSupportingDistro(common.GoProgrammingLanguage)

	onlyEmptyRules := []odigosv1.InstrumentationRule{cicRule(nil), cicRule(nil)}
	assert.Nil(t, CalculateCustomInstrumentationsConfig(supporting, &onlyEmptyRules),
		"no rule carries probes, so there is nothing to configure")

	noRules := []odigosv1.InstrumentationRule{}
	assert.Nil(t, CalculateCustomInstrumentationsConfig(supporting, &noRules))

	// An empty rule ahead of a populated one must not stop the populated one from being picked up.
	mixed := []odigosv1.InstrumentationRule{
		cicRule(nil),
		cicRule(&instrumentationrules.CustomInstrumentationsWithMetadata{
			Golang: []instrumentationrules.GolangCustomProbeWithMetadata{{GolangCustomProbe: cicGoProbe, CustomProbeMetadata: cicMetadata}},
		}),
		cicRule(nil),
	}
	require.Equal(t, &instrumentationrules.CustomInstrumentations{
		Golang: []instrumentationrules.GolangCustomProbe{cicGoProbe},
	}, CalculateCustomInstrumentationsConfig(supporting, &mixed))
}

// The metadata lives on the rule to explain why a probe exists; shipping it to an agent would put
// free-form operator notes into every instrumented workload's config.
func TestCalculateCustomInstrumentationsConfig_dropsRuleOnlyMetadata(t *testing.T) {
	rule := cicEveryLanguageRule()
	require.NotEmpty(t, rule.Java[0].Notes, "the fixture must carry metadata for its absence to mean anything")
	require.NotEmpty(t, rule.Java[0].Category)
	require.NotEmpty(t, rule.Java[0].AddedAt)

	rules := []odigosv1.InstrumentationRule{cicRule(rule)}
	got := CalculateCustomInstrumentationsConfig(cicSupportingDistro(common.JavaProgrammingLanguage), &rules)

	require.Equal(t, &instrumentationrules.CustomInstrumentations{
		Java: []instrumentationrules.JavaCustomProbe{cicJavaProbe},
	}, got)
}
