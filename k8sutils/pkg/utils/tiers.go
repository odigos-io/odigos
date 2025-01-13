package utils

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
)

const (
	odigosCloudApiKeySecretKey = "odigos-cloud-api-key"
	odigosOnpremTokenSecretKey = "odigos-onprem-token"
)

func GetCurrentOdigosTier(ctx context.Context, namespaces string, client *kubernetes.Clientset) (common.OdigosTier, error) {
	sec, err := getCurrentOdigosProSecret(ctx, namespaces, client)

	if err != nil {
		return "", err
	}
	if sec == nil {
		return common.CommunityOdigosTier, nil
	}

	if _, exists := sec.Data[odigosCloudApiKeySecretKey]; exists {
		return common.CloudOdigosTier, nil
	}
	if _, exists := sec.Data[odigosOnpremTokenSecretKey]; exists {
		return common.OnPremOdigosTier, nil
	}
	return common.CommunityOdigosTier, nil
}

func getCurrentOdigosProSecret(ctx context.Context, namespace string, client *kubernetes.Clientset) (*corev1.Secret, error) {
	secret, err := client.CoreV1().Secrets(namespace).Get(ctx, consts.OdigosProSecretName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// apparently, k8s is not setting the type meta for the object obtained with Get.
	secret.TypeMeta = metav1.TypeMeta{
		Kind:       "Secret",
		APIVersion: "v1",
	}
	return secret, err
}
