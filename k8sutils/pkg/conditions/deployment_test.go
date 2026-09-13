package conditions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func deploymentStatusWith(conditions ...appsv1.DeploymentCondition) appsv1.DeploymentStatus {
	return appsv1.DeploymentStatus{
		ObservedGeneration: 7,
		Conditions:         conditions,
	}
}

func deploymentCondition(condType appsv1.DeploymentConditionType, reason string) appsv1.DeploymentCondition {
	return appsv1.DeploymentCondition{
		Type:   condType,
		Status: corev1.ConditionTrue,
		Reason: reason,
	}
}

func TestGetDeploymentCondition(t *testing.T) {
	progressing := deploymentCondition(appsv1.DeploymentProgressing, "NewReplicaSetAvailable")
	available := deploymentCondition(appsv1.DeploymentAvailable, "MinimumReplicasAvailable")

	t.Run("returns the requested condition", func(t *testing.T) {
		status := deploymentStatusWith(available, progressing)

		found := GetDeploymentCondition(status, appsv1.DeploymentProgressing)

		require.NotNil(t, found)
		assert.Equal(t, progressing, *found)
	})

	t.Run("returns nil when the condition type is absent", func(t *testing.T) {
		status := deploymentStatusWith(available)

		assert.Nil(t, GetDeploymentCondition(status, appsv1.DeploymentProgressing))
	})

	t.Run("returns nil when there are no conditions at all", func(t *testing.T) {
		assert.Nil(t, GetDeploymentCondition(deploymentStatusWith(), appsv1.DeploymentProgressing))
	})

	t.Run("returns the first condition of the requested type", func(t *testing.T) {
		first := deploymentCondition(appsv1.DeploymentProgressing, "ReplicaSetUpdated")
		second := deploymentCondition(appsv1.DeploymentProgressing, "ProgressDeadlineExceeded")
		status := deploymentStatusWith(first, second)

		found := GetDeploymentCondition(status, appsv1.DeploymentProgressing)

		require.NotNil(t, found)
		assert.Equal(t, "ReplicaSetUpdated", found.Reason)
	})

	// Callers receive a pointer, and writing through it must not edit the Deployment status they
	// were handed.
	t.Run("returns a copy rather than a pointer into the status", func(t *testing.T) {
		status := deploymentStatusWith(progressing)

		found := GetDeploymentCondition(status, appsv1.DeploymentProgressing)
		require.NotNil(t, found)
		found.Reason = "MutatedByCaller"

		assert.Equal(t, "NewReplicaSetAvailable", status.Conditions[0].Reason)
	})
}

// The reason is not exposed as a constant by the apps API; it is the literal string the deployment
// controller writes when a rollout blows through progressDeadlineSeconds, and rollout detection
// compares against it.
func TestTimedOutReasonMatchesTheDeploymentControllerLiteral(t *testing.T) {
	assert.Equal(t, "ProgressDeadlineExceeded", TimedOutReason)
}
