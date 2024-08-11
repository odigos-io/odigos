package gateway

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/autoscaler/controllers/common"
	odigoscommon "github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
	odgiosK8s "github.com/odigos-io/odigos/k8sutils/pkg/conditions"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
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
	errNoPipelineConfigured = errors.New("no pipeline was configured, cannot add self telemetry pipeline")
	errNoReceiversConfigured = errors.New("no receivers were configured, cannot add self telemetry pipeline")
	errNoExportersConfigured = errors.New("no exporters were configured, cannot add self telemetry pipeline")
)

func addSelfTelemetryPipeline(c *config.Config) error {
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
					"job_name": "otelcol",
					"scrape_interval": "10s",
					"static_configs": []config.GenericMap{
						{
							"targets": []string{"127.0.0.1:8888"},
						},
					},
					"metric_relabel_configs": []config.GenericMap{
						{
							"source_labels": []string{"__name__"},
							"regex": "(.*odigos.*|^otelcol_processor_accepted.*|^otelcol_exporter_sent.*)",
							"action": "keep",
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
		Receivers: []string{"prometheus/self-metrics"},
		Processors: []string{"resource/pod-name"},
		Exporters: []string{"otlp/odigos-own-telemetry-ui"},
	}

	c.Service.Telemetry.Metrics = config.GenericMap{
		"address": "0.0.0.0:8888",
	}

	for pipelineName, pipeline := range c.Service.Pipelines {
		if pipelineName == "metrics/otelcol" {
			continue
		}
		pipeline.Processors = append(pipeline.Processors, "odigostrafficmetrics")
		c.Service.Pipelines[pipelineName] = pipeline
	}

	return nil
}

func syncConfigMap(dests *odigosv1.DestinationList, allProcessors *odigosv1.ProcessorList, gateway *odigosv1.CollectorsGroup, ctx context.Context, c client.Client, scheme *runtime.Scheme, memConfig *memoryConfigurations) (string, []odigoscommon.ObservabilitySignal, error) {
	logger := log.FromContext(ctx)

	memoryLimiterConfiguration := config.GenericMap{
		"check_interval":  "1s",
		"limit_mib":       memConfig.memoryLimiterLimitMiB,
		"spike_limit_mib": memConfig.memoryLimiterSpikeLimitMiB,
	}

	processors := common.FilterAndSortProcessorsByOrderHint(allProcessors, odigosv1.CollectorsGroupRoleClusterGateway)

	desiredData, err, status, signals := config.Calculate(
		common.ToExporterConfigurerArray(dests),
		common.ToProcessorConfigurerArray(processors),
		memoryLimiterConfiguration,
		addSelfTelemetryPipeline,
	)
	if err != nil {
		logger.Error(err, "Failed to calculate config")
		return "", nil, err
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
				err := odgiosK8s.UpdateStatusConditions(ctx, c, &dest, &dest.Status.Conditions, metav1.ConditionTrue, destinationConfiguredType, "TransformedToOtelcolConfig", "destination successfully transformed to otelcol configuration")
				if err != nil {
					logger.Error(err, "Failed to update destination success status conditions")
				}
			}
		}
	}

	desiredCM := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosClusterCollectorConfigMapName,
			Namespace: gateway.Namespace,
		},
		Data: map[string]string{
			consts.OdigosClusterCollectorConfigMapKey: desiredData,
		},
	}

	if err := ctrl.SetControllerReference(gateway, desiredCM, scheme); err != nil {
		logger.Error(err, "Failed to set controller reference")
		return "", nil, err
	}

	existing := &v1.ConfigMap{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: gateway.Namespace, Name: consts.OdigosClusterCollectorConfigMapName}, existing); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(0).Info("Creating gateway config map")
			_, err := createConfigMap(desiredCM, ctx, c)
			if err != nil {
				logger.Error(err, "Failed to create gateway config map")
				return "", nil, err
			}
			return desiredData, signals, nil
		} else {
			logger.Error(err, "Failed to get gateway config map")
			return "", nil, err
		}
	}

	logger.V(0).Info("Patching gateway config map")
	_, err = patchConfigMap(existing, desiredCM, ctx, c)
	if err != nil {
		logger.Error(err, "Failed to patch gateway config map")
		return "", nil, err
	}

	return desiredData, signals, nil
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
