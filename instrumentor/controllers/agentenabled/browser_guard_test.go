package agentenabled

import (
	"testing"

	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func deploymentWorkloadWithContainers(containers []corev1.Container) workload.Workload {
	return &workload.DeploymentWorkload{
		Deployment: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{Containers: containers},
				},
			},
		},
	}
}

func TestContainerHasTCPPort(t *testing.T) {
	tests := []struct {
		name          string
		workloadObj   workload.Workload
		containerName string
		want          bool
	}{
		{
			name:          "nil workload is treated as having a port (don't block here)",
			workloadObj:   nil,
			containerName: "web",
			want:          true,
		},
		{
			name: "container with an implicit-TCP port",
			workloadObj: deploymentWorkloadWithContainers([]corev1.Container{
				{Name: "web", Ports: []corev1.ContainerPort{{ContainerPort: 8080}}},
			}),
			containerName: "web",
			want:          true,
		},
		{
			name: "container with an explicit TCP port",
			workloadObj: deploymentWorkloadWithContainers([]corev1.Container{
				{Name: "web", Ports: []corev1.ContainerPort{{ContainerPort: 8080, Protocol: corev1.ProtocolTCP}}},
			}),
			containerName: "web",
			want:          true,
		},
		{
			name: "container with no ports",
			workloadObj: deploymentWorkloadWithContainers([]corev1.Container{
				{Name: "web"},
			}),
			containerName: "web",
			want:          false,
		},
		{
			name: "container with only a UDP port",
			workloadObj: deploymentWorkloadWithContainers([]corev1.Container{
				{Name: "web", Ports: []corev1.ContainerPort{{ContainerPort: 53, Protocol: corev1.ProtocolUDP}}},
			}),
			containerName: "web",
			want:          false,
		},
		{
			name: "container not found in pod template is treated as having a port",
			workloadObj: deploymentWorkloadWithContainers([]corev1.Container{
				{Name: "other", Ports: []corev1.ContainerPort{{ContainerPort: 8080}}},
			}),
			containerName: "web",
			want:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, containerHasTCPPort(tt.workloadObj, tt.containerName))
		})
	}
}
