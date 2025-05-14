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
	"k8s.io/apimachinery/pkg/util/intstr"
)

type centralBackendResourceManager struct {
	client        *kube.Client
	ns            string
	odigosVersion string
	managerOpts   resourcemanager.ManagerOpts
}

func NewCentralBackendResourceManager(client *kube.Client, ns string, odigosVersion string, managerOpts resourcemanager.ManagerOpts) resourcemanager.ResourceManager {
	return &centralBackendResourceManager{client: client, ns: ns, odigosVersion: odigosVersion, managerOpts: managerOpts}
}

func (m *centralBackendResourceManager) Name() string { return k8sconsts.CentralBackendName }

func (m *centralBackendResourceManager) InstallFromScratch(ctx context.Context) error {
	return m.client.ApplyResources(ctx, 1, []kube.Object{
		NewCentralBackendDeployment(m.ns, k8sconsts.OdigosImagePrefix, m.managerOpts.ImageReferences.CentralBackendImage, m.odigosVersion),
		NewCentralBackendService(m.ns),
	}, m.managerOpts)
}

func NewCentralBackendDeployment(ns, imagePrefix, imageName, version string) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.CentralBackendName,
			Namespace: ns,
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
							Image: containers.GetImageName(imagePrefix, imageName, version),
							Env: []corev1.EnvVar{
								{
									Name: k8sconsts.OdigosOnpremTokenEnvName,
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: k8sconsts.OdigosCentralSecretName,
											},
											Key: k8sconsts.OdigosOnpremTokenSecretKey,
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

func NewCentralBackendService(ns string) *corev1.Service {
	portInt, err := strconv.Atoi(k8sconsts.CentralBackendPort)
	if err != nil {
		portInt = 8081
	}
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.CentralBackendName,
			Namespace: ns,
			Labels: map[string]string{
				"app": k8sconsts.CentralBackendAppName,
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": k8sconsts.CentralBackendAppName,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       int32(portInt),
					TargetPort: intstrFromInt(portInt),
				},
			},
		},
	}
}

func intstrFromInt(val int) intstr.IntOrString {
	return intstr.IntOrString{Type: intstr.Int, IntVal: int32(val)}
}
