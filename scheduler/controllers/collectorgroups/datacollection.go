package collectorgroups

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	dataCollectionName = "odigos-data-collection"
)

func NewDataCollection(namespace string) *odigosv1.CollectorsGroup {
	return &odigosv1.CollectorsGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dataCollectionName,
			Namespace: namespace,
		},
		Spec: odigosv1.CollectorsGroupSpec{
			Role: odigosv1.CollectorsGroupRoleNodeCollector,
		},
	}
}
