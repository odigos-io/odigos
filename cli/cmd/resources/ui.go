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
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "ui-db-storage",
									MountPath: "/data",
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/healthz",
										Port: intstr.IntOrString{
											Type:   intstr.Type(0),
											IntVal: 3000,
										},
									},
								},
								InitialDelaySeconds: 15,
								TimeoutSeconds:      5,
								PeriodSeconds:       20,
								SuccessThreshold:    0,
								FailureThreshold:    0,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/readyz",
										Port: intstr.IntOrString{
											Type:   intstr.Type(0),
											IntVal: 3000,
										},
									},
								},
								PeriodSeconds:  10,
								TimeoutSeconds: 5,
							},
							SecurityContext: &corev1.SecurityContext{},
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
			{ // Needed to read odigos-configuration configmap for settings
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list"},
			},
			{ // Needed to read all odigos.io CRDs in the odigos namespace
				APIGroups: []string{"odigos.io"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Needed to read all actions.odigos.io CRDs in the odigos namespace
				APIGroups: []string{"actions.odigos.io"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Needed for reading collectors related resources
				APIGroups: []string{"apps"},
				Resources: []string{"replicasets"},
				Verbs:     []string{"get", "list"},
			},
			{ // Needed for reading HPA status of the gateway-collector
				APIGroups: []string{"autoscaling"},
				Resources: []string{"horizontalpodautoscalers"},
				Verbs:     []string{"get"},
			},
			{ // Needed for pod operations (restart pod)
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list"},
			},
			{ // Needed for reading Pods logs
				APIGroups: []string{""},
				Resources: []string{"pods/log"},
				Verbs:     []string{"get"},
			},
		}
	} else {
		rules = []rbacv1.PolicyRule{
			{ // Needed to read and update odigos-configuration configmap for settings
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list", "create", "update", "patch"},
			},
			{ // Needed for secret values in destinations
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "create", "patch", "update", "delete"},
			},
			{ // Needed for CRUD on all odigos.io CRDs in the odigos namespace
				APIGroups: []string{"odigos.io"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"odigos.io"},
				Resources: []string{"instrumentationrules", "destinations", "actions"},
				Verbs:     []string{"create", "patch", "update", "delete"},
			},
			{ // Needed for CRUD on all actions.odigos.io CRDs in the odigos namespace
				APIGroups: []string{"actions.odigos.io"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list", "create", "patch", "update", "delete"},
			},
			{ // Needed for reading ReplicaSets owned by workloads
				APIGroups: []string{"apps"},
				Resources: []string{"replicasets"},
				Verbs:     []string{"get", "list"},
			},
			{ // Needed for reading HPA status in the namespace
				APIGroups: []string{"autoscaling"},
				Resources: []string{"horizontalpodautoscalers"},
				Verbs:     []string{"get"},
			},
			{ // Needed for pod operations (restart pod)
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "delete"},
			},
			{ // Needed for reading Pods logs
				APIGroups: []string{""},
				Resources: []string{"pods/log"},
				Verbs:     []string{"get"},
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

func NewUIClusterRole(readonly bool, openshiftEnabled bool) *rbacv1.ClusterRole {
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
			{ // Needed to get and instrument sources
				APIGroups: []string{"batch"},
				Resources: []string{"cronjobs"},
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
			{ // Needed to read all odigos.io CRDs cluster-wide
				APIGroups: []string{"odigos.io"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Needed to read all actions.odigos.io CRDs cluster-wide
				APIGroups: []string{"actions.odigos.io"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list", "watch"},
			},
		}
		if openshiftEnabled {
			rules = append(rules, rbacv1.PolicyRule{
				// OpenShift DeploymentConfigs support
				APIGroups: []string{"apps.openshift.io"},
				Resources: []string{"deploymentconfigs"},
				Verbs:     []string{"get", "list"},
			})
		}
		// Argo Rollouts support
		rules = append(rules, rbacv1.PolicyRule{
			APIGroups: []string{"argoproj.io"},
			Resources: []string{"rollouts"},
			Verbs:     []string{"get", "list"},
		})
	} else {
		rules = []rbacv1.PolicyRule{
			{ // Needed to get and instrument namespaces
				APIGroups: []string{""},
				Resources: []string{"namespaces"},
				Verbs:     []string{"get", "list"},
			},
			{ // get & list : Needed to get workloads
				// patch & update: Needed to rollout restart workloads
				APIGroups: []string{"apps"},
				Resources: []string{"deployments", "statefulsets", "daemonsets"},
				Verbs:     []string{"get", "list", "patch", "update"},
			},
			{ // Needed to get and instrument sources
				APIGroups: []string{"batch"},
				Resources: []string{"cronjobs"},
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
				// delete is needed for restart pod functionality
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch", "delete"},
			},
			{ // Needed for CRUD on all odigos.io CRDs cluster-wide
				APIGroups: []string{"odigos.io"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"odigos.io"},
				Resources: []string{"instrumentationconfigs", "instrumentationinstances", "sources"},
				Verbs:     []string{"patch", "create", "delete"},
			},
			{ // Needed for CRUD on all actions.odigos.io CRDs cluster-wide
				APIGroups: []string{"actions.odigos.io"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list", "create", "patch", "delete"},
			},
		}
		if openshiftEnabled {
			rules = append(rules, rbacv1.PolicyRule{
				// OpenShift DeploymentConfigs support
				APIGroups: []string{"apps.openshift.io"},
				Resources: []string{"deploymentconfigs"},
				Verbs:     []string{"get", "list", "patch", "update"},
			})
		}
		// Argo Rollouts support
		rules = append(rules, rbacv1.PolicyRule{
			APIGroups: []string{"argoproj.io"},
			Resources: []string{"rollouts"},
			Verbs:     []string{"get", "list", "patch", "update"},
		})
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
		NewUIClusterRole(u.readonly, u.config.OpenshiftEnabled),
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
		readonly:      config.UiMode == common.UiModeReadonly,
		managerOpts:   managerOpts,
	}
}

// Pointer helper functions
func ptrint32(i int32) *int32 {
	return &i
}

func ptrint64(i int64) *int64 {
	return &i
}

func ptrbool(b bool) *bool {
	return &b
}
