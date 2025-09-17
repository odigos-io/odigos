package k8sconfig

import (
	appsv1 "k8s.io/api/apps/v1"

	"github.com/odigos-io/odigos/common"
)

var availableK8sConfigers = map[common.DestinationType]K8sConfiger{
	common.GoogleCloudDestinationType: &GoogleCloud{},
}

// K8sConfiger is the interface for modifying the gateway collector deployment.
// It is linked to a common config destination type.
type K8sConfiger interface {
	DestType() common.DestinationType
	ModifyGatewayCollectorDeployment(dest K8sExporterConfigurer, currentDeployment *appsv1.Deployment) error
}

// LoadK8sConfigers loads the available K8sConfigers, mapped to their commonconfig destination type.
func LoadK8sConfigers() map[common.DestinationType]K8sConfiger {
	return availableK8sConfigers
}
