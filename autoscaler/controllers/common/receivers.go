package common

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/config"
)

func FilterReceivers(receivers *odigosv1.ReceiverList, collectorRole odigosv1.CollectorsGroupRole) []*odigosv1.Receiver {
	filtered := []*odigosv1.Receiver{}
	for i := range receivers.Items {
		r := &receivers.Items[i]
		if r.Spec.Disabled {
			continue
		}
		for _, role := range r.Spec.CollectorRoles {
			if role == collectorRole {
				filtered = append(filtered, r)
				break
			}
		}
	}
	return filtered
}

func ToReceiverConfigurerArray(items []*odigosv1.Receiver) []config.ReceiverConfigurer {
	configurers := make([]config.ReceiverConfigurer, len(items))
	for i := range items {
		configurers[i] = items[i]
	}
	return configurers
}
