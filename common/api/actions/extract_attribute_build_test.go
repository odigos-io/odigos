package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// BuildExtractAttributeProcessorConfig moved here from the instrumentor in #6162 so that
// non-Kubernetes consumers can reuse it. Its only other coverage is an envtest-backed ginkgo
// suite in the instrumentor module, which the new consumers do not build, so the cross-field
// invariants the CRD schema cannot express are exercised here directly.

func eabRule(target, lookupKey string, format DataFormat, regex string) Extraction {
	return Extraction{
		TargetAttributeName: target,
		LookupKey:           lookupKey,
		DataFormat:          format,
		Regex:               regex,
	}
}

func eabConfig(extractions ...Extraction) *ExtractAttributeConfig {
	return &ExtractAttributeConfig{Extractions: extractions}
}

func TestBuildExtractAttributeProcessorConfig_translatesEveryFieldToTheProcessorWireNames(t *testing.T) {
	got, err := BuildExtractAttributeProcessorConfig(eabConfig(
		eabRule("order.id", "orders", FormatResourcePath, ""),
		eabRule("user.id", "user_id", FormatJSON, ""),
		eabRule("tenant.id", "tenant_id", FormatSQL, ""),
		eabRule("trace.hint", "", "", `hint=(\w+)`),
	))
	require.NoError(t, err)

	// Spelled out rather than derived from the input so that renaming a field, or wiring
	// DataFormat into the wrong rule, cannot keep both sides of the assertion in step.
	require.Equal(t, ExtractAttributeProcessorConfig{
		Extractions: []ExtractAttributeProcessorRule{
			{TargetAttributeName: "order.id", LookupKey: "orders", DataFormat: "resource_path"},
			{TargetAttributeName: "user.id", LookupKey: "user_id", DataFormat: "json"},
			{TargetAttributeName: "tenant.id", LookupKey: "tenant_id", DataFormat: "sql"},
			{TargetAttributeName: "trace.hint", Regex: `hint=(\w+)`},
		},
	}, got)
}

// The processor applies extractions in order, so a builder that reordered or deduplicated them
// would change which rule wins when two of them match the same text.
func TestBuildExtractAttributeProcessorConfig_preservesInputOrder(t *testing.T) {
	got, err := BuildExtractAttributeProcessorConfig(eabConfig(
		eabRule("third", "c", FormatJSON, ""),
		eabRule("first", "a", FormatJSON, ""),
		eabRule("second", "b", FormatJSON, ""),
	))
	require.NoError(t, err)

	names := make([]string, 0, len(got.Extractions))
	for _, e := range got.Extractions {
		names = append(names, e.TargetAttributeName)
	}
	assert.Equal(t, []string{"third", "first", "second"}, names)
}

func TestBuildExtractAttributeProcessorConfig_rejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *ExtractAttributeConfig
		wantErr string
	}{
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: "extractAttribute config is nil",
		},
		{
			name:    "no extractions",
			cfg:     eabConfig(),
			wantErr: "extractions must not be empty",
		},
		{
			name:    "missing target attribute name",
			cfg:     eabConfig(eabRule("ok", "a", FormatJSON, ""), eabRule("", "b", FormatJSON, "")),
			wantErr: "extractions[1]: targetAttributeName is required",
		},
		{
			name:    "duplicate target attribute name",
			cfg:     eabConfig(eabRule("dup", "a", FormatJSON, ""), eabRule("other", "b", FormatJSON, ""), eabRule("dup", "c", FormatJSON, "")),
			wantErr: `extractions[2]: duplicate targetAttributeName "dup" (also used by extractions[0])`,
		},
		{
			name:    "neither lookupKey nor regex",
			cfg:     eabConfig(eabRule("ok", "a", FormatJSON, ""), eabRule("empty", "", "", "")),
			wantErr: "extractions[1]: exactly one of lookupKey or regex must be set",
		},
		{
			name:    "both lookupKey and regex",
			cfg:     eabConfig(eabRule("ok", "a", FormatJSON, ""), eabRule("both", "b", FormatJSON, `(\d+)`)),
			wantErr: "extractions[1]: exactly one of lookupKey or regex must be set",
		},
		{
			name:    "lookupKey without dataFormat",
			cfg:     eabConfig(eabRule("ok", "a", FormatJSON, ""), eabRule("bare", "b", "", "")),
			wantErr: "extractions[1]: dataFormat is required when lookupKey is set",
		},
		{
			name:    "regex with dataFormat",
			cfg:     eabConfig(eabRule("ok", "a", FormatJSON, ""), eabRule("mixed", "", FormatJSON, `(\d+)`)),
			wantErr: "extractions[1]: dataFormat must be empty when regex is set",
		},
	}

	// Every rejection reports the index of the offending extraction, so each case puts a valid
	// rule at index 0: an error message hardcoded to extractions[0] would pass otherwise.
	seen := make(map[string]struct{}, len(tests))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildExtractAttributeProcessorConfig(tt.cfg)
			require.Error(t, err)
			assert.Equal(t, tt.wantErr, err.Error())
		})
		seen[tt.wantErr] = struct{}{}
	}
	// "neither" and "both" deliberately share one message; the other six must stay distinguishable,
	// because the message is all an operator gets back from the action webhook.
	assert.Len(t, seen, len(tests)-1)
}

// A duplicate name is reported against the first rule that claimed it, not the previous rule,
// so an operator with many extractions is pointed at the actual collision. The first claimant
// deliberately sits at index 1: an index recorded as a constant would read the same as the
// correct one if the collision started at index 0.
func TestBuildExtractAttributeProcessorConfig_duplicateNamesTheFirstClaimant(t *testing.T) {
	_, err := BuildExtractAttributeProcessorConfig(eabConfig(
		eabRule("filler0", "a", FormatJSON, ""),
		eabRule("dup", "b", FormatJSON, ""),
		eabRule("filler2", "c", FormatJSON, ""),
		eabRule("dup", "d", FormatJSON, ""),
	))
	require.Error(t, err)
	assert.Equal(t, `extractions[3]: duplicate targetAttributeName "dup" (also used by extractions[1])`, err.Error())
}

// Validation is not all-or-nothing: the rules before the offending one are already in the
// returned config. Callers must branch on the error rather than on the config being empty.
func TestBuildExtractAttributeProcessorConfig_returnsPartiallyBuiltConfigWithTheError(t *testing.T) {
	got, err := BuildExtractAttributeProcessorConfig(eabConfig(
		eabRule("valid", "a", FormatJSON, ""),
		eabRule("", "b", FormatJSON, ""),
	))
	require.Error(t, err)
	require.Len(t, got.Extractions, 1)
	assert.Equal(t, "valid", got.Extractions[0].TargetAttributeName)
}

// The two whole-config rejections are the exception: they return nothing at all.
func TestBuildExtractAttributeProcessorConfig_returnsEmptyConfigForWholeConfigRejections(t *testing.T) {
	for _, cfg := range []*ExtractAttributeConfig{nil, eabConfig()} {
		got, err := BuildExtractAttributeProcessorConfig(cfg)
		require.Error(t, err)
		assert.Equal(t, ExtractAttributeProcessorConfig{}, got)
	}
}

// The "lookupKey must be empty when regex is set" guard is unreachable: past the
// "exactly one of lookupKey or regex" check exactly one of the two is set, so regex and
// lookupKey can never both be present. This pins that there is no input that reaches it, which
// is what makes removing the guard safe and what makes a future rewrite of the earlier check
// visible here.
func TestBuildExtractAttributeProcessorConfig_regexWithLookupKeyIsCaughtByTheEarlierGuard(t *testing.T) {
	_, err := BuildExtractAttributeProcessorConfig(eabConfig(
		eabRule("both", "a", "", `(\d+)`),
	))
	require.Error(t, err)
	assert.Equal(t, "extractions[0]: exactly one of lookupKey or regex must be set", err.Error())
	assert.NotContains(t, err.Error(), "lookupKey must be empty")
}

// An empty-but-present extractions list is rejected, while a one-element list is accepted:
// the length check must be an emptiness check, not a nil check.
func TestBuildExtractAttributeProcessorConfig_distinguishesEmptyFromAbsentExtractions(t *testing.T) {
	_, errNilSlice := BuildExtractAttributeProcessorConfig(&ExtractAttributeConfig{Extractions: nil})
	require.EqualError(t, errNilSlice, "extractions must not be empty")

	_, errEmptySlice := BuildExtractAttributeProcessorConfig(&ExtractAttributeConfig{Extractions: []Extraction{}})
	require.EqualError(t, errEmptySlice, "extractions must not be empty")

	got, err := BuildExtractAttributeProcessorConfig(eabConfig(eabRule("one", "a", FormatJSON, "")))
	require.NoError(t, err)
	assert.Len(t, got.Extractions, 1)
}

// DataFormat is carried through as its string value. A DataFormat the CRD enum does not allow
// is not rejected here, so the kubebuilder marker is the only thing constraining it; this pins
// which of the two layers is doing the work.
func TestBuildExtractAttributeProcessorConfig_passesDataFormatThroughUnvalidated(t *testing.T) {
	got, err := BuildExtractAttributeProcessorConfig(eabConfig(
		eabRule("x", "k", DataFormat("not_a_real_format"), ""),
	))
	require.NoError(t, err)
	require.Len(t, got.Extractions, 1)
	assert.Equal(t, "not_a_real_format", got.Extractions[0].DataFormat)
}

// The three DataFormat constants are the values the CRD enum allows, and the processor matches
// on these exact strings.
func TestDataFormatConstantsMatchTheProcessorWireValues(t *testing.T) {
	assert.Equal(t, DataFormat("resource_path"), FormatResourcePath)
	assert.Equal(t, DataFormat("json"), FormatJSON)
	assert.Equal(t, DataFormat("sql"), FormatSQL)
}
