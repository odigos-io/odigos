package utils

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/tj/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var instrumentationConfigResource = schema.GroupResource{
	Group:    "odigos.io",
	Resource: "instrumentationconfigs",
}

// K8SUpdateErrorHandler is the shared tail of ~20 reconcilers. Classifying an error into the wrong
// bucket either drops work silently or fills the logs with stack traces for an expected race, so
// each bucket is pinned by the result it produces and not only by the absence of an error.
func TestK8SUpdateErrorHandler(t *testing.T) {
	updateConflict := apierrors.NewConflict(instrumentationConfigResource, "deployment-frontend",
		errors.New("the object has been modified"))
	notFound := apierrors.NewNotFound(instrumentationConfigResource, "deployment-frontend")

	t.Run("a conflict is retried without surfacing an error", func(t *testing.T) {
		result, err := K8SUpdateErrorHandler(updateConflict)

		assert.NoError(t, err)
		assert.True(t, result.Requeue)
		assert.Equal(t, time.Duration(0), result.RequeueAfter)
	})

	t.Run("a deleted object ends the reconcile", func(t *testing.T) {
		result, err := K8SUpdateErrorHandler(notFound)

		assert.NoError(t, err)
		assert.False(t, result.Requeue)
	})

	t.Run("another agent already running is an expected outcome", func(t *testing.T) {
		result, err := K8SUpdateErrorHandler(ErrOtherAgentRun)

		assert.NoError(t, err)
		assert.False(t, result.Requeue)
	})

	t.Run("another agent already running is recognised through a wrapped error", func(t *testing.T) {
		result, err := K8SUpdateErrorHandler(fmt.Errorf("injecting device: %w", ErrOtherAgentRun))

		assert.NoError(t, err)
		assert.False(t, result.Requeue)
	})

	t.Run("any other error is returned for the controller runtime to log and back off", func(t *testing.T) {
		updateErr := errors.New("the webhook rejected the update")

		result, err := K8SUpdateErrorHandler(updateErr)

		assert.Equal(t, updateErr, err)
		assert.False(t, result.Requeue)
	})

	t.Run("an unrelated api error is not mistaken for a conflict", func(t *testing.T) {
		invalid := apierrors.NewInvalid(schema.GroupKind{Group: "odigos.io", Kind: "InstrumentationConfig"},
			"deployment-frontend", nil)

		result, err := K8SUpdateErrorHandler(invalid)

		assert.Error(t, err)
		assert.False(t, result.Requeue)
	})

	t.Run("a status error carrying the conflict reason is retried", func(t *testing.T) {
		statusErr := &apierrors.StatusError{ErrStatus: metav1.Status{
			Status: metav1.StatusFailure,
			Code:   409,
			Reason: metav1.StatusReasonConflict,
		}}

		result, err := K8SUpdateErrorHandler(statusErr)

		assert.NoError(t, err)
		assert.True(t, result.Requeue)
	})

	t.Run("no error at all is not a requeue", func(t *testing.T) {
		result, err := K8SUpdateErrorHandler(nil)

		assert.NoError(t, err)
		assert.False(t, result.Requeue)
	})
}

func TestK8SNoEffectiveConfigErrorHandler(t *testing.T) {
	t.Run("a missing effective config is retried after a delay", func(t *testing.T) {
		result, err := K8SNoEffectiveConfigErrorHandler(ErrOdigosEffectiveConfigNotFound)

		assert.NoError(t, err)
		assert.True(t, result.Requeue)
		// The delay itself is a tuning knob; that there is one is not, because an immediate
		// requeue would hot loop until the scheduler renders the config.
		assert.True(t, result.RequeueAfter > 0)
	})

	t.Run("any other error is returned as is", func(t *testing.T) {
		readErr := errors.New("the apiserver is unavailable")

		result, err := K8SNoEffectiveConfigErrorHandler(readErr)

		assert.Equal(t, readErr, err)
		assert.False(t, result.Requeue)
		assert.Equal(t, time.Duration(0), result.RequeueAfter)
	})

	t.Run("no error at all is not a requeue", func(t *testing.T) {
		result, err := K8SNoEffectiveConfigErrorHandler(nil)

		assert.NoError(t, err)
		assert.False(t, result.Requeue)
	})
}
