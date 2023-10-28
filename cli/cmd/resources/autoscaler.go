package resources

import (
	"context"

	"github.com/keyval-dev/odigos/cli/pkg/containers"
	"github.com/keyval-dev/odigos/cli/pkg/kube"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var AutoscalerImage string

const (
	AutoScalerServiceAccountName = "odigos-autoscaler"
	AutoScalerServiceName        = "auto-scaler"
	AutoScalerDeploymentName     = "odigos-autoscaler"
	AutoScalerAppLabelValue      = "odigos-autoscaler"
	AutoScalerContainerName      = "manager"
)

func NewAutoscalerServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: AutoScalerServiceAccountName,
		},
	}
}

func NewAutoscalerRole() *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "odigos-autoscaler",
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
					"configmaps",
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
				APIGroups: []string{""},
				Resources: []string{
					"services",
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
					"apps",
				},
				Resources: []string{
					"daemonsets",
				},
			},
			{
				Verbs: []string{
					"get",
					"patch",
					"update",
				},
				APIGroups: []string{
					"apps",
				},
				Resources: []string{
					"daemonsets/status",
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
					"apps",
				},
				Resources: []string{
					"deployments",
				},
			},
			{
				Verbs: []string{
					"get",
					"patch",
					"update",
				},
				APIGroups: []string{
					"apps",
				},
				Resources: []string{
					"deployments/status",
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
		},
	}
}

func NewAutoscalerRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "odigos-autoscaler",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: "odigos-autoscaler",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "odigos-autoscaler",
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
			Name: "odigos-autoscaler",
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
			Name: "odigos-autoscaler",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "odigos-autoscaler",
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "odigos-autoscaler",
		},
	}
}

func NewAutoscalerLeaderElectionRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "odigos-autoscaler-leader-election",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: "odigos-autoscaler",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "odigos-leader-election-role",
		},
	}
}

func NewAutoscalerDeployment(version string) *appsv1.Deployment {
	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: AutoScalerDeploymentName,

			Annotations: map[string]string{
				"odigos.io/skip": "true",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrint32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": AutoScalerAppLabelValue,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": AutoScalerAppLabelValue,
					},
					Annotations: map[string]string{
						"kubectl.kubernetes.io/default-container": AutoScalerContainerName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  AutoScalerContainerName,
							Image: containers.GetImageName(AutoscalerImage, version),
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
									"memory": *resource.NewQuantity(134217728, resource.BinarySI),
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
					ServiceAccountName:            "odigos-autoscaler",
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: ptrbool(true),
					},
				},
			},
			Strategy:        appsv1.DeploymentStrategy{},
			MinReadySeconds: 0,
		},
	}

	if containers.ImagePrefix != "" {
		dep.Spec.Template.Spec.Containers[0].Args = append(dep.Spec.Template.Spec.Containers[0].Args,
			"--image-prefix="+containers.ImagePrefix)
	}

	return dep
}

type autoScalerResourceManager struct {
	client  *kube.Client
	ns      string
	version string
}

func NewAutoScalerResourceManager(client *kube.Client, ns string, version string) ResourceManager {
	return &autoScalerResourceManager{client: client, ns: ns, version: version}
}

func (a *autoScalerResourceManager) Name() string { return "AutoScaler" }

func (a *autoScalerResourceManager) InstallFromScratch(ctx context.Context) error {

	sa := NewAutoscalerServiceAccount()
	err := a.client.ApplyResource(ctx, a.ns, a.version, sa)
	if err != nil {
		return err
	}

	role := NewAutoscalerRole()
	err = a.client.ApplyResource(ctx, a.ns, a.version, role)
	if err != nil {
		return err
	}

	roleBinding := NewAutoscalerRoleBinding()
	err = a.client.ApplyResource(ctx, a.ns, a.version, roleBinding)
	if err != nil {
		return err
	}

	clusterRole := NewAutoscalerClusterRole()
	err = a.client.ApplyResource(ctx, "", a.version, clusterRole)
	if err != nil {
		return err
	}

	clusterRoleBinding := NewAutoscalerClusterRoleBinding(a.ns)
	err = a.client.ApplyResource(ctx, "", a.version, clusterRoleBinding)
	if err != nil {
		return err
	}

	leaderElectionRoleBinding := NewAutoscalerLeaderElectionRoleBinding()
	err = a.client.ApplyResource(ctx, a.ns, a.version, leaderElectionRoleBinding)
	if err != nil {
		return err
	}

	dep := NewAutoscalerDeployment(a.version)
	err = a.client.ApplyResource(ctx, a.ns, a.version, dep)
	if err != nil {
		return err
	}

	return nil
}
