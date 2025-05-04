package resources

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/pkg/containers"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
)

type uiResourceManager struct {
	client        *kube.Client
	ns            string
	config        *common.OdigosConfiguration
	odigosVersion string
	readonly      bool
	managerOpts   resourcemanager.ManagerOpts
}

func (u *uiResourceManager) Name() string {
	return "UI"
}

func NewUIDeployment(ns string, version string, imagePrefix string, imageName string, nodeSelector map[string]string) *appsv1.Deployment {
	if nodeSelector == nil {
		nodeSelector = make(map[string]string)
	}
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.UIDeploymentName,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/name": k8sconsts.UIAppLabelValue,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrint32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": k8sconsts.UIAppLabelValue,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": k8sconsts.UIAppLabelValue,
					},
					Annotations: map[string]string{
						"kubectl.kubernetes.io/default-container": k8sconsts.UIContainerName,
					},
				},
				Spec: corev1.PodSpec{
					NodeSelector: nodeSelector,
					Containers: []corev1.Container{
						{
							Name:  k8sconsts.UIContainerName,
							Image: containers.GetImageName(imagePrefix, imageName, version),
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
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "ui-db-storage",
									MountPath: "/data",
								},
							},
						},
					},
					TerminationGracePeriodSeconds: ptrint64(10),
					Volumes: []corev1.Volume{
						{
							Name: "ui-db-storage",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: resource.NewQuantity(50*1024*1024, resource.BinarySI), // 50 MiB in bytes
								},
							},
						},
					},
					ServiceAccountName: k8sconsts.UIServiceAccountName,
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
			Name:      k8sconsts.UIServiceAccountName,
			Namespace: ns,
		},
	}
}

func NewUIRole(ns string, readonly bool) *rbacv1.Role {
	rules := []rbacv1.PolicyRule{}

	if readonly {
		rules = []rbacv1.PolicyRule{
			{ // Needed to read odigos-config configmap for settings
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list"},
			},
			{ // Needed for secret values in destinations
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list"},
			},
			{ // Needed for CRUD on instr. rule and destinations
				APIGroups: []string{"odigos.io"},
				Resources: []string{"instrumentationrules", "destinations"},
				Verbs:     []string{"get", "list"},
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
				Verbs:     []string{"get", "list"},
			},
		}
	} else {
		rules = []rbacv1.PolicyRule{
			{ // Needed to read odigos-config configmap for settings
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list"},
			},
			{ // Needed for secret values in destinations
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "create", "patch", "update", "delete"},
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
		}
	}

	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "odigos-ui",
			Namespace: ns,
		},
		Rules: rules,
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
				Name:      k8sconsts.UIServiceAccountName,
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

func NewUIClusterRole(readonly bool) *rbacv1.ClusterRole {
	rules := []rbacv1.PolicyRule{}

	if readonly {
		rules = []rbacv1.PolicyRule{
			{ // Needed to get and instrument namespaces
				APIGroups: []string{""},
				Resources: []string{"namespaces"},
				Verbs:     []string{"get", "list"},
			},
			{ // Needed to get and instrument sources
				APIGroups: []string{"apps"},
				Resources: []string{"deployments", "statefulsets", "daemonsets"},
				Verbs:     []string{"get", "list"},
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
			{ // Needed to instrument / uninstrument sources
				APIGroups: []string{"odigos.io"},
				Resources: []string{"sources"},
				Verbs:     []string{"get", "list"},
			},
		}
	} else {
		rules = []rbacv1.PolicyRule{
			{ // Needed to get and instrument namespaces
				APIGroups: []string{""},
				Resources: []string{"namespaces"},
				Verbs:     []string{"get", "list"},
			},
			{ // Needed to get workloads
				APIGroups: []string{"apps"},
				Resources: []string{"deployments", "statefulsets", "daemonsets"},
				Verbs:     []string{"get", "list"},
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
			{ // Needed to instrument / uninstrument sources.
				// Patch is needed to update service name.
				APIGroups: []string{"odigos.io"},
				Resources: []string{"sources"},
				Verbs:     []string{"get", "list", "create", "patch", "delete"},
			},
		}
	}

	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "odigos-ui",
		},
		Rules: rules,
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
				Name:      k8sconsts.UIServiceAccountName,
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
			Name:      k8sconsts.UIServiceName,
			Namespace: ns,
			Labels: map[string]string{
				"app": k8sconsts.UIAppLabelValue,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": k8sconsts.UIAppLabelValue,
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
		NewUIRole(u.ns, u.readonly),
		NewUIRoleBinding(u.ns),
		NewUIClusterRole(u.readonly),
		NewUIClusterRoleBinding(u.ns),
		NewUIDeployment(u.ns, u.odigosVersion, u.config.ImagePrefix, u.managerOpts.ImageReferences.UIImage, u.config.NodeSelector),
		NewUIService(u.ns),
	}
	return u.client.ApplyResources(ctx, u.config.ConfigVersion, resources, u.managerOpts)
}

func NewUIResourceManager(client *kube.Client, ns string, config *common.OdigosConfiguration, odigosVersion string, managerOpts resourcemanager.ManagerOpts) resourcemanager.ResourceManager {
	return &uiResourceManager{
		client:        client,
		ns:            ns,
		config:        config,
		odigosVersion: odigosVersion,
		readonly:      config.UiMode == common.ReadonlyUiMode,
		managerOpts:   managerOpts,
	}
}
