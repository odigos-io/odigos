package clustercollector

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

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
		"level": "detailed",
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
		if !isOdigosTrafficMetricsProcessorRelevant(pipelineName, destinationPipelineNames) {
			continue
		}
		pipeline.Processors = append(pipeline.Processors, "odigostrafficmetrics")
		c.Service.Pipelines[pipelineName] = pipeline
	}

	return nil
}

func syncConfigMap(enabledDests *odigosv1.DestinationList, allProcessors *odigosv1.ProcessorList, gateway *odigosv1.CollectorsGroup, ctx context.Context, c client.Client, scheme *runtime.Scheme) ([]odigoscommon.ObservabilitySignal, error) {
	logger := log.FromContext(ctx)

	dataStreams, err := calculateDataStreams(enabledDests)
	if err != nil {
		logger.Error(err, "Failed to build group details")
		return nil, err
	}

	processors := common.FilterAndSortProcessorsByOrderHint(allProcessors, odigosv1.CollectorsGroupRoleClusterGateway)

	gatewayOptions := pipelinegen.GatewayConfigOptions{
		ServiceGraphDisabled:  gateway.Spec.ServiceGraphDisabled,
		ClusterMetricsEnabled: gateway.Spec.ClusterMetricsEnabled,
		OdigosNamespace:       env.GetCurrentNamespace(),
	}

	desiredData, err, status, signals := pipelinegen.GetGatewayConfig(
		common.ToExporterConfigurerArray(enabledDests),
		common.ToProcessorConfigurerArray(processors),
		func(c *config.Config, destinationPipelineNames []string, signalsRootPipelines []string) error {
			return addSelfTelemetryPipeline(c, gateway.Spec.CollectorOwnMetricsPort, destinationPipelineNames, signalsRootPipelines)
		},
		dataStreams, gatewayOptions,
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
	for _, dest := range enabledDests.Items {
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

// calculateDataStreams generates a slice of data streams.
//
// Example return structure:
//
//	[]DataStreams{
//	    {
//	        Name: "dataStreamA",
//	        Destinations: []Destination{
//	            {DestinationName: "coralogix",
//				 ConfiguredSignals: []ObservabilitySignal{TracesObservabilitySignal, LogsObservabilitySignal}},
//	        },
//	    },
//	    {
//	        Name: "dataStreamB",
//	        Destinations: []Destination{
//	            {DestinationName: "jaeger",
//				 ConfiguredSignals: []ObservabilitySignal{TracesObservabilitySignal}},
//	        },
//	    },
//	}
func calculateDataStreams(
	dests *odigosv1.DestinationList,
) ([]pipelinegen.DataStreams, error) {

	dataStreamsMap := make(map[string]*pipelinegen.DataStreams)

	// Handle case where no enabled destinations exist
	if dests == nil || len(dests.Items) == 0 {
		return []pipelinegen.DataStreams{}, nil
	}

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
			// Get or create data stream using map directly
			dataStreamDetails, exists := dataStreamsMap[dataStream]
			if !exists {
				dataStreamDetails = &pipelinegen.DataStreams{
					Name:         dataStream,
					Destinations: []pipelinegen.Destination{},
				}

				dataStreamsMap[dataStream] = dataStreamDetails
			}

			if !destinationExists(dataStreamDetails.Destinations, dest.Name) {
				dataStreamDetails.Destinations = append(dataStreamDetails.Destinations, pipelinegen.Destination{
					DestinationName:   dest.Name,
					ConfiguredSignals: dest.GetSignals(),
				})
			}
		}
	}

	// Convert map to slice, this is the final result as will be used in the configmap
	dataStreamDetailsList := make([]pipelinegen.DataStreams, 0, len(dataStreamsMap))
	for _, ds := range dataStreamsMap {
		dataStreamDetailsList = append(dataStreamDetailsList, *ds)
	}

	// Sort by Name to ensure consistent ordering
	slices.SortFunc(dataStreamDetailsList, func(a, b pipelinegen.DataStreams) int {
		return strings.Compare(a.Name, b.Name)
	})

	return dataStreamDetailsList, nil
}

func destinationExists(existingDestinations []pipelinegen.Destination, destinationName string) bool {
	for _, v := range existingDestinations {
		if v.DestinationName == destinationName {
			return true
		}
	}
	return false
}

func isOdigosTrafficMetricsProcessorRelevant(name string, destinationPipelines []string) bool {
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
