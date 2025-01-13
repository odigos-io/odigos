package resources

import (
	"context"
	"slices"

	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/containers"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	k8sconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	AutoScalerDeploymentName         = "odigos-autoscaler"
	AutoScalerServiceAccountName     = AutoScalerDeploymentName
	AutoScalerAppLabelValue          = AutoScalerDeploymentName
	AutoScalerRoleName               = AutoScalerDeploymentName
	AutoScalerRoleBindingName        = AutoScalerDeploymentName
	AutoScalerClusterRoleName        = AutoScalerDeploymentName
	AutoScalerClusterRoleBindingName = AutoScalerDeploymentName
	AutoScalerServiceName            = "auto-scaler"
	AutoScalerContainerName          = "manager"
)

func NewAutoscalerServiceAccount(ns string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      AutoScalerServiceAccountName,
			Namespace: ns,
		},
	}
}

func NewAutoscalerRole(ns string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      AutoScalerRoleName,
			Namespace: ns,
		},
		Rules: []rbacv1.PolicyRule{
			{ // Needed to manage the configmaps of the data-collector and gateway-collector
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list", "watch", "create", "patch", "update", "delete"},
			},
			{ // Needed to manage the k8s-service of gateway-collector
				APIGroups: []string{""},
				Resources: []string{"services"},
				Verbs:     []string{"get", "list", "watch", "create", "patch", "update", "delete", "deletecollection"},
			},
			{ // Needed to manage the daemonsets for data-collector
				APIGroups: []string{"apps"},
				Resources: []string{"daemonsets"},
				Verbs:     []string{"get", "list", "watch", "create", "patch", "update", "delete", "deletecollection"},
			},
			{ // Needed to read the "readiness" status of the collectorsgroup
				APIGroups: []string{"apps"},
				Resources: []string{"daemonsets/status"},
				Verbs:     []string{"get"},
			},
			{ // Needed to manage the deployments for data-collector
				APIGroups: []string{"apps"},
				Resources: []string{"deployments"},
				Verbs:     []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"},
			},
			{ // Needed to read the "readiness" status of the collectorsgroup
				APIGroups: []string{"apps"},
				Resources: []string{"deployments/status"},
				Verbs:     []string{"get"},
			},
			{ // Needed to apply autoscaling to the gateway-collector
				APIGroups: []string{"autoscaling"},
				Resources: []string{"horizontalpodautoscalers"},
				Verbs:     []string{"create", "patch", "update", "delete"},
			},
			{ // Needed to track changes and restart gateway-collector
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Needed to sync the gateway-collector configuration
				APIGroups: []string{"odigos.io"},
				Resources: []string{"destinations"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Needed to track destination-config changes and update the status accordingly
				APIGroups: []string{"odigos.io"},
				Resources: []string{"destinations/status"},
				Verbs:     []string{"get", "patch", "update"},
			},
			{ // Needed to identify changes to pipeline-actions and update the data & gateway collectors configmap
				APIGroups: []string{"odigos.io"},
				Resources: []string{"processors"},
				Verbs:     []string{"get", "list", "watch", "create", "patch", "update"},
			},
			{ // Needed to read actions transform them to processors
				APIGroups: []string{"actions.odigos.io"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Needed to updated the status of the actions (confirms the user-made-changes)
				APIGroups: []string{"actions.odigos.io"},
				Resources: []string{"*/status"},
				Verbs:     []string{"get", "patch", "update"},
			},
			{ // Needed to watch for changes made in the the collectorgroups and apply them to the cluster
				APIGroups: []string{"odigos.io"},
				Resources: []string{"collectorsgroups"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // After applying the collectorgroups tot he cluster, we need to update the status of the operation
				APIGroups: []string{"odigos.io"},
				Resources: []string{"collectorsgroups/status"},
				Verbs:     []string{"get", "patch", "update"},
			},
		},
	}
}

func NewAutoscalerRoleBinding(ns string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      AutoScalerRoleBindingName,
			Namespace: ns,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: AutoScalerServiceAccountName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     AutoScalerRoleName,
		},
	}
}

func NewAutoscalerClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: AutoScalerClusterRoleName,
		},
		Rules: []rbacv1.PolicyRule{
			{ // Needed to read the applications, to populate the receivers.filelog in the data-collector configmap
				APIGroups: []string{"odigos.io"},
				Resources: []string{"instrumentationconfigs"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
}

func NewAutoscalerClusterRoleBinding(ns string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: AutoScalerClusterRoleBindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      AutoScalerServiceAccountName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     AutoScalerClusterRoleName,
		},
	}
}

func NewAutoscalerLeaderElectionRoleBinding(ns string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "odigos-autoscaler-leader-election",
			Namespace: ns,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: AutoScalerServiceAccountName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "odigos-leader-election-role",
		},
	}
}

func NewAutoscalerDeployment(ns string, version string, imagePrefix string, imageName string, disableNameProcessor bool) *appsv1.Deployment {

	optionalEnvs := []corev1.EnvVar{}

	if disableNameProcessor {
		// temporary until we migrate java and dotnet to OpAMP
		optionalEnvs = append(optionalEnvs, corev1.EnvVar{
			Name:  "DISABLE_NAME_PROCESSOR",
			Value: "true",
		})
	}

	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      AutoScalerDeploymentName,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/name": AutoScalerAppLabelValue,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrint32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": AutoScalerAppLabelValue,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": AutoScalerAppLabelValue,
					},
					Annotations: map[string]string{
						"kubectl.kubernetes.io/default-container": AutoScalerContainerName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  AutoScalerContainerName,
							Image: containers.GetImageName(imagePrefix, imageName, version),
							Command: []string{
								"/app",
							},
							Args: []string{
								"--health-probe-bind-address=:8081",
								"--metrics-bind-address=127.0.0.1:8080",
								"--leader-elect",
							},
							Env: append([]corev1.EnvVar{
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: AutoScalerServiceName,
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
							}, optionalEnvs...),
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
					ServiceAccountName:            AutoScalerServiceAccountName,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: ptrbool(true),
					},
				},
			},
			Strategy:        appsv1.DeploymentStrategy{},
			MinReadySeconds: 0,
		},
	}

	if imagePrefix != "" {
		dep.Spec.Template.Spec.Containers[0].Args = append(dep.Spec.Template.Spec.Containers[0].Args,
			"--image-prefix="+imagePrefix)
	}

	return dep
}

type autoScalerResourceManager struct {
	client        *kube.Client
	ns            string
	config        *common.OdigosConfiguration
	odigosVersion string
}

func NewAutoScalerResourceManager(client *kube.Client, ns string, config *common.OdigosConfiguration, odigosVersion string) resourcemanager.ResourceManager {
	return &autoScalerResourceManager{client: client, ns: ns, config: config, odigosVersion: odigosVersion}
}

func (a *autoScalerResourceManager) Name() string { return "AutoScaler" }

func (a *autoScalerResourceManager) InstallFromScratch(ctx context.Context) error {

	disableNameProcessor := slices.Contains(a.config.Profiles, "disable-name-processor") || slices.Contains(a.config.Profiles, "kratos")

	resources := []kube.Object{
		NewAutoscalerServiceAccount(a.ns),
		NewAutoscalerRole(a.ns),
		NewAutoscalerRoleBinding(a.ns),
		NewAutoscalerClusterRole(),
		NewAutoscalerClusterRoleBinding(a.ns),
		NewAutoscalerLeaderElectionRoleBinding(a.ns),
		NewAutoscalerDeployment(a.ns, a.odigosVersion, a.config.ImagePrefix, a.config.AutoscalerImage, disableNameProcessor),
	}
	return a.client.ApplyResources(ctx, a.config.ConfigVersion, resources)
}
