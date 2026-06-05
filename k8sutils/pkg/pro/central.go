package pro

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

// GetCentralOnPremToken reads the on-prem token from the odigos-central secret.
func GetCentralOnPremToken(ctx context.Context, client kubernetes.Interface, namespace string) (string, error) {
	secret, err := client.CoreV1().Secrets(namespace).Get(ctx, k8sconsts.OdigosCentralSecretName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", fmt.Errorf(
				"central on-prem token secret %q not found in namespace %q; reinstall Odigos Central or set onPremToken via helm",
				k8sconsts.OdigosCentralSecretName,
				namespace,
			)
		}
		return "", fmt.Errorf("failed to read central on-prem token secret: %w", err)
	}

	for _, key := range []string{k8sconsts.OdigosOnpremTokenEnvName, k8sconsts.OdigosOnpremTokenSecretKey} {
		if data, ok := secret.Data[key]; ok && len(data) > 0 {
			return string(data), nil
		}
	}

	return "", fmt.Errorf(
		"central secret %q in namespace %q has no on-prem token (expected key %q or %q)",
		k8sconsts.OdigosCentralSecretName,
		namespace,
		k8sconsts.OdigosOnpremTokenEnvName,
		k8sconsts.OdigosOnpremTokenSecretKey,
	)
}
