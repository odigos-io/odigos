package centralodigos

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/containers"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type centralProxyResourceManager struct {
	client        *kube.Client
	ns            string
	odigosVersion string
	config        *common.OdigosConfiguration
	managerOpts   resourcemanager.ManagerOpts
}

func NewCentralProxyResourceManager(client *kube.Client, ns string, config *common.OdigosConfiguration, odigosVersion string, managerOpts resourcemanager.ManagerOpts) resourcemanager.ResourceManager {
	return &centralProxyResourceManager{client: client, ns: ns,
		config:        config,
		odigosVersion: odigosVersion,
		managerOpts:   managerOpts}
}

func (m *centralProxyResourceManager) Name() string { return k8sconsts.CentralProxyAppName }

func (m *centralProxyResourceManager) InstallFromScratch(ctx context.Context) error {
	resources := []kube.Object{
		NewCentralProxyServiceAccount(m.ns),
		NewCentralProxyRoleBinding(m.ns),
		NewCentralProxyRole(m.ns),
		NewCentralProxyClusterRoleBinding(m.ns),
		NewCentralProxyClusterRole(),
		NewCentralProxyDeployment(m.ns, m.odigosVersion, m.config.ImagePrefix, m.managerOpts.ImageReferences.CentralProxyImage, m.config.NodeSelector),
	}

	return m.client.ApplyResources(ctx, m.config.ConfigVersion, resources, m.managerOpts)
}

func NewCentralProxyDeployment(ns string, version string, imagePrefix string, imageName string, nodeSelector map[string]string) *appsv1.Deployment {
	if nodeSelector == nil {
		nodeSelector = make(map[string]string)
	}
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.CentralProxyDeploymentName,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/name": k8sconsts.CentralProxyLabelAppNameValue,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrint32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": k8sconsts.CentralProxyLabelAppNameValue,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": k8sconsts.CentralProxyLabelAppNameValue,
					},
				},
				Spec: corev1.PodSpec{
					NodeSelector:       nodeSelector,
					ServiceAccountName: k8sconsts.CentralProxyServiceAccountName,
					Containers: []corev1.Container{
						{
							Name:  k8sconsts.CentralProxyContainerName,
							Image: containers.GetImageName(imagePrefix, imageName, version),
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
			APIVersion: "rbac.authorization.k8s.io/v1",
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
			APIVersion: "rbac.authorization.k8s.io/v1",
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
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

func NewCentralProxyClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sconsts.CentralProxyRoleName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces"},
				Verbs:     []string{"list", "watch"},
			},
			{
				APIGroups: []string{"apps"},
				Resources: []string{"deployments", "statefulsets", "daemonsets"},
				Verbs:     []string{"list", "watch"},
			},
			{
				APIGroups: []string{"batch"},
				Resources: []string{"cronjobs"},
				Verbs:     []string{"list", "watch"},
			},
			{
				APIGroups: []string{"odigos.io"},
				Resources: []string{"instrumentationconfigs"},
				Verbs:     []string{"list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get"},
			},
		},
	}
}

func NewCentralProxyClusterRoleBinding(ns string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sconsts.CentralProxyRoleBindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      k8sconsts.CentralProxyServiceAccountName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     k8sconsts.CentralProxyRoleName,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

func ptrint32(i int32) *int32 {
	return &i
}
