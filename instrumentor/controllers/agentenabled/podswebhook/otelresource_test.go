package podswebhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

func TestGetWorkloadKindAttributeKey(t *testing.T) {
	// the resource attribute key of the workload is what associates the telemetry with the
	// workload in the UI, so a wrong key silently detaches the data from its source.
	tests := []struct {
		kind        k8sconsts.WorkloadKind
		expectedKey string
	}{
		{k8sconsts.WorkloadKindDeployment, "k8s.deployment.name"},
		{k8sconsts.WorkloadKindStatefulSet, "k8s.statefulset.name"},
		{k8sconsts.WorkloadKindDaemonSet, "k8s.daemonset.name"},
		{k8sconsts.WorkloadKindCronJob, "k8s.cronjob.name"},
		{k8sconsts.WorkloadKindJob, "k8s.job.name"},
		// OpenShift DeploymentConfig deliberately reuses the deployment key
		{k8sconsts.WorkloadKindDeploymentConfig, "k8s.deployment.name"},
		{k8sconsts.WorkloadKindArgoRollout, "k8s.argoproj.rollout.name"},
	}

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			assert.Equal(t, tt.expectedKey, string(getWorkloadKindAttributeKey(tt.kind)))
		})
	}
}

func TestGetOwnerReferenceAttributes(t *testing.T) {
	t.Run("job owner adds name and uid", func(t *testing.T) {
		attrs := getOwnerReferenceAttributes([]metav1.OwnerReference{
			{Kind: "Job", Name: "my-cronjob-28123", UID: "job-uid"},
		})

		assert.Equal(t, "k8s.job.name=my-cronjob-28123,k8s.job.uid=job-uid", getResourceAttributesEnvVarValue(attrs))
	})

	t.Run("replicaset owner adds name and uid", func(t *testing.T) {
		attrs := getOwnerReferenceAttributes([]metav1.OwnerReference{
			{Kind: "ReplicaSet", Name: "my-deployment-7d4c8b5f9b", UID: "rs-uid"},
		})

		assert.Equal(t, "k8s.replicaset.name=my-deployment-7d4c8b5f9b,k8s.replicaset.uid=rs-uid",
			getResourceAttributesEnvVarValue(attrs))
	})

	t.Run("other owner kinds are ignored", func(t *testing.T) {
		attrs := getOwnerReferenceAttributes([]metav1.OwnerReference{
			{Kind: "StatefulSet", Name: "my-statefulset", UID: "sts-uid"},
			{Kind: "Node", Name: "my-node", UID: "node-uid"},
		})

		assert.Empty(t, attrs)
	})

	t.Run("no owner references", func(t *testing.T) {
		assert.Empty(t, getOwnerReferenceAttributes(nil))
	})
}

func TestInjectOtelResourceAndServiceNameEnvVars(t *testing.T) {
	pw := k8sconsts.PodWorkload{Name: "my-deployment", Kind: k8sconsts.WorkloadKindDeployment, Namespace: "my-ns"}

	t.Run("sets the service name and the full resource attributes", func(t *testing.T) {
		container := &corev1.Container{Name: "my-container"}

		InjectOtelResourceAndServiceNameEnvVars(GetEnvVarNamesSet(container), container, "nodejs-community", pw,
			"my-service", []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "my-deployment-7d4c8b5f9b", UID: "rs-uid"}})

		require.Len(t, container.Env, 2)
		assert.Equal(t, corev1.EnvVar{Name: "OTEL_SERVICE_NAME", Value: "my-service"}, container.Env[0])
		assert.Equal(t, corev1.EnvVar{
			Name: "OTEL_RESOURCE_ATTRIBUTES",
			Value: "k8s.pod.name=$(ODIGOS_POD_NAME)," +
				"k8s.container.name=my-container," +
				"k8s.namespace.name=my-ns," +
				"k8s.deployment.name=my-deployment," +
				"odigos.workload.kind=Deployment," +
				"odigos.workload.name=my-deployment," +
				"k8s.replicaset.name=my-deployment-7d4c8b5f9b," +
				"k8s.replicaset.uid=rs-uid",
		}, container.Env[1])
	})

	t.Run("a user defined service name is not overwritten", func(t *testing.T) {
		container := &corev1.Container{
			Name: "my-container",
			Env:  []corev1.EnvVar{{Name: "OTEL_SERVICE_NAME", Value: "user-service"}},
		}

		InjectOtelResourceAndServiceNameEnvVars(GetEnvVarNamesSet(container), container, "nodejs-community", pw, "my-service", nil)

		assert.Equal(t, "user-service", container.Env[0].Value)
	})

	t.Run("user defined resource attributes are passed to odigos own env var instead", func(t *testing.T) {
		container := &corev1.Container{
			Name: "my-container",
			Env:  []corev1.EnvVar{{Name: "OTEL_RESOURCE_ATTRIBUTES", Value: "user.attr=value"}},
		}

		InjectOtelResourceAndServiceNameEnvVars(GetEnvVarNamesSet(container), container, "nodejs-community", pw, "my-service", nil)

		assert.Equal(t, "user.attr=value", container.Env[0].Value)
		odigosAttrs := findEnvVar(t, container, "ODIGOS_RESOURCE_ATTRIBUTES")
		assert.Contains(t, odigosAttrs.Value, "k8s.deployment.name=my-deployment")
		assert.Len(t, container.Env, 3)
	})
}

func findEnvVar(t *testing.T, container *corev1.Container, name string) corev1.EnvVar {
	t.Helper()
	envVar := findEnvVarOrNil(container, name)
	require.NotNil(t, envVar, "env var %s not found in container %s", name, container.Name)
	return *envVar
}

func findEnvVarOrNil(container *corev1.Container, name string) *corev1.EnvVar {
	for i := range container.Env {
		if container.Env[i].Name == name {
			return &container.Env[i]
		}
	}
	return nil
}
