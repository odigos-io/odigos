package collectorgroups

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewClusterCollectorGroup(namespace string) *odigosv1.CollectorsGroup {
	return &odigosv1.CollectorsGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosClusterCollectorCollectorGroupName,
			Namespace: namespace,
		},
		Spec: odigosv1.CollectorsGroupSpec{
			Role: odigosv1.CollectorsGroupRoleClusterGateway,
		},
	}
}
