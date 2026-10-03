package actions

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
)

const testActionsNamespace = "odigos-system"

func newK8sAttributesAction(name string, signals []common.ObservabilitySignal) *odigosv1.Action {
	return &odigosv1.Action{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "Action",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testActionsNamespace,
		},
		Spec: odigosv1.ActionSpec{
			ActionName: name,
			Signals:    signals,
			K8sAttributes: &actionv1.K8sAttributesConfig{
				CollectContainerAttributes: true,
			},
		},
	}
}

func newActionsFakeClient(t *testing.T, objs ...client.Object) client.Client {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, odigosv1.AddToScheme(scheme))
	require.NoError(t, actionv1.AddToScheme(scheme))

	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

// The unified "odigos-k8sattributes" Processor is written with server side apply and
// spec.signals is an atomic list, so an unstable order rewrites the Processor spec on
// every reconcile. The Action controller Owns() that Processor, so an unstable order is
// a self-sustaining reconcile loop, not a cosmetic diff.
func TestConvertActionToProcessor_K8sAttributesSignalsAreDeterministic(t *testing.T) {
	action := newK8sAttributesAction("multi-signal", []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
		common.MetricsObservabilitySignal,
		common.LogsObservabilitySignal,
	})
	c := newActionsFakeClient(t, action)

	first, err := convertActionToProcessor(context.Background(), c, action)
	require.NoError(t, err)
	require.Len(t, first.Spec.Signals, 3)

	// a single pass can match by luck - 3 signals means a 1/6 chance per reconcile.
	for i := 0; i < 100; i++ {
		got, err := convertActionToProcessor(context.Background(), c, action)
		require.NoError(t, err)
		require.Equal(t, first.Spec.Signals, got.Spec.Signals,
			"spec.signals order changed between reconciles (iteration %d)", i)
	}
}

// Signals are aggregated across every enabled K8sAttributes action in the namespace, so
// the union has to be stable too.
func TestConvertActionToProcessor_K8sAttributesSignalsAggregatedAcrossActions(t *testing.T) {
	traces := newK8sAttributesAction("traces-only", []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
	})
	logsAndMetrics := newK8sAttributesAction("logs-and-metrics", []common.ObservabilitySignal{
		common.LogsObservabilitySignal,
		common.MetricsObservabilitySignal,
	})
	c := newActionsFakeClient(t, traces, logsAndMetrics)

	expected := []common.ObservabilitySignal{
		common.LogsObservabilitySignal,
		common.MetricsObservabilitySignal,
		common.TracesObservabilitySignal,
	}

	for i := 0; i < 100; i++ {
		got, err := convertActionToProcessor(context.Background(), c, traces)
		require.NoError(t, err)
		require.Equal(t, expected, got.Spec.Signals, "iteration %d", i)
	}
}

// A single signal can never expose the ordering bug, so assert the sort did not disturb it.
func TestConvertActionToProcessor_K8sAttributesSingleSignal(t *testing.T) {
	action := newK8sAttributesAction("single-signal", []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
	})
	c := newActionsFakeClient(t, action)

	got, err := convertActionToProcessor(context.Background(), c, action)
	require.NoError(t, err)
	require.Equal(t, []common.ObservabilitySignal{common.TracesObservabilitySignal}, got.Spec.Signals)
}

// A disabled action contributes no signals, which is what keeps the shared processor out
// of every pipeline. Locking this in so the sort cannot mask a regression there.
func TestConvertActionToProcessor_K8sAttributesDisabledActionContributesNoSignals(t *testing.T) {
	action := newK8sAttributesAction("disabled", []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
		common.LogsObservabilitySignal,
	})
	action.Spec.Disabled = true
	c := newActionsFakeClient(t, action)

	got, err := convertActionToProcessor(context.Background(), c, action)
	require.NoError(t, err)
	require.Empty(t, got.Spec.Signals)
}
