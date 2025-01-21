package k8sconsts

import (
	"github.com/odigos-io/odigos/common/consts"
)

const (
	OdigosDeploymentConfigMapName                  = "odigos-deployment"
	OdigosDeploymentConfigMapVersionKey            = consts.OdigosVersionEnvVarName
	OdigosDeploymentConfigMapTierKey               = consts.OdigosTierEnvVarName
	OdigosDeploymentConfigMapInstallationMethodKey = "installation-method"
	OdigosDeploymentConfigMapKubernetesVersionKey  = "kubernetes-version"
)
