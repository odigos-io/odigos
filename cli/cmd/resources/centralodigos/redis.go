package centralodigos

import (
	"context"
	"strconv"

	"github.com/odigos-io/odigos/api/k8sconsts"
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

func (m *redisResourceManager) Name() string { return k8sconsts.RedisResourceManagerName }

func (m *redisResourceManager) InstallFromScratch(ctx context.Context) error {
	resources := []kube.Object{
		NewRedisDeployment(m.ns),
		NewRedisService(m.ns),
	}

	return m.client.ApplyResources(ctx, 1, resources, m.managerOpts)
}

func NewRedisDeployment(ns string) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.RedisDeploymentName,
			Namespace: ns,
			Labels:    map[string]string{"app": k8sconsts.RedisAppName},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrint32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": k8sconsts.RedisAppName},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": k8sconsts.RedisAppName},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    k8sconsts.RedisContainerName,
							Image:   k8sconsts.RedisImage,
							Command: []string{k8sconsts.RedisCommand},
							Args:    []string{"--port", strconv.Itoa(k8sconsts.RedisPort)},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: k8sconsts.RedisPort,
									Name:          k8sconsts.RedisPortName,
								},
							},
						},
					},
				},
			},
		},
	}
}

func NewRedisService(ns string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.RedisServiceName,
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": k8sconsts.RedisAppName},
			Ports: []corev1.ServicePort{
				{
					Name:       k8sconsts.RedisPortName,
					Port:       k8sconsts.RedisPort,
					TargetPort: intstr.FromInt(k8sconsts.RedisPort),
				},
			},
		},
	}
}
