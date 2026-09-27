package clustercollector

import (
	"os"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	corev1 "k8s.io/api/core/v1"
)

func collectorLicenseEnv(tier common.OdigosTier) []corev1.EnvVar {
	if !tier.IsEnterprise() {
		return nil
	}
	if os.Getenv("ODIGOS_LICENSE_PROVIDER") == "aws-marketplace" {
		return []corev1.EnvVar{
			{Name: "ODIGOS_LICENSE_PROVIDER", Value: "aws-marketplace"},
		}
	}
	return []corev1.EnvVar{{
		Name: k8sconsts.OdigosOnpremTokenEnvName,
		ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: k8sconsts.OdigosProSecretName},
			Key:                  k8sconsts.OdigosOnpremTokenSecretKey,
		}},
	}}
}
