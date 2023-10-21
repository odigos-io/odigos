package resources

import (
	"context"
	"fmt"

	"github.com/keyval-dev/odigos/cli/pkg/containers"
	"github.com/keyval-dev/odigos/cli/pkg/kube"

	"github.com/keyval-dev/odigos/cli/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

const (
	odigletServiceName   = "odiglet"
	odigletDaemonSetName = "odiglet"
	odigletContainerName = "odiglet"
)

var OdigletImage string

func NewOdigletServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "odiglet",
			Labels: labels.OdigosSystem,
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
			Name:   "odiglet",
			Labels: labels.OdigosSystem,
		},
		Rules: []rbacv1.PolicyRule{
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
					"list",
					"watch",
				},
				APIGroups: []string{""},
				Resources: []string{
					"namespaces",
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
			Name:   "odiglet",
			Labels: labels.OdigosSystem,
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

func NewOdigletDaemonSet(version string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: odigletDaemonSetName,
			Labels: map[string]string{
				"app":                       "odiglet",
				labels.OdigosSystemLabelKey: labels.OdigosSystemLabelValue,
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "odiglet",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "odiglet",
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
					Volumes: []corev1.Volume{
						{
							Name: "run-dir",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/run",
								},
							},
						},
						{
							Name: "var-dir",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var",
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
					},
					Containers: []corev1.Container{
						{
							Name:  odigletContainerName,
							Image: containers.GetImageName(OdigletImage, version),
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: odigletServiceName,
								},
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
							Resources: corev1.ResourceRequirements{},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:             "run-dir",
									MountPath:        "/run",
									MountPropagation: ptrMountPropagationMode("Bidirectional"),
								},
								{
									Name:             "var-dir",
									MountPath:        "/var",
									MountPropagation: ptrMountPropagationMode("Bidirectional"),
								},
								{
									Name:             "odigos",
									MountPath:        "/var/odigos",
									MountPropagation: ptrMountPropagationMode("Bidirectional"),
								},
								{
									Name:      "kernel-debug",
									MountPath: "/sys/kernel/debug",
								},
							},
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
				},
			},
		},
	}
}
func ptrMountPropagationMode(p corev1.MountPropagationMode) *corev1.MountPropagationMode {
	return &p
}

type odigletResourceManager struct {
	client *kube.Client
	ns     string
}

func NewOdigletResourceManager(client *kube.Client, ns string) ResourceManager {
	return &odigletResourceManager{client: client, ns: ns}
}

func (a *odigletResourceManager) InstallFromScratch(ctx context.Context) error {
	return nil
}

func (a *odigletResourceManager) ApplyMigrationStep(ctx context.Context, sourceVersion string) error {
	return nil
}

func (a *odigletResourceManager) RollbackMigrationStep(ctx context.Context, sourceVersion string) error {
	return nil
}

func (a *odigletResourceManager) PatchOdigosVersionToTarget(ctx context.Context, newOdigosVersion string) error {
	fmt.Println("Patching Odigos odiglet daemonset")
	jsonPatchDocumentBytes := patchTemplateSpecImageTag(AutoscalerImage, newOdigosVersion, odigletContainerName)
	_, err := a.client.AppsV1().DaemonSets(a.ns).Patch(ctx, odigletDaemonSetName, k8stypes.JSONPatchType, jsonPatchDocumentBytes, metav1.PatchOptions{})
	return err
}
