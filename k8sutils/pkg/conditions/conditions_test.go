package conditions

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

const (
	conditionsTestNamespace = "odigos-system"
	conditionsTestName      = "source-deployment-frontend"
	transformedConditionMsg = "collector configuration applied"
)

func conditionsTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, odigosv1.AddToScheme(scheme))
	return scheme
}

func conditionsTestSource() *odigosv1.Source {
	return &odigosv1.Source{
		ObjectMeta: metav1.ObjectMeta{
			Name:       conditionsTestName,
			Namespace:  conditionsTestNamespace,
			Generation: 7,
		},
	}
}

// statusUpdateCounter builds a client that records how many status updates reached the apiserver,
// which is the only way to see that an unchanged condition is not written back.
func statusUpdateCounter(t *testing.T, obj client.Object, updateErr error) (client.Client, *int) {
	t.Helper()
	updates := 0
	c := fake.NewClientBuilder().
		WithScheme(conditionsTestScheme(t)).
		WithObjects(obj).
		WithStatusSubresource(obj).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(ctx context.Context, cl client.Client, subResourceName string,
				o client.Object, opts ...client.SubResourceUpdateOption) error {
				updates++
				if updateErr != nil {
					return updateErr
				}
				return cl.Status().Update(ctx, o, opts...)
			},
		}).
		Build()
	return c, &updates
}

func storedConditions(t *testing.T, c client.Client, obj *odigosv1.Source) []metav1.Condition {
	t.Helper()
	stored := &odigosv1.Source{}
	require.NoError(t, c.Get(context.Background(),
		client.ObjectKey{Namespace: obj.Namespace, Name: obj.Name}, stored))
	return stored.Status.Conditions
}

func TestUpdateStatusConditionsWritesANewCondition(t *testing.T) {
	source := conditionsTestSource()
	c, updates := statusUpdateCounter(t, source, nil)

	err := UpdateStatusConditions(context.Background(), c, source, &source.Status.Conditions,
		metav1.ConditionTrue, "Transformed", "ConfigApplied", transformedConditionMsg)

	require.NoError(t, err)
	assert.Equal(t, 1, *updates)

	stored := storedConditions(t, c, source)
	require.Len(t, stored, 1)
	assert.Equal(t, "Transformed", stored[0].Type)
	assert.Equal(t, metav1.ConditionTrue, stored[0].Status)
	assert.Equal(t, "ConfigApplied", stored[0].Reason)
	assert.Equal(t, transformedConditionMsg, stored[0].Message)
	assert.Equal(t, int64(7), stored[0].ObservedGeneration)
}

// Controllers call this on every reconcile. Writing an unchanged condition back would trigger
// another watch event and spin the reconcile loop.
func TestUpdateStatusConditionsSkipsTheApiserverWhenNothingChanged(t *testing.T) {
	source := conditionsTestSource()
	c, updates := statusUpdateCounter(t, source, nil)

	for range 3 {
		require.NoError(t, UpdateStatusConditions(context.Background(), c, source, &source.Status.Conditions,
			metav1.ConditionTrue, "Transformed", "ConfigApplied", transformedConditionMsg))
	}

	assert.Equal(t, 1, *updates)
	assert.Len(t, storedConditions(t, c, source), 1)
}

func TestUpdateStatusConditionsWritesEveryChangedField(t *testing.T) {
	tests := []struct {
		name    string
		status  metav1.ConditionStatus
		reason  string
		message string
	}{
		{
			name:    "a changed status",
			status:  metav1.ConditionFalse,
			reason:  "ConfigApplied",
			message: transformedConditionMsg,
		},
		{
			name:    "a changed reason",
			status:  metav1.ConditionTrue,
			reason:  "ConfigRejected",
			message: transformedConditionMsg,
		},
		{
			name:    "a changed message",
			status:  metav1.ConditionTrue,
			reason:  "ConfigApplied",
			message: "collector configuration rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := conditionsTestSource()
			c, updates := statusUpdateCounter(t, source, nil)

			require.NoError(t, UpdateStatusConditions(context.Background(), c, source, &source.Status.Conditions,
				metav1.ConditionTrue, "Transformed", "ConfigApplied", transformedConditionMsg))
			require.NoError(t, UpdateStatusConditions(context.Background(), c, source, &source.Status.Conditions,
				tt.status, "Transformed", tt.reason, tt.message))

			assert.Equal(t, 2, *updates)
			stored := storedConditions(t, c, source)
			require.Len(t, stored, 1)
			assert.Equal(t, tt.status, stored[0].Status)
			assert.Equal(t, tt.reason, stored[0].Reason)
			assert.Equal(t, tt.message, stored[0].Message)
		})
	}
}

// The observed generation is what tells a reader whether the condition describes the spec they are
// looking at, so a bumped generation has to be written back even if nothing else moved.
func TestUpdateStatusConditionsTracksTheObservedGeneration(t *testing.T) {
	source := conditionsTestSource()
	c, updates := statusUpdateCounter(t, source, nil)

	require.NoError(t, UpdateStatusConditions(context.Background(), c, source, &source.Status.Conditions,
		metav1.ConditionTrue, "Transformed", "ConfigApplied", transformedConditionMsg))

	source.Generation = 8
	require.NoError(t, UpdateStatusConditions(context.Background(), c, source, &source.Status.Conditions,
		metav1.ConditionTrue, "Transformed", "ConfigApplied", transformedConditionMsg))

	assert.Equal(t, 2, *updates)
	stored := storedConditions(t, c, source)
	require.Len(t, stored, 1)
	assert.Equal(t, int64(8), stored[0].ObservedGeneration)
}

func TestUpdateStatusConditionsLeavesOtherConditionTypesAlone(t *testing.T) {
	source := conditionsTestSource()
	c, _ := statusUpdateCounter(t, source, nil)

	require.NoError(t, UpdateStatusConditions(context.Background(), c, source, &source.Status.Conditions,
		metav1.ConditionTrue, "Transformed", "ConfigApplied", transformedConditionMsg))
	require.NoError(t, UpdateStatusConditions(context.Background(), c, source, &source.Status.Conditions,
		metav1.ConditionFalse, "MarkedForInstrumentation", "NoSource", "no source CR marks this workload"))

	stored := storedConditions(t, c, source)
	require.Len(t, stored, 2)
	byType := map[string]metav1.Condition{}
	for _, condition := range stored {
		byType[condition.Type] = condition
	}
	assert.Equal(t, metav1.ConditionTrue, byType["Transformed"].Status)
	assert.Equal(t, "ConfigApplied", byType["Transformed"].Reason)
	assert.Equal(t, metav1.ConditionFalse, byType["MarkedForInstrumentation"].Status)
	assert.Equal(t, "NoSource", byType["MarkedForInstrumentation"].Reason)
}

func TestUpdateStatusConditionsPropagatesTheUpdateError(t *testing.T) {
	source := conditionsTestSource()
	updateErr := errors.New("conflict on the status subresource")
	c, updates := statusUpdateCounter(t, source, updateErr)

	err := UpdateStatusConditions(context.Background(), c, source, &source.Status.Conditions,
		metav1.ConditionTrue, "Transformed", "ConfigApplied", transformedConditionMsg)

	require.ErrorIs(t, err, updateErr)
	assert.Equal(t, 1, *updates)
}
