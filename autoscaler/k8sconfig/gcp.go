package k8sconfig

import (
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
)

const (
	gcpApplicationCredentialsKey = "GCP_APPLICATION_CREDENTIALS"
	gcpCredentialsMountPath      = "/secrets"
)

type GoogleCloud struct{}

func (g *GoogleCloud) DestType() common.DestinationType {
	return common.GoogleCloudDestinationType
}

func (g *GoogleCloud) ModifyGatewayCollectorDeployment(dest K8sExporterConfigurer, currentDeployment *appsv1.Deployment) error {
	config := dest.GetConfig()
	// If GCP_APPLICATION_CREDENTIALS is set, mount the secret and set the environment variable
	// NOTE: Currently, only one GCP Destination may have Application Credentials configured. This is a limitation of the GCP Collector Exporter,
	// which relies on the GOOGLE_APPLICATION_CREDENTIALS environment variable to be set.
	// To support multiple GCP Destinations with different credentials (which is uncommon but not totally unreasonable), we would need to
	// create multiple Gateway Collector Deployments, one for each GCP Destination.
	if val, exists := config[gcpApplicationCredentialsKey]; exists && val != "" && dest.GetSecretRef() != nil {
		containerIndex := -1
		for i := 0; i < len(currentDeployment.Spec.Template.Spec.Containers); i++ {
			if currentDeployment.Spec.Template.Spec.Containers[i].Name == k8sconsts.OdigosClusterCollectorContainerName {
				containerIndex = i
				break
			}
		}
		if containerIndex == -1 {
			return fmt.Errorf("gateway collector container '%s' not found", k8sconsts.OdigosClusterCollectorContainerName)
		}

		secretRefName := strings.ReplaceAll(dest.GetSecretRef().Name, ".", "-")

		// Add volume mount if it doesn't exist
		for _, volumeMount := range currentDeployment.Spec.Template.Spec.Containers[containerIndex].VolumeMounts {
			if volumeMount.Name == secretRefName {
				return fmt.Errorf("GCP credentials volume mount %s already exists."+
					"Only one GCP Destination may have Application Credentials configured", volumeMount.Name)
			}
		}
		currentDeployment.Spec.Template.Spec.Containers[containerIndex].VolumeMounts = append(currentDeployment.Spec.Template.Spec.Containers[containerIndex].VolumeMounts, corev1.VolumeMount{
			Name:      secretRefName,
			MountPath: gcpCredentialsMountPath,
		})

		// Add volume if it doesn't exist
		for i := range currentDeployment.Spec.Template.Spec.Volumes {
			if currentDeployment.Spec.Template.Spec.Volumes[i].Name == secretRefName {
				return fmt.Errorf("GCP credentials volume %s already exists."+
					"Only one GCP Destination may have Application Credentials configured", currentDeployment.Spec.Template.Spec.Volumes[i].Name)
			}
		}
		currentDeployment.Spec.Template.Spec.Volumes = append(currentDeployment.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: secretRefName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: dest.GetSecretRef().Name,
					Items: []corev1.KeyToPath{
						{
							Key:  gcpApplicationCredentialsKey,
							Path: gcpApplicationCredentialsKey,
						},
					},
				},
			},
		})

		// Add environment variable pointing to the mounted credentials if it doesn't exist
		for _, env := range currentDeployment.Spec.Template.Spec.Containers[containerIndex].Env {
			if env.Name == gcpApplicationCredentialsKey {
				return fmt.Errorf("GCP credentials environment variable %s already exists."+
					"Only one GCP Destination may have Application Credentials configured", env.Name)
			}
		}
		currentDeployment.Spec.Template.Spec.Containers[containerIndex].Env = append(currentDeployment.Spec.Template.Spec.Containers[containerIndex].Env, corev1.EnvVar{
			Name:  gcpApplicationCredentialsKey,
			Value: gcpCredentialsMountPath + "/" + gcpApplicationCredentialsKey,
		})
	}
	return nil
}
