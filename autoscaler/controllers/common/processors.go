package common

import (
	"context"
	"sort"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func FilterAndSortProcessorsByOrderHint(processors *odigosv1.ProcessorList, collectorRole odigosv1.CollectorsGroupRole) []*odigosv1.Processor {
	filteredProcessors := []*odigosv1.Processor{}
	for i, processor := range processors.Items {

		// do not include disabled processors
		if processor.Spec.Disabled {
			continue
		}

		// take only processors that participate in this collector role
		for _, role := range processor.Spec.CollectorRoles {
			if role == collectorRole {
				filteredProcessors = append(filteredProcessors, &processors.Items[i])
			}
		}
	}

	// Now sort the filteredProcessors by the OrderHint property
	sort.Slice(filteredProcessors, func(i, j int) bool {
		return filteredProcessors[i].Spec.OrderHint < filteredProcessors[j].Spec.OrderHint
	})

	return filteredProcessors
}

func DeleteProcessorByType(c client.Client, processorType string, namespace string) error {
	var processors odigosv1.ProcessorList
	if err := c.List(context.TODO(), &processors, client.InNamespace(namespace)); err != nil {
		return client.IgnoreNotFound(err)
	}

	if processor := GetProcessorIfExists(&processors, processorType); processor != nil {
		return c.Delete(context.TODO(), processor)
	}

	return nil
}

func GetProcessorIfExists(allProcessors *odigosv1.ProcessorList, processorType string) *odigosv1.Processor {
	for _, processor := range allProcessors.Items {
		if processor.Spec.Type == processorType {
			return &processor
		}
	}
	return nil
}
