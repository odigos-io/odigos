package centralodigos

import (
	"context"
	"fmt"
	"strconv"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/containers"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	autoscalingv2beta1 "k8s.io/api/autoscaling/v2beta1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/version"
)

type centralBackendResourceManager struct {
	client        *kube.Client
	ns            string
	odigosVersion string
	managerOpts   resourcemanager.ManagerOpts
}

func NewCentralBackendResourceManager(client *kube.Client, ns string, odigosVersion string, managerOpts resourcemanager.ManagerOpts) resourcemanager.ResourceManager {
	return &centralBackendResourceManager{client: client, ns: ns, odigosVersion: odigosVersion, managerOpts: managerOpts}
}

func (m *centralBackendResourceManager) Name() string { return k8sconsts.CentralBackendName }

func (m *centralBackendResourceManager) InstallFromScratch(ctx context.Context) error {
	return m.client.ApplyResources(ctx, 1, []kube.Object{
		NewCentralBackendServiceAccount(m.ns),
		NewCentralBackendRole(m.ns),
		NewCentralBackendRoleBinding(m.ns),
		NewCentralBackendDeployment(m.ns, k8sconsts.OdigosImagePrefix, m.managerOpts.ImageReferences.CentralBackendImage, m.odigosVersion),
		NewCentralBackendService(m.ns),
		NewCentralBackendHPA(m.ns, m.client),
	}, m.managerOpts)
}

func NewCentralBackendDeployment(ns, imagePrefix, imageName, version string) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.CentralBackendName,
			Namespace: ns,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrint32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": k8sconsts.CentralBackendAppName},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": k8sconsts.CentralBackendAppName},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: k8sconsts.CentralBackendServiceAccountName,
					Containers: []corev1.Container{
						{
							Name:  k8sconsts.CentralBackendAppName,
							Image: containers.GetImageName(imagePrefix, imageName, version),
							Env: []corev1.EnvVar{
								{
									Name: k8sconsts.OdigosOnpremTokenEnvName,
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: k8sconsts.OdigosCentralSecretName,
											},
											Key: k8sconsts.OdigosOnpremTokenSecretKey,
										},
									},
								},
								// Keycloak configuration
								{
									Name:  "KEYCLOAK_HOST",
									Value: fmt.Sprintf("http://%s:%d", k8sconsts.KeycloakServiceName, k8sconsts.KeycloakPort),
								},
								{
									Name:  "USE_K8S_SECRETS",
									Value: "true",
								},
								{
									Name:  "KEYCLOAK_SECRET_NAMESPACE",
									Value: ns,
								},
								{
									Name:  "KEYCLOAK_SECRET_NAME",
									Value: k8sconsts.KeycloakSecretName,
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

func NewCentralBackendService(ns string) *corev1.Service {
	portInt, err := strconv.Atoi(k8sconsts.CentralBackendPort)
	if err != nil {
		portInt = 8081
	}
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.CentralBackendName,
			Namespace: ns,
			Labels: map[string]string{
				"app": k8sconsts.CentralBackendAppName,
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": k8sconsts.CentralBackendAppName,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       int32(portInt),
					TargetPort: intstrFromInt(portInt),
				},
			},
		},
	}
}

func intstrFromInt(val int) intstr.IntOrString {
	return intstr.IntOrString{Type: intstr.Int, IntVal: int32(val)}
}

func NewCentralBackendServiceAccount(ns string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.CentralBackendServiceAccountName,
			Namespace: ns,
		},
	}
}

func NewCentralBackendRole(ns string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.CentralBackendRoleName,
			Namespace: ns,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get"},
				APIGroups: []string{""},
				Resources: []string{"secrets"},
			},
		},
	}
}

func NewCentralBackendRoleBinding(ns string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.CentralBackendRoleBindingName,
			Namespace: ns,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      k8sconsts.CentralBackendServiceAccountName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     k8sconsts.CentralBackendRoleName,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}



func NewCentralBackendHPA(ns string, client *kube.Client) kube.Object {
	minReplicas := ptrint32(1)
	maxReplicas := int32(10)
	targetUtilization := int32(k8sconsts.CentralBackendDefaultCpuTargetUtilization)

	sv, err := client.Discovery().ServerVersion()
	var parsed *version.Version
	if err == nil {
		parsed, _ = version.Parse(sv.GitVersion)
	}

	// HPA apiVersion selection by server version:
	// - < 1.23   → autoscaling/v2beta1 (no Behavior fields)
	// - 1.23–1.24 → autoscaling/v2beta2 (limited Behavior support)
	// - ≥ 1.25 or unknown → autoscaling/v2 (full Behavior)


	if parsed == nil || !parsed.LessThan(version.MustParse("1.25.0")) {
		return &autoscalingv2.HorizontalPodAutoscaler{
			TypeMeta:   metav1.TypeMeta{APIVersion: "autoscaling/v2", Kind: "HorizontalPodAutoscaler"},
			ObjectMeta: metav1.ObjectMeta{Name: k8sconsts.CentralBackendName, Namespace: ns},
			Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: k8sconsts.CentralBackendName},
				MinReplicas:    minReplicas,
				MaxReplicas:    maxReplicas,
				Metrics: []autoscalingv2.MetricSpec{
					{
						Type: autoscalingv2.ResourceMetricSourceType,
						Resource: &autoscalingv2.ResourceMetricSource{
							Name:   corev1.ResourceCPU,
							Target: autoscalingv2.MetricTarget{Type: autoscalingv2.UtilizationMetricType, AverageUtilization: &targetUtilization},
						},
					},
				},
				Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
					ScaleUp: &autoscalingv2.HPAScalingRules{
						StabilizationWindowSeconds: int32Ptr(30),
						SelectPolicy:               selectPolicyPtr(autoscalingv2.MaxChangePolicySelect),
						Policies: []autoscalingv2.HPAScalingPolicy{
							{Type: autoscalingv2.PercentScalingPolicy, Value: 100, PeriodSeconds: 30},
							{Type: autoscalingv2.PodsScalingPolicy, Value: 2, PeriodSeconds: 30},
						},
					},
					ScaleDown: &autoscalingv2.HPAScalingRules{
						StabilizationWindowSeconds: int32Ptr(600),
						SelectPolicy:               selectPolicyPtr(autoscalingv2.MinChangePolicySelect),
						Policies: []autoscalingv2.HPAScalingPolicy{
							{Type: autoscalingv2.PercentScalingPolicy, Value: 10, PeriodSeconds: 60},
							{Type: autoscalingv2.PodsScalingPolicy, Value: 1, PeriodSeconds: 60},
						},
					},
				},
			},
		}
	}

	if parsed != nil && !parsed.LessThan(version.MustParse("1.23.0")) && parsed.LessThan(version.MustParse("1.25.0")) {
		return &autoscalingv2beta2.HorizontalPodAutoscaler{
			TypeMeta:   metav1.TypeMeta{APIVersion: "autoscaling/v2beta2", Kind: "HorizontalPodAutoscaler"},
			ObjectMeta: metav1.ObjectMeta{Name: k8sconsts.CentralBackendName, Namespace: ns},
			Spec: autoscalingv2beta2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2beta2.CrossVersionObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: k8sconsts.CentralBackendName},
				MinReplicas:    minReplicas,
				MaxReplicas:    maxReplicas,
				Metrics: []autoscalingv2beta2.MetricSpec{
					{
						Type: autoscalingv2beta2.ResourceMetricSourceType,
						Resource: &autoscalingv2beta2.ResourceMetricSource{
							Name: corev1.ResourceCPU,
							Target: autoscalingv2beta2.MetricTarget{Type: autoscalingv2beta2.UtilizationMetricType, AverageUtilization: &targetUtilization},
						},
					},
				},
			},
		}
	}

	return &autoscalingv2beta1.HorizontalPodAutoscaler{
		TypeMeta:   metav1.TypeMeta{APIVersion: "autoscaling/v2beta1", Kind: "HorizontalPodAutoscaler"},
		ObjectMeta: metav1.ObjectMeta{Name: k8sconsts.CentralBackendName, Namespace: ns},
		Spec: autoscalingv2beta1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2beta1.CrossVersionObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: k8sconsts.CentralBackendName},
			MinReplicas:    minReplicas,
			MaxReplicas:    maxReplicas,
			Metrics: []autoscalingv2beta1.MetricSpec{
				{
					Type: autoscalingv2beta1.ResourceMetricSourceType,
					Resource: &autoscalingv2beta1.ResourceMetricSource{
						Name:                     corev1.ResourceCPU,
						TargetAverageUtilization: &targetUtilization,
					},
				},
			},
		},
	}
}

func int32Ptr(v int32) *int32 { return &v }

func selectPolicyPtr(p autoscalingv2.ScalingPolicySelect) *autoscalingv2.ScalingPolicySelect {
	return &p
}
