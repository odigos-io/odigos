package action

import (
	"os"
	"reflect"
	"testing"

	actionscatalog "github.com/odigos-io/odigos/actions"
	actionsv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigosactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The action catalog is a process-wide singleton that each binary's main loads once; resolving an
// Action against it silently yields "not a config extension" until Load has run.
func TestMain(m *testing.M) {
	if err := actionscatalog.Load(); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// SpanRenamer is applied to the agent as dynamic per-container config, not through a collector
// processor, so it is deliberately absent from the catalog that drives the UI and the collector.
const catalogTypeWithoutCatalogEntry = odigosactions.ActionSpanRenamer

func actionWithSpec(spec odigosv1.ActionSpec) *odigosv1.Action {
	return &odigosv1.Action{Spec: spec}
}

// specWithOnlyConfig returns an ActionSpec carrying a single zero-valued action config, addressed
// by its Go field name. The admission webhook rejects a spec with more or fewer than one config,
// so this is the only shape CatalogType ever sees in a cluster.
func specWithOnlyConfig(t *testing.T, fieldName string) odigosv1.ActionSpec {
	t.Helper()

	spec := odigosv1.ActionSpec{}
	field := reflect.ValueOf(&spec).Elem().FieldByName(fieldName)
	require.True(t, field.IsValid(), "ActionSpec has no field %s", fieldName)
	require.Equal(t, reflect.Pointer, field.Kind(), "ActionSpec.%s is not an action config pointer", fieldName)
	field.Set(reflect.New(field.Type().Elem()))

	return spec
}

// actionConfigFieldNames lists the ActionSpec fields that select which action a CR is: every
// pointer field. The remaining fields (name, notes, disabled, signals) are action-agnostic.
func actionConfigFieldNames() []string {
	specType := reflect.TypeOf(odigosv1.ActionSpec{})

	names := make([]string, 0, specType.NumField())
	for i := 0; i < specType.NumField(); i++ {
		field := specType.Field(i)
		if field.Type.Kind() == reflect.Pointer {
			names = append(names, field.Name)
		}
	}
	return names
}

// ****************
// CatalogType
// ****************

func TestCatalogType(t *testing.T) {
	tests := []struct {
		fieldName string
		expected  string
	}{
		{fieldName: "AddClusterInfo", expected: "AddClusterInfo"},
		{fieldName: "DeleteAttribute", expected: "DeleteAttribute"},
		{fieldName: "RenameAttribute", expected: "RenameAttribute"},
		{fieldName: "PiiMasking", expected: "PiiMasking"},
		// The CRD constant for this action is "K8sAttributes" while the catalog entry is
		// "K8sAttributesResolver"; CatalogType must produce the catalog spelling.
		{fieldName: "K8sAttributes", expected: "K8sAttributesResolver"},
		{fieldName: "URLTemplatization", expected: "URLTemplatization"},
		{fieldName: "SpanRenamer", expected: "SpanRenamer"},
		{fieldName: "ExtractAttribute", expected: "ExtractAttribute"},
		{fieldName: "DbQueryTemplatization", expected: "DbQueryTemplatization"},
		{fieldName: "InferDbAttributes", expected: "InferDbAttributes"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			spec := specWithOnlyConfig(t, tt.fieldName)
			assert.Equal(t, tt.expected, CatalogType(actionWithSpec(spec)))
		})
	}
}

func TestCatalogTypeNoConfigSet(t *testing.T) {
	assert.Equal(t, "", CatalogType(actionWithSpec(odigosv1.ActionSpec{})))
	assert.False(t, IsConfigExtension(actionWithSpec(odigosv1.ActionSpec{})))
}

// A config added to the CRD without a matching CatalogType case resolves to the empty type, which
// makes the action invisible to every catalog-driven decision (config-extension routing, the
// declared signals) instead of failing loudly. Reflecting over the spec keeps this test honest as
// action types are added.
func TestEveryActionSpecConfigResolvesToACatalogType(t *testing.T) {
	fieldNames := actionConfigFieldNames()
	require.NotEmpty(t, fieldNames)

	for _, fieldName := range fieldNames {
		t.Run(fieldName, func(t *testing.T) {
			spec := specWithOnlyConfig(t, fieldName)
			assert.NotEmpty(t, CatalogType(actionWithSpec(spec)),
				"ActionSpec.%s has no case in CatalogType, so an Action using it resolves to no catalog entry", fieldName)
		})
	}
}

// The two sides of the catalog contract are maintained independently: the YAML files declare
// metadata.type and CatalogType hardcodes the strings it maps to. An entry no Action CR can
// resolve to is dead metadata - its processors are never wired into the collector and the UI
// renders a form for an action the backend cannot recognise.
func TestEveryCatalogEntryIsReachableFromAnActionSpec(t *testing.T) {
	reachable := map[string]string{}
	for _, fieldName := range actionConfigFieldNames() {
		spec := specWithOnlyConfig(t, fieldName)
		reachable[CatalogType(actionWithSpec(spec))] = fieldName
	}

	catalog := actionscatalog.Get()
	require.NotEmpty(t, catalog, "action catalog is empty")

	for _, entry := range catalog {
		assert.Contains(t, reachable, entry.Metadata.Type,
			"catalog type %q is not produced by CatalogType for any ActionSpec config", entry.Metadata.Type)
	}
}

// The opposite direction of the same contract: an action config whose catalog entry is missing
// silently loses its collector processors and its declared signals. SpanRenamer is the only
// intentional omission, so pin the exception rather than the absence.
func TestOnlySpanRenamerHasNoCatalogEntry(t *testing.T) {
	for _, fieldName := range actionConfigFieldNames() {
		t.Run(fieldName, func(t *testing.T) {
			spec := specWithOnlyConfig(t, fieldName)
			catalogType := CatalogType(actionWithSpec(spec))

			_, found := actionscatalog.GetActionByType(catalogType)
			assert.Equal(t, catalogType != catalogTypeWithoutCatalogEntry, found,
				"catalog entry for %q: got found=%v", catalogType, found)
		})
	}
}

// ****************
// ConfigExtensionProcessorTypes / IsConfigExtension
// ****************

func TestConfigExtensionProcessorTypes(t *testing.T) {
	tests := []struct {
		fieldName string
		expected  []string
	}{
		{fieldName: "PiiMasking", expected: []string{consts.OdigosPiiMaskingProcessorType}},
		{fieldName: "DbQueryTemplatization", expected: []string{consts.OdigosSQLQueryProcessorType}},
		{fieldName: "InferDbAttributes", expected: []string{consts.OdigosSQLQueryProcessorType}},
		// Processor-backed actions: the autoscaler converts these into Processor CRs instead, so
		// reporting a config-extension processor here would double-apply them.
		{fieldName: "AddClusterInfo", expected: nil},
		{fieldName: "DeleteAttribute", expected: nil},
		{fieldName: "RenameAttribute", expected: nil},
		{fieldName: "K8sAttributes", expected: nil},
		{fieldName: "URLTemplatization", expected: nil},
		{fieldName: "ExtractAttribute", expected: nil},
		// Applied as dynamic agent config rather than through any collector processor.
		{fieldName: "SpanRenamer", expected: nil},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			action := actionWithSpec(specWithOnlyConfig(t, tt.fieldName))

			if len(tt.expected) == 0 {
				assert.Empty(t, ConfigExtensionProcessorTypes(action))
			} else {
				assert.Equal(t, tt.expected, ConfigExtensionProcessorTypes(action))
			}
			assert.Equal(t, len(tt.expected) > 0, IsConfigExtension(action))
		})
	}
}

func TestConfigExtensionProcessorTypesNoConfigSet(t *testing.T) {
	assert.Empty(t, ConfigExtensionProcessorTypes(actionWithSpec(odigosv1.ActionSpec{})))
}

// A disabled action is still a config-extension action. The autoscaler uses this to decide
// between the config-extension path and converting the action into a Processor CR: were a
// disabled PiiMasking action to report false, it would be turned into a legacy Processor CR and
// masking would keep running in the collector after the user disabled it.
func TestIsConfigExtensionIgnoresDisabled(t *testing.T) {
	spec := specWithOnlyConfig(t, "PiiMasking")
	spec.Disabled = true

	action := actionWithSpec(spec)
	assert.True(t, IsConfigExtension(action))
	assert.Equal(t, []string{consts.OdigosPiiMaskingProcessorType}, ConfigExtensionProcessorTypes(action))
}

// The catalog names collector processor types as free-form YAML strings. A name the collector does
// not register makes the gateway fail to build its pipeline, so the catalog may only reference
// processor types Odigos actually ships.
func TestCatalogConfigExtensionProcessorTypesAreKnownCollectorProcessors(t *testing.T) {
	known := map[string]bool{
		consts.OdigosPiiMaskingProcessorType: true,
		consts.OdigosSQLQueryProcessorType:   true,
	}

	for _, entry := range actionscatalog.Get() {
		for _, processor := range entry.Spec.Processors {
			if processor.ConfigMechanism != "odigosConfigExtension" {
				continue
			}
			assert.True(t, known[processor.Type],
				"action %q references unknown collector processor type %q", entry.Metadata.Type, processor.Type)
		}
	}
}

// Guard the one place CatalogType cannot use a shared constant: the K8sAttributes CRD constant and
// the catalog type differ, so a well-meaning cleanup that swaps the literal for the constant would
// break every catalog lookup for the action.
func TestK8sAttributesCatalogTypeDiffersFromTheCRDConstant(t *testing.T) {
	catalogType := CatalogType(actionWithSpec(specWithOnlyConfig(t, "K8sAttributes")))

	assert.NotEqual(t, actionsv1.ActionNameK8sAttributes, catalogType)
	_, found := actionscatalog.GetActionByType(catalogType)
	assert.True(t, found, "catalog has no entry for %q", catalogType)
}
