package resources

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources/odigospro"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/containers"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func NewInstrumentorServiceAccount(ns string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.InstrumentorServiceAccountName,
			Namespace: ns,
		},
	}
}

func NewInstrumentorLeaderElectionRoleBinding(ns string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "odigos-instrumentor-leader-election",
			Namespace: ns,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: k8sconsts.InstrumentorServiceAccountName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "odigos-leader-election-role",
		},
	}
}

func NewInstrumentorRole(ns string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.InstrumentorRoleName,
			Namespace: ns,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"configmaps"},
				ResourceNames: []string{consts.OdigosEffectiveConfigName},
				Verbs:         []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // used by cert-controller to rotate the webhook certificate
				APIGroups:     []string{""},
				Resources:     []string{"secrets"},
				ResourceNames: []string{k8sconsts.InstrumentorWebhookSecretName},
				Verbs:         []string{"update"},
			},
			{ // Used to delete the deprecated webhook secret
				APIGroups:     []string{""},
				Resources:     []string{"secrets"},
				ResourceNames: []string{k8sconsts.DeprecatedInstrumentorWebhookSecretName},
				Verbs:         []string{"delete"},
			},
			{ // check for odiglet daemonset ready before starting the instrumentation
				APIGroups:     []string{"apps"},
				Resources:     []string{"daemonsets"},
				ResourceNames: []string{k8sconsts.OdigletDaemonSetName},
				Verbs:         []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"odigos.io"},
				Resources: []string{"collectorsgroups"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"odigos.io"},
				Resources: []string{"collectorsgroups/status"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Needed for odigos own telemetry events reporting. Consider moving to scheduler
				APIGroups: []string{"odigos.io"},
				Resources: []string{"destinations"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"odigos.io"},
				Resources: []string{"instrumentationrules"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
}

func NewInstrumentorRoleBinding(ns string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.InstrumentorRoleBindingName,
			Namespace: ns,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: k8sconsts.InstrumentorServiceAccountName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     k8sconsts.InstrumentorRoleName,
		},
	}
}

func NewInstrumentorClusterRole(openshiftEnabled bool) *rbacv1.ClusterRole {
	openshiftRules := []rbacv1.PolicyRule{}
	if openshiftEnabled {
		openshiftRules = append(openshiftRules, rbacv1.PolicyRule{
			// Required for OwnerReferencesPermissionEnforcement (on by default in OpenShift)
			// When we create an InstrumentationConfig, we set the OwnerReference to the related workload.
			// Controller-runtime sets BlockDeletion: true. So with this Admission Plugin we need permission to
			// update finalizers on the workloads so that they can block deletion.
			// see https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#ownerreferencespermissionenforcement
			APIGroups: []string{"apps"},
			Resources: []string{"statefulsets/finalizers", "daemonsets/finalizers", "deployments/finalizers"},
			Verbs:     []string{"update"},
		}, rbacv1.PolicyRule{
			// OpenShift DeploymentConfigs support
			APIGroups: []string{"apps.openshift.io"},
			Resources: []string{"deploymentconfigs", "deploymentconfigs/finalizers"},
			Verbs:     []string{"get", "list", "watch", "update", "patch"},
		})
	}

	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sconsts.InstrumentorClusterRoleName,
		},
		Rules: append([]rbacv1.PolicyRule{
			{ // Used in events reporting for own telemetry
				APIGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"list", "watch", "get"},
			},
			{ // Read instrumentation labels from namespaces
				APIGroups: []string{""},
				Resources: []string{"namespaces"},
				Verbs:     []string{"list", "watch", "get"},
			},
			{ // reconcile rollouts and instrumentation delpoyment status by actual odigos pods
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Read instrumentation labels from statefulsets and apply pod spec changes
				APIGroups: []string{"batch"},
				Resources: []string{"cronjobs"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Read instrumentation labels from daemonsets and apply pod spec changes
				APIGroups: []string{"apps"},
				Resources: []string{"daemonsets"},
				Verbs:     []string{"get", "list", "watch", "update", "patch"},
			},
			{ // Read instrumentation labels from deployments and apply pod spec changes
				APIGroups: []string{"apps"},
				Resources: []string{"deployments"},
				Verbs:     []string{"get", "list", "watch", "update", "patch"},
			},
			{ // Read instrumentation labels from statefulsets and apply pod spec changes
				APIGroups: []string{"apps"},
				Resources: []string{"statefulsets"},
				Verbs:     []string{"get", "list", "watch", "update", "patch"},
			},
			{
				APIGroups: []string{"operator.odigos.io"},
				Resources: []string{"odigos/finalizers"},
				Verbs:     []string{"update"},
			},
			{ // Update the status of the instrumentation configs after device injection
				APIGroups: []string{"odigos.io"},
				Resources: []string{"instrumentationconfigs/status"},
				Verbs:     []string{"get", "patch", "update"},
			},
			{
				APIGroups: []string{"odigos.io"},
				Resources: []string{"instrumentationconfigs"},
				Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch"},
			},
			{
				APIGroups: []string{"odigos.io"},
				Resources: []string{"sources"},
				Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch"},
			},
			{
				APIGroups: []string{"odigos.io"},
				Resources: []string{"sources/finalizers"},
				Verbs:     []string{"update"},
			},
			{
				APIGroups: []string{"admissionregistration.k8s.io"},
				Resources: []string{"mutatingwebhookconfigurations"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups:     []string{"admissionregistration.k8s.io"},
				Resources:     []string{"mutatingwebhookconfigurations"},
				ResourceNames: []string{k8sconsts.InstrumentorSourceMutatingWebhookName, k8sconsts.InstrumentorMutatingWebhookName},
				Verbs:         []string{"update"},
			},
			{
				APIGroups: []string{"admissionregistration.k8s.io"},
				Resources: []string{"validatingwebhookconfigurations"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups:     []string{"admissionregistration.k8s.io"},
				Resources:     []string{"validatingwebhookconfigurations"},
				ResourceNames: []string{k8sconsts.InstrumentorSourceValidatingWebhookName},
				Verbs:         []string{"update"},
			},
			{
				APIGroups: []string{"argoproj.io"},
				Resources: []string{"rollouts"},
				Verbs:     []string{"get", "list", "watch", "patch"},
			},
		}, openshiftRules...),
	}
}

func NewInstrumentorClusterRoleBinding(ns string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sconsts.InstrumentorClusterRoleBindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      k8sconsts.InstrumentorServiceAccountName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     k8sconsts.InstrumentorClusterRoleName,
		},
	}
}

func NewInstrumentorService(ns string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.InstrumentorServiceName,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/name": k8sconsts.InstrumentorAppLabelValue,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "webhook-server",
					Port:       9443,
					TargetPort: intstr.FromInt(9443),
				},
				{
					Name:       "metrics",
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
				},
			},
			Selector: map[string]string{
				"app.kubernetes.io/name": k8sconsts.InstrumentorAppLabelValue,
			},
		},
	}
}

func NewSourceValidatingWebhookConfiguration(ns string) *admissionregistrationv1.ValidatingWebhookConfiguration {
	webhook := &admissionregistrationv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ValidatingWebhookConfiguration",
			APIVersion: "admissionregistration.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sconsts.InstrumentorSourceValidatingWebhookName,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "odigos-source-validating-webhook",
				"app.kubernetes.io/instance":   k8sconsts.InstrumentorSourceValidatingWebhookName,
				"app.kubernetes.io/component":  "webhook",
				"app.kubernetes.io/created-by": "instrumentor",
				"app.kubernetes.io/part-of":    "odigos",
			},
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{
				Name: "odigos-source-validating-webhook.odigos.io",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Name:      k8sconsts.InstrumentorServiceName,
						Namespace: ns,
						Path:      ptrString("/validate-odigos-io-v1alpha1-source"),
						Port:      intPtr(9443),
					},
				},
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1.OperationType{
							admissionregistrationv1.Create,
							admissionregistrationv1.Update,
						},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{"odigos.io"},
							APIVersions: []string{"v1alpha1"},
							Resources:   []string{"sources"},
							Scope:       ptrGeneric(admissionregistrationv1.NamespacedScope),
						},
					},
				},
				FailurePolicy:  ptrGeneric(admissionregistrationv1.Fail),
				SideEffects:    ptrGeneric(admissionregistrationv1.SideEffectClassNone),
				TimeoutSeconds: intPtr(10),
				AdmissionReviewVersions: []string{
					"v1",
				},
			},
		},
	}

	return webhook
}

func NewSourceMutatingWebhookConfiguration(ns string) *admissionregistrationv1.MutatingWebhookConfiguration {
	webhook := &admissionregistrationv1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MutatingWebhookConfiguration",
			APIVersion: "admissionregistration.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sconsts.InstrumentorSourceMutatingWebhookName,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "odigos-source-mutating-webhook",
				"app.kubernetes.io/instance":   k8sconsts.InstrumentorSourceMutatingWebhookName,
				"app.kubernetes.io/component":  "webhook",
				"app.kubernetes.io/created-by": "instrumentor",
				"app.kubernetes.io/part-of":    "odigos",
			},
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: "odigos-source-mutating-webhook.odigos.io",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Name:      k8sconsts.InstrumentorServiceName,
						Namespace: ns,
						Path:      ptrString("/mutate-odigos-io-v1alpha1-source"),
						Port:      intPtr(9443),
					},
				},
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1.OperationType{
							admissionregistrationv1.Create,
							admissionregistrationv1.Update,
						},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{"odigos.io"},
							APIVersions: []string{"v1alpha1"},
							Resources:   []string{"sources"},
							Scope:       ptrGeneric(admissionregistrationv1.NamespacedScope),
						},
					},
				},
				FailurePolicy:      ptrGeneric(admissionregistrationv1.Fail),
				ReinvocationPolicy: ptrGeneric(admissionregistrationv1.NeverReinvocationPolicy),
				SideEffects:        ptrGeneric(admissionregistrationv1.SideEffectClassNone),
				TimeoutSeconds:     intPtr(10),
				AdmissionReviewVersions: []string{
					"v1",
				},
			},
		},
	}

	return webhook
}

func NewPodMutatingWebhookConfiguration(ns string) *admissionregistrationv1.MutatingWebhookConfiguration {
	webhook := &admissionregistrationv1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MutatingWebhookConfiguration",
			APIVersion: "admissionregistration.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sconsts.InstrumentorMutatingWebhookName,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "odigos-pod-mutating-webhook",
				"app.kubernetes.io/instance":   k8sconsts.InstrumentorMutatingWebhookName,
				"app.kubernetes.io/component":  "webhook",
				"app.kubernetes.io/created-by": "instrumentor",
				"app.kubernetes.io/part-of":    "odigos",
				"odigos.io/system-object":      "true",
			},
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: "odigos-pod-mutating-webhook.odigos.io",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Name:      k8sconsts.InstrumentorServiceName,
						Namespace: ns,
						Path:      ptrString("/mutate--v1-pod"),
						Port:      intPtr(9443),
					},
				},
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1.OperationType{
							admissionregistrationv1.Create,
						},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
							Scope:       ptrGeneric(admissionregistrationv1.NamespacedScope),
						},
					},
				},
				FailurePolicy:      ptrGeneric(admissionregistrationv1.Ignore),
				ReinvocationPolicy: ptrGeneric(admissionregistrationv1.NeverReinvocationPolicy),
				SideEffects:        ptrGeneric(admissionregistrationv1.SideEffectClassNone),
				TimeoutSeconds:     intPtr(10),
				AdmissionReviewVersions: []string{
					"v1",
				},
			},
		},
	}

	return webhook
}

func NewInstrumentorTLSSecret(ns string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.InstrumentorWebhookSecretName,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "instrumentor-cert",
				"app.kubernetes.io/instance":   "instrumentor-cert",
				"app.kubernetes.io/component":  "certificate",
				"app.kubernetes.io/created-by": "instrumentor",
				"app.kubernetes.io/part-of":    "odigos",
			},
		},
	}
}

func NewInstrumentorDeployment(ns string, version string, telemetryEnabled bool, imagePrefix string, imageName string, tier common.OdigosTier, nodeSelector map[string]string, initContainerImage string, waspEnabled *bool) *appsv1.Deployment {
	if nodeSelector == nil {
		nodeSelector = make(map[string]string)
	}
	args := []string{
		"--health-probe-bind-address=:8081",
		"--metrics-bind-address=0.0.0.0:8080",
		"--leader-elect",
	}

	if !telemetryEnabled {
		args = append(args, "--telemetry-disabled")
	}

	if waspEnabled != nil && *waspEnabled {
		args = append(args, "--wasp-enabled")
	}

	dynamicEnv := []corev1.EnvVar{}
	if tier == common.CloudOdigosTier {
		dynamicEnv = append(dynamicEnv, odigospro.CloudTokenAsEnvVar())
	} else if tier == common.OnPremOdigosTier {
		dynamicEnv = append(dynamicEnv, odigospro.OnPremTokenAsEnvVar())
	}

	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.InstrumentorDeploymentName,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/name": k8sconsts.InstrumentorAppLabelValue,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrint32(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": k8sconsts.InstrumentorAppLabelValue,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": k8sconsts.InstrumentorAppLabelValue,
					},
					Annotations: map[string]string{
						"kubectl.kubernetes.io/default-container": k8sconsts.InstrumentorContainerName,
					},
				},
				Spec: corev1.PodSpec{
					NodeSelector: nodeSelector,
					Affinity: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
								{
									Weight: 100,
									PodAffinityTerm: corev1.PodAffinityTerm{
										LabelSelector: &metav1.LabelSelector{
											MatchLabels: map[string]string{
												"app.kubernetes.io/name": k8sconsts.InstrumentorAppLabelValue,
											},
										},
										TopologyKey: "kubernetes.io/hostname",
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  k8sconsts.InstrumentorContainerName,
							Image: containers.GetImageName(imagePrefix, imageName, version),
							Command: []string{
								"/app",
							},
							Args: args,
							Env: append([]corev1.EnvVar{
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: k8sconsts.InstrumentorOtelServiceName,
								},
								{
									Name: "CURRENT_NS",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								// This env var is used to set the image (ubi9 or not) of the init container (odigos-agents)
								{
									Name:  k8sconsts.OdigosInitContainerEnvVarName,
									Value: containers.GetImageName(imagePrefix, initContainerImage, version),
								},
								// TODO: this tier env var should be removed once we complete the transition to
								// enterprise and community images, and the webhook code won't rely on this env var
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
							}, dynamicEnv...),
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: ownTelemetryOtelConfig,
										},
									},
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "webhook-server",
									ContainerPort: 9443,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      k8sconsts.InstrumentorWebhookVolumeName,
									ReadOnly:  true,
									MountPath: "/tmp/k8s-webhook-server/serving-certs",
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
											IntVal: 8081,
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
					ServiceAccountName:            k8sconsts.InstrumentorServiceAccountName,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: ptrbool(true),
					},
					Volumes: []corev1.Volume{
						{
							Name: k8sconsts.InstrumentorWebhookVolumeName,
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName:  k8sconsts.InstrumentorWebhookSecretName,
									DefaultMode: ptrint32(420),
								},
							},
						},
					},
				},
			},
			Strategy:        appsv1.DeploymentStrategy{},
			MinReadySeconds: 0,
		},
	}

	return dep
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
	client        *kube.Client
	ns            string
	config        *common.OdigosConfiguration
	odigosVersion string
	tier          common.OdigosTier
	managerOpts   resourcemanager.ManagerOpts
}

func NewInstrumentorResourceManager(client *kube.Client, ns string, config *common.OdigosConfiguration, tier common.OdigosTier, odigosVersion string, managerOpts resourcemanager.ManagerOpts) resourcemanager.ResourceManager {
	return &instrumentorResourceManager{
		client:        client,
		ns:            ns,
		config:        config,
		odigosVersion: odigosVersion,
		tier:          tier,
		managerOpts:   managerOpts,
	}
}

func (a *instrumentorResourceManager) Name() string { return "Instrumentor" }

func (a *instrumentorResourceManager) InstallFromScratch(ctx context.Context) error {
	resources := []kube.Object{
		NewInstrumentorServiceAccount(a.ns),
		NewInstrumentorLeaderElectionRoleBinding(a.ns),
		NewInstrumentorRole(a.ns),
		NewInstrumentorRoleBinding(a.ns),
		NewInstrumentorClusterRole(a.config.OpenshiftEnabled),
		NewInstrumentorClusterRoleBinding(a.ns),
		NewInstrumentorDeployment(a.ns, a.odigosVersion, a.config.TelemetryEnabled, a.config.ImagePrefix, a.managerOpts.ImageReferences.InstrumentorImage, a.tier, a.config.NodeSelector, a.managerOpts.ImageReferences.InitContainerImage, a.config.WaspEnabled),
		NewInstrumentorService(a.ns),
	}

	resources = append([]kube.Object{NewInstrumentorTLSSecret(a.ns),
		NewPodMutatingWebhookConfiguration(a.ns),
		NewSourceMutatingWebhookConfiguration(a.ns),
		NewSourceValidatingWebhookConfiguration(a.ns),
	},
		resources...)

	return a.client.ApplyResources(ctx, a.config.ConfigVersion, resources, a.managerOpts)
}
