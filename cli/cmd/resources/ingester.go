package resources

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/containers"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ingesterResourceManager struct {
	client      *kube.Client
	ns          string
	config      *common.OdigosConfiguration
	managerOpts resourcemanager.ManagerOpts
}

func (u *ingesterResourceManager) Name() string {
	return "Ingester"
}

func NewIngesterDeployment(ns string, version string, imagePrefix string, imageName string, nodeSelector map[string]string) *appsv1.Deployment {
	if nodeSelector == nil {
		nodeSelector = make(map[string]string)
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.IngesterDeploymentName,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/name": k8sconsts.IngesterAppLabelValue,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrint32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": k8sconsts.IngesterAppLabelValue,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": k8sconsts.IngesterAppLabelValue,
					},
					Annotations: map[string]string{
						"kubectl.kubernetes.io/default-container": k8sconsts.IngesterContainerName,
					},
				},
				Spec: corev1.PodSpec{
					NodeSelector: nodeSelector,
					Containers: []corev1.Container{
						{
							Name:  k8sconsts.IngesterContainerName,
							Image: containers.GetImageName(imagePrefix, imageName, version),
							Ports: []corev1.ContainerPort{
								{
									Name:          "otlp",
									ContainerPort: consts.OTLPPort,
								},
								{
									Name:          "api",
									ContainerPort: k8sconsts.IngesterApiPort,
								},
							},
						},
					},
					TerminationGracePeriodSeconds: ptrint64(10),
					ServiceAccountName:            k8sconsts.IngesterServiceAccountName,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: ptrbool(true),
					},
				},
			},
			Strategy:        appsv1.DeploymentStrategy{},
			MinReadySeconds: 0,
		},
	}
}

func NewIngesterServiceAccount(ns string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.IngesterServiceAccountName,
			Namespace: ns,
		},
	}
}

func NewIngesterService(ns string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.IngesterServiceName,
			Namespace: ns,
			Labels: map[string]string{
				"app": k8sconsts.IngesterAppLabelValue,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": k8sconsts.IngesterAppLabelValue,
			},
			Ports: []corev1.ServicePort{
				{
					Name: "otlp",
					Port: consts.OTLPPort,
				},
				{
					Name: "api",
					Port: k8sconsts.IngesterApiPort,
				},
			},
		},
	}
}

func (u *ingesterResourceManager) InstallFromScratch(ctx context.Context) error {
	resources := []kube.Object{
		NewIngesterServiceAccount(u.ns),
		NewIngesterDeployment(u.ns, k8sconsts.JaegerVersion, k8sconsts.JaegerPrefix, k8sconsts.JaegerImage, u.config.NodeSelector),
		NewIngesterService(u.ns),
	}
	return u.client.ApplyResources(ctx, u.config.ConfigVersion, resources, u.managerOpts)
}

func NewIngesterResourceManager(client *kube.Client, ns string, config *common.OdigosConfiguration, managerOpts resourcemanager.ManagerOpts) resourcemanager.ResourceManager {
	return &ingesterResourceManager{
		client:      client,
		ns:          ns,
		config:      config,
		managerOpts: managerOpts,
	}
}
