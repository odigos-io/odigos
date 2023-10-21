package resources

import (
	"context"
	"fmt"

	"github.com/keyval-dev/odigos/cli/pkg/containers"
	"github.com/keyval-dev/odigos/cli/pkg/kube"

	"github.com/keyval-dev/odigos/cli/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var AutoscalerImage string

const (
	autoScalerServiceName    = "auto-scaler"
	autoScalerDeploymentName = "odigos-autoscaler"
)

func NewAutoscalerServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "odigos-autoscaler",
			Labels: labels.OdigosSystem,
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
			Name:   "odigos-autoscaler",
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
			Name:   "odigos-autoscaler",
			Labels: labels.OdigosSystem,
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
			Name:   "odigos-autoscaler",
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
			Name:   "odigos-autoscaler",
			Labels: labels.OdigosSystem,
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
			Name:   "odigos-autoscaler-leader-election",
			Labels: labels.OdigosSystem,
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
			Name: autoScalerDeploymentName,
			Labels: map[string]string{
				"app":                       "odigos-autoscaler",
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
					"app": "odigos-autoscaler",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "odigos-autoscaler",
					},
					Annotations: map[string]string{
						"kubectl.kubernetes.io/default-container": "manager",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "manager",
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
									Value: autoScalerServiceName,
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
	client *kube.Client
	ns     string
}

func NewAutoScalerResourceManager(client *kube.Client, ns string) ResourceManager {
	return &autoScalerResourceManager{client: client, ns: ns}
}

func (a *autoScalerResourceManager) InstallFromScratch(ctx context.Context) error {
	return nil
}

func (a *autoScalerResourceManager) ApplyMigrationStep(ctx context.Context, sourceVersion string) error {
	return nil
}

func (a *autoScalerResourceManager) RollbackMigrationStep(ctx context.Context, sourceVersion string) error {
	return nil
}

func (a *autoScalerResourceManager) PatchOdigosVersionToTarget(ctx context.Context, newOdigosVersion string) error {
	fmt.Println("Patching Odigos autoscaler deployment")
	jsonPatchDocumentBytes := patchTemplateSpecImageTag(AutoscalerImage, newOdigosVersion)
	_, err := a.client.AppsV1().Deployments(a.ns).Patch(ctx, autoScalerDeploymentName, k8stypes.JSONPatchType, jsonPatchDocumentBytes, metav1.PatchOptions{})
	return err
}
