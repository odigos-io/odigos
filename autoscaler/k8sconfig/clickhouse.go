package k8sconfig

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
)

const (
	clickhouseCaPemKey = "CLICKHOUSE_CA_PEM"
)

type Clickhouse struct{}

func validateCertificatePem(pemData []byte) error {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return errors.New("failed to decode PEM block")
	}
	if block.Type != "CERTIFICATE" {
		return errors.New("PEM block is not a certificate")
	}
	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}
	return nil
}

func (c *Clickhouse) DestType() common.DestinationType {
	return common.ClickhouseDestinationType
}

// ModifyGatewayCollectorDeployment modifies the gateway collector deployment to mount the ClickHouse CA certificate secret.
func (c *Clickhouse) ModifyGatewayCollectorDeployment(ctx context.Context, k8sClient client.Client, dest K8sExporterConfigurer, currentDeployment *appsv1.Deployment) error {
	// Check if destination has a non-empty secret ref
	if dest.GetSecretRef() == nil || dest.GetSecretRef().Name == "" {
		return nil
	}

	// Try to get the secret
	secret := &corev1.Secret{}
	err := k8sClient.Get(ctx, client.ObjectKey{
		Name:      dest.GetSecretRef().Name,
		Namespace: currentDeployment.Namespace,
	}, secret)
	if err != nil {
		return fmt.Errorf("failed to get secret %s: %w", dest.GetSecretRef().Name, err)
	}

	// Check if the CA PEM key is set in the secret
	if secret.Data == nil {
		return nil
	}

	caPemData, keyExists := secret.Data[clickhouseCaPemKey]
	if !keyExists {
		return nil
	}

	// Validate the CA certificate PEM format
	if err := validateCertificatePem(caPemData); err != nil {
		return fmt.Errorf("invalid CA certificate for ClickHouse destination %s: %w", dest.GetID(), err)
	}

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

	// Kubernetes volume names must not contain dots, so replace them with dashes
	sanitizedID := strings.ReplaceAll(dest.GetID(), ".", "-")
	volumeName := config.ClickhouseCaSecretVolumeName + "-" + sanitizedID

	// Add volume mount if it doesn't exist
	for _, volumeMount := range currentDeployment.Spec.Template.Spec.Containers[containerIndex].VolumeMounts {
		if volumeMount.Name == volumeName {
			// Volume mount already exists
			return nil
		}
	}
	// Mount path includes destination ID to support multiple clickhouse destinations
	mountPath := config.ClickhouseCaMountPath + "/" + dest.GetID()
	currentDeployment.Spec.Template.Spec.Containers[containerIndex].VolumeMounts = append(
		currentDeployment.Spec.Template.Spec.Containers[containerIndex].VolumeMounts,
		corev1.VolumeMount{
			Name:      volumeName,
			MountPath: mountPath,
			ReadOnly:  true,
		},
	)

	// Add volume if it doesn't exist
	for i := range currentDeployment.Spec.Template.Spec.Volumes {
		if currentDeployment.Spec.Template.Spec.Volumes[i].Name == volumeName {
			// Volume already exists
			return nil
		}
	}
	currentDeployment.Spec.Template.Spec.Volumes = append(currentDeployment.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: dest.GetSecretRef().Name,
				Items: []corev1.KeyToPath{
					{
						Key:  clickhouseCaPemKey,
						Path: clickhouseCaPemKey,
					},
				},
			},
		},
	})

	return nil
}
