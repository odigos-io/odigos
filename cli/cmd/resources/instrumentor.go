package resources

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources/odigospro"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/containers"
	"github.com/odigos-io/odigos/cli/pkg/crypto"
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

func NewInstrumentorClusterRole(ownerPermissionEnforcement bool) *rbacv1.ClusterRole {
	finalizersUpdate := []rbacv1.PolicyRule{}
	if ownerPermissionEnforcement {
		finalizersUpdate = append(finalizersUpdate, rbacv1.PolicyRule{
			// Required for OwnerReferencesPermissionEnforcement (on by default in OpenShift)
			// When we create an InstrumentationConfig, we set the OwnerReference to the related workload.
			// Controller-runtime sets BlockDeletion: true. So with this Admission Plugin we need permission to
			// update finalizers on the workloads so that they can block deletion.
			// see https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#ownerreferencespermissionenforcement
			APIGroups: []string{"apps"},
			Resources: []string{"statefulsets/finalizers", "daemonsets/finalizers", "deployments/finalizers"},
			Verbs:     []string{"update"},
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
			{ // React to runtime detection in user workloads in all namespaces
				APIGroups: []string{"odigos.io"},
				Resources: []string{"instrumentedapplications"},
				Verbs:     []string{"delete", "get", "list", "watch"},
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
		}, finalizersUpdate...),
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
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "webhook-server",
					Port:       9443,
					TargetPort: intstr.FromInt(9443),
				},
			},
			Selector: map[string]string{
				"app.kubernetes.io/name": k8sconsts.InstrumentorAppLabelValue,
			},
		},
	}
}

func NewSourceValidatingWebhookConfiguration(ns string, caBundle []byte) *admissionregistrationv1.ValidatingWebhookConfiguration {
	webhook := &admissionregistrationv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ValidatingWebhookConfiguration",
			APIVersion: "admissionregistration.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sconsts.InstrumentorSourceValidatingWebhookName,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "source-validating-webhook",
				"app.kubernetes.io/instance":   k8sconsts.InstrumentorSourceValidatingWebhookName,
				"app.kubernetes.io/component":  "webhook",
				"app.kubernetes.io/created-by": "instrumentor",
				"app.kubernetes.io/part-of":    "odigos",
			},
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{
				Name: "source-validating-webhook.odigos.io",
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
				FailurePolicy:  ptrGeneric(admissionregistrationv1.Ignore),
				SideEffects:    ptrGeneric(admissionregistrationv1.SideEffectClassNone),
				TimeoutSeconds: intPtr(10),
				AdmissionReviewVersions: []string{
					"v1",
				},
			},
		},
	}

	if caBundle == nil {
		webhook.Annotations = map[string]string{
			"cert-manager.io/inject-ca-from": fmt.Sprintf("%s/serving-cert", ns),
		}
	} else {
		webhook.Webhooks[0].ClientConfig.CABundle = caBundle
	}

	return webhook
}

func NewSourceMutatingWebhookConfiguration(ns string, caBundle []byte) *admissionregistrationv1.MutatingWebhookConfiguration {
	webhook := &admissionregistrationv1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MutatingWebhookConfiguration",
			APIVersion: "admissionregistration.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sconsts.InstrumentorSourceMutatingWebhookName,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "source-mutating-webhook",
				"app.kubernetes.io/instance":   k8sconsts.InstrumentorSourceMutatingWebhookName,
				"app.kubernetes.io/component":  "webhook",
				"app.kubernetes.io/created-by": "instrumentor",
				"app.kubernetes.io/part-of":    "odigos",
			},
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: "source-mutating-webhook.odigos.io",
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
				FailurePolicy:      ptrGeneric(admissionregistrationv1.Ignore),
				ReinvocationPolicy: ptrGeneric(admissionregistrationv1.IfNeededReinvocationPolicy),
				SideEffects:        ptrGeneric(admissionregistrationv1.SideEffectClassNone),
				TimeoutSeconds:     intPtr(10),
				AdmissionReviewVersions: []string{
					"v1",
				},
			},
		},
	}

	if caBundle == nil {
		webhook.Annotations = map[string]string{
			"cert-manager.io/inject-ca-from": fmt.Sprintf("%s/serving-cert", ns),
		}
	} else {
		webhook.Webhooks[0].ClientConfig.CABundle = caBundle
	}

	return webhook
}

func NewPodMutatingWebhookConfiguration(ns string, caBundle []byte) *admissionregistrationv1.MutatingWebhookConfiguration {
	webhook := &admissionregistrationv1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MutatingWebhookConfiguration",
			APIVersion: "admissionregistration.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sconsts.InstrumentorMutatingWebhookName,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "pod-mutating-webhook",
				"app.kubernetes.io/instance":   k8sconsts.InstrumentorMutatingWebhookName,
				"app.kubernetes.io/component":  "webhook",
				"app.kubernetes.io/created-by": "instrumentor",
				"app.kubernetes.io/part-of":    "odigos",
			},
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: "pod-mutating-webhook.odigos.io",
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
							admissionregistrationv1.Update,
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
				ReinvocationPolicy: ptrGeneric(admissionregistrationv1.IfNeededReinvocationPolicy),
				SideEffects:        ptrGeneric(admissionregistrationv1.SideEffectClassNone),
				TimeoutSeconds:     intPtr(10),
				AdmissionReviewVersions: []string{
					"v1",
				},
			},
		},
	}

	if caBundle == nil {
		webhook.Annotations = map[string]string{
			"cert-manager.io/inject-ca-from": fmt.Sprintf("%s/serving-cert", ns),
		}
	} else {
		webhook.Webhooks[0].ClientConfig.CABundle = caBundle
	}

	return webhook
}

func NewInstrumentorTLSSecret(ns string, cert *crypto.Certificate) *corev1.Secret {
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
			Annotations: map[string]string{
				"helm.sh/hook":               "pre-install,pre-upgrade",
				"helm.sh/hook-delete-policy": "before-hook-creation",
			},
		},
		Data: map[string][]byte{
			"tls.crt": []byte(cert.Cert),
			"tls.key": []byte(cert.Key),
		},
	}
}

func NewInstrumentorDeployment(ns string, version string, telemetryEnabled bool, imagePrefix string, imageName string, tier common.OdigosTier) *appsv1.Deployment {
	args := []string{
		"--health-probe-bind-address=:8081",
		"--metrics-bind-address=127.0.0.1:8080",
		"--leader-elect",
	}

	if !telemetryEnabled {
		args = append(args, "--telemetry-disabled")
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
								TimeoutSeconds:      0,
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
								PeriodSeconds: 10,
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
		NewInstrumentorDeployment(a.ns, a.odigosVersion, a.config.TelemetryEnabled, a.config.ImagePrefix, a.managerOpts.ImageReferences.InstrumentorImage, a.tier),
		NewInstrumentorService(a.ns),
	}

	ca, err := crypto.GenCA(k8sconsts.InstrumentorCertificateName, 365)
	if err != nil {
		return fmt.Errorf("failed to generate CA: %w", err)
	}

	altNames := []string{
		fmt.Sprintf("%s.%s.svc", k8sconsts.InstrumentorServiceName, a.ns),
		fmt.Sprintf("%s.%s.svc.cluster.local", k8sconsts.InstrumentorServiceName, a.ns),
	}

	cert, err := crypto.GenerateSignedCertificate("serving-cert", nil, altNames, 365, ca)
	if err != nil {
		return fmt.Errorf("failed to generate signed certificate: %w", err)
	}

	resources = append([]kube.Object{NewInstrumentorTLSSecret(a.ns, &cert),
		NewPodMutatingWebhookConfiguration(a.ns, []byte(cert.Cert)),
		NewSourceMutatingWebhookConfiguration(a.ns, []byte(cert.Cert)),
		NewSourceValidatingWebhookConfiguration(a.ns, []byte(cert.Cert)),
	},
		resources...)

	return a.client.ApplyResources(ctx, a.config.ConfigVersion, resources, a.managerOpts)
}
