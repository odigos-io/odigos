package k8sconfig

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/odigos-io/odigos/common/config"
)

type K8sExporterConfigurer interface {
	config.ExporterConfigurer
	GetSecretRef() *corev1.LocalObjectReference
}
