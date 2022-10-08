package resources

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NamespaceLabelKey   = "odigos.io/installation-namespace"
	NamespaceLabelValue = "true"
)

func NewNamespace(name string) *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				NamespaceLabelKey: NamespaceLabelValue,
			},
		},
	}
}
