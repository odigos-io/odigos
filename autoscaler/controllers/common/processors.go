package common

import (
	"encoding/json"
	"fmt"
	"sort"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
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

func FindFirstProcessorByType(allProcessors *odigosv1.ProcessorList, processorType string) *odigosv1.Processor {
	for _, processor := range allProcessors.Items {
		if processor.Spec.Type == processorType {
			return &processor
		}
	}
	return nil
}

func GetGenericBatchProcessor() odigosv1.Processor {
	emptyConfig, _ := json.Marshal(make(map[string]interface{}))

	return odigosv1.Processor{
		ObjectMeta: metav1.ObjectMeta{
			Name: "generic-batch-processor",
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "Processor",
		},
		Spec: odigosv1.ProcessorSpec{
			Type: "batch",
			CollectorRoles: []odigosv1.CollectorsGroupRole{
				odigosv1.CollectorsGroupRoleClusterGateway,
				odigosv1.CollectorsGroupRoleNodeCollector},
			OrderHint:       0,
			ProcessorConfig: runtime.RawExtension{Raw: emptyConfig},
			Signals: []common.ObservabilitySignal{
				common.TracesObservabilitySignal,
				common.MetricsObservabilitySignal,
				common.LogsObservabilitySignal,
			},
		},
	}
}

type MatchCondition struct {
	Name      string `mapstructure:"name"`
	Namespace string `mapstructure:"namespace"`
	Kind      string `mapstructure:"kind"`
}

func AddFilterProcessors(allProcessors *odigosv1.ProcessorList, dests *odigosv1.DestinationList, sources *odigosv1.SourceList) {
	for _, dest := range dests.Items {
		//TODO: remove this log
		logger := log.Log.WithValues("destination", dest.Name)
		logger.Info("Processing destination for filter processor")
		//TODO: remove this log
		matchedSources := filterSources(sources.Items, dest.Spec.SourceSelector)

		// if len(matchedSources) == 0 {
		// 	//TODO: remove this log
		// 	logger.Info("No matching sources found for destination. Skipping processor creation.")

		// 	//TODO: remove this log
		// 	continue
		// }
		//TODO: remove this log
		logger.Info("Matched sources for destination", "matchedSources", matchedSources)
		//TODO: remove this log
		var matchConditions []map[string]string
		for _, source := range matchedSources {
			//TODO: remove this log
			logger.Info("Adding match condition for source", "sourceName", source.Spec.Workload.Name, "namespace", source.Spec.Workload.Namespace, "kind", source.Spec.Workload.Kind)
			//TODO: remove this log

			matchCondition := map[string]string{
				"name":      source.Spec.Workload.Name,
				"namespace": source.Spec.Workload.Namespace,
				"kind":      string(source.Spec.Workload.Kind),
			}
			matchConditions = append(matchConditions, matchCondition)
		}

		filterConfig := map[string]interface{}{
			"match_conditions": matchConditions,
		}

		allProcessors.Items = append(allProcessors.Items, odigosv1.Processor{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("odigossourcetodestinationfilter-%s", dest.Name),
			},
			Spec: odigosv1.ProcessorSpec{
				Type:            "odigossourcetodestinationfilterprocessor",
				ProcessorConfig: runtime.RawExtension{Raw: marshalConfig(filterConfig)},
				CollectorRoles: []odigosv1.CollectorsGroupRole{
					odigosv1.CollectorsGroupRoleClusterGateway,
				},
				OrderHint: len(allProcessors.Items) + 1,
				Signals:   dest.Spec.Signals,
			},
		})
		//TODO: remove this log
		logger.Info("Filter processor added successfully", "processorName", fmt.Sprintf("odigossourcetodestinationfilter-%s", dest.Name))
		//TODO: remove this log

	}
}

func filterSources(sources []odigosv1.Source, selector *odigosv1.SourceSelector) []odigosv1.Source {
	if selector == nil || selector.Mode == "all" {
		return sources
	}

	var filtered []odigosv1.Source
	for _, source := range sources {
		if selector.Mode == "namespaces" && contains(selector.Namespaces, source.Spec.Workload.Namespace) {
			filtered = append(filtered, source)
		} else if selector.Mode == "groups" {
			// Handle nil or empty groups gracefully
			if source.Spec.Groups == nil || len(source.Spec.Groups) == 0 {
				continue
			}
			if containsAny(selector.Groups, source.Spec.Groups) {
				filtered = append(filtered, source)
			}
		}
	}
	return filtered
}

func contains(arr []string, val string) bool {
	for _, item := range arr {
		if item == val {
			return true
		}
	}
	return false
}

func containsAny(arr1, arr2 []string) bool {
	for _, item1 := range arr1 {
		for _, item2 := range arr2 {
			if item1 == item2 {
				return true
			}
		}
	}
	return false
}

func buildFilterConditions(sources []odigosv1.Source) []string {
	var conditions []string
	for _, source := range sources {
		conditions = append(conditions, source.Name)
	}
	return conditions
}

func marshalConfig(config map[string]interface{}) []byte {
	data, err := json.Marshal(config)
	if err != nil {
		log.Log.Error(err, "Failed to marshal processor config")
	}
	return data
}
