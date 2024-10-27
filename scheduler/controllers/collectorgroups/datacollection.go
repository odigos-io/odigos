package collectorgroups

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewNodeCollectorGroup(odigosConfig common.OdigosConfiguration) *odigosv1.CollectorsGroup {

	ownMetricsPort := consts.OdigosNodeCollectorOwnTelemetryPortDefault
	if odigosConfig.CollectorNode != nil && odigosConfig.CollectorNode.CollectorOwnMetricsPort != 0 {
		ownMetricsPort = odigosConfig.CollectorNode.CollectorOwnMetricsPort
	}

	return &odigosv1.CollectorsGroup{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CollectorsGroup",
			APIVersion: "odigos.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosNodeCollectorDaemonSetName,
			Namespace: env.GetCurrentNamespace(),
		},
		Spec: odigosv1.CollectorsGroupSpec{
			Role:                    odigosv1.CollectorsGroupRoleNodeCollector,
			CollectorOwnMetricsPort: ownMetricsPort,
		},
	}
}

func ShouldHaveNodeCollectorGroup(gatewayReady bool, numberofInstrumentedApps int) bool {
	return gatewayReady && numberofInstrumentedApps > 0
}
