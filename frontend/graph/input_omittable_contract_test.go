package graph

import (
	"reflect"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gqlgen only generates graphql.Omittable for the fields listed under `models:` in gqlgen.yml.
// Every rule update merge in the backend branches on IsSet() to tell "the client omitted this
// field, keep the stored value" from "the client sent null, clear it". A nullable field that is
// missing from gqlgen.yml is generated as a plain pointer, so its omitted and null cases collapse
// into one and the merge silently keeps the old value — the bug fixed in #5954. These tests derive
// the expectation from the live schema, so a nullable field added to one of these inputs fails
// here instead of shipping with the regression.

var omitOmittablePkgPath = reflect.TypeOf(graphql.Omittable[struct{}]{}).PkgPath()

func omitIsOmittable(fieldType reflect.Type) bool {
	return fieldType.PkgPath() == omitOmittablePkgPath && strings.HasPrefix(fieldType.Name(), "Omittable[")
}

// omitGoFieldFor resolves the generated Go field for a GraphQL input field name. gqlgen capitalises
// initialisms (httpServer -> HTTPServer), so names are compared case-insensitively.
func omitGoFieldFor(t *testing.T, goType reflect.Type, graphqlFieldName string) reflect.StructField {
	t.Helper()

	for i := range goType.NumField() {
		if strings.EqualFold(goType.Field(i).Name, graphqlFieldName) {
			return goType.Field(i)
		}
	}
	t.Fatalf("no generated Go field on %s for schema field %q", goType.Name(), graphqlFieldName)
	return reflect.StructField{}
}

// The sampling rule inputs are wholly partial-update: the edit form sends null for "All
// Operations", "all sources", "drop all" and a cleared note, so every nullable field needs the
// omitted/null distinction.
func TestEveryNullableSamplingRuleInputFieldIsOmittable(t *testing.T) {
	schema := NewExecutableSchema(Config{}).Schema()

	inputs := map[string]reflect.Type{
		"NoisyOperationRuleInput":          reflect.TypeOf(model.NoisyOperationRuleInput{}),
		"HighlyRelevantOperationRuleInput": reflect.TypeOf(model.HighlyRelevantOperationRuleInput{}),
		"CostReductionRuleInput":           reflect.TypeOf(model.CostReductionRuleInput{}),
	}

	nonNullableSeen := 0
	for inputName, goType := range inputs {
		t.Run(inputName, func(t *testing.T) {
			definition := schema.Types[inputName]
			require.NotNil(t, definition, "input %s is not in the schema", inputName)
			require.Positive(t, len(definition.Fields))
			// A Go field with no schema field of its own would never be unmarshalled at all.
			require.Equal(t, goType.NumField(), len(definition.Fields),
				"%s has %d schema fields but %d generated Go fields", inputName, len(definition.Fields), goType.NumField())

			nullableSeen := 0
			for _, schemaField := range definition.Fields {
				goField := omitGoFieldFor(t, goType, schemaField.Name)

				if schemaField.Type.NonNull {
					nonNullableSeen++
					assert.False(t, omitIsOmittable(goField.Type),
						"%s.%s is non-nullable, so it can never be omitted-vs-null and must not be Omittable", inputName, schemaField.Name)
					continue
				}

				nullableSeen++
				assert.True(t, omitIsOmittable(goField.Type),
					"%s.%s is nullable and must be marked `omittable: true` in gqlgen.yml, otherwise an explicit null is indistinguishable from an omitted field and the update keeps the stored value",
					inputName, schemaField.Name)
			}
			assert.Positive(t, nullableSeen, "%s has no nullable field, so this test proves nothing", inputName)
		})
	}

	assert.Positive(t, nonNullableSeen,
		"no non-nullable sampling rule input field exists, so the must-not-be-Omittable arm never ran")
}

// The instrumentation rule input is deliberately mixed, and the asymmetry is the contract: the two
// selectors take the partial-update semantics, while the type payloads must keep their stored value
// on null because clients (the profiling symbol table) send null for the payloads that don't apply
// to the rule's type. Marking a payload omittable would wipe it on every such update.
func TestInstrumentationRuleInputSeparatesSelectorsFromTypePayloads(t *testing.T) {
	goType := reflect.TypeOf(model.InstrumentationRuleInput{})
	schema := NewExecutableSchema(Config{}).Schema()
	definition := schema.Types["InstrumentationRuleInput"]
	require.NotNil(t, definition)

	clearableSelectors := []string{"sourcesScopes", "instrumentationLibraries"}
	preservedOnNull := []string{"codeAttributes", "headersCollection", "payloadCollection", "customInstrumentations", "networkMetrics"}

	for _, fieldName := range clearableSelectors {
		assert.True(t, omitIsOmittable(omitGoFieldFor(t, goType, fieldName).Type),
			"a null %s widens the rule (entire cluster / all libraries), so it must be Omittable", fieldName)
	}
	for _, fieldName := range preservedOnNull {
		assert.False(t, omitIsOmittable(omitGoFieldFor(t, goType, fieldName).Type),
			"%s must keep its stored value on null, so it must not be Omittable", fieldName)
	}

	// Both lists name real schema fields, and together they cover every field the two policies
	// apply to, so a renamed or newly added selector cannot slip past unclassified.
	classified := make(map[string]bool, len(clearableSelectors)+len(preservedOnNull))
	for _, fieldName := range append(append([]string{}, clearableSelectors...), preservedOnNull...) {
		require.NotNil(t, definition.Fields.ForName(fieldName), "schema field %q no longer exists", fieldName)
		classified[fieldName] = true
	}
	for _, schemaField := range definition.Fields {
		if classified[schemaField.Name] {
			continue
		}
		assert.False(t, omitIsOmittable(omitGoFieldFor(t, goType, schemaField.Name).Type),
			"%s is Omittable but is not classified by this test; decide whether a null clears it or keeps the stored value", schemaField.Name)
	}
}
