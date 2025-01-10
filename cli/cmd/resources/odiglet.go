package resources

import (
	"context"
	"slices"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/odigos-io/odigos/cli/pkg/autodetect"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/common/consts"

	"github.com/odigos-io/odigos/cli/cmd/resources/odigospro"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/containers"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sversion "k8s.io/apimachinery/pkg/util/version"
)

const (
	OdigletDaemonSetName          = "odiglet"
	OdigletAppLabelValue          = OdigletDaemonSetName
	OdigletServiceAccountName     = OdigletDaemonSetName
	OdigletRoleName               = OdigletDaemonSetName
	OdigletRoleBindingName        = OdigletDaemonSetName
	OdigletClusterRoleName        = OdigletDaemonSetName
	OdigletClusterRoleBindingName = OdigletDaemonSetName
	OdigletContainerName          = "odiglet"
	OdigletImageName              = "keyval/odigos-odiglet"
	OdigletEnterpriseImageName    = "keyval/odigos-enterprise-odiglet"
)

func NewOdigletServiceAccount(ns string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      OdigletServiceAccountName,
			Namespace: ns,
		},
	}
}

func NewOdigletRole(ns string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      OdigletRoleName,
			Namespace: ns,
		},
		Rules: []rbacv1.PolicyRule{
			{
				// Needed for reading the enabled signals for each source
				// TODO: rely on inctr. config instead of collectorsgroups, then remove this
				APIGroups: []string{"odigos.io"},
				Resources: []string{"collectorsgroups", "collectorsgroups/status"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Needed to read the odigos_config for ignored containers
				APIGroups:     []string{""},
				Resources:     []string{"configmaps"},
				ResourceNames: []string{consts.OdigosConfigurationName},
				Verbs:         []string{"get", "list", "watch"},
			},
		},
	}
}

func NewOdigletRoleBinding(ns string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      OdigletRoleBindingName,
			Namespace: ns,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      OdigletServiceAccountName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     OdigletRoleName,
		},
	}
}

func NewOdigletClusterRole(psp bool) *rbacv1.ClusterRole {
	clusterrole := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: OdigletClusterRoleName,
		},
		Rules: []rbacv1.PolicyRule{
			{ // Needed for language detection
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Needed for language detection
				APIGroups: []string{""},
				Resources: []string{"pods/status"},
				Verbs:     []string{"get"},
			},
			{ // Needed for language detection
				// TODO: remove this once Tamir/PR is read for new language detection
				APIGroups: []string{"apps"},
				Resources: []string{"deployments", "daemonsets", "statefulsets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Needed for language detection
				// TODO: remove this once Tamir/PR is read for new language detection
				APIGroups: []string{"apps"},
				Resources: []string{"deployments/status", "daemonsets/status", "statefulsets/status"},
				Verbs:     []string{"get"},
			},
			{ // Needed for virtual device registration
				APIGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Needed for storage of the process instrumentation state
				APIGroups: []string{"odigos.io"},
				Resources: []string{"instrumentationinstances"},
				Verbs:     []string{"create", "get", "list", "patch", "update", "watch", "delete"},
			},
			{ // Needed for storage of the process instrumentation state
				APIGroups: []string{"odigos.io"},
				Resources: []string{"instrumentationinstances/status"},
				Verbs:     []string{"get", "patch", "update"},
			},
			{ // Need for storage of runtime details / language detection (future update)
				APIGroups: []string{"odigos.io"},
				Resources: []string{"instrumentationconfigs"},
				Verbs:     []string{"get", "list", "watch", "patch", "update"},
			},
			{ // Need for storage of runtime details / language detection (future update)
				APIGroups: []string{"odigos.io"},
				Resources: []string{"instrumentationconfigs/status"},
				Verbs:     []string{"get", "patch", "update"},
			},
		},
	}

	if psp {
		clusterrole.Rules = append(clusterrole.Rules, rbacv1.PolicyRule{
			// Needed for clients who enable pod security policies
			APIGroups:     []string{"policy"},
			Resources:     []string{"podsecuritypolicies"},
			ResourceNames: []string{"privileged"},
			Verbs:         []string{"use"},
		})
	}

	return clusterrole
}

func NewOdigletClusterRoleBinding(ns string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: OdigletClusterRoleBindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      OdigletServiceAccountName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     OdigletClusterRoleName,
		},
	}
}

func NewSCCRoleBinding(ns string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "system:openshift:scc:privileged",
			Namespace: ns,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      OdigletServiceAccountName,
				Namespace: ns,
			},
			{
				Kind:      "ServiceAccount",
				Name:      "odigos-data-collection",
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "system:openshift:scc:privileged",
		},
	}
}

func NewSCClusterRoleBinding(ns string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "system:openshift:scc:anyuid:" + ns,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "Group",
				Name:      "system:serviceaccounts:" + ns,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "system:openshift:scc:anyuid",
		},
	}
}

func NewResourceQuota(ns string) *corev1.ResourceQuota {
	return &corev1.ResourceQuota{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ResourceQuota",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "odigos-quota",
			Namespace: ns,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				"pods": resource.MustParse("10k"),
			},
			ScopeSelector: &corev1.ScopeSelector{
				MatchExpressions: []corev1.ScopedResourceSelectorRequirement{
					{
						ScopeName: corev1.ResourceQuotaScopePriorityClass,
						Operator:  corev1.ScopeSelectorOpIn,
						Values:    []string{"system-node-critical"},
					},
				},
			},
		},
	}
}

func NewOdigletDaemonSet(ns string, version string, imagePrefix string, imageName string, odigosTier common.OdigosTier, openshiftEnabled bool, goAutoIncludeCodeAttributes bool, clusterDetails *autodetect.ClusterDetails) *appsv1.DaemonSet {

	dynamicEnv := []corev1.EnvVar{}
	if odigosTier == common.CloudOdigosTier {
		dynamicEnv = append(dynamicEnv, odigospro.CloudTokenAsEnvVar())
	} else if odigosTier == common.OnPremOdigosTier {
		dynamicEnv = append(dynamicEnv, odigospro.OnPremTokenAsEnvVar())
	}

	if goAutoIncludeCodeAttributes {
		dynamicEnv = append(dynamicEnv, corev1.EnvVar{
			Name:  "OTEL_GO_AUTO_INCLUDE_CODE_ATTRIBUTES",
			Value: "true",
		})
	}

	odigosSeLinuxHostVolumes := []corev1.Volume{}
	odigosSeLinuxHostVolumeMounts := []corev1.VolumeMount{}
	if openshiftEnabled || clusterDetails.Kind == autodetect.KindOpenShift {
		odigosSeLinuxHostVolumes = append(odigosSeLinuxHostVolumes, selinuxHostVolumes()...)
		odigosSeLinuxHostVolumeMounts = append(odigosSeLinuxHostVolumeMounts, selinuxHostVolumeMounts()...)
	}

	// 50% of the nodes can be unavailable during the update.
	// if we do not set it, the default value is 1.
	// 1 means that if 1 daemonset pod fails to update, the whole rollout will be broken.
	// this can happen when a single node has memory pressure, scheduling issues, not enough resources, etc.
	// by setting it to 50% we can tolerate more failures and the rollout will be more stable.
	maxUnavailable := intstr.FromString("50%")
	// maxSurge is the number of pods that can be created above the desired number of pods.
	// we do not want more then 1 odiglet pod on the same node as it is not supported by the eBPF.
	// Only set maxSurge if Kubernetes version is >= 1.22
	// Prepare RollingUpdate based on version support for maxSurge
	rollingUpdate := &appsv1.RollingUpdateDaemonSet{
		MaxUnavailable: &maxUnavailable,
	}
	k8sversionInCluster := clusterDetails.K8SVersion
	if k8sversionInCluster != nil && k8sversionInCluster.AtLeast(k8sversion.MustParse("v1.22")) {
		maxSurge := intstr.FromInt(0)
		rollingUpdate.MaxSurge = &maxSurge
	}

	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      OdigletDaemonSetName,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/name": OdigletAppLabelValue,
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": OdigletAppLabelValue,
				},
			},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type:          appsv1.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: rollingUpdate,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": OdigletAppLabelValue,
					},
				},
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"kubernetes.io/os": "linux",
					},
					Tolerations: []corev1.Toleration{
						{
							// This toleration with 'Exists' operator and no key/effect specified
							// will match ALL taints, allowing pods to be scheduled on any node
							// regardless of its taints (including master/control-plane nodes)
							Operator: corev1.TolerationOpExists,
						},
					},
					Volumes: append([]corev1.Volume{
						{
							Name: "run-dir",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/run",
								},
							},
						},
						{
							Name: "pod-resources",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/pod-resources",
								},
							},
						},
						{
							Name: "device-plugins-dir",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/device-plugins",
								},
							},
						},
						{
							Name: "odigos",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/odigos",
								},
							},
						},
						{
							Name: "kernel-debug",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/sys/kernel/debug",
								},
							},
						},
					}, odigosSeLinuxHostVolumes...),
					InitContainers: []corev1.Container{
						{
							Name:  "init",
							Image: containers.GetImageName(imagePrefix, imageName, version),
							Command: []string{
								"/root/odiglet",
							},
							Args: []string{
								"init",
							},
							Resources: corev1.ResourceRequirements{},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "odigos",
									MountPath: "/var/odigos",
								},
							},
							ImagePullPolicy: "IfNotPresent",
						},
					},
					Containers: []corev1.Container{
						{
							Name:  OdigletContainerName,
							Image: containers.GetImageName(imagePrefix, imageName, version),
							Env: append([]corev1.EnvVar{
								{
									Name: "NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name: "NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
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
									Name:  "OTEL_LOG_LEVEL",
									Value: "info",
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
							Resources: corev1.ResourceRequirements{},
							VolumeMounts: append([]corev1.VolumeMount{
								{
									Name:      "run-dir",
									MountPath: "/run",
								},
								{
									Name:      "device-plugins-dir",
									MountPath: "/var/lib/kubelet/device-plugins",
								},
								{
									Name:      "pod-resources",
									MountPath: "/var/lib/kubelet/pod-resources",
									ReadOnly:  true,
								},
								{
									Name:      "odigos",
									MountPath: "/var/odigos",
									ReadOnly:  true,
								},
								{
									Name:      "kernel-debug",
									MountPath: "/sys/kernel/debug",
								},
							}, odigosSeLinuxHostVolumeMounts...),
							ImagePullPolicy: "IfNotPresent",
							SecurityContext: &corev1.SecurityContext{
								Privileged: ptrbool(true),
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{
										"SYS_PTRACE",
									},
								},
							},
						},
					},
					DNSPolicy:          "ClusterFirstWithHostNet",
					ServiceAccountName: OdigletServiceAccountName,
					HostNetwork:        true,
					HostPID:            true,
					PriorityClassName:  "system-node-critical",
				},
			},
		},
	}
}

// used to inject the host volumes into odigos components for selinux update
func selinuxHostVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "host",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/",
				},
			},
		},
		{
			Name: "selinux",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/selinux",
				},
			},
		},
	}
}

// used to inject the host volumemounts into odigos components for selinux update
func selinuxHostVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "host",
			MountPath: "/host",
			ReadOnly:  true,
		},
		{
			Name:             "selinux",
			MountPath:        "/host/etc/selinux",
			MountPropagation: ptrMountPropagationMode("Bidirectional"),
		},
	}
}

func ptrMountPropagationMode(p corev1.MountPropagationMode) *corev1.MountPropagationMode {
	return &p
}

type odigletResourceManager struct {
	client        *kube.Client
	ns            string
	config        *common.OdigosConfiguration
	odigosTier    common.OdigosTier
	odigosVersion string
}

func NewOdigletResourceManager(client *kube.Client, ns string, config *common.OdigosConfiguration, odigosTier common.OdigosTier, odigosVersion string) resourcemanager.ResourceManager {
	return &odigletResourceManager{client: client, ns: ns, config: config, odigosTier: odigosTier, odigosVersion: odigosVersion}
}

func (a *odigletResourceManager) Name() string { return "Odiglet" }

func (a *odigletResourceManager) InstallFromScratch(ctx context.Context) error {

	odigletImage := a.config.OdigletImage
	// if the user specified an image, use it. otherwise, use the default image.
	// prev v1.0.4 - the cli would automatically store "keyval/odigos-odiglet" instead of empty value,
	// thus we need to treat the default image name as empty value.
	if odigletImage == "" || odigletImage == OdigletImageName {
		if a.odigosTier == common.CommunityOdigosTier {
			odigletImage = OdigletImageName
		} else {
			odigletImage = OdigletEnterpriseImageName
		}
	}

	resources := []kube.Object{
		NewOdigletServiceAccount(a.ns),
		NewOdigletRole(a.ns),
		NewOdigletRoleBinding(a.ns),
		NewOdigletClusterRole(a.config.Psp),
		NewOdigletClusterRoleBinding(a.ns),
	}

	clusterKind := cmdcontext.ClusterKindFromContext(ctx)

	// if openshift is enabled, we need to create additional SCC cluster role binding first
	if a.config.OpenshiftEnabled || clusterKind == autodetect.KindOpenShift {
		resources = append(resources, NewSCCRoleBinding(a.ns))
		resources = append(resources, NewSCClusterRoleBinding(a.ns))
	}

	// if gke, create resource quota
	if clusterKind == autodetect.KindGKE {
		resources = append(resources, NewResourceQuota(a.ns))
	}

	// temporary hack - check if the profiles named "code-attributes" or "kratos" are enabled.
	// in the future, the go code attribute collection should be handled on an otel-sdk level
	// instead of setting a global environment variable.
	// once this is done, we can remove this check.
	goAutoIncludeCodeAttributes := a.config.GoAutoIncludeCodeAttributes
	if slices.Contains(a.config.Profiles, "code-attributes") || slices.Contains(a.config.Profiles, "kratos") {
		goAutoIncludeCodeAttributes = true
	}

	// before creating the daemonset, we need to create the service account, cluster role and cluster role binding
	resources = append(resources,
		NewOdigletDaemonSet(a.ns, a.odigosVersion, a.config.ImagePrefix, odigletImage, a.odigosTier, a.config.OpenshiftEnabled, goAutoIncludeCodeAttributes,
			&autodetect.ClusterDetails{
				Kind:       clusterKind,
				K8SVersion: cmdcontext.K8SVersionFromContext(ctx),
			}))

	return a.client.ApplyResources(ctx, a.config.ConfigVersion, resources)
}
