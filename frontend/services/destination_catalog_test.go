package services

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/destinations"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// destinations.Get() reads a package global that only Load() populates, and Load()
// replaces that global wholesale. Load it once per test binary rather than per test
// so no test can observe a half-loaded catalog.
var dsvcLoadOnce sync.Once

func dsvcEmbeddedCatalog(t *testing.T) []destinations.Destination {
	t.Helper()

	dsvcLoadOnce.Do(func() { _ = destinations.Load() })

	catalog := destinations.Get()
	require.NotEmpty(t, catalog, "the embedded destination catalog failed to load")
	return catalog
}

// dsvcField builds a catalog field. required is written into componentProps the way
// the yaml files do, so the production `.(bool)` type assertion sees a real bool.
func dsvcField(name string, secret bool, required bool) destinations.Field {
	return destinations.Field{
		Name:           name,
		DisplayName:    name + " display",
		ComponentType:  "input",
		ComponentProps: map[string]interface{}{"required": required},
		Secret:         secret,
	}
}

func dsvcDest(fields ...destinations.Field) *destinations.Destination {
	dest := &destinations.Destination{}
	dest.Metadata.Type = "synthetic-dest"
	dest.Spec.Fields = fields
	return dest
}

// ---------------------------------------------------------------------------
// VerifyDestinationDataScheme
// ---------------------------------------------------------------------------

func TestVerifyDestinationDataSchemeReportsEveryRequiredFieldThatIsMissingOrBlank(t *testing.T) {
	destTypeConfig := dsvcDest(
		dsvcField("REQUIRED_ABSENT", false, true),
		dsvcField("REQUIRED_BLANK", false, true),
		dsvcField("REQUIRED_PRESENT", false, true),
		dsvcField("OPTIONAL_ABSENT", false, false),
	)

	errs := VerifyDestinationDataScheme("synthetic-dest", destTypeConfig, map[string]string{
		"REQUIRED_BLANK":   "",
		"REQUIRED_PRESENT": "value",
	})

	messages := make([]string, 0, len(errs))
	for _, err := range errs {
		messages = append(messages, err.Error())
	}

	// Exactly two, so neither the satisfied required field nor the absent optional
	// one can start producing an error unnoticed.
	require.Len(t, messages, 2, "got %v", messages)
	assert.Contains(t, messages, "field REQUIRED_ABSENT is required")
	assert.Contains(t, messages, "field REQUIRED_BLANK is required")
}

// The yaml schema has no type backing for componentProps, so `required` is read with
// a `.(bool)` assertion that silently fails open for any other encoding. Pinning the
// fail-open half next to the working half keeps a future `required: "true"` from
// quietly turning a mandatory credential into an optional one.
func TestVerifyDestinationDataSchemeTreatsANonBooleanRequiredPropAsNotRequired(t *testing.T) {
	for _, tt := range []struct {
		name         string
		requiredProp interface{}
		wantErrors   int
	}{
		{name: "bool true is the only encoding that enforces", requiredProp: true, wantErrors: 1},
		{name: "bool false does not enforce", requiredProp: false, wantErrors: 0},
		{name: "the string \"true\" does not enforce", requiredProp: "true", wantErrors: 0},
		{name: "the integer 1 does not enforce", requiredProp: 1, wantErrors: 0},
		{name: "a nil value does not enforce", requiredProp: nil, wantErrors: 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			destTypeConfig := dsvcDest(destinations.Field{
				Name:           "API_KEY",
				ComponentProps: map[string]interface{}{"required": tt.requiredProp},
			})

			errs := VerifyDestinationDataScheme("synthetic-dest", destTypeConfig, map[string]string{})

			assert.Len(t, errs, tt.wantErrors)
		})
	}
}

func TestVerifyDestinationDataSchemeTreatsAFieldWithNoRequiredPropAsNotRequired(t *testing.T) {
	destTypeConfig := dsvcDest(destinations.Field{Name: "API_KEY"})

	assert.Empty(t, VerifyDestinationDataScheme("synthetic-dest", destTypeConfig, map[string]string{}))
}

func TestVerifyDestinationDataSchemeRejectsDataFieldsTheCatalogDoesNotDeclare(t *testing.T) {
	destTypeConfig := dsvcDest(dsvcField("KNOWN", false, false))

	errs := VerifyDestinationDataScheme("synthetic-dest", destTypeConfig, map[string]string{
		"KNOWN":   "value",
		"UNKNOWN": "value",
	})

	require.Len(t, errs, 1)
	// One ordered phrase: asserting the two substrings separately would not see the
	// field name and the destination type swapped in the format string.
	assert.EqualError(t, errs[0], "field UNKNOWN is not found in config for destination type 'synthetic-dest'")
}

func TestVerifyDestinationDataSchemeReportsMissingAndUnknownFieldsTogether(t *testing.T) {
	destTypeConfig := dsvcDest(dsvcField("REQUIRED", false, true))

	errs := VerifyDestinationDataScheme("synthetic-dest", destTypeConfig, map[string]string{"UNKNOWN": "value"})

	require.Len(t, errs, 2)
	assert.NotEqual(t, errs[0].Error(), errs[1].Error())
}

func TestVerifyDestinationDataSchemeAcceptsACompleteSubmission(t *testing.T) {
	destTypeConfig := dsvcDest(
		dsvcField("REQUIRED", false, true),
		dsvcField("OPTIONAL", false, false),
	)

	errs := VerifyDestinationDataScheme("synthetic-dest", destTypeConfig, map[string]string{
		"REQUIRED": "value",
		"OPTIONAL": "",
	})

	assert.Empty(t, errs)
}

// ---------------------------------------------------------------------------
// TransformFieldsToDataAndSecrets
// ---------------------------------------------------------------------------

// Whole-map equality on both outputs: a per-key assertion cannot see the two
// branches of the `if fieldConfig.Secret` cross-wired, because every field would
// still be present somewhere.
func TestTransformFieldsToDataAndSecretsRoutesEachFieldByItsOwnSecretFlag(t *testing.T) {
	destTypeConfig := dsvcDest(
		dsvcField("PLAIN_ONE", false, false),
		dsvcField("SECRET_ONE", true, false),
		dsvcField("PLAIN_TWO", false, false),
		dsvcField("SECRET_TWO", true, false),
	)

	data, secrets := TransformFieldsToDataAndSecrets(destTypeConfig, map[string]string{
		"PLAIN_ONE":  "plain-one-value",
		"SECRET_ONE": "secret-one-value",
		"PLAIN_TWO":  "plain-two-value",
		"SECRET_TWO": "secret-two-value",
	})

	assert.Equal(t, map[string]string{
		"PLAIN_ONE": "plain-one-value",
		"PLAIN_TWO": "plain-two-value",
	}, data)
	assert.Equal(t, map[string]string{
		"SECRET_ONE": "secret-one-value",
		"SECRET_TWO": "secret-two-value",
	}, secrets)
}

func TestTransformFieldsToDataAndSecretsDropsEmptyValuesFromBothMaps(t *testing.T) {
	destTypeConfig := dsvcDest(
		dsvcField("PLAIN", false, false),
		dsvcField("SECRET", true, false),
	)

	data, secrets := TransformFieldsToDataAndSecrets(destTypeConfig, map[string]string{
		"PLAIN":  "",
		"SECRET": "",
	})

	assert.Empty(t, data)
	assert.Empty(t, secrets)
}

// The transform is not a validator: a submitted field the catalog never declares is
// dropped without a word. VerifyDestinationDataScheme is the only thing standing
// between a typo'd field name and a destination created with that field missing, so
// both halves are asserted here against the same input.
func TestTransformFieldsToDataAndSecretsSilentlyDropsFieldsTheCatalogDoesNotDeclare(t *testing.T) {
	destTypeConfig := dsvcDest(dsvcField("KNOWN", false, false))
	fields := map[string]string{"KNOWN": "kept", "UNDECLARED": "dropped"}

	data, secrets := TransformFieldsToDataAndSecrets(destTypeConfig, fields)

	assert.Equal(t, map[string]string{"KNOWN": "kept"}, data)
	assert.Empty(t, secrets)
	assert.Len(t, VerifyDestinationDataScheme("synthetic-dest", destTypeConfig, fields), 1,
		"the transform drops the undeclared field silently, so the verifier must be the one to reject it")
}

// Driven over all 64 embedded destinations rather than one fixture: this is the
// function that decides whether a credential is written into the Destination CR in
// plain text or into a Secret, and the catalog is the only thing that tells it which.
func TestEveryEmbeddedDestinationSplitsItsFieldsIntoExactlyOneOfDataOrSecrets(t *testing.T) {
	catalog := dsvcEmbeddedCatalog(t)

	for _, destConfig := range catalog {
		t.Run(string(destConfig.Metadata.Type), func(t *testing.T) {
			if len(destConfig.Spec.Fields) == 0 {
				t.Skip("destination declares no fields")
			}

			submitted := map[string]string{}
			wantData := map[string]string{}
			wantSecrets := map[string]string{}
			for i, field := range destConfig.Spec.Fields {
				// A distinct value per field so a copied value cannot pass.
				value := fmt.Sprintf("value-%d", i)
				submitted[field.Name] = value
				if field.Secret {
					wantSecrets[field.Name] = value
				} else {
					wantData[field.Name] = value
				}
			}

			data, secrets := TransformFieldsToDataAndSecrets(&destConfig, submitted)

			assert.Equal(t, wantData, data)
			assert.Equal(t, wantSecrets, secrets)
			assert.Len(t, data, len(wantData))
			assert.Equal(t, len(submitted), len(data)+len(secrets),
				"every submitted field must land in exactly one of the two maps")
		})
	}
}

// dsvcPasswordFieldsNotMarkedSecret lists the catalog fields the UI renders as a
// password input but that are NOT declared `secret: true`, so their value is stored
// in Destination.Spec.Data in plain text instead of in a Secret. Each entry is a
// live divergence on main, not an exemption: TestEveryKnownPlaintextPasswordFieldStillDiverges
// fails as soon as one is fixed, so an entry cannot outlive its bug.
var dsvcPasswordFieldsNotMarkedSecret = map[string][]string{
	// destinations/data/gigapipe.yaml declares QRYN_API_KEY with
	// componentProps.type: password and required: true, but no `secret: true` --
	// unlike its sibling QRYN_API_SECRET, which has it.
	"qryn": {"QRYN_API_KEY"},
}

func dsvcIsKnownPlaintextPassword(destType common.DestinationType, fieldName string) bool {
	for _, known := range dsvcPasswordFieldsNotMarkedSecret[string(destType)] {
		if known == fieldName {
			return true
		}
	}
	return false
}

// A field the UI masks as a password is a credential by construction. If it is not
// marked `secret: true` it is written to Destination.Spec.Data, which is an ordinary
// CR: it is not encrypted at rest with the Secret, it is covered by Destination RBAC
// rather than Secret RBAC, and the read-only UI still serves it.
func TestEveryPasswordFieldInTheCatalogIsStoredAsASecret(t *testing.T) {
	catalog := dsvcEmbeddedCatalog(t)

	passwordFields := 0
	for _, destConfig := range catalog {
		for _, field := range destConfig.Spec.Fields {
			if componentType, _ := field.ComponentProps["type"].(string); componentType != "password" {
				continue
			}
			passwordFields++
			if dsvcIsKnownPlaintextPassword(destConfig.Metadata.Type, field.Name) {
				continue
			}

			_, secrets := TransformFieldsToDataAndSecrets(&destConfig, map[string]string{field.Name: "credential"})

			assert.Equal(t, map[string]string{field.Name: "credential"}, secrets,
				"%s/%s is rendered as a password but is not stored as a secret; mark it `secret: true` in destinations/data",
				destConfig.Metadata.Type, field.Name)
		}
	}

	assert.Positive(t, passwordFields, "no password field was examined, the assertion above is vacuous")
}

func TestEveryKnownPlaintextPasswordFieldStillDiverges(t *testing.T) {
	catalog := dsvcEmbeddedCatalog(t)

	for destType, fieldNames := range dsvcPasswordFieldsNotMarkedSecret {
		for _, fieldName := range fieldNames {
			found := false
			for _, destConfig := range catalog {
				if string(destConfig.Metadata.Type) != destType {
					continue
				}
				for _, field := range destConfig.Spec.Fields {
					if field.Name != fieldName {
						continue
					}
					found = true
					componentType, _ := field.ComponentProps["type"].(string)
					assert.False(t, componentType == "password" && field.Secret,
						"%s/%s is now stored as a secret; remove it from dsvcPasswordFieldsNotMarkedSecret",
						destType, fieldName)
				}
			}
			assert.True(t, found, "%s/%s no longer exists; remove it from dsvcPasswordFieldsNotMarkedSecret", destType, fieldName)
		}
	}
}

// ---------------------------------------------------------------------------
// DestinationTypeConfigToCategoryItem
// ---------------------------------------------------------------------------

func TestDestinationTypeConfigToCategoryItemCarriesEveryFieldProperty(t *testing.T) {
	field := destinations.Field{
		Name:            "API_KEY",
		DisplayName:     "API Key",
		ComponentType:   "input",
		ComponentProps:  map[string]interface{}{"required": true},
		Secret:          true,
		InitialValue:    "initial",
		RenderCondition: []string{"OTHER_FIELD", "==", "yes"},
		// Deliberately different from RenderCondition: two same-typed sibling
		// slices cannot be told apart if they share a fixture value.
		HideFromReadData:     []string{"SOME_FIELD"},
		CustomReadDataLabels: []*destinations.CustomReadDataLabel{{Condition: "cond", Title: "title", Value: "value"}},
	}

	destConfig := destinations.Destination{}
	destConfig.Spec.Fields = []destinations.Field{field}

	item := DestinationTypeConfigToCategoryItem(destConfig)

	require.Len(t, item.Fields, 1)
	got := item.Fields[0]

	// Reflection sweep: a property added to the yaml model and forgotten here would
	// otherwise be invisible, because it would be absent from both the fixture and
	// the expectation.
	value := reflect.ValueOf(*got)
	require.Positive(t, value.NumField())
	for i := 0; i < value.NumField(); i++ {
		assert.False(t, value.Field(i).IsZero(),
			"%s was not carried from the yaml field to the UI model", value.Type().Field(i).Name)
	}

	assert.Equal(t, "API_KEY", got.Name)
	assert.Equal(t, "API Key", got.DisplayName)
	assert.Equal(t, "input", got.ComponentType)
	assert.JSONEq(t, `{"required":true}`, got.ComponentProperties)
	assert.True(t, got.Secret)
	assert.Equal(t, "initial", got.InitialValue)
	assert.Equal(t, []string{"OTHER_FIELD", "==", "yes"}, got.RenderCondition)
	assert.Equal(t, []string{"SOME_FIELD"}, got.HideFromReadData)
	require.Len(t, got.CustomReadDataLabels, 1)
	assert.Equal(t, "cond", got.CustomReadDataLabels[0].Condition)
	assert.Equal(t, "title", got.CustomReadDataLabels[0].Title)
	assert.Equal(t, "value", got.CustomReadDataLabels[0].Value)
}

func TestDestinationTypeConfigToCategoryItemCarriesEveryTopLevelProperty(t *testing.T) {
	destConfig := destinations.Destination{}
	destConfig.Metadata.Type = "datadog"
	destConfig.Metadata.DisplayName = "Datadog"
	destConfig.Spec.Image = "datadog.svg"
	destConfig.Spec.TestConnectionSupported = true
	destConfig.Spec.Fields = []destinations.Field{{Name: "API_KEY"}}

	item := DestinationTypeConfigToCategoryItem(destConfig)

	assert.Equal(t, "datadog", item.Type)
	assert.Equal(t, "Datadog", item.DisplayName)
	assert.Equal(t, GetImageURL("datadog.svg"), item.ImageURL)
	assert.True(t, item.TestConnectionSupported)
	require.Len(t, item.Fields, 1)
}

// Four sibling bools copied one after another: with any two of them true at once a
// cross-wired pair is invisible, so each run has exactly one supported signal.
func TestDestinationTypeConfigToCategoryItemMapsEachSignalOnItsOwn(t *testing.T) {
	for _, tt := range []struct {
		name string
		set  func(*destinations.Destination)
		want model.SupportedSignals
	}{
		{
			name: "traces",
			set:  func(d *destinations.Destination) { d.Spec.Signals.Traces.Supported = true },
			want: model.SupportedSignals{
				Traces:   &model.ObservabilitySignalSupport{Supported: true},
				Metrics:  &model.ObservabilitySignalSupport{Supported: false},
				Logs:     &model.ObservabilitySignalSupport{Supported: false},
				Profiles: &model.ObservabilitySignalSupport{Supported: false},
			},
		},
		{
			name: "metrics",
			set:  func(d *destinations.Destination) { d.Spec.Signals.Metrics.Supported = true },
			want: model.SupportedSignals{
				Traces:   &model.ObservabilitySignalSupport{Supported: false},
				Metrics:  &model.ObservabilitySignalSupport{Supported: true},
				Logs:     &model.ObservabilitySignalSupport{Supported: false},
				Profiles: &model.ObservabilitySignalSupport{Supported: false},
			},
		},
		{
			name: "logs",
			set:  func(d *destinations.Destination) { d.Spec.Signals.Logs.Supported = true },
			want: model.SupportedSignals{
				Traces:   &model.ObservabilitySignalSupport{Supported: false},
				Metrics:  &model.ObservabilitySignalSupport{Supported: false},
				Logs:     &model.ObservabilitySignalSupport{Supported: true},
				Profiles: &model.ObservabilitySignalSupport{Supported: false},
			},
		},
		{
			name: "profiles",
			set:  func(d *destinations.Destination) { d.Spec.Signals.Profiles.Supported = true },
			want: model.SupportedSignals{
				Traces:   &model.ObservabilitySignalSupport{Supported: false},
				Metrics:  &model.ObservabilitySignalSupport{Supported: false},
				Logs:     &model.ObservabilitySignalSupport{Supported: false},
				Profiles: &model.ObservabilitySignalSupport{Supported: true},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			destConfig := destinations.Destination{}
			tt.set(&destConfig)

			item := DestinationTypeConfigToCategoryItem(destConfig)

			require.NotNil(t, item.SupportedSignals)
			assert.Equal(t, tt.want, *item.SupportedSignals)
		})
	}
}

// The conversion drops a whole field when json.Marshal of its componentProps fails,
// without reporting anything. No embedded destination hits that today, so the gate
// is on the count: a future componentProps value that yaml.v2 decodes into a
// map[interface{}]interface{} would silently remove the input from the UI form.
func TestEveryEmbeddedDestinationRendersEveryOneOfItsFieldsToTheUI(t *testing.T) {
	catalog := dsvcEmbeddedCatalog(t)

	totalFields := 0
	for _, destConfig := range catalog {
		item := DestinationTypeConfigToCategoryItem(destConfig)
		totalFields += len(destConfig.Spec.Fields)

		require.Len(t, item.Fields, len(destConfig.Spec.Fields),
			"%s: the UI form would be missing a field the catalog declares", destConfig.Metadata.Type)

		for i, field := range destConfig.Spec.Fields {
			assert.Equal(t, field.Name, item.Fields[i].Name, "%s: field order changed", destConfig.Metadata.Type)
		}
	}

	assert.Positive(t, totalFields)
}

func TestDestinationTypeConfigToCategoryItemDropsAFieldWhoseComponentPropsCannotBeMarshalled(t *testing.T) {
	destConfig := destinations.Destination{}
	destConfig.Spec.Fields = []destinations.Field{
		{Name: "GOOD", ComponentProps: map[string]interface{}{"required": true}},
		// yaml.v2 decodes a nested mapping into map[interface{}]interface{}, which
		// encoding/json refuses.
		{Name: "BAD", ComponentProps: map[string]interface{}{"nested": map[interface{}]interface{}{1: "one"}}},
	}

	item := DestinationTypeConfigToCategoryItem(destConfig)

	require.Len(t, item.Fields, 1, "the unmarshallable field is dropped without an error")
	assert.Equal(t, "GOOD", item.Fields[0].Name)
}

// ---------------------------------------------------------------------------
// GetDestinationCategories / GetDestinationTypeConfig
// ---------------------------------------------------------------------------

func TestGetDestinationCategoriesReturnsEveryEmbeddedDestinationExactlyOnceUnderItsOwnCategory(t *testing.T) {
	catalog := dsvcEmbeddedCatalog(t)

	wantCategoryOf := map[string]string{}
	for _, destConfig := range catalog {
		wantCategoryOf[string(destConfig.Metadata.Type)] = destConfig.Metadata.Category
	}

	resp := GetDestinationCategories()

	gotCategoryOf := map[string]string{}
	for _, category := range resp.Categories {
		for _, item := range category.Items {
			_, duplicate := gotCategoryOf[item.Type]
			assert.False(t, duplicate, "%s appears in more than one category", item.Type)
			gotCategoryOf[item.Type] = category.Name
		}
	}

	assert.Equal(t, wantCategoryOf, gotCategoryOf)
}

// The description map is hardcoded in the frontend while the category values live in
// destinations/data/*.yaml, with nothing tying the two together. A yaml introducing a
// third category would render a category header with an empty description.
func TestGetDestinationCategoriesDescribesEveryCategoryTheCatalogUses(t *testing.T) {
	dsvcEmbeddedCatalog(t)

	resp := GetDestinationCategories()

	require.NotEmpty(t, resp.Categories)
	describedBy := map[string]string{}
	for _, category := range resp.Categories {
		assert.NotEmpty(t, category.Description,
			"category %q has no description in GetDestinationCategories", category.Name)
		assert.NotEmpty(t, category.Items, "category %q is empty", category.Name)

		// Distinctness, so a description looked up under a constant key rather
		// than the category's own name cannot pass.
		previous, shared := describedBy[category.Description]
		assert.False(t, shared, "categories %q and %q share a description", previous, category.Name)
		describedBy[category.Description] = category.Name
	}
	assert.Len(t, describedBy, len(resp.Categories))
}

func TestGetDestinationTypeConfigResolvesEveryEmbeddedTypeToItself(t *testing.T) {
	catalog := dsvcEmbeddedCatalog(t)

	for _, destConfig := range catalog {
		resolved, err := GetDestinationTypeConfig(destConfig.Metadata.Type)

		require.NoError(t, err)
		require.NotNil(t, resolved)
		assert.Equal(t, destConfig.Metadata.Type, resolved.Metadata.Type)
		assert.Equal(t, destConfig.Metadata.DisplayName, resolved.Metadata.DisplayName)
	}
}

func TestGetDestinationTypeConfigRejectsLookalikeTypes(t *testing.T) {
	dsvcEmbeddedCatalog(t)

	for _, lookalike := range []common.DestinationType{
		"Datadog",     // wrong case
		"datadog ",    // trailing space
		"datadogs",    // plural
		"data-dog",    // hyphenated
		"data_dog",    // underscored
		"gigapipe",    // the file name of the "qryn" entry, not its type
		"",            // empty
		"not-a-thing", // absent
	} {
		resolved, err := GetDestinationTypeConfig(lookalike)

		assert.Nil(t, resolved, "%q resolved to a destination type", lookalike)
		require.Error(t, err)
		assert.EqualError(t, err, fmt.Sprintf("destination type %s not found", lookalike))
	}
}

// ---------------------------------------------------------------------------
// K8sDestinationToEndpointFormat
// ---------------------------------------------------------------------------

func dsvcK8sDestination(spec v1alpha1.DestinationSpec) v1alpha1.Destination {
	return v1alpha1.Destination{
		ObjectMeta: metav1.ObjectMeta{Name: "odigos.io.dest.datadog-abcde"},
		Spec:       spec,
	}
}

func TestK8sDestinationToEndpointFormatMergesSecretsOverData(t *testing.T) {
	dest := dsvcK8sDestination(v1alpha1.DestinationSpec{
		Type:            "datadog",
		DestinationName: "my datadog",
		Data:            map[string]string{"DATADOG_SITE": "datadoghq.com", "SHARED": "from-data"},
	})

	got := K8sDestinationToEndpointFormat(dest, map[string]string{
		"DATADOG_API_KEY": "super-secret",
		"SHARED":          "from-secret",
	})

	var fields map[string]string
	require.NoError(t, json.Unmarshal([]byte(got.Fields), &fields))
	assert.Equal(t, map[string]string{
		"DATADOG_SITE":    "datadoghq.com",
		"DATADOG_API_KEY": "super-secret",
		"SHARED":          "from-secret",
	}, fields)

	assert.Equal(t, "odigos.io.dest.datadog-abcde", got.ID)
	assert.Equal(t, "datadog", got.Type)
	assert.Equal(t, "my datadog", got.Name)
}

func TestK8sDestinationToEndpointFormatReportsEachExportedSignalOnItsOwn(t *testing.T) {
	for _, tt := range []struct {
		signal common.ObservabilitySignal
		want   model.ExportedSignals
	}{
		{common.TracesObservabilitySignal, model.ExportedSignals{Traces: true}},
		{common.MetricsObservabilitySignal, model.ExportedSignals{Metrics: true}},
		{common.LogsObservabilitySignal, model.ExportedSignals{Logs: true}},
		{common.ProfilesObservabilitySignal, model.ExportedSignals{Profiles: true}},
	} {
		t.Run(string(tt.signal), func(t *testing.T) {
			dest := dsvcK8sDestination(v1alpha1.DestinationSpec{
				Type:    "datadog",
				Signals: []common.ObservabilitySignal{tt.signal},
			})

			got := K8sDestinationToEndpointFormat(dest, nil)

			require.NotNil(t, got.ExportedSignals)
			assert.Equal(t, tt.want, *got.ExportedSignals)
		})
	}
}

func TestK8sDestinationToEndpointFormatMapsEveryConditionStatus(t *testing.T) {
	dest := dsvcK8sDestination(v1alpha1.DestinationSpec{Type: "datadog"})
	dest.Status.Conditions = []metav1.Condition{
		{Type: "Unknown", Status: metav1.ConditionUnknown, Reason: "Pending", Message: "pending message"},
		{Type: "True", Status: metav1.ConditionTrue, Reason: "Ready", Message: "ready message"},
		{Type: "False", Status: metav1.ConditionFalse, Reason: "Broken", Message: "broken message"},
		{
			// A reason the shared helper classifies as "disabled" overrides the
			// status that the metav1 condition would otherwise produce.
			Type:    "DisabledReason",
			Status:  metav1.ConditionFalse,
			Reason:  string(v1alpha1.RuntimeDetectionReasonNoRunningPods),
			Message: "no running pods",
		},
	}

	got := K8sDestinationToEndpointFormat(dest, nil)

	require.Len(t, got.Conditions, 4)
	assert.Equal(t, model.ConditionStatusLoading, got.Conditions[0].Status)
	assert.Equal(t, model.ConditionStatusSuccess, got.Conditions[1].Status)
	assert.Equal(t, model.ConditionStatusError, got.Conditions[2].Status)
	assert.Equal(t, model.ConditionStatusDisabled, got.Conditions[3].Status)

	// Reason and message are adjacent same-typed strings; assert both on one
	// condition so they cannot be swapped.
	assert.Equal(t, "False", got.Conditions[2].Type)
	require.NotNil(t, got.Conditions[2].Reason)
	require.NotNil(t, got.Conditions[2].Message)
	assert.Equal(t, "Broken", *got.Conditions[2].Reason)
	assert.Equal(t, "broken message", *got.Conditions[2].Message)
}

func TestK8sDestinationToEndpointFormatDefaultsDisabledToFalseWhenUnset(t *testing.T) {
	disabled := true

	assert.False(t, K8sDestinationToEndpointFormat(dsvcK8sDestination(v1alpha1.DestinationSpec{Type: "datadog"}), nil).Disabled)
	assert.True(t, K8sDestinationToEndpointFormat(dsvcK8sDestination(v1alpha1.DestinationSpec{
		Type:     "datadog",
		Disabled: &disabled,
	}), nil).Disabled)
}

func TestK8sDestinationToEndpointFormatReportsTheDataStreamsOfTheDestination(t *testing.T) {
	dest := dsvcK8sDestination(v1alpha1.DestinationSpec{
		Type:           "datadog",
		SourceSelector: &v1alpha1.SourceSelector{DataStreams: []string{"alpha", "beta", "alpha"}},
	})

	got := K8sDestinationToEndpointFormat(dest, nil)

	require.Len(t, got.DataStreamNames, 2)
	assert.Equal(t, "alpha", *got.DataStreamNames[0])
	assert.Equal(t, "beta", *got.DataStreamNames[1])
}

func TestK8sDestinationToEndpointFormatResolvesTheCatalogEntryOfAKnownType(t *testing.T) {
	dsvcEmbeddedCatalog(t)

	got := K8sDestinationToEndpointFormat(dsvcK8sDestination(v1alpha1.DestinationSpec{Type: "datadog"}), nil)

	require.NotNil(t, got.DestinationType)
	assert.Equal(t, "datadog", got.DestinationType.Type)
	assert.NotEmpty(t, got.DestinationType.Fields)
}

// GetDestinationByType also returns a "found" bool, which this caller discards. A
// Destination CR applied with a type the catalog does not know therefore renders as
// a destination whose type block is blank rather than as an error.
func TestK8sDestinationToEndpointFormatYieldsABlankTypeBlockForAnUnknownType(t *testing.T) {
	dsvcEmbeddedCatalog(t)

	got := K8sDestinationToEndpointFormat(dsvcK8sDestination(v1alpha1.DestinationSpec{Type: "not-a-thing"}), nil)

	assert.Equal(t, "not-a-thing", got.Type, "the spec type is still reported")
	require.NotNil(t, got.DestinationType)
	assert.Empty(t, got.DestinationType.Type)
	assert.Empty(t, got.DestinationType.DisplayName)
	assert.Empty(t, got.DestinationType.Fields)
}

// ---------------------------------------------------------------------------
// ExportedSignalsObjectToSlice
// ---------------------------------------------------------------------------

func TestExportedSignalsObjectToSliceEmitsEachSignalOnItsOwn(t *testing.T) {
	for _, tt := range []struct {
		name  string
		input model.ExportedSignalsInput
		want  []common.ObservabilitySignal
	}{
		{"none", model.ExportedSignalsInput{}, nil},
		{"traces", model.ExportedSignalsInput{Traces: true}, []common.ObservabilitySignal{common.TracesObservabilitySignal}},
		{"metrics", model.ExportedSignalsInput{Metrics: true}, []common.ObservabilitySignal{common.MetricsObservabilitySignal}},
		{"logs", model.ExportedSignalsInput{Logs: true}, []common.ObservabilitySignal{common.LogsObservabilitySignal}},
		{"profiles", model.ExportedSignalsInput{Profiles: true}, []common.ObservabilitySignal{common.ProfilesObservabilitySignal}},
		{
			"all four",
			model.ExportedSignalsInput{Traces: true, Metrics: true, Logs: true, Profiles: true},
			[]common.ObservabilitySignal{
				common.TracesObservabilitySignal,
				common.MetricsObservabilitySignal,
				common.LogsObservabilitySignal,
				common.ProfilesObservabilitySignal,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ExportedSignalsObjectToSlice(&tt.input))
		})
	}
}

// The slice the UI submits and the bools the UI reads back have to agree, or a
// destination saved for one signal comes back selected for another.
func TestExportedSignalsSurviveTheRoundTripThroughTheDestinationSpec(t *testing.T) {
	input := model.ExportedSignalsInput{Traces: true, Metrics: false, Logs: true, Profiles: true}

	dest := dsvcK8sDestination(v1alpha1.DestinationSpec{
		Type:    "datadog",
		Signals: ExportedSignalsObjectToSlice(&input),
	})

	got := K8sDestinationToEndpointFormat(dest, nil)

	require.NotNil(t, got.ExportedSignals)
	assert.Equal(t, model.ExportedSignals(input), *got.ExportedSignals)
}
