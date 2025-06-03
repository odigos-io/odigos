package clustercollector

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/autoscaler/controllers/common"
	odigoscommon "github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/consts"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
	pipelinegen "github.com/odigos-io/odigos/common/pipelinegen"
	odgiosK8s "github.com/odigos-io/odigos/k8sutils/pkg/conditions"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	destinationConfiguredType = "DestinationConfigured"
)

var (
	errNoPipelineConfigured  = errors.New("no pipeline was configured, cannot add self telemetry pipeline")
	errNoReceiversConfigured = errors.New("no receivers were configured, cannot add self telemetry pipeline")
	errNoExportersConfigured = errors.New("no exporters were configured, cannot add self telemetry pipeline")
)

func addSelfTelemetryPipeline(c *config.Config, ownTelemetryPort int32, destinationPipelineNames []string, signalsRootPipelines []string) error {
	if c.Service.Pipelines == nil {
		return errNoPipelineConfigured
	}
	if c.Receivers == nil {
		return errNoReceiversConfigured
	}
	if c.Exporters == nil {
		return errNoExportersConfigured
	}
	c.Receivers["prometheus/self-metrics"] = config.GenericMap{
		"config": config.GenericMap{
			"scrape_configs": []config.GenericMap{
				{
					"job_name":        "otelcol",
					"scrape_interval": "10s",
					"static_configs": []config.GenericMap{
						{
							"targets": []string{fmt.Sprintf("127.0.0.1:%d", ownTelemetryPort)},
						},
					},
					"metric_relabel_configs": []config.GenericMap{
						{
							"source_labels": []string{"__name__"},
							"regex":         "(.*odigos.*|^otelcol_exporter_sent.*)",
							"action":        "keep",
						},
					},
				},
			},
		},
	}
	if c.Processors == nil {
		c.Processors = make(config.GenericMap)
	}
	c.Processors["resource/pod-name"] = config.GenericMap{
		"attributes": []config.GenericMap{
			{
				"key":    "k8s.pod.name",
				"value":  "${POD_NAME}",
				"action": "upsert",
			},
		},
	}
	// odigostrafficmetrics processor should be the last processor in the pipeline
	// as it helps to calculate the size of the data being exported.
	// In case of performance impact caused by this processor, we should modify this config to reduce the sampling ratio.
	c.Processors["odigostrafficmetrics"] = struct{}{}
	c.Exporters["otlp/odigos-own-telemetry-ui"] = config.GenericMap{
		"endpoint": fmt.Sprintf("ui.%s:%d", env.GetCurrentNamespace(), odigosconsts.OTLPPort),
		"tls": config.GenericMap{
			"insecure": true,
		},
		"retry_on_failure": config.GenericMap{
			"enabled": false,
		},
	}
	c.Service.Pipelines["metrics/otelcol"] = config.Pipeline{
		Receivers:  []string{"prometheus/self-metrics"},
		Processors: []string{"resource/pod-name"},
		Exporters:  []string{"otlp/odigos-own-telemetry-ui"},
	}

	c.Service.Telemetry.Metrics = config.GenericMap{
		"readers": []config.GenericMap{
			{
				"pull": config.GenericMap{
					"exporter": config.GenericMap{
						"prometheus": config.GenericMap{
							"host": "0.0.0.0",
							"port": ownTelemetryPort,
						},
					},
				},
			},
		},
	}

	// Add the odigostrafficmetrics processor to both root and destination pipelines to track telemetry data flow
	for pipelineName, pipeline := range c.Service.Pipelines {
		if !isOdigosTrafficMetricsProcessorRelevant(pipelineName, signalsRootPipelines, destinationPipelineNames) {
			continue
		}
		pipeline.Processors = append(pipeline.Processors, "odigostrafficmetrics")
		c.Service.Pipelines[pipelineName] = pipeline
	}

	return nil
}

func syncConfigMap(dests *odigosv1.DestinationList, allProcessors *odigosv1.ProcessorList, gateway *odigosv1.CollectorsGroup, ctx context.Context, c client.Client, scheme *runtime.Scheme) ([]odigoscommon.ObservabilitySignal, error) {
	logger := log.FromContext(ctx)
	memoryLimiterConfiguration := common.GetMemoryLimiterConfig(gateway.Spec.ResourcesSettings)

	dataStreams, err := GetDataStreamsWithDestinations(ctx, c, dests, logger)
	if err != nil {
		logger.Error(err, "Failed to build group details")
		return nil, err
	}

	processors := common.FilterAndSortProcessorsByOrderHint(allProcessors, odigosv1.CollectorsGroupRoleClusterGateway)

	desiredData, err, status, signals := pipelinegen.GetGatewayConfig(
		common.ToExporterConfigurerArray(dests),
		common.ToProcessorConfigurerArray(processors),
		memoryLimiterConfiguration,
		func(c *config.Config, destinationPipelineNames []string, signalsRootPipelines []string) error {
			return addSelfTelemetryPipeline(c, gateway.Spec.CollectorOwnMetricsPort, destinationPipelineNames, signalsRootPipelines)
		},
		dataStreams,
	)

	if err != nil {
		logger.Error(err, "Failed to calculate config")
		return nil, err
	}

	for destName, destErr := range status.Destination {
		if destErr != nil {
			logger.Error(destErr, "Failed to calculate config for destination", "destination", destName)
		}
	}
	for name, err := range status.Processor {
		if err != nil {
			logger.Info(err.Error(), "processor", name)
		}
	}

	// Update destination status conditions in k8s
	for _, dest := range dests.Items {
		if destErr, found := status.Destination[dest.ObjectMeta.Name]; found {
			if destErr != nil {
				err := odgiosK8s.UpdateStatusConditions(ctx, c, &dest, &dest.Status.Conditions, metav1.ConditionFalse, destinationConfiguredType, "ErrConfigDestination", destErr.Error())
				if err != nil {
					logger.Error(err, "Failed to update destination error status conditions")
				}
			} else {
				err := odgiosK8s.UpdateStatusConditions(ctx, c, &dest, &dest.Status.Conditions, metav1.ConditionTrue, destinationConfiguredType, "TransformedToOtelcolConfig", "Destination successfully transformed to otelcol configuration")
				if err != nil {
					logger.Error(err, "Failed to update destination success status conditions")
				}
			}
		}
	}

	desiredCM := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosClusterCollectorConfigMapName,
			Namespace: gateway.Namespace,
		},
		Data: map[string]string{
			k8sconsts.OdigosClusterCollectorConfigMapKey: desiredData,
		},
	}

	if err := ctrl.SetControllerReference(gateway, desiredCM, scheme); err != nil {
		logger.Error(err, "Failed to set controller reference")
		return nil, err
	}

	existing := &v1.ConfigMap{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: gateway.Namespace, Name: k8sconsts.OdigosClusterCollectorConfigMapName}, existing); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(0).Info("Creating gateway config map")
			_, err := createConfigMap(desiredCM, ctx, c)
			if err != nil {
				logger.Error(err, "Failed to create gateway config map")
				return nil, err
			}
			return signals, nil
		} else {
			logger.Error(err, "Failed to get gateway config map")
			return nil, err
		}
	}

	logger.V(0).Info("Patching gateway config map")
	_, err = patchConfigMap(existing, desiredCM, ctx, c)
	if err != nil {
		logger.Error(err, "Failed to patch gateway config map")
		return nil, err
	}

	return signals, nil
}

func createConfigMap(desired *v1.ConfigMap, ctx context.Context, c client.Client) (*v1.ConfigMap, error) {
	if err := c.Create(ctx, desired); err != nil {
		return nil, err
	}

	return desired, nil
}

func patchConfigMap(existing *v1.ConfigMap, desired *v1.ConfigMap, ctx context.Context, c client.Client) (*v1.ConfigMap, error) {
	if reflect.DeepEqual(existing.Data, desired.Data) &&
		reflect.DeepEqual(existing.ObjectMeta.OwnerReferences, desired.ObjectMeta.OwnerReferences) {
		log.FromContext(ctx).V(0).Info("Gateway config maps already match")
		return existing, nil
	}
	updated := existing.DeepCopy()
	updated.Data = desired.Data
	updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences
	patch := client.MergeFrom(existing)
	if err := c.Patch(ctx, updated, patch); err != nil {
		return nil, err
	}

	return updated, nil
}

// GetDataStreamsWithDestinations generates a slice of data streams.
//
// Example return structure:
//
//	[]DataStreams{
//	    {
//	        Name: "dataStreamA",
//	        Sources: []SourceFilter{
//	            {Namespace: "ns1", Kind: "Deployment", Name: "frontend"},
//	            {Namespace: "ns1", Kind: "DaemonSet", Name: "log-agent"},
//	        },
//	        Destinations: []Destination{
//	            {DestinationName: "coralogix",
//				 ConfiguredSignals: []ObservabilitySignal{TracesObservabilitySignal, LogsObservabilitySignal}},
//	        },
//	    },
//	    {
//	        Name: "dataStreamB",
//	        Sources: []SourceFilter{
//	            {Namespace: "ns2", Kind: "StatefulSet", Name: "db"},
//	        },
//	        Destinations: []Destination{
//	            {DestinationName: "jaeger",
//				 ConfiguredSignals: []ObservabilitySignal{TracesObservabilitySignal}},
//	        },
//	    },
//	}
func GetDataStreamsWithDestinations(
	ctx context.Context,
	kubeClient client.Client,
	dests *odigosv1.DestinationList,
	logger logr.Logger,
) ([]pipelinegen.DataStreams, error) {

	dataStreamDetailsList := []pipelinegen.DataStreams{}
	seenDataStreams := make(map[string]struct{})

	for _, dest := range dests.Items {

		// If the destination has no source selector, use the default data stream
		// Otherwise, use the data streams specified in the source selector
		dataStreams := []string{}
		if dest.Spec.SourceSelector == nil {
			dataStreams = append(dataStreams, consts.DefaultDataStream)
		} else {
			dataStreams = dest.Spec.SourceSelector.DataStreams
		}

		for _, dataStream := range dataStreams {
			dataStreamDetails := findOrCreateDataStream(&dataStreamDetailsList, dataStream)

			if !destinationExists(dataStreamDetails.Destinations, dest.Name) {
				dataStreamDetails.Destinations = append(dataStreamDetails.Destinations, pipelinegen.Destination{
					DestinationName:   dest.Name,
					ConfiguredSignals: dest.GetSignals(),
				})
			}

			if _, alreadySeen := seenDataStreams[dataStream]; !alreadySeen {
				seenDataStreams[dataStream] = struct{}{}

				sourcesFilters, namespacesFilters, err := getSourcesForDataStream(ctx, kubeClient, dataStream, logger)
				if err != nil {
					return nil, err
				}
				dataStreamDetails.Sources = sourcesFilters
				dataStreamDetails.Namespaces = namespacesFilters
			}
		}
	}

	return dataStreamDetailsList, nil
}

// getSourcesForDataStream fetches all sources [workload and namespace] that are labeled with the given data stream name.
func getSourcesForDataStream(
	ctx context.Context,
	kubeClient client.Client,
	dataStream string,
	logger logr.Logger,
) ([]pipelinegen.SourceFilter, []pipelinegen.NamespaceFilter, error) {

	sourceList := &odigosv1.SourceList{}
	namespacesSources := &odigosv1.SourceList{}

	if dataStream == consts.DefaultDataStream {
		var err error
		sourceList, namespacesSources, err = getSourcesForDefaultDataStream(ctx, kubeClient, dataStream)
		if err != nil {
			return nil, nil, err
		}
	} else {
		labelSelector := labels.Set{fmt.Sprintf("%s%s", k8sconsts.SourceDataStreamLabelPrefix, dataStream): "true"}.AsSelector()
		err := kubeClient.List(ctx, sourceList, &client.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			logger.Error(err, "Failed to fetch sources for DataStream", "dataStream", dataStream)
			return nil, nil, err
		}
	}

	namespacesFilters := make([]pipelinegen.NamespaceFilter, 0, len(namespacesSources.Items))
	for _, source := range namespacesSources.Items {
		namespacesFilters = append(namespacesFilters, pipelinegen.NamespaceFilter{
			Namespace: source.Spec.Workload.Namespace,
		})
	}

	sourcesFilters := make([]pipelinegen.SourceFilter, 0, len(sourceList.Items))
	for _, source := range sourceList.Items {
		sourcesFilters = append(sourcesFilters, pipelinegen.SourceFilter{
			Namespace: source.Spec.Workload.Namespace,
			Kind:      string(source.Spec.Workload.Kind),
			Name:      source.Spec.Workload.Name,
		})
	}

	return sourcesFilters, namespacesFilters, nil
}

func destinationExists(list []pipelinegen.Destination, item string) bool {
	for _, v := range list {
		if v.DestinationName == item {
			return true
		}
	}
	return false
}

func isOdigosTrafficMetricsProcessorRelevant(name string, rootPipelines []string, destinationPipelines []string) bool {
	// we should not add the odigostrafficmetrics processor to the metrics/otelcol pipeline
	if name == "metrics/otelcol" {
		return false
	}
	// we should add the odigostrafficmetrics processor to all destination pipelines
	if slices.Contains(destinationPipelines, name) {
		return true
	}
	return false
}

func findOrCreateDataStream(dataStreams *[]pipelinegen.DataStreams, name string) *pipelinegen.DataStreams {
	for i := range *dataStreams {
		if (*dataStreams)[i].Name == name {
			return &(*dataStreams)[i]
		}
	}
	newDataStream := pipelinegen.DataStreams{
		Name:         name,
		Sources:      []pipelinegen.SourceFilter{},
		Destinations: []pipelinegen.Destination{},
	}
	*dataStreams = append(*dataStreams, newDataStream)
	return &(*dataStreams)[len(*dataStreams)-1]
}

// For the default data stream, include all sources that don't have any data stream labels assigned
// and sources that have the default data stream label assigned
func getSourcesForDefaultDataStream(ctx context.Context, kubeClient client.Client, group string) (*odigosv1.SourceList, *odigosv1.SourceList, error) {

	defaultStreamSources := &odigosv1.SourceList{}
	err := kubeClient.List(ctx, defaultStreamSources, &client.ListOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list all sources: %w", err)
	}

	workloadSources := []odigosv1.Source{}

	// this is done for namespace that were selected as "future select"
	// in that case a single source will be created for the namespace.
	namespacesSources := []odigosv1.Source{}

	for _, src := range defaultStreamSources.Items {

		if src.Spec.Workload.Kind == k8sconsts.WorkloadKindNamespace {
			namespacesSources = append(namespacesSources, src)
			continue
		}

		hasDataStreamLabel := false
		explicitMatch := false

		for key, value := range src.Labels {
			if strings.HasPrefix(key, k8sconsts.SourceDataStreamLabelPrefix) {
				hasDataStreamLabel = true
				if key == fmt.Sprintf("%s%s", k8sconsts.SourceDataStreamLabelPrefix, group) && value == "true" {
					explicitMatch = true
				}
			}
		}

		// If the source has no data stream labels assigned or has the default data stream label assigned, include it in the filtered list
		if !hasDataStreamLabel || explicitMatch {
			workloadSources = append(workloadSources, src)
		}
	}

	return &odigosv1.SourceList{Items: workloadSources}, &odigosv1.SourceList{Items: namespacesSources}, nil
}
