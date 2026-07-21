package actions

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigosactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	actionsapi "github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/common/consts"
)

func TestReconcilePiiMaskingStopsAfterSharedProcessorSync(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, odigosv1.AddToScheme(scheme))

	action := &odigosv1.Action{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pii-masking-regression",
			Namespace: consts.DefaultOdigosNamespace,
		},
		Spec: odigosv1.ActionSpec{
			ActionName: "PII masking regression",
			Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
			PiiMasking: &odigosactions.PiiMaskingConfig{
				PiiMaskingConfig: actionsapi.PiiMaskingConfig{
					PiiCategories: []actionsapi.PiiCategory{actionsapi.EmailMasking},
				},
			},
		},
	}
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&odigosv1.Action{}).
		WithObjects(action).
		Build()
	reconciler := &ActionReconciler{Client: fakeClient}

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: client.ObjectKeyFromObject(action),
	})
	require.NoError(t, err)

	sharedProcessor := &odigosv1.Processor{}
	require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKey{
		Name:      consts.PiiMaskingProcessorName,
		Namespace: consts.DefaultOdigosNamespace,
	}, sharedProcessor))

	reconciledAction := &odigosv1.Action{}
	require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKeyFromObject(action), reconciledAction))
	condition := meta.FindStatusCondition(reconciledAction.Status.Conditions, odigosv1.ActionTransformedToProcessorType)
	require.NotNil(t, condition)
	assert.Equal(t, metav1.ConditionTrue, condition.Status)
	assert.Equal(t, string(odigosv1.ActionTransformedToProcessorReasonProcessorCreated), condition.Reason)
}
