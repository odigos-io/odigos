package centralodigos

import (
	"context"

	"github.com/google/uuid"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

	// Resolve the admin password: reuse existing, use provided, or generate new
	adminPassword, err := m.resolveAdminPassword(ctx)
	if err != nil {
		return err
	}

	resolvedConfig := m.config
	resolvedConfig.AdminPassword = adminPassword

	resources := []kube.Object{
		NewKeycloakSecret(m.ns, resolvedConfig),
		NewKeycloakDeployment(m.ns, resolvedConfig, withPvc),
		NewKeycloakService(m.ns),
	}
	if withPvc {
		resources = append(resources, NewKeycloakPVC(m.ns, resolvedConfig))
	}
	return m.client.ApplyResources(ctx, 1, resources, m.managerOpts)
}

// resolveAdminPassword determines the admin password to use:
// 1. If secret exists, reuse the existing password (prevents mismatch with Keycloak)
// 2. If password was provided via flag, use it
// 3. Otherwise, generate a random UUID as password
func (m *keycloakResourceManager) resolveAdminPassword(ctx context.Context) (string, error) {
	existingSecret, err := m.client.CoreV1().Secrets(m.ns).Get(ctx, k8sconsts.KeycloakSecretName, metav1.GetOptions{})
	if err == nil {
		if password, ok := existingSecret.Data[k8sconsts.KeycloakAdminPasswordKey]; ok {
			return string(password), nil
		}
	} else if !apierrors.IsNotFound(err) {
		return "", err
	}

	if m.config.AdminPassword != "" {
		return m.config.AdminPassword, nil
	}

	// Generate a random password using UUID
	return uuid.New().String(), nil
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
								{
									Name:  "KC_HOSTNAME_STRICT_HTTPS",
									Value: "false",
								},
								{
									Name:  "KC_HOSTNAME_STRICT_BACKCHANNEL",
									Value: "true",
								},
								{
									Name:  "KC_PROXY",
									Value: "edge",
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
