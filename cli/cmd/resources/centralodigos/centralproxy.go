package centralodigos

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type centralProxyResourceManager struct {
	client      *kube.Client
	ns          string
	managerOpts resourcemanager.ManagerOpts
}

func NewCentralProxyResourceManager(client *kube.Client, ns string, managerOpts resourcemanager.ManagerOpts) resourcemanager.ResourceManager {
	return &centralProxyResourceManager{client: client, ns: ns, managerOpts: managerOpts}
}

func (m *centralProxyResourceManager) Name() string { return k8sconsts.CentralProxyAppName }

func (m *centralProxyResourceManager) InstallFromScratch(ctx context.Context) error {
	resources := []kube.Object{
		NewCentralProxyServiceAccount(m.ns),
		NewCentralProxyRoleBinding(m.ns),
		NewCentralProxyRole(m.ns),
		NewCentralProxyDeployment(m.ns),
	}

	return m.client.ApplyResources(ctx, 1, resources, m.managerOpts)
}

func NewCentralProxyDeployment(ns string) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.CentralProxyDeploymentName,
			Namespace: ns,
			Labels: map[string]string{
				k8sconsts.CentralProxyLabelAppNameKey: k8sconsts.CentralProxyLabelAppNameValue,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrint32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					k8sconsts.CentralProxyLabelAppNameKey: k8sconsts.CentralProxyLabelAppNameValue,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						k8sconsts.CentralProxyLabelAppNameKey: k8sconsts.CentralProxyLabelAppNameValue,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: k8sconsts.CentralProxyServiceAccountName,
					Containers: []corev1.Container{
						{
							Name:  k8sconsts.CentralProxyContainerName,
							Image: k8sconsts.CentralProxyContainerImage,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: k8sconsts.CentralProxyContainerPort,
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

func NewCentralProxyServiceAccount(ns string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.CentralProxyServiceAccountName,
			Namespace: ns,
		},
	}
}

func NewCentralProxyRole(ns string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: k8sconsts.CentralProxyRBACAPIGroup + "/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.CentralProxyRoleName,
			Namespace: ns,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get"},
				APIGroups: []string{""},
				Resources: []string{k8sconsts.CentralProxyConfigMapResource},
			},
		},
	}
}

func NewCentralProxyRoleBinding(ns string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: k8sconsts.CentralProxyRBACAPIGroup + "/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.CentralProxyRoleBindingName,
			Namespace: ns,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      k8sconsts.CentralProxyServiceAccountName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     k8sconsts.CentralProxyRoleName,
			APIGroup: k8sconsts.CentralProxyRBACAPIGroup,
		},
	}
}
