package resources

import (
	"fmt"

	"github.com/keyval-dev/odigos/cli/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	uiImage = "ghcr.io/keyval-dev/odigos/ui"
)

func NewUIServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "odigos-ui",
			Labels: labels.OdigosSystem,
		},
	}
}

func NewUIRole() *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "odigos-ui",
			Labels: labels.OdigosSystem,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs: []string{
					"create",
					"delete",
					"get",
					"list",
					"patch",
					"update",
					"watch",
				},
				APIGroups: []string{""},
				Resources: []string{
					"secrets",
				},
			},
			{
				Verbs: []string{
					"create",
					"delete",
					"get",
					"list",
					"patch",
					"update",
					"watch",
				},
				APIGroups: []string{
					"odigos.io",
				},
				Resources: []string{
					"collectorsgroups",
				},
			},
			{
				Verbs: []string{
					"update",
				},
				APIGroups: []string{
					"odigos.io",
				},
				Resources: []string{
					"collectorsgroups/finalizers",
				},
			},
			{
				Verbs: []string{
					"get",
					"patch",
					"update",
				},
				APIGroups: []string{
					"odigos.io",
				},
				Resources: []string{
					"collectorsgroups/status",
				},
			},
			{
				Verbs: []string{
					"create",
					"delete",
					"get",
					"list",
					"patch",
					"update",
					"watch",
				},
				APIGroups: []string{
					"odigos.io",
				},
				Resources: []string{
					"destinations",
				},
			},
			{
				Verbs: []string{
					"update",
				},
				APIGroups: []string{
					"odigos.io",
				},
				Resources: []string{
					"destinations/finalizers",
				},
			},
			{
				Verbs: []string{
					"get",
					"patch",
					"update",
				},
				APIGroups: []string{
					"odigos.io",
				},
				Resources: []string{
					"destinations/status",
				},
			},
			{
				Verbs: []string{
					"create",
					"delete",
					"get",
					"list",
					"patch",
					"update",
					"watch",
				},
				APIGroups: []string{
					"odigos.io",
				},
				Resources: []string{
					"odigosconfigurations",
				},
			},
			{
				Verbs: []string{
					"update",
				},
				APIGroups: []string{
					"odigos.io",
				},
				Resources: []string{
					"odigosconfigurations/finalizers",
				},
			},
		},
	}
}

func NewUIRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "odigos-ui",
			Labels: labels.OdigosSystem,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: "odigos-ui",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "odigos-ui",
		},
	}
}

func NewUIClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "odigos-ui",
			Labels: labels.OdigosSystem,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs: []string{
					"create",
					"delete",
					"get",
					"list",
					"patch",
					"update",
					"watch",
				},
				APIGroups: []string{
					"odigos.io",
				},
				Resources: []string{
					"instrumentedapplications",
				},
			},
			{
				Verbs: []string{
					"update",
				},
				APIGroups: []string{
					"odigos.io",
				},
				Resources: []string{
					"instrumentedapplications/finalizers",
				},
			},
			{
				Verbs: []string{
					"get",
					"patch",
					"update",
				},
				APIGroups: []string{
					"odigos.io",
				},
				Resources: []string{
					"instrumentedapplications/status",
				},
			},
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
					"patch",
					"update",
				},
				APIGroups: []string{""},
				Resources: []string{
					"namespaces",
				},
			},
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
					"patch",
					"update",
				},
				APIGroups: []string{"apps"},
				Resources: []string{
					"deployments",
					"daemonsets",
					"statefulsets",
				},
			},
			{
				Verbs: []string{"get"},
				APIGroups: []string{
					"apps",
				},
				Resources: []string{
					"daemonsets/status",
					"deployments/status",
					"statefulsets/status",
				},
			},
		},
	}
}

func NewUIClusterRoleBinding(ns string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "odigos-ui",
			Labels: labels.OdigosSystem,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "odigos-ui",
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "odigos-ui",
		},
	}
}

func NewUIDeployment(version string) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "odigos-ui",
			Labels: map[string]string{
				"app":                       "odigos-ui",
				labels.OdigosSystemLabelKey: labels.OdigosSystemLabelValue,
			},
			Annotations: map[string]string{
				"odigos.io/skip": "true",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrint32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "odigos-ui",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "odigos-ui",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "odigos-ui",
							Image: fmt.Sprintf("%s:%s", uiImage, version),
							Ports: []corev1.ContainerPort{
								{
									Name:          "ui",
									HostPort:      0,
									ContainerPort: 3000,
								},
							},
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
							Resources: corev1.ResourceRequirements{},
						},
					},
					DeprecatedServiceAccount: "odigos-ui",
				},
			},
			Strategy:        appsv1.DeploymentStrategy{},
			MinReadySeconds: 0,
		},
	}
}

func NewUIService() *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "odigos-ui",
			Labels: labels.OdigosSystem,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Protocol: "TCP",
					Port:     3000,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Type(0),
						IntVal: 3000,
					},
					NodePort: 0,
				},
			},
			Selector: map[string]string{
				"app": "odigos-ui",
			},
			HealthCheckNodePort: 0,
		},
	}
}
