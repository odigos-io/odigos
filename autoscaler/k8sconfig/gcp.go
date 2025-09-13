package k8sconfig

import (
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

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
		// Add volume mount if it doesn't exist
		for _, volumeMount := range currentDeployment.Spec.Template.Spec.Containers[0].VolumeMounts {
			if volumeMount.Name == strings.ReplaceAll(dest.GetSecretRef().Name, ".", "-") {
				return fmt.Errorf("GCP credentials volume mount %s already exists."+
					"Only one GCP Destination may have Application Credentials configured", volumeMount.Name)
			}
		}
		currentDeployment.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
			{
				Name:      strings.ReplaceAll(dest.GetSecretRef().Name, ".", "-"),
				MountPath: gcpCredentialsMountPath,
			},
		}

		// Add volume if it doesn't exist
		for i := range currentDeployment.Spec.Template.Spec.Volumes {
			if currentDeployment.Spec.Template.Spec.Volumes[i].Name == strings.ReplaceAll(dest.GetSecretRef().Name, ".", "-") {
				return fmt.Errorf("GCP credentials volume %s already exists."+
					"Only one GCP Destination may have Application Credentials configured", currentDeployment.Spec.Template.Spec.Volumes[i].Name)
			}
		}
		currentDeployment.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: strings.ReplaceAll(dest.GetSecretRef().Name, ".", "-"),
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
			},
		}

		// Add environment variable pointing to the mounted credentials if it doesn't exist
		for _, env := range currentDeployment.Spec.Template.Spec.Containers[0].Env {
			if env.Name == gcpApplicationCredentialsKey {
				return fmt.Errorf("GCP credentials environment variable %s already exists."+
					"Only one GCP Destination may have Application Credentials configured", env.Name)
			}
		}
		currentDeployment.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{
			{
				Name:  gcpApplicationCredentialsKey,
				Value: gcpCredentialsMountPath + "/" + gcpApplicationCredentialsKey,
			},
		}
	}
	return nil
}
