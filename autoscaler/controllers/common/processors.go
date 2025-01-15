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
	"github.com/odigos-io/odigos/common/config"
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

func GenerateRoutingProcessors(
	ctx context.Context,
	kubeClient client.Client,
	dests *odigosv1.DestinationList,
) map[string]config.GenericMap {
	logger := log.FromContext(ctx)
	routingProcessors := make(map[string]config.GenericMap)

	for _, dest := range dests.Items {

		if dest.Spec.SourceSelector == nil {
			continue
		}

		matchConditions := []string{}
		if len(dest.Spec.SourceSelector.Namespaces) > 0 {
			for _, namespace := range dest.Spec.SourceSelector.Namespaces {
				matchConditions = append(matchConditions, fmt.Sprintf("%s/*/*", namespace))
			}
		}
		if len(dest.Spec.SourceSelector.Groups) > 0 {
			matchedSources := fetchSourcesByGroups(ctx, kubeClient, dest.Spec.SourceSelector.Groups, logger)
			for _, source := range matchedSources {
				key := fmt.Sprintf("%s/%s/%s", source.Spec.Workload.Namespace, source.Spec.Workload.Name, source.Spec.Workload.Kind)
				matchConditions = append(matchConditions, key)
			}
		}

		sanitizedProcessorName := strings.ReplaceAll(dest.GetID(), ".", "-")
		processorName := fmt.Sprintf("odigosroutingfilterprocessor/%s", sanitizedProcessorName)

		routingProcessors[processorName] = config.GenericMap{
			"match_conditions": matchConditions,
		}
	}

	return routingProcessors
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
	sourceMap := make(map[string]odigosv1.Source)
	for _, group := range groups {
		labelSelector := labels.Set{fmt.Sprintf("odigos.io/group-%s", group): "true"}.AsSelector()

		sourceList := &odigosv1.SourceList{}
		err := kubeClient.List(ctx, sourceList, &client.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			logger.Error(err, "Failed to fetch sources for group", "group", group)
			continue
		}

		for _, source := range sourceList.Items {
			key := fmt.Sprintf("%s/%s", source.Namespace, source.Name)
			if _, exists := sourceMap[key]; !exists {
				sourceMap[key] = source
			}
		}
	}
	sources := make([]odigosv1.Source, 0, len(sourceMap))
	for _, source := range sourceMap {
		sources = append(sources, source)
	}
	return sources
}
