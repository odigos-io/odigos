package api_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"

	"github.com/odigos-io/odigos/common/api"
	"github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/common/api/agentsignalconfig"
)

// On Kubernetes these configs travel as JSON, so the json tags decide the key names. On VMs the
// same configs are written to disk and read back with yaml, which ignores json tags and binds an
// untagged field to its lowercased field name instead. #4862, #6137 and #6160 each fixed a batch
// of structs by hand for exactly this reason; these tests pin which fields of the action config
// tree still disagree, so the next struct added to it cannot quietly join them.
//
// Scope is the action configs (common/api/actions) and the fields of the two roots that hold
// them. The sampling and instrumentationrules subtrees have the same problem and are not pinned
// here.

const vycActionsPkg = "github.com/odigos-io/odigos/common/api/actions"

// vycKnownGaps are the fields whose yaml key differs from the json key the Kubernetes API and the
// CRD documentation use, so a config written with the documented key does not reach them on a VM.
//
// To fix one, add `yaml:"<jsonName>,omitempty"` next to its json tag and delete the entry here.
var vycKnownGaps = []string{
	// The four action configs a VM adopting SourceCollectorConfig reads. Every one of them is
	// unreachable under its documented key, which is what makes the leaf tags added by #6137 and
	// #6160 ineffective on the VM path they were added for.
	"SourceCollectorConfig.UrlTemplatization: json=urlTemplatization yaml=urltemplatization",
	"SourceCollectorConfig.DbQueryTemplatization: json=dbQueryTemplatization yaml=dbquerytemplatization",
	"SourceCollectorConfig.InferDbAttributes: json=inferDbAttributes yaml=inferdbattributes",
	"SourceCollectorConfig.PiiMasking: json=piiMasking yaml=piimasking",

	// The same two action configs on the agent side.
	"AgentTracesConfig.UrlTemplatization: json=urlTemplatization yaml=urltemplatization",
	"AgentTracesConfig.SpanRenamer: json=spanRenamer yaml=spanrenamer",

	// SpanRenamerConfig is the only action config under these roots that was never given yaml
	// tags, so it stays unreadable even once its parent field is fixed.
	"SpanRenamerConfig.ScopeRules: json=scopeRules yaml=scoperules",
	"SpanRenamerScopeRules.ScopeName: json=scopeName yaml=scopename",
	"SpanRenamerScopeRules.RegexReplacements: json=regexReplacements yaml=regexreplacements",
	"SpanRenamerRegexReplacement.TemplateText: json=templateText yaml=templatetext",
	"SpanRenamerRegexReplacement.RegexPattern: json=regexPattern yaml=regexpattern",
}

// vycRoots are the two config trees a VM reads back from disk: the per-source collector config
// (#6155 created SourceCollectorConfig so the VM could adopt it) and the per-container agent
// traces config.
func vycRoots() []reflect.Type {
	return []reflect.Type{
		reflect.TypeOf(api.SourceCollectorConfig{}),
		reflect.TypeOf(agentsignalconfig.AgentTracesConfig{}),
	}
}

// vycEffectiveYAMLName is the key gopkg.in/yaml.v2 and go.yaml.in/yaml/v3 bind a field to: the
// yaml tag when there is one, and the lowercased field name when there is not.
func vycEffectiveYAMLName(f reflect.StructField) string {
	if tag, ok := f.Tag.Lookup("yaml"); ok {
		return strings.Split(tag, ",")[0]
	}
	return strings.ToLower(f.Name)
}

func vycJSONName(f reflect.StructField) string {
	return strings.Split(f.Tag.Get("json"), ",")[0]
}

// vycScan walks the two roots and reports every field whose yaml key differs from its json key.
// It only descends into the action config package, because that is the population the yaml-tag
// fixes were about. It also returns the action config types it reached, so a walk that stops
// resolving cannot read as "no disagreements".
func vycScan(t *testing.T) (gaps []string, reached map[string]bool) {
	t.Helper()

	seen := map[reflect.Type]bool{}
	reached = map[string]bool{}

	var walk func(typ reflect.Type, isRoot bool)
	walk = func(typ reflect.Type, isRoot bool) {
		for typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Map {
			typ = typ.Elem()
		}
		if typ.Kind() != reflect.Struct || seen[typ] {
			return
		}
		seen[typ] = true
		if typ.PkgPath() == vycActionsPkg {
			reached[typ.Name()] = true
		}

		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			if !f.IsExported() {
				continue
			}

			fieldType := f.Type
			for fieldType.Kind() == reflect.Pointer || fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Map {
				fieldType = fieldType.Elem()
			}
			inActionsPkg := fieldType.PkgPath() == vycActionsPkg

			// A root contributes only the fields that hold an action config; everything else
			// under it belongs to a subtree this contract does not cover.
			if isRoot && !inActionsPkg {
				continue
			}

			jsonName := vycJSONName(f)
			if jsonName == "-" {
				continue
			}
			if yamlName := vycEffectiveYAMLName(f); yamlName != jsonName {
				gaps = append(gaps, fmt.Sprintf("%s.%s: json=%s yaml=%s", typ.Name(), f.Name, jsonName, yamlName))
			}

			if inActionsPkg {
				walk(f.Type, false)
			}
		}
	}

	for _, root := range vycRoots() {
		walk(root, true)
	}
	sort.Strings(gaps)
	return gaps, reached
}

// vycExpectedTypes are the action config types the two roots reach. An empty gap list only means
// something if the walk actually got to all of them.
var vycExpectedTypes = []string{
	"UrlTemplatizationConfig", "DefaultTemplatizationConfig", "DefaultTemplatizationSkipPolicyConfig",
	"DbQueryTemplatizationConfig", "InferDbAttributesConfig",
	"PiiMaskingConfig", "CustomFormatMasking", "CustomRegexMasking",
	"SpanRenamerConfig", "SpanRenamerScopeRules", "SpanRenamerRegexReplacement",
}

func TestActionConfigYAMLKeysMatchTheirJSONKeys(t *testing.T) {
	gaps, reached := vycScan(t)

	// Anti-vacuity: a walk that stopped resolving would report no disagreements at all.
	for _, typeName := range vycExpectedTypes {
		require.True(t, reached[typeName], "the walk never reached %s", typeName)
	}

	want := append([]string(nil), vycKnownGaps...)
	sort.Strings(want)
	assert.Equal(t, want, gaps,
		"a field of the action config tree now disagrees with vycKnownGaps; add yaml:\"<jsonName>\" "+
			"to a new field, or delete its entry from vycKnownGaps once it is fixed")
}

// The fixes that have already landed must stay landed: these action configs are reachable from
// the roots and must contribute no gaps of their own. Without this half, the test above would
// pass just as well if nothing in the tree had yaml tags at all.
func TestTheAlreadyFixedActionConfigsHaveNoYAMLGaps(t *testing.T) {
	gaps, reached := vycScan(t)

	fixed := []string{
		"UrlTemplatizationConfig", "DefaultTemplatizationConfig", "DefaultTemplatizationSkipPolicyConfig",
		"DbQueryTemplatizationConfig", "PiiMaskingConfig", "CustomFormatMasking", "CustomRegexMasking",
	}
	for _, typeName := range fixed {
		require.True(t, reached[typeName], "the walk never reached %s", typeName)
		for _, gap := range gaps {
			assert.False(t, strings.HasPrefix(gap, typeName+"."),
				"%s was given yaml tags by #6137/#6160 and must keep them: %s", typeName, gap)
		}
	}
}

// Every pinned gap must still be a real disagreement, so fixing one fails here and forces the
// entry to be deleted rather than left behind as a silent exemption.
func TestPinnedYAMLGapsAreStillReal(t *testing.T) {
	gaps, _ := vycScan(t)
	actual := map[string]bool{}
	for _, gap := range gaps {
		actual[gap] = true
	}

	for _, pinned := range vycKnownGaps {
		assert.True(t, actual[pinned],
			"%q no longer disagrees; delete it from vycKnownGaps", pinned)
	}
	require.NotEmpty(t, vycKnownGaps, "an empty pin list makes the contract above vacuous")
}

// A VM is handed the same config shape Kubernetes uses, where the keys come from the json tags.
// Decoding that document as yaml drops every action config, because yaml looks for the
// lowercased key instead. This is why the yaml tags on the leaves do not help yet.
func TestActionConfigsAreUnreachableUnderTheirDocumentedKeys(t *testing.T) {
	populated := api.SourceCollectorConfig{
		UrlTemplatization: &actions.UrlTemplatizationConfig{
			Templates: []string{"/users/{id}"},
			Default:   &actions.DefaultTemplatizationConfig{Disabled: true},
		},
		DbQueryTemplatization: &actions.DbQueryTemplatizationConfig{TemplatizeLiterals: true},
		InferDbAttributes:     &actions.InferDbAttributesConfig{},
		PiiMasking: &actions.PiiMaskingConfig{
			PiiCategories: []actions.PiiCategory{actions.EmailMasking},
		},
	}

	asJSON, err := json.Marshal(populated)
	require.NoError(t, err)
	require.Contains(t, string(asJSON), `"urlTemplatization"`)

	// JSON is valid YAML, so this is the same document a VM would be given.
	var decoded api.SourceCollectorConfig
	require.NoError(t, yaml.Unmarshal(asJSON, &decoded))

	assert.Nil(t, decoded.UrlTemplatization, "SourceCollectorConfig.UrlTemplatization now decodes; drop its vycKnownGaps entry")
	assert.Nil(t, decoded.DbQueryTemplatization, "SourceCollectorConfig.DbQueryTemplatization now decodes; drop its vycKnownGaps entry")
	assert.Nil(t, decoded.InferDbAttributes, "SourceCollectorConfig.InferDbAttributes now decodes; drop its vycKnownGaps entry")
	assert.Nil(t, decoded.PiiMasking, "SourceCollectorConfig.PiiMasking now decodes; drop its vycKnownGaps entry")
}

// The leaf tags themselves are correct: once the parent key is spelled the way yaml expects, the
// camelCase keys inside the action config are read back exactly. This separates "the parent field
// is missing a tag" from "yaml cannot read these structs at all".
func TestLeafYAMLTagsAreCorrectOnceTheParentKeyIsLowercased(t *testing.T) {
	document := strings.Join([]string{
		"urltemplatization:",
		"  templatizationRules:",
		"  - /users/{id}",
		"  default:",
		"    skipPolicy:",
		"      skipHttpStatusCodes: [404, 500]",
		"dbquerytemplatization:",
		"  templatizeLiterals: true",
		"  removePostgresCastOperator: true",
		"piimasking:",
		"  piiCategories: [EMAIL]",
		"  customFormatMaskings:",
		"  - lookupKey: user_id",
		"    dataFormat: json",
		"",
	}, "\n")

	var decoded api.SourceCollectorConfig
	require.NoError(t, yaml.Unmarshal([]byte(document), &decoded))

	require.NotNil(t, decoded.UrlTemplatization)
	assert.Equal(t, []string{"/users/{id}"}, decoded.UrlTemplatization.Templates)
	require.NotNil(t, decoded.UrlTemplatization.Default)
	require.NotNil(t, decoded.UrlTemplatization.Default.SkipPolicy)
	assert.Equal(t, []int{404, 500}, decoded.UrlTemplatization.Default.SkipPolicy.SkipHttpStatusCodes)

	require.NotNil(t, decoded.DbQueryTemplatization)
	assert.True(t, decoded.DbQueryTemplatization.TemplatizeLiterals)
	assert.True(t, decoded.DbQueryTemplatization.RemovePostgresCastOperator)

	require.NotNil(t, decoded.PiiMasking)
	assert.Equal(t, []actions.PiiCategory{actions.EmailMasking}, decoded.PiiMasking.PiiCategories)
	require.Len(t, decoded.PiiMasking.CustomFormatMaskings, 1)
	assert.Equal(t, "user_id", decoded.PiiMasking.CustomFormatMaskings[0].LookupKey)
	assert.Equal(t, actions.FormatJSON, decoded.PiiMasking.CustomFormatMaskings[0].DataFormat)
}

// SpanRenamerConfig sits next to UrlTemplatizationConfig in AgentTracesConfig but was left out of
// both yaml-tag fixes, so even with the parent key spelled the way yaml wants, its contents are
// dropped. The other action configs under the same parent are shown surviving the same document,
// so this cannot pass merely because the document is malformed.
func TestSpanRenamerIsTheActionConfigLeftWithoutYAMLTags(t *testing.T) {
	document := strings.Join([]string{
		"spanrenamer:",
		"  scopeRules:",
		"  - scopeName: my.scope",
		"    regexReplacements:",
		"    - templateText: /orders/{id}",
		"      regexPattern: /orders/[0-9]+",
		"urltemplatization:",
		"  templatizationRules:",
		"  - /users/{id}",
		"",
	}, "\n")

	var decoded agentsignalconfig.AgentTracesConfig
	require.NoError(t, yaml.Unmarshal([]byte(document), &decoded))

	require.NotNil(t, decoded.SpanRenamer, "yaml binds the parent to the lowercased name, so the block itself is found")
	assert.Nil(t, decoded.SpanRenamer.ScopeRules,
		"SpanRenamerConfig now reads its camelCase keys; drop its vycKnownGaps entries")

	require.NotNil(t, decoded.UrlTemplatization)
	assert.Equal(t, []string{"/users/{id}"}, decoded.UrlTemplatization.Templates)
}
