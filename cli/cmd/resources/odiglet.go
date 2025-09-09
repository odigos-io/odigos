package resources

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/pkg/autodetect"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/unixfd"

	"github.com/odigos-io/odigos/cli/cmd/resources/odigospro"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/containers"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/sizing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sversion "k8s.io/apimachinery/pkg/util/version"
)

func NewOdigletServiceAccount(ns string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigletServiceAccountName,
			Namespace: ns,
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
			Name:      k8sconsts.OdigletRoleBindingName,
			Namespace: ns,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      k8sconsts.OdigletServiceAccountName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     k8sconsts.OdigletRoleName,
		},
	}
}

func NewOdigletClusterRole(psp, ownerPermissionEnforcement bool) *rbacv1.ClusterRole {
	finalizersUpdate := []rbacv1.PolicyRule{}
	if ownerPermissionEnforcement {
		finalizersUpdate = append(finalizersUpdate, rbacv1.PolicyRule{
			// Required for OwnerReferencesPermissionEnforcement (on by default in OpenShift)
			// When we create an InstrumentationInstance, we set the OwnerReference to the related pod.
			// Controller-runtime sets BlockDeletion: true. So with this Admission Plugin we need permission to
			// update finalizers on the workloads so that they can block deletion.
			// seehttps://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#ownerreferencespermissionenforcement
			APIGroups: []string{""},
			Resources: []string{"pods/finalizers"},
			Verbs:     []string{"update"},
		})
	}

	clusterrole := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sconsts.OdigletClusterRoleName,
		},
		Rules: append([]rbacv1.PolicyRule{
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
			{ // Needed for virtual device registration + taint removal in case of Karpenter
				APIGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"get", "list", "watch", "patch", "update"},
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
			//  **** Needed for data collection container
			{ // TODO: remove this after we remove honeycomb custom exporter config
				// located at: autoscaler/controllers/datacollection/custom/honeycomb.go
				APIGroups: []string{""},
				Resources: []string{"nodes/stats", "nodes/proxy"},
				Verbs:     []string{"get", "list"},
			},
			{ // Need for k8s attributes processor
				APIGroups: []string{""},
				Resources: []string{"pods", "namespaces"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Need for k8s attributes processor
				APIGroups: []string{"apps"},
				Resources: []string{"replicasets", "deployments", "daemonsets", "statefulsets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Needed for load balancer
				APIGroups: []string{""},
				Resources: []string{"endpoints"},
				Verbs:     []string{"get", "list", "watch"},
			},
			//  **** Needed for data collection container
		}, finalizersUpdate...),
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
			Name: k8sconsts.OdigletClusterRoleBindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      k8sconsts.OdigletServiceAccountName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     k8sconsts.OdigletClusterRoleName,
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
				Name:      k8sconsts.OdigletServiceAccountName,
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

func NewOdigletDaemonSet(odigletOptions *OdigletDaemonSetOptions) *appsv1.DaemonSet {
	if odigletOptions.NodeSelector == nil {
		odigletOptions.NodeSelector = make(map[string]string)
	}

	sizePreset := odigletOptions.NodeCollectorSizing

	// Data-collection default: enable all signals (traces, metrics, logs)
	// TODO: control by configuration later per request
	signalsEnabled := map[common.ObservabilitySignal]bool{
		common.TracesObservabilitySignal:  true,
		common.MetricsObservabilitySignal: true,
		common.LogsObservabilitySignal:    true,
	}

	logsEnabled := signalsEnabled[common.LogsObservabilitySignal]
	metricsEnabled := signalsEnabled[common.MetricsObservabilitySignal]
	privilegedRequired := logsEnabled || metricsEnabled

	if _, ok := odigletOptions.NodeSelector["kubernetes.io/os"]; !ok {
		odigletOptions.NodeSelector["kubernetes.io/os"] = "linux"
	}
	dynamicEnv := []corev1.EnvVar{}
	if odigletOptions.Tier == common.CloudOdigosTier {
		dynamicEnv = append(dynamicEnv, odigospro.CloudTokenAsEnvVar())
	} else if odigletOptions.Tier == common.OnPremOdigosTier {
		dynamicEnv = append(dynamicEnv, odigospro.OnPremTokenAsEnvVar())
	}

	additionalVolumes := make([]corev1.Volume, 0)
	additionalVolumeMounts := make([]corev1.VolumeMount, 0)

	odigosSeLinuxHostVolumes := []corev1.Volume{}
	odigosSeLinuxHostVolumeMounts := []corev1.VolumeMount{}
	if odigletOptions.OpenShiftEnabled || odigletOptions.ClusterDetails.Kind == autodetect.KindOpenShift {
		odigosSeLinuxHostVolumes = append(odigosSeLinuxHostVolumes, selinuxHostVolumes()...)
		odigosSeLinuxHostVolumeMounts = append(odigosSeLinuxHostVolumeMounts, selinuxHostVolumeMounts()...)
	}
	additionalVolumes = append(additionalVolumes, odigosSeLinuxHostVolumes...)
	additionalVolumeMounts = append(additionalVolumeMounts, odigosSeLinuxHostVolumeMounts...)

	customContainerRuntimeSocketVolumes := []corev1.Volume{}
	customContainerRunetimeSocketVolumeMounts := []corev1.VolumeMount{}
	if odigletOptions.CustomContainerRuntimeSocketPath != "" {
		customContainerRuntimeSocketVolumes = setCustomContainerRuntimeSocketVolume(odigletOptions.CustomContainerRuntimeSocketPath)
		customContainerRunetimeSocketVolumeMounts = setCustomContainerRuntimeSocketVolumeMount(odigletOptions.CustomContainerRuntimeSocketPath)
		dynamicEnv = append(dynamicEnv,
			corev1.EnvVar{
				Name:  k8sconsts.CustomContainerRuntimeSocketEnvVar,
				Value: odigletOptions.CustomContainerRuntimeSocketPath})
	}
	additionalVolumes = append(additionalVolumes, customContainerRuntimeSocketVolumes...)
	additionalVolumeMounts = append(additionalVolumeMounts, customContainerRunetimeSocketVolumeMounts...)

	goOffsetsVolume := corev1.Volume{
		Name: k8sconsts.GoOffsetsConfigMap,
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: k8sconsts.GoOffsetsConfigMap,
				},
			},
		},
	}
	goOffsetsVolumeMount := corev1.VolumeMount{
		Name:      k8sconsts.GoOffsetsConfigMap,
		MountPath: k8sconsts.OffsetFileMountPath,
	}
	additionalVolumes = append(additionalVolumes, goOffsetsVolume)
	additionalVolumeMounts = append(additionalVolumeMounts, goOffsetsVolumeMount)
	dynamicEnv = append(dynamicEnv, v1.EnvVar{Name: k8sconsts.GoOffsetsEnvVar, Value: k8sconsts.OffsetFileMountPath + "/" + k8sconsts.GoOffsetsFileName})

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
	k8sversionInCluster := odigletOptions.ClusterDetails.K8SVersion
	if k8sversionInCluster != nil && k8sversionInCluster.AtLeast(k8sversion.MustParse("v1.22")) {
		maxSurge := intstr.FromInt(0)
		rollingUpdate.MaxSurge = &maxSurge
	}

	// Build the pod volumes (base + conditional for logs/metrics/traces)
	baseVolumes := []corev1.Volume{
		{
			Name: "run-dir",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{Path: "/run"},
			},
		},
		{
			Name: "device-plugins-dir",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{Path: "/var/lib/kubelet/device-plugins"},
			},
		},
		{
			Name: "odigos",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{Path: k8sconsts.OdigosAgentsDirectory},
			},
		},
		{
			Name: "kernel-debug",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{Path: "/sys/kernel/debug"},
			},
		},
		{
			Name: "exchange-dir",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
	// Logs volumes
	if logsEnabled {
		baseVolumes = append(baseVolumes,
			corev1.Volume{
				Name: "varlog",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{Path: "/var/log"},
				},
			},
			corev1.Volume{
				Name: "varlibdockercontainers",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{Path: "/var/lib/docker/containers"},
				},
			},
		)
	}
	// Metrics volumes
	if metricsEnabled {
		baseVolumes = append(baseVolumes,
			corev1.Volume{
				Name: "hostfs",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{Path: "/"},
				},
			},
		)
	}

	// Build the data-collection container mounts (only for its container)
	dataCollectionMounts := []corev1.VolumeMount{
		{Name: "exchange-dir", MountPath: unixfd.ExchangeDir},
	}
	if logsEnabled {
		dataCollectionMounts = append(dataCollectionMounts,
			corev1.VolumeMount{Name: "varlog", MountPath: "/var/log", ReadOnly: true},
			corev1.VolumeMount{Name: "varlibdockercontainers", MountPath: "/var/lib/docker/containers", ReadOnly: true},
		)
	}
	if metricsEnabled {
		dataCollectionMounts = append(dataCollectionMounts,
			corev1.VolumeMount{Name: "hostfs", MountPath: "/hostfs", ReadOnly: true},
		)
	}

	ds := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigletDaemonSetName,
			Namespace: odigletOptions.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name": k8sconsts.OdigletAppLabelValue,
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": k8sconsts.OdigletAppLabelValue,
				},
			},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type:          appsv1.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: rollingUpdate,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name":           k8sconsts.OdigletAppLabelValue,
						k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleNodeCollector),
					},
					Annotations: map[string]string{
						"kubectl.kubernetes.io/default-container": k8sconsts.OdigletContainerName,
					},
				},
				Spec: corev1.PodSpec{
					NodeSelector: odigletOptions.NodeSelector,
					Tolerations: []corev1.Toleration{
						{
							// This toleration with 'Exists' operator and no key/effect specified
							// will match ALL taints, allowing pods to be scheduled on any node
							// regardless of its taints (including master/control-plane nodes)
							Operator: corev1.TolerationOpExists,
						},
					},
					Volumes: append(baseVolumes, additionalVolumes...),
					InitContainers: []corev1.Container{
						{
							Name:  "init",
							Image: containers.GetImageName(odigletOptions.ImagePrefix, odigletOptions.OdigletImage, odigletOptions.Version),
							Command: []string{
								"/root/odiglet",
							},
							Args: []string{
								"init",
							},
							Env: []corev1.EnvVar{
								{
									Name: k8sconsts.NodeNameEnvVar,
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
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
									Name: "CURRENT_NS",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									// let the Go runtime know how many CPUs are available,
									// without this, Go will assume all the cores are available.
									Name: "GOMAXPROCS",
									ValueFrom: &corev1.EnvVarSource{
										ResourceFieldRef: &corev1.ResourceFieldSelector{
											ContainerName: "init",
											// limitCPU, Kubernetes automatically rounds up the value to an integer
											// (700m -> 1, 1200m -> 2)
											Resource: "limits.cpu",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("200m"),
									"memory": resource.MustParse("200Mi"),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("200m"),
									"memory": resource.MustParse("200Mi"),
								},
							},
							VolumeMounts: append([]corev1.VolumeMount{
								{
									Name:      "odigos",
									MountPath: k8sconsts.OdigosAgentsDirectory,
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
					Containers: []corev1.Container{
						{
							Name: k8sconsts.OdigletContainerName,
							Command: []string{
								"/root/odiglet",
							},
							Args: []string{
								"--health-probe-bind-port=" + strconv.Itoa(odigletOptions.HealthProbeBindPort),
							},
							Image: containers.GetImageName(odigletOptions.ImagePrefix, odigletOptions.OdigletImage, odigletOptions.Version),
							Env: append([]corev1.EnvVar{
								{
									Name: k8sconsts.NodeNameEnvVar,
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
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/healthz",
										Port: intstr.IntOrString{
											Type:   intstr.Type(0),
											IntVal: int32(odigletOptions.HealthProbeBindPort),
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
											IntVal: int32(odigletOptions.HealthProbeBindPort),
										},
									},
								},
								InitialDelaySeconds: 15,
								TimeoutSeconds:      5,
								PeriodSeconds:       20,
								SuccessThreshold:    0,
								FailureThreshold:    0,
							},
							VolumeMounts: append([]corev1.VolumeMount{
								{
									Name:      "run-dir",
									MountPath: "/run",
								},
								{
									Name:      "odigos",
									MountPath: k8sconsts.OdigosAgentsDirectory,
									ReadOnly:  true,
								},
								{
									Name:      "kernel-debug",
									MountPath: "/sys/kernel/debug",
								},
								{
									Name:      "exchange-dir",
									MountPath: unixfd.ExchangeDir,
								},
							}, additionalVolumeMounts...),
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
						{
							Name:    k8sconsts.OdigosNodeCollectorContainerName,
							Image:   containers.GetImageName(odigletOptions.ImagePrefix, odigletOptions.CollectorImage, odigletOptions.Version),
							Command: []string{k8sconsts.OdigosNodeCollectorContainerCommand},
							Args: []string{
								fmt.Sprintf(
									"--config=%s:%s/%s/%s",
									k8sconsts.OdigosCollectorConfigMapProviderScheme,
									odigletOptions.Namespace,
									k8sconsts.OdigosNodeCollectorConfigMapName,
									k8sconsts.OdigosNodeCollectorConfigMapKey,
								),
							},
							Env: []corev1.EnvVar{
								{
									Name: k8sconsts.NodeNameEnvVar,
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"},
									},
								},
								{
									Name: k8sconsts.NodeIPEnvVar,
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
									},
								},
								{
									Name:  "GOMEMLIMIT",
									Value: fmt.Sprintf("%dMiB", sizePreset.GoMemLimitMib),
								},
								{
									Name: "GOMAXPROCS",
									ValueFrom: &corev1.EnvVarSource{
										ResourceFieldRef: &corev1.ResourceFieldSelector{
											ContainerName: k8sconsts.OdigosNodeCollectorContainerName,
											Resource:      "limits.cpu",
										},
									},
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.FromInt(13133),
									},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.FromInt(13133),
									},
								},
							},
							// For PoC we leave Resources empty or set simple defaults; you can thread values later.
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse(fmt.Sprintf("%dm", sizePreset.RequestCPUm)),
									"memory": resource.MustParse(fmt.Sprintf("%dMi", sizePreset.RequestMemoryMiB)),
								},
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse(fmt.Sprintf("%dm", sizePreset.LimitCPUm)),
									"memory": resource.MustParse(fmt.Sprintf("%dMi", sizePreset.LimitMemoryMiB)),
								},
							},
							VolumeMounts: dataCollectionMounts,
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privilegedRequired,
							},
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
					},
					DNSPolicy:          "ClusterFirstWithHostNet",
					ServiceAccountName: k8sconsts.OdigletServiceAccountName,
					HostPID:            true,
					PriorityClassName:  "system-node-critical",
				},
			},
		},
	}

	// If mount method is not set (default is k8s-virtual-device), or it is k8s-virtual-device, we need to install the device plugin
	if odigletOptions.MountMethod == nil || *odigletOptions.MountMethod == common.K8sVirtualDeviceMountMethod {
		ds.Spec.Template.Spec.Containers = append(ds.Spec.Template.Spec.Containers, corev1.Container{
			Name:  k8sconsts.OdigletDevicePluginContainerName,
			Image: containers.GetImageName(odigletOptions.ImagePrefix, odigletOptions.OdigletImage, odigletOptions.Version),
			Command: []string{
				"/root/deviceplugin",
			},
			Env: []corev1.EnvVar{
				{
					Name: k8sconsts.NodeNameEnvVar,
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
			},
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					"cpu":    resource.MustParse("100m"),
					"memory": resource.MustParse("300Mi"),
				},
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("40m"),
					"memory": resource.MustParse("200Mi"),
				},
			},
			LivenessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					Exec: &corev1.ExecAction{
						Command: []string{k8sconsts.GrpcHealthBinaryPath, "-addr=" + k8sconsts.GrpcHealthProbePath, "-connect-timeout=" + strconv.Itoa(k8sconsts.GrpcHealthProbeTimeout) + "s", "-rpc-timeout=" + strconv.Itoa(k8sconsts.GrpcHealthProbeTimeout) + "s"},
					},
				},
				InitialDelaySeconds: 10,
				FailureThreshold:    3,
				PeriodSeconds:       10,
				TimeoutSeconds:      10,
			},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					Exec: &corev1.ExecAction{
						Command: []string{k8sconsts.GrpcHealthBinaryPath, "-addr=" + k8sconsts.GrpcHealthProbePath, "-connect-timeout=" + strconv.Itoa(k8sconsts.GrpcHealthProbeTimeout) + "s", "-rpc-timeout=" + strconv.Itoa(k8sconsts.GrpcHealthProbeTimeout) + "s"},
					},
				},
				InitialDelaySeconds: 10,
				FailureThreshold:    3,
				PeriodSeconds:       10,
				TimeoutSeconds:      10,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "device-plugins-dir",
					MountPath: "/var/lib/kubelet/device-plugins",
				},
			},
			ImagePullPolicy: "IfNotPresent",
		},
		)
	}

	// if inetrnal trffic policy is not yet supported in the cluster, fall back to host network
	if k8sversionInCluster == nil || k8sversionInCluster.LessThan(k8sversion.MustParse("v1.26")) {
		ds.Spec.Template.Spec.HostNetwork = true
	}

	return ds
}

// NewOdigletGoOffsetsConfigMap returns the custom Go Offsets ConfigMap mounted by Odiglet.
// If one already exists, it will return that object (to support upgrades while preserving existing file).
// Otherwise, it returns a configmap with a blank file, which instructs Odiglet to use the default offsets.
func NewOdigletGoOffsetsConfigMap(ctx context.Context, client *kube.Client, ns string) (*v1.ConfigMap, error) {
	existingCm := &v1.ConfigMap{}
	existingCm, err := client.Clientset.CoreV1().ConfigMaps(ns).Get(ctx, k8sconsts.GoOffsetsConfigMap, metav1.GetOptions{})

	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}

	goOffsetContent := ""
	if err == nil {
		if _, exists := existingCm.Data[k8sconsts.GoOffsetsFileName]; exists {
			goOffsetContent = existingCm.Data[k8sconsts.GoOffsetsFileName]
		}
	}

	return &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.GoOffsetsConfigMap,
			Namespace: ns,
		},
		Data: map[string]string{
			k8sconsts.GoOffsetsFileName: goOffsetContent,
		},
	}, nil
}

func NewOdigletLocalTrafficService(ns string) *corev1.Service {
	localTrafficPolicy := v1.ServiceInternalTrafficPolicyLocal
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigletLocalTrafficServiceName,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/name": k8sconsts.OdigletAppLabelValue,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app.kubernetes.io/name": k8sconsts.OdigletAppLabelValue,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "op-amp",
					Port:       int32(consts.OpAMPPort),
					TargetPort: intstr.FromInt(consts.OpAMPPort),
				},
				{
					Name:       "metrics",
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
				},
			},
			InternalTrafficPolicy: &localTrafficPolicy,
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

// used to inject the host volumes into odigos components for custom container runtime socket path
func setCustomContainerRuntimeSocketVolume(customContainerRuntimeSocketPath string) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "custom-container-runtime-socket",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: filepath.Dir(customContainerRuntimeSocketPath),
				},
			},
		},
	}
}

func setCustomContainerRuntimeSocketVolumeMount(customContainerRuntimeSocketPath string) []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "custom-container-runtime-socket",
			MountPath: filepath.Dir(customContainerRuntimeSocketPath),
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
	managerOpts   resourcemanager.ManagerOpts
}

func NewOdigletResourceManager(client *kube.Client, ns string, config *common.OdigosConfiguration, odigosTier common.OdigosTier, odigosVersion string, managerOpts resourcemanager.ManagerOpts) resourcemanager.ResourceManager {
	return &odigletResourceManager{client: client, ns: ns, config: config, odigosTier: odigosTier, odigosVersion: odigosVersion, managerOpts: managerOpts}
}

func (a *odigletResourceManager) Name() string { return "Odiglet" }

func (a *odigletResourceManager) InstallFromScratch(ctx context.Context) error {
	goOffsetConfigMap, err := NewOdigletGoOffsetsConfigMap(ctx, a.client, a.ns)
	if err != nil {
		return err
	}
	resources := []kube.Object{
		NewOdigletServiceAccount(a.ns),
		NewOdigletRoleBinding(a.ns),
		NewOdigletClusterRole(a.config.Psp, a.config.OpenshiftEnabled),
		NewOdigletClusterRoleBinding(a.ns),
		goOffsetConfigMap,
	}

	k8sVersion := cmdcontext.K8SVersionFromContext(ctx)
	if k8sVersion != nil && k8sVersion.AtLeast(k8sversion.MustParse("v1.26")) {
		resources = append(resources, NewOdigletLocalTrafficService(a.ns))
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

	// if the health probe bind port is not set, use the default value
	if a.config.OdigletHealthProbeBindPort == 0 {
		a.config.OdigletHealthProbeBindPort = k8sconsts.OdigletDefaultHealthProbeBindPort
	}

	// Calculate the node collector sizing by starting from the preset sizing
	// and then applying any overrides defined in the Odigos configuration.
	collectorSizing := sizing.ComputeResourceSizePreset(a.config)
	nodeCollectorSizing := collectorSizing.CollectorNodeConfig

	odigletOptions := &OdigletDaemonSetOptions{
		Namespace:        a.ns,
		Version:          a.odigosVersion,
		ImagePrefix:      a.config.ImagePrefix,
		OdigletImage:     a.managerOpts.ImageReferences.OdigletImage,
		CollectorImage:   a.managerOpts.ImageReferences.CollectorImage,
		Tier:             a.odigosTier,
		OpenShiftEnabled: a.config.OpenshiftEnabled,
		ClusterDetails: &autodetect.ClusterDetails{
			Kind:       clusterKind,
			K8SVersion: k8sVersion,
		},
		CustomContainerRuntimeSocketPath: a.config.CustomContainerRuntimeSocketPath,
		NodeSelector:                     a.config.NodeSelector,
		HealthProbeBindPort:              a.config.OdigletHealthProbeBindPort,
		MountMethod:                      a.config.MountMethod,
		NodeCollectorSizing:              nodeCollectorSizing,
	}

	// before creating the daemonset, we need to create the service account, cluster role and cluster role binding
	resources = append(resources,
		NewOdigletDaemonSet(odigletOptions))

	return a.client.ApplyResources(ctx, a.config.ConfigVersion, resources, a.managerOpts)
}
