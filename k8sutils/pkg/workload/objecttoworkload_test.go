package workload_test

import (
	"testing"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	"github.com/tj/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// The fixtures below give every replica counter on the status a distinct value so
// that an adapter reading the wrong field returns a wrong number rather than the
// expected one. AvailableReplicas gates whether odigos considers a workload
// rolled out, so reading the wrong counter stalls or prematurely completes a rollout.

func podTemplateSpec(containerName string) corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: containerName, Image: "some-image"}},
		},
	}
}

func TestObjectToWorkloadDeployment(t *testing.T) {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "my-deployment", Namespace: "default"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "my-deployment"}},
			Template: podTemplateSpec("deployment-container"),
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            10,
			UpdatedReplicas:     11,
			ReadyReplicas:       12,
			AvailableReplicas:   13,
			UnavailableReplicas: 14,
		},
	}

	w, err := workload.ObjectToWorkload(dep)
	assert.NoError(t, err)

	typed, ok := w.(*workload.DeploymentWorkload)
	assert.True(t, ok)
	assert.Same(t, dep, typed.Deployment)

	assert.Equal(t, int32(13), w.AvailableReplicas())
	assert.Same(t, &dep.Spec.Template.Spec, w.PodSpec())
	assert.Same(t, dep.Spec.Selector, w.LabelSelector())
}

func TestObjectToWorkloadDaemonSet(t *testing.T) {
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "my-daemonset", Namespace: "default"},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "my-daemonset"}},
			Template: podTemplateSpec("daemonset-container"),
		},
		Status: appsv1.DaemonSetStatus{
			CurrentNumberScheduled: 20,
			NumberMisscheduled:     21,
			DesiredNumberScheduled: 22,
			NumberReady:            23,
			UpdatedNumberScheduled: 24,
			NumberAvailable:        25,
			NumberUnavailable:      26,
		},
	}

	w, err := workload.ObjectToWorkload(ds)
	assert.NoError(t, err)

	typed, ok := w.(*workload.DaemonSetWorkload)
	assert.True(t, ok)
	assert.Same(t, ds, typed.DaemonSet)

	// a daemonset has no replicas, the number of ready pods is the equivalent.
	assert.Equal(t, int32(23), w.AvailableReplicas())
	assert.Same(t, &ds.Spec.Template.Spec, w.PodSpec())
	assert.Same(t, ds.Spec.Selector, w.LabelSelector())
}

func TestObjectToWorkloadStatefulSet(t *testing.T) {
	ss := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "my-statefulset", Namespace: "default"},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "my-statefulset"}},
			Template: podTemplateSpec("statefulset-container"),
		},
		Status: appsv1.StatefulSetStatus{
			Replicas:          30,
			ReadyReplicas:     31,
			CurrentReplicas:   32,
			UpdatedReplicas:   33,
			AvailableReplicas: 34,
		},
	}

	w, err := workload.ObjectToWorkload(ss)
	assert.NoError(t, err)

	typed, ok := w.(*workload.StatefulSetWorkload)
	assert.True(t, ok)
	assert.Same(t, ss, typed.StatefulSet)

	assert.Equal(t, int32(31), w.AvailableReplicas())
	assert.Same(t, &ss.Spec.Template.Spec, w.PodSpec())
	assert.Same(t, ss.Spec.Selector, w.LabelSelector())
}

func TestObjectToWorkloadDeploymentConfig(t *testing.T) {
	template := podTemplateSpec("deploymentconfig-container")
	dc := &openshiftappsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "my-deploymentconfig", Namespace: "default"},
		Spec: openshiftappsv1.DeploymentConfigSpec{
			Selector: map[string]string{"app": "my-deploymentconfig"},
			Template: &template,
		},
		Status: openshiftappsv1.DeploymentConfigStatus{
			Replicas:            40,
			UpdatedReplicas:     41,
			AvailableReplicas:   42,
			UnavailableReplicas: 43,
			ReadyReplicas:       44,
		},
	}

	w, err := workload.ObjectToWorkload(dc)
	assert.NoError(t, err)

	typed, ok := w.(*workload.DeploymentConfigWorkload)
	assert.True(t, ok)
	assert.Same(t, dc, typed.DeploymentConfig)

	assert.Equal(t, int32(42), w.AvailableReplicas())
	assert.Same(t, &dc.Spec.Template.Spec, w.PodSpec())

	// a DeploymentConfig selector is a plain label map, so it has to be wrapped
	// into a LabelSelector for callers that select pods with it.
	assert.Equal(t,
		&metav1.LabelSelector{MatchLabels: map[string]string{"app": "my-deploymentconfig"}},
		w.LabelSelector())
}

func TestObjectToWorkloadArgoRollout(t *testing.T) {
	rollout := &argorolloutsv1alpha1.Rollout{
		ObjectMeta: metav1.ObjectMeta{Name: "my-rollout", Namespace: "default"},
		Spec: argorolloutsv1alpha1.RolloutSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "my-rollout"}},
			Template: podTemplateSpec("rollout-container"),
		},
		Status: argorolloutsv1alpha1.RolloutStatus{
			Replicas:          50,
			UpdatedReplicas:   51,
			ReadyReplicas:     52,
			AvailableReplicas: 53,
			HPAReplicas:       54,
		},
	}

	w, err := workload.ObjectToWorkload(rollout)
	assert.NoError(t, err)

	typed, ok := w.(*workload.ArgoRolloutWorkload)
	assert.True(t, ok)
	assert.Same(t, rollout, typed.Rollout)

	assert.Equal(t, int32(53), w.AvailableReplicas())
	assert.Same(t, &rollout.Spec.Template.Spec, w.PodSpec())
	assert.Same(t, rollout.Spec.Selector, w.LabelSelector())
}

func TestObjectToWorkloadCronJob(t *testing.T) {
	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Name: "my-cronjob", Namespace: "default"},
		Spec: batchv1.CronJobSpec{
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{Template: podTemplateSpec("cronjob-container")},
			},
		},
		Status: batchv1.CronJobStatus{
			Active: []corev1.ObjectReference{
				{Name: "my-cronjob-1"},
				{Name: "my-cronjob-2"},
			},
		},
	}

	w, err := workload.ObjectToWorkload(cronJob)
	assert.NoError(t, err)

	typed, ok := w.(*workload.CronJobWorkloadV1)
	assert.True(t, ok)
	assert.Same(t, cronJob, typed.CronJob)

	// a cronjob has no replicas, its currently running jobs are the equivalent.
	assert.Equal(t, int32(2), w.AvailableReplicas())
	assert.Same(t, &cronJob.Spec.JobTemplate.Spec.Template.Spec, w.PodSpec())

	// a cronjob's pods cannot be selected by labels from the cronjob itself.
	assert.Nil(t, w.LabelSelector())
}

func TestObjectToWorkloadCronJobWithNoActiveJobs(t *testing.T) {
	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Name: "my-cronjob", Namespace: "default"},
	}

	w, err := workload.ObjectToWorkload(cronJob)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), w.AvailableReplicas())
}

func TestObjectToWorkloadStaticPod(t *testing.T) {
	pod := staticPodPod("kube-apiserver-node-1", "node-1")
	pod.Spec.Containers = []corev1.Container{{Name: "kube-apiserver", Image: "some-image"}}
	pod.Status.Phase = corev1.PodRunning

	w, err := workload.ObjectToWorkload(pod)
	assert.NoError(t, err)

	typed, ok := w.(*workload.StaticPodWorkload)
	assert.True(t, ok)
	assert.Same(t, pod, typed.Pod)

	assert.Equal(t, int32(1), w.AvailableReplicas())
	assert.Same(t, &pod.Spec, w.PodSpec())

	// a static pod is its own workload, there is nothing to select.
	assert.Nil(t, w.LabelSelector())
}

// A static pod stands in for a single replica, so it only counts as available
// while it is actually running.
func TestObjectToWorkloadStaticPodAvailableReplicasFollowsThePhase(t *testing.T) {
	tests := []struct {
		phase corev1.PodPhase
		want  int32
	}{
		{phase: corev1.PodRunning, want: 1},
		{phase: corev1.PodPending, want: 0},
		{phase: corev1.PodSucceeded, want: 0},
		{phase: corev1.PodFailed, want: 0},
		{phase: corev1.PodUnknown, want: 0},
		{phase: corev1.PodPhase(""), want: 0},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			pod := staticPodPod("kube-apiserver-node-1", "node-1")
			pod.Status.Phase = tt.phase

			w, err := workload.ObjectToWorkload(pod)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, w.AvailableReplicas())
		})
	}
}

// Only static pods can be workloads. An ordinary pod belongs to a controller that
// is the real workload, so accepting it here would instrument the same
// application twice under two different identities.
func TestObjectToWorkloadRejectsNonStaticPods(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "my-deployment-abc123-xyz",
			Namespace:       "default",
			OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "my-deployment-abc123"}},
		},
	}

	w, err := workload.ObjectToWorkload(pod)
	assert.Nil(t, w)
	assert.EqualError(t, err, "currently not supporting standalone pods which are not static as workloads")
}

func TestObjectToWorkloadRejectsUnsupportedKinds(t *testing.T) {
	tests := []struct {
		name   string
		object client.Object
	}{
		{name: "config map", object: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "cm"}}},
		{name: "replicaset", object: &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "rs"}}},
		{
			// a Namespace is a valid odigos workload *kind* for source level
			// configuration, but it has no pod spec so it is not a Workload.
			name:   "namespace",
			object: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := workload.ObjectToWorkload(tt.object)
			assert.Nil(t, w)
			assert.EqualError(t, err, "unknown kind")
		})
	}
}
