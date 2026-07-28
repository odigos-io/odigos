package common

import (
	"testing"

	actionscatalog "github.com/odigos-io/odigos/actions"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMain(m *testing.M) {
	if err := actionscatalog.Load(); err != nil {
		panic(err)
	}
	m.Run()
}

func TestConvertActionsToConfigExtensionProcessors_SQLPlacement(t *testing.T) {
	actionList := odigosv1.ActionList{
		Items: []odigosv1.Action{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "db-query"},
				Spec: odigosv1.ActionSpec{
					Signals:               []common.ObservabilitySignal{common.TracesObservabilitySignal},
					DbQueryTemplatization: &actions.DbQueryTemplatizationConfig{},
				},
			},
		},
	}

	t.Run("gateway when span metrics disabled", func(t *testing.T) {
		procs := ConvertActionsToConfigExtensionProcessors(actionList, false)
		require.Len(t, procs, 1)
		assert.Equal(t, consts.OdigosSQLQueryProcessorType, procs[0].Spec.Type)
		assert.Equal(t, []odigosv1.CollectorsGroupRole{odigosv1.CollectorsGroupRoleClusterGateway}, procs[0].Spec.CollectorRoles)
		assert.Equal(t, 1, procs[0].Spec.OrderHint)
	})

	t.Run("node collector when span metrics enabled", func(t *testing.T) {
		procs := ConvertActionsToConfigExtensionProcessors(actionList, true)
		require.Len(t, procs, 1)
		assert.Equal(t, consts.OdigosSQLQueryProcessorType, procs[0].Spec.Type)
		assert.Equal(t, []odigosv1.CollectorsGroupRole{odigosv1.CollectorsGroupRoleNodeCollector}, procs[0].Spec.CollectorRoles)
		assert.Equal(t, 1, procs[0].Spec.OrderHint)
	})
}

func TestConvertActionsToConfigExtensionProcessors_InferDbAttributesPlacement(t *testing.T) {
	actionList := odigosv1.ActionList{
		Items: []odigosv1.Action{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "infer-db"},
				Spec: odigosv1.ActionSpec{
					Signals:            []common.ObservabilitySignal{common.TracesObservabilitySignal},
					InferDbAttributes:  &actions.InferDbAttributesConfig{},
				},
			},
		},
	}

	procs := ConvertActionsToConfigExtensionProcessors(actionList, true)
	require.Len(t, procs, 1)
	assert.Equal(t, consts.OdigosSQLQueryProcessorType, procs[0].Spec.Type)
	assert.Equal(t, []odigosv1.CollectorsGroupRole{odigosv1.CollectorsGroupRoleNodeCollector}, procs[0].Spec.CollectorRoles)
}

func TestConvertActionsToConfigExtensionProcessors_PiiMaskingStaysOnGateway(t *testing.T) {
	actionList := odigosv1.ActionList{
		Items: []odigosv1.Action{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "pii"},
				Spec: odigosv1.ActionSpec{
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					PiiMasking: &actions.PiiMaskingConfig{},
				},
			},
		},
	}

	procs := ConvertActionsToConfigExtensionProcessors(actionList, true)
	require.Len(t, procs, 1)
	assert.Equal(t, consts.OdigosPiiMaskingProcessorType, procs[0].Spec.Type)
	assert.Equal(t, []odigosv1.CollectorsGroupRole{odigosv1.CollectorsGroupRoleClusterGateway}, procs[0].Spec.CollectorRoles)
	assert.Equal(t, 1, procs[0].Spec.OrderHint)
}

func TestConvertActionsToConfigExtensionProcessors_DisabledIgnored(t *testing.T) {
	actionList := odigosv1.ActionList{
		Items: []odigosv1.Action{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "db-query"},
				Spec: odigosv1.ActionSpec{
					Disabled:              true,
					Signals:               []common.ObservabilitySignal{common.TracesObservabilitySignal},
					DbQueryTemplatization: &actions.DbQueryTemplatizationConfig{},
				},
			},
		},
	}

	procs := ConvertActionsToConfigExtensionProcessors(actionList, true)
	assert.Empty(t, procs)
}

func TestConfigExtensionActionCollectorRole(t *testing.T) {
	sqlAction := &odigosv1.Action{
		Spec: odigosv1.ActionSpec{
			DbQueryTemplatization: &actions.DbQueryTemplatizationConfig{},
		},
	}
	assert.Equal(t, odigosv1.CollectorsGroupRoleClusterGateway, ConfigExtensionActionCollectorRole(sqlAction, false))
	assert.Equal(t, odigosv1.CollectorsGroupRoleNodeCollector, ConfigExtensionActionCollectorRole(sqlAction, true))

	piiAction := &odigosv1.Action{
		Spec: odigosv1.ActionSpec{
			PiiMasking: &actions.PiiMaskingConfig{},
		},
	}
	assert.Equal(t, odigosv1.CollectorsGroupRoleClusterGateway, ConfigExtensionActionCollectorRole(piiAction, true))
}

func TestNodeCollectorsGroupSpanMetricsEnabled(t *testing.T) {
	assert.False(t, NodeCollectorsGroupSpanMetricsEnabled(nil))
	assert.False(t, NodeCollectorsGroupSpanMetricsEnabled(&odigosv1.CollectorsGroup{}))
	assert.False(t, NodeCollectorsGroupSpanMetricsEnabled(&odigosv1.CollectorsGroup{
		Spec: odigosv1.CollectorsGroupSpec{Metrics: &odigosv1.CollectorsGroupMetricsCollectionSettings{}},
	}))
	assert.True(t, NodeCollectorsGroupSpanMetricsEnabled(&odigosv1.CollectorsGroup{
		Spec: odigosv1.CollectorsGroupSpec{
			Metrics: &odigosv1.CollectorsGroupMetricsCollectionSettings{
				SpanMetrics: &common.MetricsSourceSpanMetricsConfiguration{},
			},
		},
	}))
}
