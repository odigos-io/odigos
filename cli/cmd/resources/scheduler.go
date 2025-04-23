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

func NewSchedulerServiceAccount(ns string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.SchedulerServiceAccountName,
			Namespace: ns,
		},
	}
}

func NewSchedulerLeaderElectionRoleBinding(ns string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "odigos-scheduler-leader-election",
			Namespace: ns,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: k8sconsts.SchedulerServiceAccountName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "odigos-leader-election-role",
		},
	}
}

func NewSchedulerRole(ns string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.SchedulerRoleName,
			Namespace: ns,
		},
		Rules: []rbacv1.PolicyRule{
			{ // Needed to react and reconcile odigos-config changes to effective config
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Needed to apply effective config after reconciling (defaulting and profile applying) and react to it
				APIGroups:     []string{""},
				Resources:     []string{"configmaps"},
				ResourceNames: []string{consts.OdigosEffectiveConfigName, k8sconsts.OdigosDeploymentConfigMapName},
				Verbs:         []string{"patch", "create", "update"},
			},
			{ // Needed because the scheduler is managing the collectorsgroups
				APIGroups: []string{"odigos.io"},
				Resources: []string{"collectorsgroups"},
				Verbs:     []string{"get", "list", "create", "patch", "update", "watch", "delete"},
			},
			{ // Needed to read the status of the gateway collector, to wake the data collector
				APIGroups: []string{"odigos.io"},
				Resources: []string{"collectorsgroups/status"},
				Verbs:     []string{"get"},
			},
			{ // Needed to wake the gateway collector (based on the presence of any destination)
				APIGroups: []string{"odigos.io"},
				Resources: []string{"destinations"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // apply profiles
				APIGroups: []string{"odigos.io"},
				Resources: []string{"processors", "instrumentationrules"},
				Verbs:     []string{"get", "list", "watch", "patch", "delete", "create"},
			},
			{ // read odigos pro token
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
}

func NewSchedulerRoleBinding(ns string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.SchedulerRoleBindingName,
			Namespace: ns,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: k8sconsts.SchedulerServiceAccountName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     k8sconsts.SchedulerRoleName,
		},
	}
}

func NewSchedulerClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sconsts.SchedulerClusterRoleName,
		},
		Rules: []rbacv1.PolicyRule{
			{ // Needed to track presence/status of configs to wake the data/gateway collectors
				APIGroups: []string{"odigos.io"},
				Resources: []string{"instrumentationconfigs"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
}

func NewSchedulerClusterRoleBinding(ns string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sconsts.SchedulerClusterRoleBindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      k8sconsts.SchedulerServiceAccountName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     k8sconsts.SchedulerClusterRoleName,
		},
	}
}

func NewSchedulerDeployment(ns string, version string, imagePrefix string, imageName string, nodeSelector map[string]string) *appsv1.Deployment {
	if nodeSelector == nil {
		nodeSelector = make(map[string]string)
	}
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.SchedulerDeploymentName,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/name": k8sconsts.SchedulerAppLabelValue,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrint32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": k8sconsts.SchedulerAppLabelValue,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": k8sconsts.SchedulerAppLabelValue,
					},
					Annotations: map[string]string{
						"kubectl.kubernetes.io/default-container": k8sconsts.SchedulerContainerName,
					},
				},
				Spec: corev1.PodSpec{
					NodeSelector: nodeSelector,
					Containers: []corev1.Container{
						{
							Name:  k8sconsts.SchedulerContainerName,
							Image: containers.GetImageName(imagePrefix, imageName, version),
							Command: []string{
								"/app",
							},
							Args: []string{
								"--health-probe-bind-address=:8081",
								"--metrics-bind-address=127.0.0.1:8080",
								"--leader-elect",
							},
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: k8sconsts.SchedulerServiceName,
								},
								{
									Name: "CURRENT_NS",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									Name: consts.OdigosTierEnvVarName,
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: k8sconsts.OdigosDeploymentConfigMapName,
											},
											Key: k8sconsts.OdigosDeploymentConfigMapTierKey,
										},
									},
								},
								{
									Name: consts.OdigosVersionEnvVarName,
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: k8sconsts.OdigosDeploymentConfigMapName,
											},
											Key: k8sconsts.OdigosDeploymentConfigMapVersionKey,
										},
									},
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: ownTelemetryOtelConfig,
										},
									},
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
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/healthz",
										Port: intstr.IntOrString{
											Type:   intstr.Type(0),
											IntVal: 8081,
										},
									},
								},
								InitialDelaySeconds: 15,
								TimeoutSeconds:      0,
								PeriodSeconds:       20,
								SuccessThreshold:    0,
								FailureThreshold:    0,
							},
							SecurityContext: &corev1.SecurityContext{},
						},
					},
					TerminationGracePeriodSeconds: ptrint64(10),
					ServiceAccountName:            k8sconsts.SchedulerServiceAccountName,
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

type schedulerResourceManager struct {
	client        *kube.Client
	ns            string
	config        *common.OdigosConfiguration
	odigosVersion string
	managerOpts   resourcemanager.ManagerOpts
}

func NewSchedulerResourceManager(client *kube.Client, ns string, config *common.OdigosConfiguration, odigosVersion string, managerOpts resourcemanager.ManagerOpts) resourcemanager.ResourceManager {
	return &schedulerResourceManager{client: client, ns: ns, config: config, odigosVersion: odigosVersion, managerOpts: managerOpts}
}

func (a *schedulerResourceManager) Name() string { return "Scheduler" }

func (a *schedulerResourceManager) InstallFromScratch(ctx context.Context) error {
	resources := []kube.Object{
		NewSchedulerServiceAccount(a.ns),
		NewSchedulerLeaderElectionRoleBinding(a.ns),
		NewSchedulerRole(a.ns),
		NewSchedulerRoleBinding(a.ns),
		NewSchedulerClusterRole(),
		NewSchedulerClusterRoleBinding(a.ns),
		NewSchedulerDeployment(a.ns, a.odigosVersion, a.config.ImagePrefix, a.managerOpts.ImageReferences.SchedulerImage, a.config.NodeSelector),
	}
	return a.client.ApplyResources(ctx, a.config.ConfigVersion, resources, a.managerOpts)
}
