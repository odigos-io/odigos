package centralodigos

import (
	"context"

	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type redisResourceManager struct {
	client      *kube.Client
	ns          string
	managerOpts resourcemanager.ManagerOpts
}

func NewRedisResourceManager(client *kube.Client, ns string, managerOpts resourcemanager.ManagerOpts) resourcemanager.ResourceManager {
	return &redisResourceManager{client: client, ns: ns, managerOpts: managerOpts}
}

func (m *redisResourceManager) Name() string { return "Redis" }

func (m *redisResourceManager) InstallFromScratch(ctx context.Context) error {
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis",
			Namespace: m.ns,
			Labels:    map[string]string{"app": "redis"},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrint32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "redis"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "redis"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "redis",
							Image: "redis:latest",
							Env: []corev1.EnvVar{
								{
									Name:  "REDIS_PORT",
									Value: "6379",
								},
							},
							Command: []string{"redis-server"},
							Args:    []string{"--port", "$(REDIS_PORT)"},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 6379,
									Name:          "redis",
								},
							},
						},
					},
				},
			},
		},
	}

	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis",
			Namespace: m.ns,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "redis"},
			Ports: []corev1.ServicePort{
				{
					Name:       "redis",
					Port:       6379,
					TargetPort: intstr.FromInt(6379),
				},
			},
		},
	}

	return m.client.ApplyResources(ctx, 1, []kube.Object{deployment, service}, m.managerOpts)
}
