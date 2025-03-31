package centralodigos

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type centralUIResourceManager struct {
	client      *kube.Client
	ns          string
	managerOpts resourcemanager.ManagerOpts
}

func NewCentralUIResourceManager(client *kube.Client, ns string, managerOpts resourcemanager.ManagerOpts) resourcemanager.ResourceManager {
	return &centralUIResourceManager{client: client, ns: ns, managerOpts: managerOpts}
}

func (m *centralUIResourceManager) Name() string { return k8sconsts.CentralUIAppName }

func (m *centralUIResourceManager) InstallFromScratch(ctx context.Context) error {
	return m.client.ApplyResources(ctx, 1, []kube.Object{
		NewCentralUIDeployment(m.ns),
	}, m.managerOpts)
}

func NewCentralUIDeployment(ns string) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.CentralUIDeploymentName,
			Namespace: ns,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrint32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					k8sconsts.CentralUILabelAppKey: k8sconsts.CentralUILabelAppValue,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						k8sconsts.CentralUILabelAppKey: k8sconsts.CentralUILabelAppValue,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  k8sconsts.CentralUI,
							Image: "staging-registry.odigos.io/central-ui:dev",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(k8sconsts.CentralCPURequest),
									corev1.ResourceMemory: resource.MustParse(k8sconsts.CentralMemoryRequest),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(k8sconsts.CentralCPULimit),
									corev1.ResourceMemory: resource.MustParse(k8sconsts.CentralMemoryLimit),
								},
							},
						},
					},
				},
			},
		},
	}
}

func ptrint32(i int32) *int32 {
	return &i
}
