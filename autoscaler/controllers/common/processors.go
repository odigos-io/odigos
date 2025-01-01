package common

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func AddFilterProcessors(ctx context.Context, kubeClient client.Client, allProcessors *odigosv1.ProcessorList, dests *odigosv1.DestinationList) {
	for _, dest := range dests.Items {
		logger := log.Log.WithValues("destination", dest.Name)
		logger.Info("Processing destination for filter processor")

		if dest.Spec.SourceSelector == nil || contains(dest.Spec.SourceSelector.Modes, "all") {
			logger.Info("Skipping destination as SourceSelector is nil or set to 'all'")
			continue
		}

		var matchedSources []odigosv1.Source
		if contains(dest.Spec.SourceSelector.Modes, "namespaces") {
			matchedSources = append(matchedSources, fetchSourcesByNamespaces(ctx, kubeClient, dest.Spec.SourceSelector.Namespaces, logger)...)
		}
		if contains(dest.Spec.SourceSelector.Modes, "groups") {
			matchedSources = append(matchedSources, fetchSourcesByGroups(ctx, kubeClient, dest.Spec.SourceSelector.Groups, logger)...)
		}

		logger.Info("Matched sources for destination", "matchedSources", matchedSources)

		matchConditions := make(map[string]bool)
		for _, source := range matchedSources {
			logger.Info("Adding match condition for source", "sourceName", source.Spec.Workload.Name, "namespace", source.Spec.Workload.Namespace, "kind", source.Spec.Workload.Kind)

			key := fmt.Sprintf("%s/%s/%s", source.Spec.Workload.Namespace, source.Spec.Workload.Name, source.Spec.Workload.Kind)
			matchConditions[key] = true
		}

		if len(matchConditions) == 0 {
			logger.Info("No matched sources for destination. Skipping processor creation.")
			continue
		}

		filterConfig := map[string]interface{}{
			"match_conditions": matchConditions,
		}

		allProcessors.Items = append(allProcessors.Items, odigosv1.Processor{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("odigosroutingfilterprocessor-%s", dest.Name),
			},
			Spec: odigosv1.ProcessorSpec{
				Type:            "odigosroutingfilterprocessor",
				ProcessorConfig: runtime.RawExtension{Raw: marshalConfig(filterConfig)},
				CollectorRoles: []odigosv1.CollectorsGroupRole{
					odigosv1.CollectorsGroupRoleClusterGateway,
				},
				OrderHint: len(allProcessors.Items) + 1,
				Signals:   dest.Spec.Signals,
			},
		})

	}
}

func fetchSourcesByNamespaces(ctx context.Context, kubeClient client.Client, namespaces []string, logger logr.Logger) []odigosv1.Source {
	var sources []odigosv1.Source
	for _, ns := range namespaces {
		sourceList := &odigosv1.SourceList{}
		err := kubeClient.List(ctx, sourceList, &client.ListOptions{Namespace: ns})
		if err != nil {
			logger.Error(err, "Failed to fetch sources by namespace", "namespace", ns)
			continue
		}
		sources = append(sources, sourceList.Items...)
	}
	return sources
}

func fetchSourcesByGroups(ctx context.Context, kubeClient client.Client, groups []string, logger logr.Logger) []odigosv1.Source {
	selectors := make([]string, len(groups))
	for i, group := range groups {
		selectors[i] = fmt.Sprintf("odigos.io/group-%s=true", group)
	}
	labelSelector := labels.SelectorFromSet(labels.Set{
		strings.Join(selectors, ","): "",
	})

	sourceList := &odigosv1.SourceList{}
	err := kubeClient.List(ctx, sourceList, &client.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		logger.Error(err, "Failed to fetch sources by groups", "groups", groups)
		return nil
	}
	return sourceList.Items
}

func marshalConfig(config map[string]interface{}) []byte {
	data, err := json.Marshal(config)
	if err != nil {
		log.Log.Error(err, "Failed to marshal processor config")
	}
	return data
}

func contains(arr []string, val string) bool {
	for _, item := range arr {
		if item == val {
			return true
		}
	}
	return false
}
