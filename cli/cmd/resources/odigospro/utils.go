package odigospro

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetCurrentOdigosTier(ctx context.Context, client *kube.Client, ns string) (common.OdigosTier, error) {
	sec, err := getCurrentOdigosProSecret(ctx, client, ns)
	if err != nil {
		return "", err
	}
	if sec == nil {
		return common.CommunityOdigosTier, nil
	}

	if _, exists := sec.Data[k8sconsts.OdigosCloudApiKeySecretKey]; exists {
		return common.CloudOdigosTier, nil
	}
	if _, exists := sec.Data[k8sconsts.OdigosOnpremTokenSecretKey]; exists {
		return common.OnPremOdigosTier, nil
	}
	return common.CommunityOdigosTier, nil
}

// used to inject the cloud token as env var into odigos components
func CloudTokenAsEnvVar() corev1.EnvVar {
	return corev1.EnvVar{
		Name: k8sconsts.OdigosCloudTokenEnvName,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: k8sconsts.OdigosProSecretName,
				},
				Key: k8sconsts.OdigosCloudApiKeySecretKey,
			},
		},
	}
}

// used to inject the onprem token as env var into odigos components
func OnPremTokenAsEnvVar() corev1.EnvVar {
	return corev1.EnvVar{
		Name: k8sconsts.OdigosOnpremTokenEnvName,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: k8sconsts.OdigosProSecretName,
				},
				Key: k8sconsts.OdigosOnpremTokenSecretKey,
			},
		},
	}
}

func getCurrentOdigosProSecret(ctx context.Context, client *kube.Client, ns string) (*corev1.Secret, error) {
	secret, err := client.CoreV1().Secrets(ns).Get(ctx, k8sconsts.OdigosProSecretName, metav1.GetOptions{})
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
