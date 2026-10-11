package pro

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

func NewEnterpriseRegistryPullSecret(namespace, token string) (*corev1.Secret, error) {
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("odigos:%s", token)))
	dockerConfigJSON, err := json.Marshal(map[string]any{
		"auths": map[string]any{
			k8sconsts.OdigosImagePrefix: map[string]string{
				"username": "odigos",
				"password": token,
				"auth":     auth,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal enterprise registry docker config: %w", err)
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosEnterpriseRegistryPullSecretName,
			Namespace: namespace,
			Labels:    EnterpriseRegistryPullSecretLabels(),
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: dockerConfigJSON,
		},
	}, nil
}

// EnterpriseRegistryPullSecretLabels returns labels matching Helm-managed pull secrets.
func EnterpriseRegistryPullSecretLabels() map[string]string {
	return map[string]string{
		k8sconsts.OdigosSystemLabelKey: k8sconsts.OdigosSystemLabelValue,
	}
}
