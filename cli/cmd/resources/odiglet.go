package resources

import (
	"context"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/odigos-io/odigos/cli/pkg/autodetect"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/cli/cmd/resources/odigospro"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/containers"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	OdigletServiceName         = "odiglet"
	OdigletDaemonSetName       = "odiglet"
	OdigletAppLabelValue       = "odiglet"
	OdigletContainerName       = "odiglet"
	OdigletImageName           = "keyval/odigos-odiglet"
	OdigletEnterpriseImageName = "keyval/odigos-enterprise-odiglet"
)

func NewOdigletServiceAccount(ns string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "odiglet",
			Namespace: ns,
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
			Name: "odiglet",
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
				APIGroups: []string{"odigos.io"},
				Resources: []string{"odigosconfigurations", "collectorsgroups", "collectorsgroups/status"},
			},
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
				APIGroups: []string{""},
				Resources: []string{
					"pods",
				},
			},
			{
				Verbs: []string{
					"get",
				},
				APIGroups: []string{""},
				Resources: []string{
					"pods/status",
				},
			},
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
				APIGroups: []string{""},
				Resources: []string{
					"nodes",
				},
			},
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
				APIGroups: []string{"apps"},
				Resources: []string{"replicasets"},
			},
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
				APIGroups: []string{"apps"},
				Resources: []string{"deployments"},
			},
			{
				Verbs: []string{
					"get",
				},
				APIGroups: []string{"apps"},
				Resources: []string{
					"deployments/status",
				},
			},
			{
				Verbs: []string{
					"get",
				},
				APIGroups: []string{"apps"},
				Resources: []string{
					"deployments/finalizers",
				},
			},
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
				APIGroups: []string{"apps"},
				Resources: []string{"statefulsets"},
			},
			{
				Verbs: []string{
					"get",
				},
				APIGroups: []string{"apps"},
				Resources: []string{
					"statefulsets/status",
				},
			},
			{
				Verbs: []string{
					"get",
				},
				APIGroups: []string{"apps"},
				Resources: []string{
					"statefulsets/finalizers",
				},
			},
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
				APIGroups: []string{"apps"},
				Resources: []string{"daemonsets"},
			},
			{
				Verbs: []string{
					"get",
				},
				APIGroups: []string{"apps"},
				Resources: []string{
					"daemonsets/status",
				},
			},
			{
				Verbs: []string{
					"get",
				},
				APIGroups: []string{"apps"},
				Resources: []string{
					"daemonsets/finalizers",
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
					"odigos.io",
				},
				Resources: []string{
					"instrumentedapplications",
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
					"get",
					"list",
					"patch",
					"update",
					"watch",
					"delete",
				},
				APIGroups: []string{
					"odigos.io",
				},
				Resources: []string{
					"instrumentationinstances",
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
					"instrumentationinstances/status",
				},
			},
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
				APIGroups: []string{""},
				Resources: []string{
					"namespaces",
				},
			},
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
					"patch",
					"update",
				},
				APIGroups: []string{"odigos.io"},
				Resources: []string{
					"instrumentationconfigs",
				},
			},
		},
	}

	if psp {
		clusterrole.Rules = append(clusterrole.Rules, rbacv1.PolicyRule{
			Verbs: []string{
				"use",
			},
			APIGroups: []string{
				"policy",
			},
			Resources: []string{
				"podsecuritypolicies",
			},
			ResourceNames: []string{
				"privileged",
			},
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
			Name: "odiglet",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "odiglet",
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "odiglet",
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
				Name:      "odiglet",
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

func NewOdigletDaemonSet(ns string, version string, imagePrefix string, imageName string, odigosTier common.OdigosTier, openshiftEnabled bool, goAutoIncludeCodeAttributes bool) *appsv1.DaemonSet {

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
	if openshiftEnabled || autodetect.CurrentKubernetesVersion.Kind == autodetect.KindOpenShift {
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
	maxSurge := intstr.FromInt(0)

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
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxUnavailable: &maxUnavailable,
					MaxSurge:       &maxSurge,
				},
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
							Key:      "node.kubernetes.io/os",
							Operator: corev1.TolerationOpEqual,
							Value:    "windows",
							Effect:   corev1.TaintEffectNoSchedule,
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
					ServiceAccountName: "odiglet",
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
	client     *kube.Client
	ns         string
	config     *odigosv1.OdigosConfigurationSpec
	odigosTier common.OdigosTier
}

func NewOdigletResourceManager(client *kube.Client, ns string, config *odigosv1.OdigosConfigurationSpec, odigosTier common.OdigosTier) resourcemanager.ResourceManager {
	return &odigletResourceManager{client: client, ns: ns, config: config, odigosTier: odigosTier}
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

	resources := []client.Object{
		NewOdigletServiceAccount(a.ns),
		NewOdigletClusterRole(a.config.Psp),
		NewOdigletClusterRoleBinding(a.ns),
	}

	// if openshift is enabled, we need to create additional SCC cluster role binding first
	if a.config.OpenshiftEnabled || autodetect.CurrentKubernetesVersion.Kind == autodetect.KindOpenShift {
		resources = append(resources, NewSCCRoleBinding(a.ns))
		resources = append(resources, NewSCClusterRoleBinding(a.ns))
	}

	// if gke, create resource quota
	if autodetect.CurrentKubernetesVersion.Kind == autodetect.KindGKE {
		resources = append(resources, NewResourceQuota(a.ns))
	}

	// before creating the daemonset, we need to create the service account, cluster role and cluster role binding
	resources = append(resources,
		NewOdigletDaemonSet(a.ns, a.config.OdigosVersion, a.config.ImagePrefix, odigletImage, a.odigosTier, a.config.OpenshiftEnabled, a.config.GoAutoIncludeCodeAttributes))

	return a.client.ApplyResources(ctx, a.config.ConfigVersion, resources)
}
