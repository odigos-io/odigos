package resources

import (
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	OdigosDeploymentConfigMapName = "odigos-deployment"
)

func NewOdigosDeploymentConfigMap(odigosVersion string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: v1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: OdigosDeploymentConfigMapName,
		},
		Data: map[string]string{
			"ODIGOS_VERSION": odigosVersion,
		},
	}
}
