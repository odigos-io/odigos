package instrumentationrules

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// #6145 split custom probes into a rule-facing form that carries bookkeeping metadata and an
// agent-facing form that must not. ToConfig is the only thing standing between the two, and it
// repeats the same unwrap four times, once per language.

// The probes below are the agent-facing values. Each test wraps them in metadata to build the rule
// and then asserts ToConfig hands back exactly these, so the expectation never comes from the same
// unwrap the production code performs.
var (
	cpmGoProbes = []GolangCustomProbe{
		{PackageName: "net/http", FunctionName: "ListenAndServe"},
		{PackageName: "database/sql", ReceiverName: "DB", ReceiverMethodName: "QueryContext"},
	}
	cpmJavaProbes = []JavaCustomProbe{
		{ClassName: "com.acme.Pricing", MethodName: "calculate"},
		{ClassName: "com.acme.Checkout", MethodName: "submit"},
	}
	cpmCppProbes = []CppCustomProbe{
		{Signature: "std::vector::push_back"},
		{Signature: "SSL_write"},
	}
	cpmPhpProbes = []PhpCustomProbe{
		{ClassName: `App\Service\OrderService`, FunctionName: "processOrder"},
		{FunctionName: "my_global_function"},
	}
)

// cpmMetadata gives every probe a distinct, valid metadata value, so a projection that reached for
// the wrong index or reused one wrapper for the whole slice cannot produce a matching result.
func cpmMetadata(i int) CustomProbeMetadata {
	categories := []CustomProbeCategory{
		CustomProbeCategoryObservability,
		CustomProbeCategorySecurity,
		CustomProbeCategoryPerformance,
	}
	return CustomProbeMetadata{
		Notes:    fmt.Sprintf("added for reason %d", i),
		Category: categories[i%len(categories)],
		AddedAt:  fmt.Sprintf("2026-10-%02dT09:00:00Z", i+1),
	}
}

// cpmRule wraps the agent-facing probes back into the rule-facing form.
func cpmRule() *CustomInstrumentationsWithMetadata {
	rule := &CustomInstrumentationsWithMetadata{}
	for i, p := range cpmGoProbes {
		rule.Golang = append(rule.Golang, GolangCustomProbeWithMetadata{GolangCustomProbe: p, CustomProbeMetadata: cpmMetadata(i)})
	}
	for i, p := range cpmJavaProbes {
		rule.Java = append(rule.Java, JavaCustomProbeWithMetadata{JavaCustomProbe: p, CustomProbeMetadata: cpmMetadata(i + 2)})
	}
	for i, p := range cpmCppProbes {
		rule.Cpp = append(rule.Cpp, CppCustomProbeWithMetadata{CppCustomProbe: p, CustomProbeMetadata: cpmMetadata(i + 4)})
	}
	for i, p := range cpmPhpProbes {
		rule.Php = append(rule.Php, PhpCustomProbeWithMetadata{PhpCustomProbe: p, CustomProbeMetadata: cpmMetadata(i + 6)})
	}
	return rule
}

func TestToConfigProjectsEveryLanguageInOrder(t *testing.T) {
	t.Parallel()

	rule := cpmRule()
	require.NoError(t, rule.Verify(), "the fixture must be a rule the webhook would accept")

	got := rule.ToConfig()

	// One whole-struct comparison: a probe dropped, duplicated, reordered or taken from the wrong
	// index all surface here, and none of them would show up in a per-field spot check.
	require.Equal(t, &CustomInstrumentations{
		Golang: cpmGoProbes,
		Java:   cpmJavaProbes,
		Cpp:    cpmCppProbes,
		Php:    cpmPhpProbes,
	}, got)
}

// Each language is copied by its own block, so the only fixture that can see a block being skipped
// is one where that language is the sole occupant of the rule.
func TestToConfigCopiesEachLanguageIndependently(t *testing.T) {
	t.Parallel()

	full := cpmRule()
	tests := []struct {
		language string
		rule     *CustomInstrumentationsWithMetadata
		want     *CustomInstrumentations
	}{
		{"golang", &CustomInstrumentationsWithMetadata{Golang: full.Golang}, &CustomInstrumentations{Golang: cpmGoProbes}},
		{"java", &CustomInstrumentationsWithMetadata{Java: full.Java}, &CustomInstrumentations{Java: cpmJavaProbes}},
		{"cpp", &CustomInstrumentationsWithMetadata{Cpp: full.Cpp}, &CustomInstrumentations{Cpp: cpmCppProbes}},
		{"php", &CustomInstrumentationsWithMetadata{Php: full.Php}, &CustomInstrumentations{Php: cpmPhpProbes}},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			t.Parallel()
			// Comparing the whole struct also asserts the other three stay nil, which is what
			// pins each block to the one field it is supposed to write.
			require.Equal(t, tt.want, tt.rule.ToConfig())
		})
	}
}

// ToConfig guards each copy with `!= nil`, so a rule that declares a language with no probes is
// carried across as an empty list rather than an absent one. Downstream code distinguishes the
// two, so the guards have to be exercised with a non-nil empty slice, not just with nil.
func TestToConfigKeepsAnEmptyProbeListDistinctFromAnAbsentOne(t *testing.T) {
	t.Parallel()

	got := (&CustomInstrumentationsWithMetadata{
		Golang: []GolangCustomProbeWithMetadata{},
		Java:   nil,
	}).ToConfig()

	require.NotNil(t, got)
	require.NotNil(t, got.Golang, "a declared-but-empty list must stay declared")
	assert.Empty(t, got.Golang)
	assert.Nil(t, got.Java, "an absent list must stay absent")
	assert.Nil(t, got.Cpp)
	assert.Nil(t, got.Php)
}

// A nil rule yields a nil config while an empty rule yields a non-nil one. The merge step in the
// instrumentor branches on exactly that difference, so collapsing the two here would change which
// rules are able to create an empty probe set downstream.
func TestToConfigReturnsNilOnlyForANilRule(t *testing.T) {
	t.Parallel()

	var absent *CustomInstrumentationsWithMetadata
	assert.Nil(t, absent.ToConfig())

	require.Equal(t, &CustomInstrumentations{}, (&CustomInstrumentationsWithMetadata{}).ToConfig())
}

// The agent-facing type has no room for metadata, so the guarantee worth pinning is that the
// serialized config an agent receives carries none of the words the rule stored.
func TestToConfigOutputSerializesWithoutAnyMetadata(t *testing.T) {
	t.Parallel()

	rule := cpmRule()
	encoded, err := cpmMarshalJSON(rule.ToConfig())
	require.NoError(t, err)

	// Spelled out rather than taken from the struct tags: a test that reads the key names from the
	// production tags cannot notice those tags being renamed.
	for _, leaked := range []string{"notes", "category", "addedAt", "observability", "security", "performance", "added for reason"} {
		assert.NotContains(t, encoded, leaked, "agent config must not carry rule bookkeeping")
	}

	ruleEncoded, err := cpmMarshalJSON(rule)
	require.NoError(t, err)
	for _, kept := range []string{"notes", "category", "addedAt"} {
		assert.Contains(t, ruleEncoded, kept, "the rule itself must still carry the metadata it is dropping")
	}
}

func TestGolangCustomProbeVerifyCoversEveryArmOfTheSwitch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		probe GolangCustomProbe
		// wantErr is the whole message, so a copy-pasted or reworded arm is visible.
		wantErr string
	}{
		{
			name:  "function only",
			probe: GolangCustomProbe{PackageName: "net/http", FunctionName: "ListenAndServe"},
		},
		{
			name:  "receiver and method",
			probe: GolangCustomProbe{PackageName: "net/http", ReceiverName: "response", ReceiverMethodName: "WriteHeader"},
		},
		{
			name:    "no package name",
			probe:   GolangCustomProbe{FunctionName: "ListenAndServe"},
			wantErr: "package name is required",
		},
		{
			// The package guard is checked first, so this row is what keeps the arms ordered.
			name:    "no package name wins over too many arguments",
			probe:   GolangCustomProbe{FunctionName: "f", ReceiverName: "r", ReceiverMethodName: "m"},
			wantErr: "package name is required",
		},
		{
			name:    "function and receiver and method",
			probe:   GolangCustomProbe{PackageName: "net/http", FunctionName: "f", ReceiverName: "r", ReceiverMethodName: "m"},
			wantErr: "too many arguments; either function name or receiver name + method name are required",
		},
		{
			name:    "nothing to instrument",
			probe:   GolangCustomProbe{PackageName: "net/http"},
			wantErr: "too few arguments; either function name or receiver name + method name are required",
		},
		{
			name:    "receiver method without receiver",
			probe:   GolangCustomProbe{PackageName: "net/http", ReceiverMethodName: "WriteHeader"},
			wantErr: "both receiver name and receiver method name are required when using receiver methods",
		},
		{
			name:    "receiver without receiver method",
			probe:   GolangCustomProbe{PackageName: "net/http", ReceiverName: "response"},
			wantErr: "both receiver name and receiver method name are required when using receiver methods",
		},
		{
			// Only reachable when FunctionName is set alongside a half-specified receiver, which is
			// the arm an "is the receiver pair complete" check would be tempted to fold away.
			name:    "function plus a lone receiver",
			probe:   GolangCustomProbe{PackageName: "net/http", FunctionName: "f", ReceiverName: "r"},
			wantErr: "both receiver name and receiver method name are required when using receiver methods",
		},
	}

	accepted, rejected := 0, 0
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.probe.Verify()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tt.wantErr)
		})
		if tt.wantErr == "" {
			accepted++
		} else {
			rejected++
		}
	}
	// Without these an always-nil or always-erroring Verify would still satisfy every row that
	// happened to agree with it.
	assert.Positive(t, accepted)
	assert.Positive(t, rejected)
}

func TestProbeMetadataVerifyChecksCategoryAndAddedAt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata CustomProbeMetadata
		wantErr  string
	}{
		{name: "empty metadata is allowed"},
		{name: "notes alone", metadata: CustomProbeMetadata{Notes: "called from checkout"}},
		{name: "category only", metadata: CustomProbeMetadata{Category: CustomProbeCategorySecurity}},
		{name: "utc timestamp", metadata: CustomProbeMetadata{AddedAt: "2026-10-19T09:00:00Z"}},
		{name: "offset timestamp", metadata: CustomProbeMetadata{AddedAt: "2026-10-19T09:00:00+03:00"}},
		{name: "fractional seconds", metadata: CustomProbeMetadata{AddedAt: "2026-10-19T09:00:00.123456Z"}},
		{
			name:     "unknown category",
			metadata: CustomProbeMetadata{Category: "billing"},
			wantErr:  `invalid category "billing"`,
		},
		{
			// Categories are lowercase; accepting a differently cased spelling would diverge from
			// the enum the CRD advertises, which the API server matches exactly.
			name:     "category in the wrong case",
			metadata: CustomProbeMetadata{Category: "Security"},
			wantErr:  `invalid category "Security"`,
		},
		{
			name:     "date without a time",
			metadata: CustomProbeMetadata{AddedAt: "2026-10-19"},
			wantErr:  "addedAt must be an RFC3339 timestamp",
		},
		{
			name:     "timestamp without a zone",
			metadata: CustomProbeMetadata{AddedAt: "2026-10-19T09:00:00"},
			wantErr:  "addedAt must be an RFC3339 timestamp",
		},
		{
			// The category is checked before the timestamp; this row pins that order.
			name:     "bad category wins over bad timestamp",
			metadata: CustomProbeMetadata{Category: "billing", AddedAt: "nonsense"},
			wantErr:  `invalid category "billing"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.metadata.Verify()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// The CRD rejects anything outside its enum before Verify ever runs, so the two have to list the
// same values. Reading the marker out of the source keeps this from being two hand-written lists
// that drift apart, which is the only way the drift could happen.
func TestCategoryEnumMarkerAndVerifyAgree(t *testing.T) {
	t.Parallel()

	marker, constants := cpmCategoryEnumAndConstants(t)
	require.NotEmpty(t, marker, "no +kubebuilder:validation:Enum marker on CustomProbeCategory")

	for _, value := range marker {
		require.NoError(t, CustomProbeCategory(value).Verify(),
			"%q is advertised by the CRD enum but Verify rejects it", value)
	}
	for _, value := range constants {
		assert.Contains(t, marker, value,
			"the %q constant is accepted in Go but is not in the CRD enum, so the API server would reject it", value)
	}
	assert.ElementsMatch(t, marker, constants,
		"every CRD enum value should have a named constant and vice versa")

	// The empty category means "unset" and is deliberately accepted without appearing in the enum.
	require.NoError(t, CustomProbeCategory("").Verify())
	require.Error(t, CustomProbeCategory("observability-").Verify(),
		"a value that merely starts with an enum member must not be accepted")
}

// cpmCategoryEnumAndConstants returns the values listed on the CustomProbeCategory kubebuilder
// enum marker and the values of the CustomProbeCategory constants declared in this package.
func cpmCategoryEnumAndConstants(t *testing.T) (marker []string, constants []string) {
	t.Helper()

	const typeName = "CustomProbeCategory"
	const markerPrefix = "+kubebuilder:validation:Enum="

	file, err := parser.ParseFile(token.NewFileSet(), "custom_instrumentation.go", nil, parser.ParseComments)
	require.NoError(t, err)

	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				if s.Name.Name != typeName {
					continue
				}
				// A lone type in a GenDecl carries its doc on the GenDecl, not the TypeSpec.
				doc := s.Doc
				if doc == nil {
					doc = gen.Doc
				}
				require.NotNil(t, doc, "%s has no doc comment to read the enum from", typeName)
				for _, line := range doc.List {
					text := strings.TrimSpace(strings.TrimPrefix(line.Text, "//"))
					if after, found := strings.CutPrefix(text, markerPrefix); found {
						marker = strings.Split(after, ";")
					}
				}
			case *ast.ValueSpec:
				ident, ok := s.Type.(*ast.Ident)
				if !ok || ident.Name != typeName {
					continue
				}
				for _, value := range s.Values {
					lit, ok := value.(*ast.BasicLit)
					require.True(t, ok, "expected a literal for a %s constant", typeName)
					unquoted, err := strconv.Unquote(lit.Value)
					require.NoError(t, err)
					constants = append(constants, unquoted)
				}
			}
		}
	}
	require.NotEmpty(t, constants, "found no %s constants; the scan is not reaching them", typeName)
	return marker, constants
}

// Each *WithMetadata.Verify has to run both halves. A fixture that is wrong in only one of them is
// the only thing that can tell a delegation that dropped a half from one that kept it.
func TestProbeVerifyChecksBothTheProbeAndItsMetadata(t *testing.T) {
	t.Parallel()

	badMetadata := CustomProbeMetadata{Category: "billing"}
	goodMetadata := CustomProbeMetadata{Category: CustomProbeCategoryObservability, AddedAt: "2026-10-19T09:00:00Z"}

	tests := []struct {
		name          string
		valid         interface{ Verify() error }
		badProbe      interface{ Verify() error }
		badMetadata   interface{ Verify() error }
		probeErr      string
		badProbeIsFor string
	}{
		{
			name:        "golang",
			valid:       &GolangCustomProbeWithMetadata{GolangCustomProbe: cpmGoProbes[0], CustomProbeMetadata: goodMetadata},
			badProbe:    &GolangCustomProbeWithMetadata{CustomProbeMetadata: goodMetadata},
			badMetadata: &GolangCustomProbeWithMetadata{GolangCustomProbe: cpmGoProbes[0], CustomProbeMetadata: badMetadata},
			probeErr:    "package name is required",
		},
		{
			name:        "java",
			valid:       &JavaCustomProbeWithMetadata{JavaCustomProbe: cpmJavaProbes[0], CustomProbeMetadata: goodMetadata},
			badProbe:    &JavaCustomProbeWithMetadata{CustomProbeMetadata: goodMetadata},
			badMetadata: &JavaCustomProbeWithMetadata{JavaCustomProbe: cpmJavaProbes[0], CustomProbeMetadata: badMetadata},
			probeErr:    "class name is required",
		},
		{
			name:        "cpp",
			valid:       &CppCustomProbeWithMetadata{CppCustomProbe: cpmCppProbes[0], CustomProbeMetadata: goodMetadata},
			badProbe:    &CppCustomProbeWithMetadata{CustomProbeMetadata: goodMetadata},
			badMetadata: &CppCustomProbeWithMetadata{CppCustomProbe: cpmCppProbes[0], CustomProbeMetadata: badMetadata},
			probeErr:    "signature is required",
		},
		{
			name:        "php",
			valid:       &PhpCustomProbeWithMetadata{PhpCustomProbe: cpmPhpProbes[0], CustomProbeMetadata: goodMetadata},
			badProbe:    &PhpCustomProbeWithMetadata{CustomProbeMetadata: goodMetadata},
			badMetadata: &PhpCustomProbeWithMetadata{PhpCustomProbe: cpmPhpProbes[0], CustomProbeMetadata: badMetadata},
			probeErr:    "function name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, tt.valid.Verify())
			require.ErrorContains(t, tt.badProbe.Verify(), tt.probeErr)
			require.ErrorContains(t, tt.badMetadata.Verify(), `invalid category "billing"`)
		})
	}
}

// Verify walks one loop per language, and an invalid probe in a single language is what proves the
// loop for that language runs at all.
func TestRuleVerifyNamesTheLanguageOfTheInvalidProbe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		language string
		rule     *CustomInstrumentationsWithMetadata
		wantErr  string
	}{
		{
			language: "golang",
			rule:     &CustomInstrumentationsWithMetadata{Golang: []GolangCustomProbeWithMetadata{{}}},
			wantErr:  "invalid configuration for golang custom instrumentation: package name is required",
		},
		{
			language: "java",
			rule:     &CustomInstrumentationsWithMetadata{Java: []JavaCustomProbeWithMetadata{{}}},
			wantErr:  "invalid configuration for java custom instrumentation: class name is required",
		},
		{
			language: "php",
			rule:     &CustomInstrumentationsWithMetadata{Php: []PhpCustomProbeWithMetadata{{}}},
			wantErr:  "invalid configuration for php custom instrumentation: function name is required",
		},
		{
			language: "cpp",
			rule:     &CustomInstrumentationsWithMetadata{Cpp: []CppCustomProbeWithMetadata{{}}},
			wantErr:  "invalid configuration for cpp custom instrumentation: signature is required",
		},
	}

	seen := make(map[string]string, len(tests))
	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			err := tt.rule.Verify()
			// The whole message, so a language loop wrapping with a neighbour's prefix is caught.
			require.EqualError(t, err, tt.wantErr)
		})
		seen[tt.language] = tt.wantErr
	}
	require.Len(t, seen, len(tests), "two languages report the same message")

	// A rule whose every probe is valid must get through all four loops.
	require.NoError(t, cpmRule().Verify())

	var absent *CustomInstrumentationsWithMetadata
	require.NoError(t, absent.Verify())
}

// The agent-facing type runs its own loops, separate from the rule-facing ones above. Cpp is left
// out on purpose: #5231 is adding a cpp loop here, and asserting on its absence would collide.
func TestAgentConfigVerifyNamesTheLanguageOfTheInvalidProbe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		language string
		config   *CustomInstrumentations
		wantErr  string
	}{
		{
			language: "golang",
			config:   &CustomInstrumentations{Golang: []GolangCustomProbe{{}}},
			wantErr:  "invalid configuration for golang custom instrumentation: package name is required",
		},
		{
			language: "java",
			config:   &CustomInstrumentations{Java: []JavaCustomProbe{{}}},
			wantErr:  "invalid configuration for java custom instrumentation: class name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			t.Parallel()
			require.EqualError(t, tt.config.Verify(), tt.wantErr)
		})
	}

	require.NoError(t, cpmRule().ToConfig().Verify(), "a config projected from a valid rule must verify")

	var absent *CustomInstrumentations
	require.NoError(t, absent.Verify())
}

// Only the probes come out of the rule, so the loops must run over every element, not just the
// first. A valid probe in front of an invalid one is what shows the loop keeps going.
func TestRuleVerifyChecksEveryProbeNotJustTheFirst(t *testing.T) {
	t.Parallel()

	rule := &CustomInstrumentationsWithMetadata{
		Java: []JavaCustomProbeWithMetadata{
			{JavaCustomProbe: cpmJavaProbes[0]},
			{JavaCustomProbe: JavaCustomProbe{ClassName: "com.acme.Pricing"}},
		},
	}
	require.ErrorContains(t, rule.Verify(), "method Name is required")
}

func cpmMarshalJSON(v any) (string, error) {
	encoded, err := json.Marshal(v)
	return string(encoded), err
}
