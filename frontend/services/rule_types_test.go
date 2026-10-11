package services

import (
	"reflect"
	"strings"
	"testing"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/instrumentationrules"
	"github.com/stretchr/testify/require"
)

// instrumentationRuleInputKeys returns the top-level keys of the GraphQL rule
// input. A catalog field's `name` is the key the UI writes its value under, so
// every field has to resolve to one of these.
func instrumentationRuleInputKeys(t *testing.T) map[string]struct{} {
	t.Helper()

	inputType := reflect.TypeOf(model.InstrumentationRuleInput{})
	keys := make(map[string]struct{}, inputType.NumField())
	for i := 0; i < inputType.NumField(); i++ {
		name := strings.Split(inputType.Field(i).Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			continue
		}
		keys[name] = struct{}{}
	}
	return keys
}

func TestInstrumentationRuleCatalogMatchesGraphQLRuleTypes(t *testing.T) {
	require.NoError(t, instrumentationrules.Load())

	catalog := instrumentationrules.Get()
	require.NotEmpty(t, catalog)

	catalogTypes := make(map[model.InstrumentationRuleType]struct{}, len(catalog))
	for _, rule := range catalog {
		t.Run(rule.Metadata.Type, func(t *testing.T) {
			ruleType := model.InstrumentationRuleType(rule.Metadata.Type)
			require.True(t, ruleType.IsValid(), "catalog type is not a GraphQL InstrumentationRuleType, so the UI cannot open rules of this type")
			require.NotEqual(t, model.InstrumentationRuleTypeUnknownType, ruleType)

			require.NotContains(t, catalogTypes, ruleType, "duplicate catalog type shadows another rule in GetRuleByType")
			catalogTypes[ruleType] = struct{}{}

			require.NotEmpty(t, rule.Metadata.DisplayName)
			require.NotEmpty(t, rule.Spec.Description)
			require.NotEmpty(t, rule.Spec.DocsURL)
			require.NotEmpty(t, rule.Spec.SupportedLanguages)
		})
	}

	for _, ruleType := range model.AllInstrumentationRuleType {
		if ruleType == model.InstrumentationRuleTypeUnknownType {
			continue
		}
		require.Contains(t, catalogTypes, ruleType, "rule type the backend can report has no catalog entry, so it cannot be created from the UI")
	}
}

func TestInstrumentationRuleCatalogFieldsBindToRuleInputKeys(t *testing.T) {
	require.NoError(t, instrumentationrules.Load())

	inputKeys := instrumentationRuleInputKeys(t)

	for _, rule := range instrumentationrules.Get() {
		for _, field := range rule.Spec.Fields {
			t.Run(rule.Metadata.Type+"/"+field.DisplayName, func(t *testing.T) {
				require.Contains(t, inputKeys, field.Name, "catalog field does not name a rule input key, so the value the UI submits is dropped")
				// A rule's type is derived from which input key is set, so a toggle
				// bound to one of those keys writes the rule's own identity: turning
				// it off produces a rule that reads back as UnknownType. Disabling a
				// rule is `spec.disabled`, not a field.
				require.NotEqual(t, "toggle", field.ComponentType, "toggle bound to a rule input key clears the field that identifies the rule's type")
			})
		}
	}
}

func TestGetInstrumentationRuleTypesExposesWholeCatalog(t *testing.T) {
	require.NoError(t, instrumentationrules.Load())

	options := GetInstrumentationRuleTypes()
	require.Len(t, options, len(instrumentationrules.Get()))

	byType := make(map[string]*model.InstrumentationRuleTypeOption, len(options))
	for _, option := range options {
		require.NotNil(t, option)
		byType[option.Type] = option
	}

	networkMetrics, ok := byType[string(model.InstrumentationRuleTypeNetworkMetrics)]
	require.True(t, ok)
	require.Empty(t, networkMetrics.Fields, "the network metrics rule has no configuration of its own")
	require.Contains(t, networkMetrics.Description, "metricsSources.networkMetrics.enabled",
		"the Helm prerequisite must stay visible in the rule description")
}

func TestInstrumentationRuleConfigToTypeOption(t *testing.T) {
	option := InstrumentationRuleConfigToTypeOption(instrumentationrules.InstrumentationRule{
		Metadata: instrumentationrules.Metadata{
			Type:        "HeadersCollection",
			DisplayName: "Headers Collection",
		},
		Spec: instrumentationrules.Spec{
			Description:        "collect headers",
			DocsURL:            "https://docs.odigos.io/pipeline/rules/headerscollection",
			SupportedLanguages: []string{"go"},
			Fields: []instrumentationrules.Field{{
				Name:            "headersCollection",
				DisplayName:     "Header keys",
				ComponentType:   "multiInput",
				InitialValue:    `["*"]`,
				RenderCondition: []string{"headersCollection", "!=", ""},
				ComponentProps:  map[string]interface{}{"wrapKey": "headerKeys"},
			}},
		},
	})

	require.Equal(t, "HeadersCollection", option.Type)
	require.Equal(t, "Headers Collection", option.DisplayName)
	require.Equal(t, "collect headers", option.Description)
	require.Equal(t, "https://docs.odigos.io/pipeline/rules/headerscollection", option.DocsURL)
	require.Equal(t, []string{"go"}, option.SupportedLanguages)

	require.Len(t, option.Fields, 1)
	require.Equal(t, "headersCollection", option.Fields[0].Name)
	require.Equal(t, "Header keys", option.Fields[0].DisplayName)
	require.Equal(t, "multiInput", option.Fields[0].ComponentType)
	require.Equal(t, `["*"]`, option.Fields[0].InitialValue)
	require.Equal(t, []string{"headersCollection", "!=", ""}, option.Fields[0].RenderCondition)
	require.JSONEq(t, `{"wrapKey":"headerKeys"}`, option.Fields[0].ComponentProperties)
}

func TestInstrumentationRuleConfigToTypeOptionWithoutFieldsOrLanguages(t *testing.T) {
	option := InstrumentationRuleConfigToTypeOption(instrumentationrules.InstrumentationRule{
		Metadata: instrumentationrules.Metadata{Type: "NetworkMetrics"},
	})

	// The UI reads both as lists, so they must marshal as `[]` and not `null`.
	require.NotNil(t, option.SupportedLanguages)
	require.Empty(t, option.SupportedLanguages)
	require.NotNil(t, option.Fields)
	require.Empty(t, option.Fields)
}
