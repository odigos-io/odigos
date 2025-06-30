package resources

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
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

func NewAutoscalerServiceAccount(ns string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.AutoScalerServiceAccountName,
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
			Name:      k8sconsts.AutoScalerRoleName,
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
			{ // Needed to update the webhook certificates and manage rotation
				APIGroups:     []string{""},
				Resources:     []string{"secrets"},
				ResourceNames: []string{k8sconsts.AutoscalerWebhookSecretName},
				Verbs:         []string{"update"},
			},
			{ // Needed to migrate away from the old secret
				APIGroups:     []string{""},
				Resources:     []string{"secrets"},
				ResourceNames: []string{k8sconsts.DeprecatedAutoscalerWebhookSecretName},
				Verbs:         []string{"delete"},
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
			Name:      k8sconsts.AutoScalerRoleBindingName,
			Namespace: ns,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: k8sconsts.AutoScalerServiceAccountName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     k8sconsts.AutoScalerRoleName,
		},
	}
}

func NewAutoscalerClusterRole(ownerPermissionEnforcement bool) *rbacv1.ClusterRole {
	finalizersUpdate := []rbacv1.PolicyRule{}
	if ownerPermissionEnforcement {
		finalizersUpdate = append(finalizersUpdate, rbacv1.PolicyRule{
			// Required for OwnerReferencesPermissionEnforcement (on by default in OpenShift)
			// When we create a collector COnfigMap, we set the OwnerReference to the collectorsgroups.
			// Controller-runtime sets BlockDeletion: true. So with this Admission Plugin we need permission to
			// update finalizers on the collectorsgroup so that they can block deletion.
			// seehttps://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#ownerreferencespermissionenforcement
			APIGroups: []string{"odigos.io"},
			Resources: []string{"collectorsgroups/finalizers"},
			Verbs:     []string{"update"},
		})
	}

	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sconsts.AutoScalerClusterRoleName,
		},
		Rules: append([]rbacv1.PolicyRule{
			{ // Needed to read the applications, to populate the receivers.filelog in the data-collector configmap
				APIGroups: []string{"odigos.io"},
				Resources: []string{"instrumentationconfigs"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Needed to watch actions in order to transform them to processors
				APIGroups: []string{"odigos.io"},
				Resources: []string{"actions"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Update conditions of the action after transforming it to a processor
				APIGroups: []string{"odigos.io"},
				Resources: []string{"actions/status"},
				Verbs:     []string{"get", "patch", "update"},
			},
			{ // Cert controller syncs the webhooks certificates with the secret, require for reconciler to watch the webhooks configs
				APIGroups: []string{"admissionregistration.k8s.io"},
				Resources: []string{"validatingwebhookconfigurations"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Needed to update the webhook configuration with the new CA bundle when the certs are rotated
				APIGroups:     []string{"admissionregistration.k8s.io"},
				Resources:     []string{"validatingwebhookconfigurations"},
				ResourceNames: []string{k8sconsts.AutoscalerActionValidatingWebhookName},
				Verbs:         []string{"update"},
			},
			// Needed to read the sources for build the odigos routing processor
			{
				APIGroups: []string{"odigos.io"},
				Resources: []string{"sources"},
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
			},
		}, finalizersUpdate...),
	}
}

func NewAutoscalerClusterRoleBinding(ns string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sconsts.AutoScalerClusterRoleBindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      k8sconsts.AutoScalerServiceAccountName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     k8sconsts.AutoScalerClusterRoleName,
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
				Name: k8sconsts.AutoScalerServiceAccountName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "odigos-leader-election-role",
		},
	}
}

func NewAutoscalerDeployment(ns string, version string, imagePrefix string, imageName string, collectorImage string, nodeSelector map[string]string) *appsv1.Deployment {

	optionalEnvs := []corev1.EnvVar{}
	if nodeSelector == nil {
		nodeSelector = make(map[string]string)
	}

	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.AutoScalerDeploymentName,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/name": k8sconsts.AutoScalerAppLabelValue,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrint32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": k8sconsts.AutoScalerAppLabelValue,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": k8sconsts.AutoScalerAppLabelValue,
					},
					Annotations: map[string]string{
						"kubectl.kubernetes.io/default-container": k8sconsts.AutoScalerContainerName,
					},
				},
				Spec: corev1.PodSpec{
					NodeSelector: nodeSelector,
					Containers: []corev1.Container{
						{
							Name:  k8sconsts.AutoScalerContainerName,
							Image: containers.GetImageName(imagePrefix, imageName, version),
							Command: []string{
								"/app",
							},
							Args: []string{
								"--health-probe-bind-address=:8081",
								"--metrics-bind-address=0.0.0.0:8080",
								"--leader-elect",
							},
							Env: append([]corev1.EnvVar{
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: k8sconsts.AutoScalerServiceName,
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
									Name:  "ODIGOS_COLLECTOR_IMAGE",
									Value: containers.GetImageName(imagePrefix, collectorImage, version),
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
							Ports: []corev1.ContainerPort{
								{
									Name:          "webhook-server",
									ContainerPort: 9443,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      k8sconsts.AutoscalerWebhookVolumeName,
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
					ServiceAccountName:            k8sconsts.AutoScalerServiceAccountName,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: ptrbool(true),
					},
					Volumes: []corev1.Volume{
						{
							Name: k8sconsts.AutoscalerWebhookVolumeName,
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName:  k8sconsts.AutoscalerWebhookSecretName,
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

func NewAutoscalerService(ns string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.AutoScalerWebhookServiceName,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/name": k8sconsts.AutoScalerAppLabelValue,
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
				"app.kubernetes.io/name": k8sconsts.AutoScalerAppLabelValue,
			},
		},
	}
}

func NewAutoscalerTLSSecret(ns string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.AutoscalerWebhookSecretName,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "autoscaler-cert",
				"app.kubernetes.io/instance":   "autoscaler-cert",
				"app.kubernetes.io/component":  "certificate",
				"app.kubernetes.io/created-by": "autoscaler",
				"app.kubernetes.io/part-of":    "odigos",
			},
		},
	}
}

func NewActionValidatingWebhookConfiguration(ns string) *admissionregistrationv1.ValidatingWebhookConfiguration {
	webhook := &admissionregistrationv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ValidatingWebhookConfiguration",
			APIVersion: "admissionregistration.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sconsts.AutoscalerActionValidatingWebhookName,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "action-validating-webhook",
				"app.kubernetes.io/instance":   k8sconsts.AutoscalerActionValidatingWebhookName,
				"app.kubernetes.io/component":  "webhook",
				"app.kubernetes.io/created-by": "autoscaler",
				"app.kubernetes.io/part-of":    "odigos",
			},
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{
				Name: "action-validating-webhook.odigos.io",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Name:      k8sconsts.AutoScalerWebhookServiceName,
						Namespace: ns,
						Path:      ptrString("/validate-odigos-io-v1alpha1-action"),
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
							Resources:   []string{"actions"},
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

	return webhook
}

type autoScalerResourceManager struct {
	client        *kube.Client
	ns            string
	config        *common.OdigosConfiguration
	odigosVersion string
	managerOpts   resourcemanager.ManagerOpts
}

func NewAutoScalerResourceManager(client *kube.Client, ns string, config *common.OdigosConfiguration, odigosVersion string, managerOpts resourcemanager.ManagerOpts) resourcemanager.ResourceManager {
	return &autoScalerResourceManager{client: client, ns: ns, config: config, odigosVersion: odigosVersion, managerOpts: managerOpts}
}

func (a *autoScalerResourceManager) Name() string { return "AutoScaler" }

func (a *autoScalerResourceManager) InstallFromScratch(ctx context.Context) error {
	resources := []kube.Object{
		NewAutoscalerServiceAccount(a.ns),
		NewAutoscalerRole(a.ns),
		NewAutoscalerRoleBinding(a.ns),
		NewAutoscalerClusterRole(a.config.OpenshiftEnabled),
		NewAutoscalerClusterRoleBinding(a.ns),
		NewAutoscalerLeaderElectionRoleBinding(a.ns),
		NewAutoscalerDeployment(a.ns, a.odigosVersion, a.config.ImagePrefix, a.managerOpts.ImageReferences.AutoscalerImage, a.managerOpts.ImageReferences.CollectorImage, a.config.NodeSelector),
		NewAutoscalerService(a.ns),
		NewAutoscalerTLSSecret(a.ns),
		NewActionValidatingWebhookConfiguration(a.ns),
	}
	return a.client.ApplyResources(ctx, a.config.ConfigVersion, resources, a.managerOpts)
}
