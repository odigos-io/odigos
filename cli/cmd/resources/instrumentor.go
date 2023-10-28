package resources

import (
	"context"
	"fmt"

	"github.com/keyval-dev/odigos/cli/pkg/containers"
	"github.com/keyval-dev/odigos/cli/pkg/kube"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var InstrumentorImage string

const (
	InstrumentorServiceName    = "instrumentor"
	InstrumentorDeploymentName = "odigos-instrumentor"
	InstrumentorAppLabelValue  = "odigos-instrumentor"
	InstrumentorContainerName  = "manager"
)

func NewInstrumentorServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: InstrumentorDeploymentName,
		},
	}
}

func NewInstrumentorRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "odigos-instrumentor-leader-election",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: "odigos-instrumentor",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "odigos-leader-election-role",
		},
	}
}

func NewInstrumentorClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "odigos-instrumentor",
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs: []string{
					"list",
					"watch",
					"get",
				},
				APIGroups: []string{""},
				Resources: []string{
					"nodes",
				},
			},
			{
				Verbs: []string{
					"list",
					"watch",
					"get",
				},
				APIGroups: []string{""},
				Resources: []string{
					"namespaces",
				},
			},
			{
				Verbs: []string{
					"create",
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
					"update",
				},
				APIGroups: []string{
					"apps",
				},
				Resources: []string{
					"daemonsets/finalizers",
				},
			},
			{
				Verbs: []string{
					"get",
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
					"update",
				},
				APIGroups: []string{
					"apps",
				},
				Resources: []string{
					"deployments/finalizers",
				},
			},
			{
				Verbs: []string{
					"get",
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
					"statefulsets",
				},
			},
			{
				Verbs: []string{
					"update",
				},
				APIGroups: []string{
					"apps",
				},
				Resources: []string{
					"statefulsets/finalizers",
				},
			},
			{
				Verbs: []string{
					"get",
				},
				APIGroups: []string{
					"apps",
				},
				Resources: []string{
					"statefulsets/status",
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

func NewInstrumentorClusterRoleBinding(ns string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "odigos-instrumentor",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "odigos-instrumentor",
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "odigos-instrumentor",
		},
	}
}

func NewInstrumentorDeployment(version string, telemetryEnabled bool, sidecarInstrumentation bool, ignoredNamespaces []string) *appsv1.Deployment {
	args := []string{
		"--health-probe-bind-address=:8081",
		"--metrics-bind-address=127.0.0.1:8080",
		"--leader-elect",
	}
	for _, v := range ignoredNamespaces {
		args = append(args, fmt.Sprintf("--ignore-namespace=%s", v))
	}

	if !telemetryEnabled {
		args = append(args, "--telemetry-disabled")
	}

	if sidecarInstrumentation {
		args = append(args, "--golang-sidecar-instrumentation")
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "odigos-instrumentor",

			Annotations: map[string]string{
				"odigos.io/skip": "true",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrint32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": InstrumentorAppLabelValue,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": InstrumentorAppLabelValue,
					},
					Annotations: map[string]string{
						"kubectl.kubernetes.io/default-container": InstrumentorContainerName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  InstrumentorContainerName,
							Image: containers.GetImageName(InstrumentorImage, version),
							Command: []string{
								"/app",
							},
							Args: args,
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: InstrumentorServiceName,
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
					ServiceAccountName:            "odigos-instrumentor",
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

func ptrint32(i int32) *int32 {
	return &i
}

func ptrint64(i int64) *int64 {
	return &i
}

func ptrbool(b bool) *bool {
	return &b
}

type instrumentorResourceManager struct {
	client                 *kube.Client
	ns                     string
	version                string
	telemetryEnabled       bool
	sidecarInstrumentation bool
	ignoredNamespaces      []string
}

func NewInstrumentorResourceManager(client *kube.Client, ns string, version string, telemetryEnabled bool, sidecarInstrumentation bool, ignoredNamespaces []string) ResourceManager {
	return &instrumentorResourceManager{
		client:                 client,
		ns:                     ns,
		version:                version,
		telemetryEnabled:       telemetryEnabled,
		sidecarInstrumentation: sidecarInstrumentation,
		ignoredNamespaces:      ignoredNamespaces,
	}
}

func (a *instrumentorResourceManager) Name() string { return "Instrumentor" }

func (a *instrumentorResourceManager) InstallFromScratch(ctx context.Context) error {

	sa := NewInstrumentorServiceAccount()
	err := a.client.ApplyResource(ctx, a.ns, a.version, sa)
	if err != nil {
		return err
	}

	rb := NewInstrumentorRoleBinding()
	err = a.client.ApplyResource(ctx, a.ns, a.version, rb)
	if err != nil {
		return err
	}

	cr := NewInstrumentorClusterRole()
	err = a.client.ApplyResource(ctx, "", a.version, cr)
	if err != nil {
		return err
	}

	crb := NewInstrumentorClusterRoleBinding(a.ns)
	err = a.client.ApplyResource(ctx, "", a.version, crb)
	if err != nil {
		return err
	}

	dep := NewInstrumentorDeployment(a.version, a.telemetryEnabled, a.sidecarInstrumentation, a.ignoredNamespaces)
	err = a.client.ApplyResource(ctx, a.ns, a.version, dep)
	if err != nil {
		return err
	}

	return err
}
