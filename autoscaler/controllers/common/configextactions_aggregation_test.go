package common

import (
	"encoding/json"
	"testing"

	actionscatalog "github.com/odigos-io/odigos/actions"
	actionsv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigosactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	odigoscommon "github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Config-extension resolution reads the process-wide action catalog that each binary's main loads;
// without it every action looks like a non-config-extension one and these tests would pass vacuously.
func loadActionCatalog(t *testing.T) {
	t.Helper()
	require.NoError(t, actionscatalog.Load())
}

// Both DbQueryTemplatization and InferDbAttributes are backed by the same odigossqlquery processor,
// which is what makes the per-processor aggregation below load-bearing.
func dbQueryTemplatizationAction(name string, signals ...odigoscommon.ObservabilitySignal) odigosv1.Action {
	return odigosv1.Action{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "odigos-system"},
		Spec: odigosv1.ActionSpec{
			ActionName:            name,
			Signals:               signals,
			DbQueryTemplatization: &odigosactions.DbQueryTemplatizationConfig{},
		},
	}
}

func inferDbAttributesAction(name string, signals ...odigoscommon.ObservabilitySignal) odigosv1.Action {
	return odigosv1.Action{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "odigos-system"},
		Spec: odigosv1.ActionSpec{
			ActionName:        name,
			Signals:           signals,
			InferDbAttributes: &odigosactions.InferDbAttributesConfig{},
		},
	}
}

// A Processor-backed action, for asserting it never reaches the config-extension path.
func deleteAttributeAction(name string, signals ...odigoscommon.ObservabilitySignal) odigosv1.Action {
	return odigosv1.Action{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "odigos-system"},
		Spec: odigosv1.ActionSpec{
			ActionName:      name,
			Signals:         signals,
			DeleteAttribute: &actionsv1.DeleteAttributeConfig{AttributeNamesToDelete: []string{"user.id"}},
		},
	}
}

// Two different action types share one collector processor, so their requested signals must be
// merged into a single Processor. Emitting one Processor per action would make the later one
// overwrite the earlier (both are named after the processor type), silently dropping signals.
func TestConvertActionsToConfigExtensionProcessorsMergesActionsSharingAProcessor(t *testing.T) {
	loadActionCatalog(t)

	processors := ConvertActionsToConfigExtensionProcessors(odigosv1.ActionList{
		Items: []odigosv1.Action{
			dbQueryTemplatizationAction("templatize", odigoscommon.TracesObservabilitySignal),
			inferDbAttributesAction("infer", odigoscommon.TracesObservabilitySignal),
		},
	})

	require.Len(t, processors, 1, "both actions are backed by %s and must produce a single processor", consts.OdigosSQLQueryProcessorType)
	assert.Equal(t, consts.OdigosSQLQueryProcessorType, processors[0].Spec.Type)
	assert.Equal(t, []odigoscommon.ObservabilitySignal{odigoscommon.TracesObservabilitySignal}, processors[0].Spec.Signals,
		"a signal requested by several actions must be listed once")
}

// The same signal listed twice on one action must not be duplicated either: a repeated entry ends
// up in the rendered collector config for the pipeline.
func TestConvertActionsToConfigExtensionProcessorsDeduplicatesRepeatedSignals(t *testing.T) {
	loadActionCatalog(t)

	processors := ConvertActionsToConfigExtensionProcessors(odigosv1.ActionList{
		Items: []odigosv1.Action{
			dbQueryTemplatizationAction("templatize",
				odigoscommon.TracesObservabilitySignal,
				odigoscommon.TracesObservabilitySignal,
			),
		},
	})

	require.Len(t, processors, 1)
	assert.Equal(t, []odigoscommon.ObservabilitySignal{odigoscommon.TracesObservabilitySignal}, processors[0].Spec.Signals)
}

// A disabled action must contribute nothing. Its signals surviving here would keep the processor in
// the collector config, so the action would go on running after the user disabled it.
func TestConvertActionsToConfigExtensionProcessorsSkipsDisabledActions(t *testing.T) {
	loadActionCatalog(t)

	disabled := dbQueryTemplatizationAction("templatize", odigoscommon.TracesObservabilitySignal)
	disabled.Spec.Disabled = true

	assert.Empty(t, ConvertActionsToConfigExtensionProcessors(odigosv1.ActionList{
		Items: []odigosv1.Action{disabled},
	}), "a disabled action must not produce a processor")

	// A second, enabled action sharing the processor keeps it alive, and the disabled one still
	// contributes nothing to it.
	processors := ConvertActionsToConfigExtensionProcessors(odigosv1.ActionList{
		Items: []odigosv1.Action{disabled, inferDbAttributesAction("infer", odigoscommon.TracesObservabilitySignal)},
	})
	require.Len(t, processors, 1)
	assert.Equal(t, []odigoscommon.ObservabilitySignal{odigoscommon.TracesObservabilitySignal}, processors[0].Spec.Signals)
}

// Processor-backed actions are turned into Processor CRs by the actions controller. Producing a
// config-extension Processor for one as well would apply it twice.
func TestConvertActionsToConfigExtensionProcessorsIgnoresProcessorBackedActions(t *testing.T) {
	loadActionCatalog(t)

	assert.Empty(t, ConvertActionsToConfigExtensionProcessors(odigosv1.ActionList{
		Items: []odigosv1.Action{deleteAttributeAction("delete", odigoscommon.TracesObservabilitySignal)},
	}))
}

func TestConvertActionsToConfigExtensionProcessorsNoActions(t *testing.T) {
	loadActionCatalog(t)

	assert.Empty(t, ConvertActionsToConfigExtensionProcessors(odigosv1.ActionList{}))
}

// An action with no signals cannot be wired anywhere, so it must not leave an empty-signal
// Processor behind: the cluster gateway would then carry a processor no pipeline consumes.
func TestConvertActionsToConfigExtensionProcessorsSkipsActionsWithoutSignals(t *testing.T) {
	loadActionCatalog(t)

	assert.Empty(t, ConvertActionsToConfigExtensionProcessors(odigosv1.ActionList{
		Items: []odigosv1.Action{dbQueryTemplatizationAction("templatize")},
	}))
}

// The generated Processor is what the collector config is rendered from, so every field of it is
// part of the contract: the name identifies the processor in the pipeline, the role decides which
// collector gets it, and the config tells the collector to read the action's settings from the
// Odigos config extension rather than from the Processor CR.
func TestConvertActionsToConfigExtensionProcessorsProcessorShape(t *testing.T) {
	loadActionCatalog(t)

	processors := ConvertActionsToConfigExtensionProcessors(odigosv1.ActionList{
		Items: []odigosv1.Action{inferDbAttributesAction("infer", odigoscommon.TracesObservabilitySignal)},
	})
	require.Len(t, processors, 1)
	proc := processors[0]

	assert.Equal(t, "Processor", proc.Kind)
	assert.Equal(t, "odigos.io/v1alpha1", proc.APIVersion)
	assert.Equal(t, consts.OdigosSQLQueryProcessorType, proc.Name,
		"the Processor is named after its type so actions sharing a processor collapse onto one object")
	assert.Equal(t, consts.OdigosSQLQueryProcessorType, proc.Spec.Type)
	assert.Equal(t, []odigosv1.CollectorsGroupRole{odigosv1.CollectorsGroupRoleClusterGateway}, proc.Spec.CollectorRoles)

	var config map[string]any
	require.NoError(t, json.Unmarshal(proc.Spec.ProcessorConfig.Raw, &config))
	assert.Equal(t, map[string]any{"odigos_config_extension": k8sconsts.OdigosConfigK8sExtensionType}, config)
}

// ****************
// IsLegacyConfigExtensionProcessorType
// ****************

// Leftover Processor CRs from versions that backed these actions with real Processor CRs must be
// recognised so the migration cleanup removes them; anything else must be left alone, since
// misclassifying a live processor type deletes a working part of the collector config.
func TestIsLegacyConfigExtensionProcessorType(t *testing.T) {
	tests := []struct {
		processorType string
		expected      bool
	}{
		{processorType: consts.OdigosPiiMaskingProcessorType, expected: true},
		{processorType: consts.OdigosSQLQueryProcessorType, expected: true},
		{processorType: "", expected: false},
		{processorType: "batch", expected: false},
		{processorType: "odigosurltemplate", expected: false},
		{processorType: "attributes", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.processorType, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsLegacyConfigExtensionProcessorType(tt.processorType))
		})
	}
}

// The legacy list and the catalog describe the same migration from both ends: every processor type
// the catalog now applies through the config extension used to be a Processor CR, so a new
// config-extension processor type added to the catalog without being listed here would leave its
// stale Processor CR behind in upgraded clusters.
func TestEveryConfigExtensionProcessorTypeIsRecognisedAsLegacy(t *testing.T) {
	loadActionCatalog(t)

	for _, entry := range actionscatalog.Get() {
		for _, processor := range entry.Spec.Processors {
			if processor.ConfigMechanism != "odigosConfigExtension" {
				continue
			}
			assert.True(t, IsLegacyConfigExtensionProcessorType(processor.Type),
				"catalog action %q applies processor %q through the config extension, but it is not recognised as a legacy Processor type",
				entry.Metadata.Type, processor.Type)
		}
	}
}
