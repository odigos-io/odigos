package resources

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources/odigospro"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/containers"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func NewKeyvalProxyServiceAccount(ns string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.KeyvalProxyServiceAccountName,
			Namespace: ns,
		},
	}
}

func NewKeyvalProxyRole(ns string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.KeyvalProxyRoleName,
			Namespace: ns,
			Labels:    map[string]string{},
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs: []string{
					"create",
					"delete",
					"get",
					"update",
					"watch",
				},
				APIGroups: []string{""},
				Resources: []string{
					"secrets",
				},
			},
		},
	}
}

func NewKeyvalProxyRoleBinding(ns string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.KeyvalProxyRoleBindingName,
			Namespace: ns,
			Labels:    map[string]string{},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      k8sconsts.KeyvalProxyServiceAccountName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     k8sconsts.KeyvalProxyRoleName,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

func NewKeyvalProxyClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sconsts.KeyvalProxyClusterRoleName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs: []string{
					"list",
					"watch",
					"get",
					"update",
					"patch",
				},
				APIGroups: []string{""},
				Resources: []string{
					"namespaces",
				},
			},
			{
				Verbs: []string{
					"list",
					"watch",
					"get",
					"update",
					"patch",
				},
				APIGroups: []string{"apps"},
				Resources: []string{
					"deployments",
				},
			},
			{
				Verbs: []string{
					"list",
					"watch",
					"get",
					"update",
					"patch",
				},
				APIGroups: []string{"apps"},
				Resources: []string{
					"daemonsets",
				},
			},
			{
				Verbs: []string{
					"list",
					"watch",
					"get",
					"update",
					"patch",
				},
				APIGroups: []string{"apps"},
				Resources: []string{
					"statefulsets",
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
					"patch",
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
					"create",
					"delete",
					"get",
					"list",
					"patch",
					"update",
					"watch",
					"patch",
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
					"create",
					"delete",
					"get",
					"list",
					"patch",
					"update",
					"watch",
					"patch",
				},
				APIGroups: []string{
					"odigos.io",
				},
				Resources: []string{
					"instrumentationconfigs",
				},
			},
			{
				Verbs: []string{
					"watch",
					"get",
					"list",
					"update",
					"patch",
					"create",
					"delete",
				},
				APIGroups: []string{"actions.odigos.io"},
				Resources: []string{"addclusterinfos", "deleteattributes", "renameattributes", "probabilisticsamplers"},
			},
		},
	}
}

func NewKeyvalProxyClusterRoleBinding(ns string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sconsts.KeyvalProxyClusterRoleBindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      k8sconsts.KeyvalProxyServiceAccountName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     k8sconsts.KeyvalProxyClusterRoleName,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

func NewKeyvalProxyDeployment(version string, ns string, imagePrefix string, nodeSelector map[string]string) *appsv1.Deployment {
	if nodeSelector == nil {
		nodeSelector = make(map[string]string)
	}
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.KeyvalProxyDeploymentName,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/name": k8sconsts.KeyvalProxyAppName,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrint32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": k8sconsts.KeyvalProxyAppName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": k8sconsts.KeyvalProxyAppName,
					},
					Annotations: map[string]string{
						"kubectl.kubernetes.io/default-container": k8sconsts.KeyvalProxyAppName,
					},
				},
				Spec: corev1.PodSpec{
					NodeSelector: nodeSelector,
					Containers: []corev1.Container{
						{
							Name:  k8sconsts.KeyvalProxyAppName,
							Image: containers.GetImageName(imagePrefix, k8sconsts.KeyvalProxyImage, version),
							Args: []string{
								"--health-probe-bind-address=:8081",
								"--metrics-bind-address=127.0.0.1:8080",
								// "--leader-elect",
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
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: k8sconsts.KeyvalProxyServiceName,
								},
								odigospro.CloudTokenAsEnvVar(),
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: ownTelemetryOtelConfig,
										},
									},
								},
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: k8sconsts.OdigosDeploymentConfigMapName,
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
					ServiceAccountName:            k8sconsts.KeyvalProxyServiceAccountName,
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

type keyvalProxyResourceManager struct {
	client      *kube.Client
	ns          string
	config      *common.OdigosConfiguration
	managerOpts resourcemanager.ManagerOpts
}

func NewKeyvalProxyResourceManager(client *kube.Client, ns string, config *common.OdigosConfiguration, managerOpts resourcemanager.ManagerOpts) resourcemanager.ResourceManager {
	return &keyvalProxyResourceManager{client: client, ns: ns, config: config, managerOpts: managerOpts}
}

func (a *keyvalProxyResourceManager) Name() string { return "CloudProxy" }

func (a *keyvalProxyResourceManager) InstallFromScratch(ctx context.Context) error {
	resources := []kube.Object{
		NewKeyvalProxyServiceAccount(a.ns),
		NewKeyvalProxyRole(a.ns),
		NewKeyvalProxyRoleBinding(a.ns),
		NewKeyvalProxyClusterRole(),
		NewKeyvalProxyClusterRoleBinding(a.ns),
		NewKeyvalProxyDeployment(k8sconsts.OdigosCloudProxyVersion, a.ns, a.config.ImagePrefix, a.config.NodeSelector),
	}
	return a.client.ApplyResources(ctx, a.config.ConfigVersion, resources, a.managerOpts)
}
