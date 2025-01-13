package resources

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/odigos-io/odigos/cli/pkg/containers"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"

	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
)

const (
	UIImage              = "keyval/odigos-ui"
	UIServiceName        = "ui"
	UIDeploymentName     = "odigos-ui"
	UIAppLabelValue      = "odigos-ui"
	UIContainerName      = "ui"
	UIServiceAccountName = "odigos-ui"
)

type uiResourceManager struct {
	client        *kube.Client
	ns            string
	config        *common.OdigosConfiguration
	odigosVersion string
}

func (u *uiResourceManager) Name() string {
	return "UI"
}

func NewUIDeployment(ns string, version string, imagePrefix string) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      UIDeploymentName,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/name": UIAppLabelValue,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrint32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": UIAppLabelValue,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": UIAppLabelValue,
					},
					Annotations: map[string]string{
						"kubectl.kubernetes.io/default-container": UIContainerName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  UIContainerName,
							Image: containers.GetImageName(imagePrefix, UIImage, version),
							Args: []string{
								"--namespace=$(CURRENT_NS)",
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
							Ports: []corev1.ContainerPort{
								{
									Name:          "ui",
									ContainerPort: 3000,
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("500m"),
									"memory": *resource.NewQuantity(536870912, resource.BinarySI),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("10m"),
									"memory": *resource.NewQuantity(67108864, resource.BinarySI),
								},
							},
							SecurityContext: &corev1.SecurityContext{},
						},
					},
					TerminationGracePeriodSeconds: ptrint64(10),
					ServiceAccountName:            UIServiceAccountName,
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

func NewUIServiceAccount(ns string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      UIServiceAccountName,
			Namespace: ns,
		},
	}
}

func NewUIRole(ns string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "odigos-ui",
			Namespace: ns,
		},
		Rules: []rbacv1.PolicyRule{
			{ // Needed to read odigos-config configmap for settings
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list"},
			},
			{ // Needed for secret values in destinations
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "create", "patch", "update"},
			},
			{ // Needed for CRUD on instr. rule and destinations
				APIGroups: []string{"odigos.io"},
				Resources: []string{"instrumentationrules", "destinations"},
				Verbs:     []string{"get", "list", "create", "patch", "update", "delete"},
			},
			{ // Needed to notify UI about changes with destinations
				APIGroups: []string{"odigos.io"},
				Resources: []string{"destinations"},
				Verbs:     []string{"watch"},
			},
			{ // Needed to read Odigos entities
				APIGroups: []string{"odigos.io"},
				Resources: []string{"collectorsgroups"},
				Verbs:     []string{"get", "list"},
			},
			{ // Needed for CRUD on pipeline actions
				APIGroups: []string{"actions.odigos.io"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list", "create", "patch", "update", "delete"},
			},
		},
	}
}

func NewUIRoleBinding(ns string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "odigos-ui",
			Namespace: ns,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      UIServiceAccountName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     "odigos-ui",
			APIGroup: "rbac.authorization.k8s.io",
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
			Name: "odigos-ui",
		},
		Rules: []rbacv1.PolicyRule{
			{ // Needed to get and instrument namespaces
				APIGroups: []string{""},
				Resources: []string{"namespaces"},
				Verbs:     []string{"get", "list", "patch"},
			},
			{ // Needed to get and instrument sources
				APIGroups: []string{"apps"},
				Resources: []string{"deployments", "statefulsets", "daemonsets"},
				Verbs:     []string{"get", "list", "patch", "update"},
			},
			{ // Needed for "Describe Source" and for "Describe Odigos"
				APIGroups: []string{"apps"},
				Resources: []string{"replicasets"},
				Verbs:     []string{"get", "list"},
			},
			{ // Need "services" for "Potential Destinations"
				APIGroups: []string{""},
				Resources: []string{"services"},
				Verbs:     []string{"get", "list"},
			},
			{ // Need "pods" for "Describe Source"
				// for collector metrics - watch and list collectors pods
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Needed to read Odigos entities,
				// "watch" to notify UI about changes with sources
				APIGroups: []string{"odigos.io"},
				Resources: []string{"instrumentationconfigs", "instrumentationinstances"},
				Verbs:     []string{"get", "list", "watch"},
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
			Name: "odigos-ui",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      UIServiceAccountName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     "odigos-ui",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

func NewUIService(ns string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      UIServiceName,
			Namespace: ns,
			Labels: map[string]string{
				"app": UIAppLabelValue,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": UIAppLabelValue,
			},
			Ports: []corev1.ServicePort{
				{
					Name: k8sconsts.OdigosUiServiceName,
					Port: k8sconsts.OdigosUiServicePort,
				},
				{
					Name: "otlp",
					Port: consts.OTLPPort,
				},
			},
		},
	}
}

func (u *uiResourceManager) InstallFromScratch(ctx context.Context) error {
	resources := []kube.Object{
		NewUIServiceAccount(u.ns),
		NewUIRole(u.ns),
		NewUIRoleBinding(u.ns),
		NewUIClusterRole(),
		NewUIClusterRoleBinding(u.ns),
		NewUIDeployment(u.ns, u.odigosVersion, u.config.ImagePrefix),
		NewUIService(u.ns),
	}
	return u.client.ApplyResources(ctx, u.config.ConfigVersion, resources)
}

func NewUIResourceManager(client *kube.Client, ns string, config *common.OdigosConfiguration, odigosVersion string) resourcemanager.ResourceManager {
	return &uiResourceManager{
		client:        client,
		ns:            ns,
		config:        config,
		odigosVersion: odigosVersion,
	}
}
