package services

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/odigos-io/odigos/actions"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The catalog is loaded once per process by the server's dependency setup; without it every
// catalog assertion below would pass vacuously against an empty list.
func loadActionsCatalog(t *testing.T) []actions.Action {
	t.Helper()
	require.NoError(t, actions.Load())

	catalog := actions.Get()
	require.NotEmpty(t, catalog, "action catalog is empty")
	return catalog
}

// actionFieldsInputKeys returns the top-level json keys of the GraphQL action fields input, which
// is the set of names the UI may submit values under.
func actionFieldsInputKeys() map[string]bool {
	inputType := reflect.TypeOf(model.ActionFieldsInput{})

	keys := make(map[string]bool, inputType.NumField())
	for i := 0; i < inputType.NumField(); i++ {
		tag := inputType.Field(i).Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		if name != "" && name != "-" {
			keys[name] = true
		}
	}
	return keys
}

// checkboxOptionIDs returns the option ids of a checkboxList field. The dynamic renderer writes
// values keyed by these ids, so they are part of the API contract just like the field name is.
func checkboxOptionIDs(t *testing.T, field actions.Field) []string {
	t.Helper()

	rawOptions, ok := field.ComponentProps["options"].([]any)
	require.True(t, ok, "field %q has no options list", field.Name)

	ids := make([]string, 0, len(rawOptions))
	for _, rawOption := range rawOptions {
		option, ok := rawOption.(map[string]any)
		require.True(t, ok, "field %q has a malformed option", field.Name)
		id, ok := option["id"].(string)
		require.True(t, ok, "field %q has an option without an id", field.Name)
		ids = append(ids, id)
	}
	return ids
}

// ****************
// Catalog / GraphQL contract
// ****************

// The action picker is rendered from the catalog while every mutation is typed by the GraphQL enum.
// A catalog entry with no enum value cannot be created at all, and an enum value with no catalog
// entry never appears in the picker, so the two lists must agree exactly.
func TestActionCatalogMatchesGraphQLActionTypes(t *testing.T) {
	catalog := loadActionsCatalog(t)

	catalogTypes := make([]string, 0, len(catalog))
	for _, entry := range catalog {
		catalogTypes = append(catalogTypes, entry.Metadata.Type)
	}

	// UnknownType is the fallback the resolver returns for an Action CR it cannot classify; it is
	// never a creatable action, so it has no catalog entry.
	enumTypes := make([]string, 0, len(model.AllActionType))
	for _, actionType := range model.AllActionType {
		if actionType == model.ActionTypeUnknownType {
			continue
		}
		enumTypes = append(enumTypes, string(actionType))
	}

	assert.ElementsMatch(t, enumTypes, catalogTypes)
}

// The UI submits a value under each field's name, and the server only reads the keys declared on
// ActionFieldsInput. A field whose name is not a key is silently dropped on submit, so the form
// looks like it works and the setting never reaches the Action CR.
func TestActionCatalogFieldsBindToActionInputKeys(t *testing.T) {
	catalog := loadActionsCatalog(t)
	inputKeys := actionFieldsInputKeys()
	require.NotEmpty(t, inputKeys)

	for _, entry := range catalog {
		for _, field := range entry.Spec.Fields {
			t.Run(entry.Metadata.Type+"/"+field.Name, func(t *testing.T) {
				// A checkboxList in flatFields mode is the one field that does not write its own
				// name: it writes each checked option id as its own top-level key.
				if field.ComponentType == "checkboxList" && field.ComponentProps["valueMode"] == "flatFields" {
					for _, id := range checkboxOptionIDs(t, field) {
						assert.True(t, inputKeys[id],
							"flatFields option %q is not a key on ActionFieldsInput, so the checkbox is dropped on submit", id)
					}
					return
				}

				assert.True(t, inputKeys[field.Name],
					"catalog field %q is not a key on ActionFieldsInput, so its value is dropped on submit", field.Name)
			})
		}
	}
}

// Every field declares the component the UI renders, and every catalog entry needs the metadata the
// picker groups and labels it by. An empty value here renders a blank row or an unlabelled entry.
func TestActionCatalogEntriesAreFullyDescribed(t *testing.T) {
	catalog := loadActionsCatalog(t)

	for _, entry := range catalog {
		t.Run(entry.Metadata.Type, func(t *testing.T) {
			assert.NotEmpty(t, entry.Metadata.DisplayName)
			assert.NotEmpty(t, entry.Metadata.Category)
			assert.NotEmpty(t, entry.Spec.Subtitle)
			assert.NotEmpty(t, entry.Spec.Description)
			assert.NotEmpty(t, entry.Spec.DocsURL)

			for _, field := range entry.Spec.Fields {
				assert.NotEmpty(t, field.Name)
				assert.NotEmpty(t, field.DisplayName, "field %q", field.Name)
				assert.NotEmpty(t, field.ComponentType, "field %q", field.Name)
			}
		})
	}
}

// An action supporting no signal cannot be created: the UI offers no signal to select and the
// backend wires the action into no pipeline.
func TestEveryActionCatalogEntrySupportsASignal(t *testing.T) {
	catalog := loadActionsCatalog(t)

	for _, entry := range catalog {
		t.Run(entry.Metadata.Type, func(t *testing.T) {
			assert.NotEmpty(t, ActionConfigToTypeOption(entry).AllowedSignals)
		})
	}
}

// initialValue is hand-written JSON embedded in YAML, so nothing but a test checks it parses. A
// typo leaves the form seeded with a value the UI cannot read.
func TestActionCatalogInitialValuesAreValidJSON(t *testing.T) {
	catalog := loadActionsCatalog(t)

	for _, entry := range catalog {
		for _, field := range entry.Spec.Fields {
			if field.InitialValue == "" {
				continue
			}
			t.Run(entry.Metadata.Type+"/"+field.Name, func(t *testing.T) {
				var parsed any
				assert.NoError(t, json.Unmarshal([]byte(field.InitialValue), &parsed),
					"initialValue of %q is not valid JSON", field.Name)
			})
		}
	}
}

// A checkboxList seeds itself from initialValue by option id. An id that is not a declared option
// silently leaves the intended default unchecked, which turns the action into a partial no-op the
// moment the user saves the form without touching it.
func TestActionCatalogCheckboxInitialValuesReferenceDeclaredOptions(t *testing.T) {
	catalog := loadActionsCatalog(t)

	checked := 0
	for _, entry := range catalog {
		for _, field := range entry.Spec.Fields {
			if field.ComponentType != "checkboxList" || field.InitialValue == "" {
				continue
			}

			t.Run(entry.Metadata.Type+"/"+field.Name, func(t *testing.T) {
				declared := checkboxOptionIDs(t, field)

				var seeded []string
				switch field.ComponentProps["valueMode"] {
				case "array":
					require.NoError(t, json.Unmarshal([]byte(field.InitialValue), &seeded))
				case "flatFields":
					var flags map[string]any
					require.NoError(t, json.Unmarshal([]byte(field.InitialValue), &flags))
					for id := range flags {
						seeded = append(seeded, id)
					}
				default:
					t.Fatalf("field %q has an unknown checkboxList valueMode", field.Name)
				}

				require.NotEmpty(t, seeded)
				for _, id := range seeded {
					assert.Contains(t, declared, id,
						"initialValue of %q seeds %q, which is not a declared option", field.Name, id)
				}
			})
			checked++
		}
	}

	require.NotZero(t, checked, "no seeded checkboxList field was checked")
}

// ****************
// ActionConfigToTypeOption
// ****************

// The catalog declares signal support as a map of booleans while the API exposes a list, so the
// flattening decides which signals the UI lets the user select for an action.
func TestActionConfigToTypeOptionFlattensSupportedSignals(t *testing.T) {
	tests := []struct {
		name                            string
		traces, metrics, logs, profiles bool
		expected                        []model.SignalType
	}{
		{
			name:     "all supported",
			traces:   true,
			metrics:  true,
			logs:     true,
			profiles: true,
			expected: []model.SignalType{model.SignalTypeTraces, model.SignalTypeMetrics, model.SignalTypeLogs, model.SignalTypeProfiles},
		},
		{name: "traces only", traces: true, expected: []model.SignalType{model.SignalTypeTraces}},
		{name: "metrics only", metrics: true, expected: []model.SignalType{model.SignalTypeMetrics}},
		{name: "logs only", logs: true, expected: []model.SignalType{model.SignalTypeLogs}},
		{name: "profiles only", profiles: true, expected: []model.SignalType{model.SignalTypeProfiles}},
		{
			name:     "traces and logs",
			traces:   true,
			logs:     true,
			expected: []model.SignalType{model.SignalTypeTraces, model.SignalTypeLogs},
		},
		{name: "none supported", expected: []model.SignalType{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := actions.Action{}
			config.Spec.Signals.Traces.Supported = tt.traces
			config.Spec.Signals.Metrics.Supported = tt.metrics
			config.Spec.Signals.Logs.Supported = tt.logs
			config.Spec.Signals.Profiles.Supported = tt.profiles

			assert.Equal(t, tt.expected, ActionConfigToTypeOption(config).AllowedSignals)
		})
	}
}

func TestActionConfigToTypeOptionMapsMetadataAndFields(t *testing.T) {
	config := actions.Action{}
	config.Metadata.Type = "DeleteAttribute"
	config.Metadata.DisplayName = "Delete Attribute"
	config.Metadata.Category = "privacy"
	config.Spec.Subtitle = "subtitle"
	config.Spec.Description = "description"
	config.Spec.DocsURL = "https://docs.odigos.io/x"
	config.Spec.Fields = []actions.Field{{
		Name:            "attributeNamesToDelete",
		DisplayName:     "Attributes to delete",
		ComponentType:   "multiInput",
		ComponentProps:  map[string]interface{}{"tooltip": "names"},
		InitialValue:    "[]",
		RenderCondition: []string{"other", "=", "value"},
	}}

	option := ActionConfigToTypeOption(config)

	assert.Equal(t, "DeleteAttribute", option.Type)
	assert.Equal(t, "Delete Attribute", option.DisplayName)
	assert.Equal(t, "privacy", option.Category)
	assert.Equal(t, "subtitle", option.Subtitle)
	assert.Equal(t, "description", option.Description)
	assert.Equal(t, "https://docs.odigos.io/x", option.DocsURL)

	require.Len(t, option.Fields, 1)
	field := option.Fields[0]
	assert.Equal(t, "attributeNamesToDelete", field.Name)
	assert.Equal(t, "Attributes to delete", field.DisplayName)
	assert.Equal(t, "multiInput", field.ComponentType)
	assert.JSONEq(t, `{"tooltip":"names"}`, field.ComponentProperties)
	assert.Equal(t, "[]", field.InitialValue)
	assert.Equal(t, []string{"other", "=", "value"}, field.RenderCondition)
}

// The GraphQL schema declares fields as a non-null list, so an action without form fields must
// serialize as an empty list rather than null.
func TestActionConfigToTypeOptionFieldsAreNeverNil(t *testing.T) {
	option := ActionConfigToTypeOption(actions.Action{})

	assert.NotNil(t, option.Fields)
	assert.Empty(t, option.Fields)
}

// componentProps is free-form YAML rendered into a JSON string for the UI. A field whose props
// cannot be marshalled is skipped entirely, so the whole form control disappears; the surrounding
// fields must still be returned.
func TestActionConfigToTypeOptionSkipsUnmarshalableComponentProps(t *testing.T) {
	config := actions.Action{}
	config.Spec.Fields = []actions.Field{
		{Name: "first", ComponentType: "input"},
		{Name: "broken", ComponentType: "input", ComponentProps: map[string]interface{}{"fn": func() {}}},
		{Name: "last", ComponentType: "input"},
	}

	option := ActionConfigToTypeOption(config)

	names := make([]string, 0, len(option.Fields))
	for _, field := range option.Fields {
		names = append(names, field.Name)
	}
	assert.Equal(t, []string{"first", "last"}, names)
}

// ****************
// GetActionTypes
// ****************

// GetActionTypes is the only path the actionTypes query takes, so it must return the whole catalog
// with every entry fully mapped - not just enough of it for the picker to look populated.
func TestGetActionTypesReturnsTheWholeCatalog(t *testing.T) {
	catalog := loadActionsCatalog(t)

	options := GetActionTypes()
	require.Len(t, options, len(catalog))

	byType := make(map[string]*model.ActionTypeOption, len(options))
	for _, option := range options {
		require.NotNil(t, option)
		byType[option.Type] = option
	}

	for _, entry := range catalog {
		t.Run(entry.Metadata.Type, func(t *testing.T) {
			option, ok := byType[entry.Metadata.Type]
			require.True(t, ok, "catalog entry %q is missing from GetActionTypes", entry.Metadata.Type)

			expected := ActionConfigToTypeOption(entry)
			assert.Equal(t, &expected, option)
		})
	}
}

// Every returned option must be a distinct pointer. Taking the address of a loop variable would
// make every entry in the list report the last action.
func TestGetActionTypesReturnsDistinctOptions(t *testing.T) {
	loadActionsCatalog(t)

	seen := map[string]bool{}
	for _, option := range GetActionTypes() {
		assert.False(t, seen[option.Type], "action type %q returned twice", option.Type)
		seen[option.Type] = true
	}
	assert.Len(t, seen, len(actions.Get()))
}
