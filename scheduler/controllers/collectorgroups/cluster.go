package collectorgroups

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The cluster gateway collector runs as a deployment and the pod is exposed as a service.
// Thus it cannot collide with other ports on the same node, and we can use an handy default port.
const ClusterCollectorDefaultOwnMetricsPort = 8888

func NewClusterCollectorGroup(namespace string) *odigosv1.CollectorsGroup {
	return &odigosv1.CollectorsGroup{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CollectorsGroup",
			APIVersion: "odigos.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosClusterCollectorCollectorGroupName,
			Namespace: namespace,
		},
		Spec: odigosv1.CollectorsGroupSpec{
			Role:                    odigosv1.CollectorsGroupRoleClusterGateway,
			CollectorOwnMetricsPort: ClusterCollectorDefaultOwnMetricsPort,
		},
	}
}
