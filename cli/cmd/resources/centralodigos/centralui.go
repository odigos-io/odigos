package centralodigos

import (
	"context"

	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

func (m *centralUIResourceManager) Name() string { return "CentralUI" }

func (m *centralUIResourceManager) InstallFromScratch(ctx context.Context) error {
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "central-ui",
			Namespace: m.ns,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrint32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "central-ui"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "central-ui"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "central-ui",
							// Image: containers.GetImageName(m.managerOpts.ImageReferences.ImagePrefix, m.managerOpts.ImageReferences.UIImage, k8sconsts.OdigosCloudProxyVersion),
							Image:           "central-ui:dev",
							ImagePullPolicy: corev1.PullNever,
						},
					},
				},
			},
		},
	}
	return m.client.ApplyResources(ctx, 1, []kube.Object{deployment}, m.managerOpts)
}

func ptrint32(i int32) *int32 {
	return &i
}
