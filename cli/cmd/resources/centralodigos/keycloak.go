package centralodigos

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AuthConfig struct {
	AdminUsername    string
	AdminPassword    string
	StorageClassName *string
}

type keycloakResourceManager struct {
	client      *kube.Client
	ns          string
	managerOpts resourcemanager.ManagerOpts
	config      AuthConfig
}

func NewKeycloakResourceManager(client *kube.Client, ns string, managerOpts resourcemanager.ManagerOpts, config AuthConfig) resourcemanager.ResourceManager {
	return &keycloakResourceManager{
		client:      client,
		ns:          ns,
		managerOpts: managerOpts,
		config:      config,
	}
}

func (m *keycloakResourceManager) Name() string { return k8sconsts.KeycloakResourceManagerName }

func (m *keycloakResourceManager) InstallFromScratch(ctx context.Context) error {
	withPvc := m.config.StorageClassName != nil && *m.config.StorageClassName != ""
	resources := []kube.Object{
		NewKeycloakSecret(m.ns, m.config),
		NewKeycloakDeployment(m.ns, m.config, withPvc),
		NewKeycloakService(m.ns),
	}
	if withPvc {
		resources = append(resources, NewKeycloakPVC(m.ns, m.config))
	}
	return m.client.ApplyResources(ctx, 1, resources, m.managerOpts)
}

func NewKeycloakSecret(ns string, config AuthConfig) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.KeycloakSecretName,
			Namespace: ns,
			Labels:    map[string]string{"app": k8sconsts.KeycloakAppName},
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			k8sconsts.KeycloakAdminUsernameKey: config.AdminUsername,
			k8sconsts.KeycloakAdminPasswordKey: config.AdminPassword,
		},
	}
}

func NewKeycloakDeployment(ns string, config AuthConfig, withPvc bool) *appsv1.Deployment {
	fsGroup := int64(1000)
	runAsNonRoot := true
	allowPrivilegeEscalation := false

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.KeycloakDeploymentName,
			Namespace: ns,
			Labels:    map[string]string{"app": k8sconsts.KeycloakAppName},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptrint32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": k8sconsts.KeycloakAppName},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": k8sconsts.KeycloakAppName},
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup: &fsGroup,
					},
					Containers: []corev1.Container{
						{
							Name:  k8sconsts.KeycloakContainerName,
							Image: k8sconsts.KeycloakImage,
							Args:  []string{"start", "--optimized", "--http-enabled=true"},
							SecurityContext: &corev1.SecurityContext{
								RunAsNonRoot:             &runAsNonRoot,
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
							},
							Env: []corev1.EnvVar{
								{
									Name: "KEYCLOAK_ADMIN",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: k8sconsts.KeycloakSecretName,
											},
											Key: k8sconsts.KeycloakAdminUsernameKey,
										},
									},
								},
								{
									Name: "KEYCLOAK_ADMIN_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: k8sconsts.KeycloakSecretName,
											},
											Key: k8sconsts.KeycloakAdminPasswordKey,
										},
									},
								},
								{
									Name:  "KC_HOSTNAME",
									Value: "localhost",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          k8sconsts.KeycloakPortName,
									ContainerPort: k8sconsts.KeycloakPort,
								},
							},
						},
					},
				},
			},
		},
	}

	if withPvc {
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
			{
				Name:      k8sconsts.KeycloakDataVolumeName,
				MountPath: "/opt/keycloak/data",
			},
		}
		deployment.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: k8sconsts.KeycloakDataVolumeName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: k8sconsts.KeycloakDataPVCName,
					},
				},
			},
		}
	}

	return deployment
}

func NewKeycloakService(ns string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.KeycloakServiceName,
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": k8sconsts.KeycloakAppName},
			Ports: []corev1.ServicePort{
				{
					Name: k8sconsts.KeycloakPortName,
					Port: k8sconsts.KeycloakPort,
				},
			},
		},
	}
}

func NewKeycloakPVC(ns string, config AuthConfig) *corev1.PersistentVolumeClaim {
	pvc := &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.KeycloakDataPVCName,
			Namespace: ns,
			Labels:    map[string]string{"app": k8sconsts.KeycloakAppName},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: config.StorageClassName,
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}

	return pvc
}
