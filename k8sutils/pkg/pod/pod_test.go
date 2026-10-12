package pod

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func boolPtr(b bool) *bool {
	return &b
}

// containerStatus builds a status for a container that is Ready, with `started`
// controlling the Started field (nil means the kubelet never reported it).
func containerStatus(name string, ready bool, started *bool) corev1.ContainerStatus {
	return corev1.ContainerStatus{
		Name:    name,
		Ready:   ready,
		Started: started,
	}
}

func podWithStatuses(phase corev1.PodPhase, statuses ...corev1.ContainerStatus) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default"},
		Status: corev1.PodStatus{
			Phase:             phase,
			ContainerStatuses: statuses,
		},
	}
}

func controllerOwnerRef(kind string) metav1.OwnerReference {
	return metav1.OwnerReference{Kind: kind, Name: "owner", Controller: boolPtr(true)}
}

func TestIsCronJobPod(t *testing.T) {
	tests := []struct {
		name      string
		ownerRefs []metav1.OwnerReference
		want      bool
	}{
		{
			name:      "no owner references",
			ownerRefs: nil,
			want:      false,
		},
		{
			name:      "controlled by a Job",
			ownerRefs: []metav1.OwnerReference{controllerOwnerRef("Job")},
			want:      true,
		},
		{
			name:      "controlled by a CronJob",
			ownerRefs: []metav1.OwnerReference{controllerOwnerRef("CronJob")},
			want:      true,
		},
		{
			name:      "controlled by a ReplicaSet",
			ownerRefs: []metav1.OwnerReference{controllerOwnerRef("ReplicaSet")},
			want:      false,
		},
		{
			// a non-controller Job reference does not make the pod a Job pod,
			// so the kubelet is still expected to report Started.
			name:      "Job owner reference that is not the controller",
			ownerRefs: []metav1.OwnerReference{{Kind: "Job", Name: "owner", Controller: boolPtr(false)}},
			want:      false,
		},
		{
			name:      "Job owner reference with no Controller field",
			ownerRefs: []metav1.OwnerReference{{Kind: "Job", Name: "owner"}},
			want:      false,
		},
		{
			name: "Job controller listed after a non-controlling reference",
			ownerRefs: []metav1.OwnerReference{
				{Kind: "ReplicaSet", Name: "other", Controller: boolPtr(false)},
				controllerOwnerRef("Job"),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := podWithStatuses(corev1.PodRunning)
			pod.OwnerReferences = tt.ownerRefs
			assert.Equal(t, tt.want, isCronJobPod(pod))
		})
	}
}

func TestAllContainersReady(t *testing.T) {
	tests := []struct {
		name      string
		phase     corev1.PodPhase
		ownerRefs []metav1.OwnerReference
		statuses  []corev1.ContainerStatus
		want      bool
	}{
		{
			name:     "no container statuses reported yet",
			phase:    corev1.PodRunning,
			statuses: nil,
			want:     false,
		},
		{
			name:     "pending pod with a ready container status",
			phase:    corev1.PodPending,
			statuses: []corev1.ContainerStatus{containerStatus("app", true, boolPtr(true))},
			want:     false,
		},
		{
			name:     "succeeded pod with a ready container status",
			phase:    corev1.PodSucceeded,
			statuses: []corev1.ContainerStatus{containerStatus("app", true, boolPtr(true))},
			want:     false,
		},
		{
			name:     "running pod with a ready and started container",
			phase:    corev1.PodRunning,
			statuses: []corev1.ContainerStatus{containerStatus("app", true, boolPtr(true))},
			want:     true,
		},
		{
			name:     "running pod with a container that is not ready",
			phase:    corev1.PodRunning,
			statuses: []corev1.ContainerStatus{containerStatus("app", false, boolPtr(true))},
			want:     false,
		},
		{
			name:     "running pod with a ready container that never started",
			phase:    corev1.PodRunning,
			statuses: []corev1.ContainerStatus{containerStatus("app", true, nil)},
			want:     false,
		},
		{
			name:     "running pod with a ready container whose Started is false",
			phase:    corev1.PodRunning,
			statuses: []corev1.ContainerStatus{containerStatus("app", true, boolPtr(false))},
			want:     false,
		},
		{
			name:  "all of several containers ready and started",
			phase: corev1.PodRunning,
			statuses: []corev1.ContainerStatus{
				containerStatus("app", true, boolPtr(true)),
				containerStatus("sidecar", true, boolPtr(true)),
			},
			want: true,
		},
		{
			// the loop must inspect every container, not just the first one.
			name:  "second container is not ready",
			phase: corev1.PodRunning,
			statuses: []corev1.ContainerStatus{
				containerStatus("app", true, boolPtr(true)),
				containerStatus("sidecar", false, boolPtr(true)),
			},
			want: false,
		},
		{
			name:  "second container never started",
			phase: corev1.PodRunning,
			statuses: []corev1.ContainerStatus{
				containerStatus("app", true, boolPtr(true)),
				containerStatus("sidecar", true, nil),
			},
			want: false,
		},
		{
			// Job/CronJob pods never get Started reported, so the Started check
			// is skipped for them or they would never be considered ready.
			name:      "job pod with a ready container that never started",
			phase:     corev1.PodRunning,
			ownerRefs: []metav1.OwnerReference{controllerOwnerRef("Job")},
			statuses:  []corev1.ContainerStatus{containerStatus("app", true, nil)},
			want:      true,
		},
		{
			name:      "cronjob pod with a ready container whose Started is false",
			phase:     corev1.PodRunning,
			ownerRefs: []metav1.OwnerReference{controllerOwnerRef("CronJob")},
			statuses:  []corev1.ContainerStatus{containerStatus("app", true, boolPtr(false))},
			want:      true,
		},
		{
			// readiness is still required for job pods.
			name:      "job pod with a container that is not ready",
			phase:     corev1.PodRunning,
			ownerRefs: []metav1.OwnerReference{controllerOwnerRef("Job")},
			statuses:  []corev1.ContainerStatus{containerStatus("app", false, nil)},
			want:      false,
		},
		{
			name:      "job pod that is not running",
			phase:     corev1.PodPending,
			ownerRefs: []metav1.OwnerReference{controllerOwnerRef("Job")},
			statuses:  []corev1.ContainerStatus{containerStatus("app", true, nil)},
			want:      false,
		},
		{
			// a deployment pod gets no Started exemption.
			name:      "replicaset pod with a ready container that never started",
			phase:     corev1.PodRunning,
			ownerRefs: []metav1.OwnerReference{controllerOwnerRef("ReplicaSet")},
			statuses:  []corev1.ContainerStatus{containerStatus("app", true, nil)},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := podWithStatuses(tt.phase, tt.statuses...)
			pod.OwnerReferences = tt.ownerRefs
			assert.Equal(t, tt.want, AllContainersReady(pod))
		})
	}
}

func TestIsPodDeleting(t *testing.T) {
	now := metav1.Now()

	tests := []struct {
		name              string
		deletionTimestamp *metav1.Time
		want              bool
	}{
		{
			name:              "no deletion timestamp",
			deletionTimestamp: nil,
			want:              false,
		},
		{
			name:              "zero deletion timestamp",
			deletionTimestamp: &metav1.Time{},
			want:              false,
		},
		{
			name:              "deletion timestamp set",
			deletionTimestamp: &now,
			want:              true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := podWithStatuses(corev1.PodRunning)
			pod.DeletionTimestamp = tt.deletionTimestamp
			assert.Equal(t, tt.want, IsPodDeleting(pod))
		})
	}
}
