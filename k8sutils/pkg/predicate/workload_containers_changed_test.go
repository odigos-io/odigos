package predicate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func deploymentWithContainers(containerNames ...string) *appsv1.Deployment {
	containers := make([]corev1.Container, 0, len(containerNames))
	for _, name := range containerNames {
		containers = append(containers, corev1.Container{Name: name, Image: "test"})
	}

	return &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: containers},
			},
		},
	}
}

func cronJobWithContainers(containerNames ...string) *batchv1.CronJob {
	containers := make([]corev1.Container, 0, len(containerNames))
	for _, name := range containerNames {
		containers = append(containers, corev1.Container{Name: name, Image: "test"})
	}

	return &batchv1.CronJob{
		Spec: batchv1.CronJobSpec{
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: containers},
					},
				},
			},
		},
	}
}

func TestWorkloadContainersChangedPredicate(t *testing.T) {
	predicate := WorkloadContainersChangedPredicate{}

	testCases := []struct {
		name     string
		old      client.Object
		new      client.Object
		expected bool
	}{
		{
			name:     "container renamed",
			old:      deploymentWithContainers("app"),
			new:      deploymentWithContainers("web"),
			expected: true,
		},
		{
			name:     "container added",
			old:      deploymentWithContainers("app"),
			new:      deploymentWithContainers("app", "sidecar"),
			expected: true,
		},
		{
			name:     "container removed",
			old:      deploymentWithContainers("app", "sidecar"),
			new:      deploymentWithContainers("app"),
			expected: true,
		},
		{
			name:     "containers reordered",
			old:      deploymentWithContainers("app", "sidecar"),
			new:      deploymentWithContainers("sidecar", "app"),
			expected: true,
		},
		{
			name:     "same containers",
			old:      deploymentWithContainers("app", "sidecar"),
			new:      deploymentWithContainers("app", "sidecar"),
			expected: false,
		},
		{
			name: "same containers with a new image",
			old:  deploymentWithContainers("app"),
			new: func() client.Object {
				deployment := deploymentWithContainers("app")
				deployment.Spec.Template.Spec.Containers[0].Image = "other"
				return deployment
			}(),
			expected: false,
		},
		{
			name:     "cronjob container renamed",
			old:      cronJobWithContainers("app"),
			new:      cronJobWithContainers("web"),
			expected: true,
		},
		{
			name:     "object which is not a workload",
			old:      &corev1.Namespace{},
			new:      &corev1.Namespace{},
			expected: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			updated := predicate.Update(event.UpdateEvent{ObjectOld: testCase.old, ObjectNew: testCase.new})
			assert.Equal(t, testCase.expected, updated)
		})
	}
}

func TestWorkloadContainersChangedPredicateOnlyHandlesUpdates(t *testing.T) {
	predicate := WorkloadContainersChangedPredicate{}

	assert.False(t, predicate.Create(event.CreateEvent{Object: deploymentWithContainers("app")}))
	assert.False(t, predicate.Delete(event.DeleteEvent{Object: deploymentWithContainers("app")}))
	assert.False(t, predicate.Generic(event.GenericEvent{Object: deploymentWithContainers("app")}))
	assert.False(t, predicate.Update(event.UpdateEvent{}))
}
