package centralodigos

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type centralBackendResourceManager struct {
	client      *kube.Client
	ns          string
	managerOpts resourcemanager.ManagerOpts
}

func NewCentralBackendResourceManager(client *kube.Client, ns string, managerOpts resourcemanager.ManagerOpts) resourcemanager.ResourceManager {
	return &centralBackendResourceManager{client: client, ns: ns, managerOpts: managerOpts}
}

func (m *centralBackendResourceManager) Name() string { return k8sconsts.CentralBackendName }

func (m *centralBackendResourceManager) InstallFromScratch(ctx context.Context) error {
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.CentralBackendName,
			Namespace: m.ns,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrint32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": k8sconsts.CentralBackendAppName},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": k8sconsts.CentralBackendAppName},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  k8sconsts.CentralBackendAppName,
							Image: "staging-registry.odigos.io/central-backend:dev",
							Env: []corev1.EnvVar{
								{
									Name:  k8sconsts.CentralBackendRedisEnvName,
									Value: k8sconsts.CentralBackendRedisAddr,
								},
							},
						},
					},
				},
			},
		},
	}
	return m.client.ApplyResources(ctx, 1, []kube.Object{deployment}, m.managerOpts)
}
