package resources

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/containers"
	"github.com/odigos-io/odigos/cli/pkg/crypto"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"

	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	InstrumentorOtelServiceName        = "instrumentor"
	InstrumentorDeploymentName         = "odigos-instrumentor"
	InstrumentorAppLabelValue          = InstrumentorDeploymentName
	InstrumentorServiceName            = InstrumentorDeploymentName
	InstrumentorServiceAccountName     = InstrumentorDeploymentName
	InstrumentorRoleName               = InstrumentorDeploymentName
	InstrumentorRoleBindingName        = InstrumentorDeploymentName
	InstrumentorClusterRoleName        = InstrumentorDeploymentName
	InstrumentorClusterRoleBindingName = InstrumentorDeploymentName
	InstrumentorCertificateName        = InstrumentorDeploymentName
	InstrumentorMutatingWebhookName    = "mutating-webhook-configuration"
	InstrumentorContainerName          = "manager"
	InstrumentorWebhookSecretName      = "instrumentor-webhook-cert"
	InstrumentorWebhookVolumeName      = "webhook-cert"
)

func NewInstrumentorServiceAccount(ns string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      InstrumentorServiceAccountName,
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
				Name: InstrumentorServiceAccountName,
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
			Name:      InstrumentorRoleName,
			Namespace: ns,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"configmaps"},
				ResourceNames: []string{"odigos-config"},
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
			Name:      InstrumentorRoleBindingName,
			Namespace: ns,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: InstrumentorServiceAccountName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     InstrumentorRoleName,
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
			Name: InstrumentorClusterRoleName,
		},
		Rules: []rbacv1.PolicyRule{
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
			{ // React to runtime detection in user workloads in all namespaces
				APIGroups: []string{"odigos.io"},
				Resources: []string{"instrumentedapplications"},
				Verbs:     []string{"delete", "get", "list", "watch"},
			},
			{ // Update the status of the instrumented applications after device injection
				APIGroups: []string{"odigos.io"},
				Resources: []string{"instrumentedapplications/status"},
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
				APIGroups: []string{"odigos.io"},
				Resources: []string{"sources"},
				Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch"},
			},
			{
				APIGroups: []string{"odigos.io"},
				Resources: []string{"sources/finalizers"},
				Verbs:     []string{"update"},
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
			Name: InstrumentorClusterRoleBindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      InstrumentorServiceAccountName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     InstrumentorClusterRoleName,
		},
	}
}

func isCertManagerInstalled(ctx context.Context, c *kube.Client) bool {
	// Check if CRD is installed
	_, err := c.ApiExtensions.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, "issuers.cert-manager.io", metav1.GetOptions{})
	if err != nil {
		return false
	}

	return true
}

func NewInstrumentorIssuer(ns string) *certv1.Issuer {
	return &certv1.Issuer{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Issuer",
			APIVersion: "cert-manager.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "selfsigned-issuer",
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "issuer",
				"app.kubernetes.io/instance":   "selfsigned-issuer",
				"app.kubernetes.io/component":  "certificate",
				"app.kubernetes.io/created-by": "instrumentor",
				"app.kubernetes.io/part-of":    "odigos",
			},
		},
		Spec: certv1.IssuerSpec{
			IssuerConfig: certv1.IssuerConfig{
				SelfSigned: &certv1.SelfSignedIssuer{},
			},
		},
	}
}

func NewInstrumentorCertificate(ns string) *certv1.Certificate {
	return &certv1.Certificate{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Certificate",
			APIVersion: "cert-manager.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "serving-cert",
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "instrumentor-cert",
				"app.kubernetes.io/instance":   "instrumentor-cert",
				"app.kubernetes.io/component":  "certificate",
				"app.kubernetes.io/created-by": "instrumentor",
				"app.kubernetes.io/part-of":    "odigos",
			},
		},
		Spec: certv1.CertificateSpec{
			DNSNames: []string{
				fmt.Sprintf("odigos-instrumentor.%s.svc", ns),
				fmt.Sprintf("odigos-instrumentor.%s.svc.cluster.local", ns),
			},
			IssuerRef: cmmeta.ObjectReference{
				Kind: "Issuer",
				Name: "selfsigned-issuer",
			},
			SecretName: InstrumentorWebhookSecretName,
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
			Name:      InstrumentorServiceName,
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
				"app.kubernetes.io/name": InstrumentorAppLabelValue,
			},
		},
	}
}

func NewMutatingWebhookConfiguration(ns string, caBundle []byte) *admissionregistrationv1.MutatingWebhookConfiguration {
	webhook := &admissionregistrationv1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MutatingWebhookConfiguration",
			APIVersion: "admissionregistration.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: InstrumentorMutatingWebhookName,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "pod-mutating-webhook",
				"app.kubernetes.io/instance":   InstrumentorMutatingWebhookName,
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
						Name:      InstrumentorServiceName,
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
				ObjectSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						consts.OdigosInjectInstrumentationLabel: "true",
					},
				},
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
			Name:      InstrumentorWebhookSecretName,
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

func NewInstrumentorDeployment(ns string, version string, telemetryEnabled bool, imagePrefix string, imageName string) *appsv1.Deployment {
	args := []string{
		"--health-probe-bind-address=:8081",
		"--metrics-bind-address=127.0.0.1:8080",
		"--leader-elect",
	}

	if !telemetryEnabled {
		args = append(args, "--telemetry-disabled")
	}

	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      InstrumentorDeploymentName,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/name": InstrumentorAppLabelValue,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrint32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": InstrumentorAppLabelValue,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": InstrumentorAppLabelValue,
					},
					Annotations: map[string]string{
						"kubectl.kubernetes.io/default-container": InstrumentorContainerName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  InstrumentorContainerName,
							Image: containers.GetImageName(imagePrefix, imageName, version),
							Command: []string{
								"/app",
							},
							Args: args,
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: InstrumentorOtelServiceName,
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
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: consts.OdigosDeploymentConfigMapName,
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
									Name:      InstrumentorWebhookVolumeName,
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
							SecurityContext: &corev1.SecurityContext{},
						},
					},
					TerminationGracePeriodSeconds: ptrint64(10),
					ServiceAccountName:            InstrumentorServiceAccountName,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: ptrbool(true),
					},
					Volumes: []corev1.Volume{
						{
							Name: InstrumentorWebhookVolumeName,
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName:  InstrumentorWebhookSecretName,
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
}

func NewInstrumentorResourceManager(client *kube.Client, ns string, config *common.OdigosConfiguration, odigosVersion string) resourcemanager.ResourceManager {
	return &instrumentorResourceManager{
		client:        client,
		ns:            ns,
		config:        config,
		odigosVersion: odigosVersion,
	}
}

func (a *instrumentorResourceManager) Name() string { return "Instrumentor" }

func (a *instrumentorResourceManager) InstallFromScratch(ctx context.Context) error {
	certManagerInstalled := isCertManagerInstalled(ctx, a.client)
	resources := []kube.Object{
		NewInstrumentorServiceAccount(a.ns),
		NewInstrumentorLeaderElectionRoleBinding(a.ns),
		NewInstrumentorRole(a.ns),
		NewInstrumentorRoleBinding(a.ns),
		NewInstrumentorClusterRole(),
		NewInstrumentorClusterRoleBinding(a.ns),
		NewInstrumentorDeployment(a.ns, a.odigosVersion, a.config.TelemetryEnabled, a.config.ImagePrefix, a.config.InstrumentorImage),
		NewInstrumentorService(a.ns),
	}

	if certManagerInstalled {
		resources = append([]kube.Object{NewInstrumentorIssuer(a.ns),
			NewInstrumentorCertificate(a.ns),
			NewMutatingWebhookConfiguration(a.ns, nil),
		},
			resources...)
	} else {
		ca, err := crypto.GenCA(InstrumentorCertificateName, 365)
		if err != nil {
			return fmt.Errorf("failed to generate CA: %w", err)
		}

		altNames := []string{
			fmt.Sprintf("%s.%s.svc", InstrumentorServiceName, a.ns),
			fmt.Sprintf("%s.%s.svc.cluster.local", InstrumentorServiceName, a.ns),
		}

		cert, err := crypto.GenerateSignedCertificate("serving-cert", nil, altNames, 365, ca)
		if err != nil {
			return fmt.Errorf("failed to generate signed certificate: %w", err)
		}

		resources = append([]kube.Object{NewInstrumentorTLSSecret(a.ns, &cert),
			NewMutatingWebhookConfiguration(a.ns, []byte(cert.Cert)),
		},
			resources...)
	}

	return a.client.ApplyResources(ctx, a.config.ConfigVersion, resources)
}
