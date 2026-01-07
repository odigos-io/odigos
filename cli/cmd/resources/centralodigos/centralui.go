package centralodigos

import (
	"context"
	"strconv"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/containers"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type centralUIResourceManager struct {
	client        *kube.Client
	ns            string
	odigosVersion string
	managerOpts   resourcemanager.ManagerOpts
}

func NewCentralUIResourceManager(client *kube.Client, ns string, managerOpts resourcemanager.ManagerOpts, odigosVersion string) resourcemanager.ResourceManager {
	return &centralUIResourceManager{client: client, ns: ns, managerOpts: managerOpts, odigosVersion: odigosVersion}
}

func (m *centralUIResourceManager) Name() string { return k8sconsts.CentralUIAppName }

func (m *centralUIResourceManager) InstallFromScratch(ctx context.Context) error {
	return m.client.ApplyResources(ctx, 1, []kube.Object{
		NewCentralUIDeployment(m.ns, k8sconsts.OdigosImagePrefix, m.managerOpts.ImageReferences.CentralUIImage, m.odigosVersion, m.managerOpts.ImagePullSecrets),
		NewCentralUIService(m.ns),
	}, m.managerOpts)
}

func NewCentralUIDeployment(ns, imagePrefix, imageName, version string, imagePullSecrets []string) *appsv1.Deployment {
	var pullRefs []corev1.LocalObjectReference
	for _, n := range imagePullSecrets {
		if n != "" {
			pullRefs = append(pullRefs, corev1.LocalObjectReference{Name: n})
		}
	}
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
					"app": k8sconsts.CentralUIAppName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": k8sconsts.CentralUIAppName,
					},
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: pullRefs,
					Containers: []corev1.Container{
						{
							Name:  k8sconsts.CentralUI,
							Image: containers.GetImageName(imagePrefix, imageName, version),
							Env: []corev1.EnvVar{
								{
									Name: "CURRENT_NS",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
							},
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

func NewCentralUIService(ns string) *corev1.Service {
	portInt, err := strconv.Atoi(k8sconsts.CentralUIPort)
	if err != nil {
		portInt = 3000
	}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.CentralUIServiceName,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/name": k8sconsts.CentralUIAppName,
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "ui",
					Port:       int32(portInt),
					TargetPort: intstrFromInt(portInt),
				},
			},
			Selector: map[string]string{
				"app": k8sconsts.CentralUIAppName,
			},
		},
	}
}
