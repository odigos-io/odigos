package k8sconfig

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/common"
)

var availableK8sConfigers = map[common.DestinationType]K8sConfiger{
	common.GoogleCloudDestinationType:     &GoogleCloud{},
	common.GoogleCloudOTLPDestinationType: &GoogleCloud{},
	common.ClickhouseDestinationType:      &Clickhouse{},
}

// K8sConfiger is the interface for modifying the gateway collector deployment.
// It is linked to a common config destination type.
type K8sConfiger interface {
	DestType() common.DestinationType
	ModifyGatewayCollectorDeployment(ctx context.Context, k8sClient client.Client, dest K8sExporterConfigurer, currentDeployment *appsv1.Deployment) error
}

// LoadK8sConfigers loads the available K8sConfigers, mapped to their commonconfig destination type.
func LoadK8sConfigers() map[common.DestinationType]K8sConfiger {
	return availableK8sConfigers
}
