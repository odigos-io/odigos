package workload_test

import (
	"testing"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/tj/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func podInNamespace(name string, namespace string, owners ...metav1.OwnerReference) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			OwnerReferences: owners,
		},
	}
}

// Resolving a pod to the workload that owns it is what ties telemetry back to an
// application, so each owner kind has to land on the right odigos workload kind.
func TestGetWorkloadNameAndKind(t *testing.T) {
	tests := []struct {
		name      string
		ownerName string
		ownerKind string
		wantName  string
		wantKind  k8sconsts.WorkloadKind
	}{
		{
			// a ReplicationController is what a DeploymentConfig rolls out on OpenShift.
			name:      "replication controller resolves to its deployment config",
			ownerName: "my-deploymentconfig-7",
			ownerKind: "ReplicationController",
			wantName:  "my-deploymentconfig",
			wantKind:  k8sconsts.WorkloadKindDeploymentConfig,
		},
		{
			// a Job pod is attributed to the CronJob that scheduled it, since odigos
			// instruments the schedule rather than each individual run.
			name:      "job resolves to its cronjob",
			ownerName: "my-cronjob-28934000",
			ownerKind: "Job",
			wantName:  "my-cronjob",
			wantKind:  k8sconsts.WorkloadKindCronJob,
		},
		{
			name:      "daemonset owns its pods directly",
			ownerName: "my-daemonset",
			ownerKind: "DaemonSet",
			wantName:  "my-daemonset",
			wantKind:  k8sconsts.WorkloadKindDaemonSet,
		},
		{
			name:      "statefulset owns its pods directly",
			ownerName: "my-statefulset",
			ownerKind: "StatefulSet",
			wantName:  "my-statefulset",
			wantKind:  k8sconsts.WorkloadKindStatefulSet,
		},
		{
			// only the generated suffix is stripped, so a hyphenated workload name survives.
			name:      "hyphenated name keeps everything but the generated suffix",
			ownerName: "my-long-app-name-7d4c8b5f9b",
			ownerKind: "ReplicaSet",
			wantName:  "my-long-app-name",
			wantKind:  k8sconsts.WorkloadKindDeployment,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := podInNamespace("some-pod", "default")

			name, kind, err := workload.GetWorkloadNameAndKind(tt.ownerName, tt.ownerKind, pod)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantKind, kind)
		})
	}
}

// Owner names that should carry a generated suffix but do not cannot be attributed
// to a workload, and the error names the kind that was expected.
func TestGetWorkloadNameAndKindRejectsNamesWithoutASuffix(t *testing.T) {
	pod := podInNamespace("some-pod", "default")

	_, _, err := workload.GetWorkloadNameAndKind("nohyphen", "ReplicationController", pod)
	assert.EqualError(t, err, "DeploymentConfig name 'nohyphen' does not contain a hyphen")

	_, _, err = workload.GetWorkloadNameAndKind("nohyphen", "Job", pod)
	assert.EqualError(t, err, "CronJob name 'nohyphen' does not contain a hyphen")

	_, _, err = workload.GetWorkloadNameAndKind("nohyphen", "ReplicaSet", pod)
	assert.EqualError(t, err, "Deployment name 'nohyphen' does not contain a hyphen")
}

func TestGetWorkloadNameAndKindForNodeOwnedPods(t *testing.T) {
	t.Run("static pod is its own workload", func(t *testing.T) {
		pod := staticPodPod("kube-apiserver-node-1", "node-1")

		name, kind, err := workload.GetWorkloadNameAndKind("node-1", "Node", pod)
		assert.NoError(t, err)
		// the full pod name is used here, node name suffix included.
		assert.Equal(t, "kube-apiserver-node-1", name)
		assert.Equal(t, k8sconsts.WorkloadKindStaticPod, kind)
	})

	t.Run("node owned pod that is not static", func(t *testing.T) {
		pod := staticPodPod("kube-apiserver-node-1", "node-1")
		pod.Annotations[configSourceAnnotation] = "api"

		_, _, err := workload.GetWorkloadNameAndKind("node-1", "Node", pod)
		assert.EqualError(t, err, "node owned pod which is not static, currently not supported as a workload")
	})
}

func TestGetWorkloadNameAndKindForUnsupportedKind(t *testing.T) {
	pod := podInNamespace("some-pod", "default")

	_, _, err := workload.GetWorkloadNameAndKind("some-owner", "SomeCustomResource", pod)
	assert.Equal(t, workload.ErrKindNotSupported, err)
	assert.True(t, workload.IsErrorKindNotSupported(err))
}

func TestPodWorkloadObject(t *testing.T) {
	t.Run("pod owned by a replicaset", func(t *testing.T) {
		pod := podInNamespace("my-deployment-7d4c8b5f9b-abcde", "my-namespace",
			metav1.OwnerReference{Kind: "ReplicaSet", Name: "my-deployment-7d4c8b5f9b"})

		pw, err := workload.PodWorkloadObject(pod)
		assert.NoError(t, err)
		assert.Equal(t, &k8sconsts.PodWorkload{
			Name:      "my-deployment",
			Kind:      k8sconsts.WorkloadKindDeployment,
			Namespace: "my-namespace",
		}, pw)
	})

	t.Run("pod owned by a replicaset of an argo rollout", func(t *testing.T) {
		pod := podInNamespace("my-rollout-abc123-xyz", "my-namespace",
			metav1.OwnerReference{Kind: "ReplicaSet", Name: "my-rollout-abc123"})
		pod.Labels = map[string]string{argorolloutsv1alpha1.DefaultRolloutUniqueLabelKey: "abc123"}

		pw, err := workload.PodWorkloadObject(pod)
		assert.NoError(t, err)
		assert.Equal(t, &k8sconsts.PodWorkload{
			Name:      "my-rollout",
			Kind:      k8sconsts.WorkloadKindArgoRollout,
			Namespace: "my-namespace",
		}, pw)
	})

	// A bare pod is legal in kubernetes and is simply not a workload, which is
	// distinct from a failure to resolve one.
	t.Run("pod with no owner references", func(t *testing.T) {
		pod := podInNamespace("standalone-pod", "my-namespace")

		pw, err := workload.PodWorkloadObject(pod)
		assert.NoError(t, err)
		assert.Nil(t, pw)
	})

	// Pods can carry owner references for kinds odigos knows nothing about
	// alongside their controller, and those must be skipped rather than abort
	// the resolution.
	t.Run("unsupported owner kind listed before the controller", func(t *testing.T) {
		pod := podInNamespace("my-app-abcde", "my-namespace",
			metav1.OwnerReference{Kind: "SomeCustomResource", Name: "some-owner"},
			metav1.OwnerReference{Kind: "StatefulSet", Name: "my-statefulset"})

		pw, err := workload.PodWorkloadObject(pod)
		assert.NoError(t, err)
		assert.Equal(t, &k8sconsts.PodWorkload{
			Name:      "my-statefulset",
			Kind:      k8sconsts.WorkloadKindStatefulSet,
			Namespace: "my-namespace",
		}, pw)
	})

	t.Run("only unsupported owner kinds", func(t *testing.T) {
		pod := podInNamespace("my-app-abcde", "my-namespace",
			metav1.OwnerReference{Kind: "SomeCustomResource", Name: "some-owner"},
			metav1.OwnerReference{Kind: "AnotherCustomResource", Name: "another-owner"})

		pw, err := workload.PodWorkloadObject(pod)
		assert.NoError(t, err)
		assert.Nil(t, pw)
	})

	// An error that is not "kind not supported" means the pod does have a
	// recognised controller that could not be resolved, which the caller must see.
	t.Run("owner name without the expected suffix", func(t *testing.T) {
		pod := podInNamespace("my-app-abcde", "my-namespace",
			metav1.OwnerReference{Kind: "ReplicaSet", Name: "nohyphen"})

		pw, err := workload.PodWorkloadObject(pod)
		assert.Nil(t, pw)
		assert.EqualError(t, err, "Deployment name 'nohyphen' does not contain a hyphen")
	})

	t.Run("node owned pod that is not static", func(t *testing.T) {
		pod := staticPodPod("kube-apiserver-node-1", "kube-system")
		pod.Annotations[configSourceAnnotation] = "api"

		pw, err := workload.PodWorkloadObject(pod)
		assert.Nil(t, pw)
		assert.EqualError(t, err, "node owned pod which is not static, currently not supported as a workload")
	})
}

// The OrError variant is for callers that cannot proceed without a workload, so
// an unowned pod becomes an error instead of a nil workload.
func TestPodWorkloadObjectOrError(t *testing.T) {
	t.Run("pod owned by a daemonset", func(t *testing.T) {
		pod := podInNamespace("my-daemonset-abcde", "my-namespace",
			metav1.OwnerReference{Kind: "DaemonSet", Name: "my-daemonset"})

		pw, err := workload.PodWorkloadObjectOrError(pod)
		assert.NoError(t, err)
		assert.Equal(t, &k8sconsts.PodWorkload{
			Name:      "my-daemonset",
			Kind:      k8sconsts.WorkloadKindDaemonSet,
			Namespace: "my-namespace",
		}, pw)
	})

	t.Run("pod with no owner references", func(t *testing.T) {
		pod := podInNamespace("standalone-pod", "my-namespace")

		pw, err := workload.PodWorkloadObjectOrError(pod)
		assert.Nil(t, pw)
		assert.EqualError(t, err, "workload not found for pod my-namespace/standalone-pod")
	})

	t.Run("resolution failure is passed through", func(t *testing.T) {
		pod := podInNamespace("my-app-abcde", "my-namespace",
			metav1.OwnerReference{Kind: "ReplicaSet", Name: "nohyphen"})

		pw, err := workload.PodWorkloadObjectOrError(pod)
		assert.Nil(t, pw)
		assert.EqualError(t, err, "Deployment name 'nohyphen' does not contain a hyphen")
	})
}
