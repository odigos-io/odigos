package common

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

func GetDeploymentName(gatewayCg *odigosv1.CollectorsGroup) string {
	if gatewayCg.Spec.DeploymentName != "" {
		return gatewayCg.Spec.DeploymentName
	}
	return k8sconsts.OdigosClusterCollectorDeploymentName
}
