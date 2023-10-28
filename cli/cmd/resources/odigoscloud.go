package resources

import (
	"github.com/keyval-dev/odigos/cli/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	OdigosCloudSecretName      = "odigos-cloud-proxy"
	odigosCloudTokenEnvName    = "ODIGOS_CLOUD_TOKEN"
	odigosCloudApiKeySecretKey = "api-key"
)

func NewKeyvalSecret(apiKey string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   OdigosCloudSecretName,
			Labels: labels.OdigosSystem,
		},
		StringData: map[string]string{
			odigosCloudApiKeySecretKey: apiKey,
		},
	}
}
