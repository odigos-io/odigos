package k8sconsts

import (
	"github.com/odigos-io/odigos/common/consts"
)

const (
	OdigosDeploymentConfigMapName                    = "odigos-deployment"
	OdigosDeploymentConfigMapVersionKey              = consts.OdigosVersionEnvVarName
	OdigosDeploymentConfigMapTierKey                 = consts.OdigosTierEnvVarName
	OdigosDeploymentConfigMapInstallationMethodKey   = "installation-method"
	OdigosDeploymentConfigMapKubernetesVersionKey    = "kubernetes-version"
	OdigosDeploymentConfigMapOnPremTokenAudKey       = "onprem-token-audience"
	OdigosDeploymentConfigMapOnPremTokenExpKey       = "onprem-token-expiration"
	OdigosDeploymentConfigMapOnPremClientProfilesKey = "onprem-profiles"
	OdigosDeploymentConfigMapOdigosDeploymentIDKey   = "odigos-deployment-id"

	OdigosLocalUiInstallationStatusKey = "installation-status"
)
