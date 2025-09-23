package nodecollector

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/autoscaler/controllers/nodecollector/collectorconfig"
	odigoscommon "github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	k8sAttributesProcessorName   = "k8sattributes/odigos-k8sattributes"
	logsServiceNameProcessorName = "resource/service-name"
)

func (b *nodeCollectorBaseReconciler) SyncConfigMap(ctx context.Context, sources *odigosv1.InstrumentationConfigList, clusterCollectorSignals []odigoscommon.ObservabilitySignal, allProcessors *odigosv1.ProcessorList,
	datacollection *odigosv1.CollectorsGroup) error {
	logger := log.FromContext(ctx)

	processors := commonconf.FilterAndSortProcessorsByOrderHint(allProcessors, odigosv1.CollectorsGroupRoleNodeCollector)

	if b.autoscalerDeployment == nil {
		// we only need to get the autoscaler deployment once since it can't change while this code is running
		// (since we are running in the autoscaler pod)
		autoscalerDeployment := &appsv1.Deployment{}
		err := b.Client.Get(ctx, client.ObjectKey{Namespace: env.GetCurrentNamespace(), Name: k8sconsts.AutoScalerDeploymentName}, autoscalerDeployment)
		if err != nil {
			return err
		}
		b.autoscalerDeployment = autoscalerDeployment
	}

	desired, err := b.getDesiredConfigMap(sources, clusterCollectorSignals, processors, datacollection)
	if err != nil {
		logger.Error(err, "failed to get desired config map")
		return err
	}

	existing := &v1.ConfigMap{}
	if err := b.Client.Get(ctx, client.ObjectKey{Namespace: env.GetCurrentNamespace(), Name: k8sconsts.OdigosNodeCollectorCollectorGroupName}, existing); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(0).Info("creating config map")
			_, err := b.createConfigMap(desired, ctx)
			if err != nil {
				logger.Error(err, "failed to create config map")
				return err
			}
			return nil
		} else {
			logger.Error(err, "failed to get config map")
			return err
		}
	}

	logger.V(0).Info("Patching config map")
	_, err = b.patchConfigMap(ctx, existing, desired)
	if err != nil {
		logger.Error(err, "failed to patch config map")
		return err
	}

	return nil
}

func (b *nodeCollectorBaseReconciler) patchConfigMap(ctx context.Context, existing *v1.ConfigMap, desired *v1.ConfigMap) (*v1.ConfigMap, error) {
	if reflect.DeepEqual(existing.Data, desired.Data) &&
		reflect.DeepEqual(existing.ObjectMeta.OwnerReferences, desired.ObjectMeta.OwnerReferences) {
		log.FromContext(ctx).V(0).Info("Config maps already match")
		return existing, nil
	}
	updated := existing.DeepCopy()
	updated.Data = desired.Data
	updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences
	patch := client.MergeFrom(existing)
	if err := b.Client.Patch(ctx, updated, patch); err != nil {
		return nil, err
	}

	return updated, nil
}

func (b *nodeCollectorBaseReconciler) createConfigMap(desired *v1.ConfigMap, ctx context.Context) (*v1.ConfigMap, error) {
	if err := b.Client.Create(ctx, desired); err != nil {
		return nil, err
	}

	return desired, nil
}

func noopConfigMap() (string, error) {
	config := config.Config{
		Extensions: config.GenericMap{
			"health_check": config.GenericMap{
				"endpoint": "0.0.0.0:13133",
			},
		},
		Receivers: config.GenericMap{
			"otlp": config.GenericMap{
				"protocols": config.GenericMap{
					"grpc": config.GenericMap{
						"endpoint": "0.0.0.0:4317",
					},
					"http": config.GenericMap{
						"endpoint": "0.0.0.0:4318",
					},
				},
			},
		},
		Exporters: config.GenericMap{
			"nop": config.GenericMap{},
		},
		Service: config.Service{
			Extensions: []string{"health_check"},
			Pipelines: map[string]config.Pipeline{
				"traces": {
					Receivers:  []string{"otlp"},
					Processors: []string{},
					Exporters:  []string{"nop"},
				},
				"metrics": {
					Receivers:  []string{"otlp"},
					Processors: []string{},
					Exporters:  []string{"nop"},
				},
				"logs": {
					Receivers:  []string{"otlp"},
					Processors: []string{},
					Exporters:  []string{"nop"},
				},
			},
		},
	}

	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}

	return string(yamlData), nil
}

func (b *nodeCollectorBaseReconciler) getDesiredConfigMap(sources *odigosv1.InstrumentationConfigList, clusterCollectorSignals []odigoscommon.ObservabilitySignal, processors []*odigosv1.Processor,
	cg *odigosv1.CollectorsGroup) (*v1.ConfigMap, error) {
	if b.autoscalerDeployment == nil {
		return nil, errors.New("autoscaler deployment is not set in the reconciler, cannot set owner reference")
	}
	var err error
	var cmData string

	if cg == nil || len(clusterCollectorSignals) == 0 {
		// if collectors group is not created yet, or there are no signals to collect, return a no-op configmap
		// this can happen if no sources are instrumented yet or no destinations are added.
		cmData, err = noopConfigMap()
	} else {
		cmData, err = calculateConfigMapData(cg, sources, clusterCollectorSignals, processors, commonconf.ControllerConfig.OnGKE)
	}

	if err != nil {
		return nil, err
	}

	desired := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosNodeCollectorConfigMapName,
			Namespace: env.GetCurrentNamespace(),
		},
		Data: map[string]string{
			k8sconsts.OdigosNodeCollectorConfigMapKey: cmData,
		},
	}

	// set the autoscaler deployment as the owner of the configmap
	// since it is the one creating it and updating it.
	// cg might not yet exist and failing to have an owner will lead to un-cleaned resources on uninstall.
	if err := ctrl.SetControllerReference(b.autoscalerDeployment, &desired, b.scheme); err != nil {
		return nil, err
	}

	return &desired, nil
}

// using the k8sattributes processor to add deployment, statefulset and daemonset metadata
// has a high memory consumption in large clusters since it brings all the replicasets in the cluster into the cache.
// this has been disabled for logs, until further investigation is done on how to reduce memory consumption.
// The side effect is that logs record will lack deployment/statefulset/daemonset names and service name that could have been derived from them.
func updateOrCreateK8sAttributesForLogs(cfg *config.Config) error {

	_, k8sProcessorExists := cfg.Processors[k8sAttributesProcessorName]

	if k8sProcessorExists {
		// make sure it includes the workload names attributes in the processor "extract" section.
		// this is added automatically for logs regardless of any action configuration.
		k8sAttributesCfg, ok := cfg.Processors[k8sAttributesProcessorName].(config.GenericMap)
		if !ok {
			return fmt.Errorf("failed to cast k8s attributes processor config to GenericMap")
		}
		if _, exists := k8sAttributesCfg["extract"]; !exists {
			k8sAttributesCfg["extract"] = config.GenericMap{}
		}
		extract, ok := k8sAttributesCfg["extract"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("failed to cast k8s attributes processor extract config to GenericMap")
		}
		if _, exists := extract["metadata"]; !exists {
			extract["metadata"] = []interface{}{}
		}
		metadata, ok := extract["metadata"].([]interface{})
		if !ok {
			return fmt.Errorf("failed to cast k8s attributes processor metadata config to []interface{}")
		}
		// convert the metadata to a string array to compare it's values
		asStrArray := make([]string, len(metadata))
		for i, v := range metadata {
			asStrArray[i] = v.(string)
		}
		// add the workload names attributes to the metadata list
		if !slices.Contains(asStrArray, string(semconv.K8SDeploymentNameKey)) {
			metadata = append(metadata, string(semconv.K8SDeploymentNameKey))
		}
		if !slices.Contains(asStrArray, string(semconv.K8SStatefulSetNameKey)) {
			metadata = append(metadata, string(semconv.K8SStatefulSetNameKey))
		}
		if !slices.Contains(asStrArray, string(semconv.K8SDaemonSetNameKey)) {
			metadata = append(metadata, string(semconv.K8SDaemonSetNameKey))
		}
		extract["metadata"] = metadata                                // set the copy back to the config
		k8sAttributesCfg["extract"] = extract                         // set the copy back to the config
		cfg.Processors[k8sAttributesProcessorName] = k8sAttributesCfg // set the copy back to the config
	} else {
		// if the processor does not exist, create it with the default configuration
		cfg.Processors[k8sAttributesProcessorName] = config.GenericMap{
			"auth_type": "serviceAccount",
			"filter": config.GenericMap{
				"node_from_env_var": k8sconsts.NodeNameEnvVar,
			},
			"extract": config.GenericMap{
				"metadata": []string{
					string(semconv.K8SDeploymentNameKey),
					string(semconv.K8SStatefulSetNameKey),
					string(semconv.K8SDaemonSetNameKey),
				},
			},
			"pod_association": []config.GenericMap{
				{
					"sources": []config.GenericMap{
						{"from": "resource_attribute", "name": string(semconv.K8SPodNameKey)},
						{"from": "resource_attribute", "name": string(semconv.K8SNamespaceNameKey)},
					},
				},
			},
		}
	}

	return nil
}

func calculateConfigMapData(
	nodeCG *odigosv1.CollectorsGroup,
	sources *odigosv1.InstrumentationConfigList,
	clusterCollectorSignals []odigoscommon.ObservabilitySignal,
	processors []*odigosv1.Processor,
	onGKE bool) (string, error) {

	ownMetricsPort := nodeCG.Spec.CollectorOwnMetricsPort
	odigosNamespace := env.GetCurrentNamespace()

	manifestProcessosrs, tracesProcessors, metricsProcessors, logsProcessors, errs := config.GetCrdProcessorsConfigMap(commonconf.ToProcessorConfigurerArray(processors))
	for name, err := range errs {
		log.Log.V(0).Error(err, "processor", name)
	}

	// common config domains - always set and active
	activeConfigDomains := []config.Config{
		collectorconfig.CommonConfig(nodeCG, onGKE),
		collectorconfig.OwnMetricsConfig(ownMetricsPort),
	}

	// metrics
	metricsEnabled := slices.Contains(clusterCollectorSignals, odigoscommon.MetricsObservabilitySignal)
	metricsConfigSettings := nodeCG.Spec.Metrics
	var additionalTraceExporters []string
	if metricsEnabled && metricsConfigSettings != nil {
		metricsConfig, metricsAdditionalTraceExporters := collectorconfig.MetricsConfig(nodeCG, odigosNamespace, metricsProcessors, metricsConfigSettings)
		activeConfigDomains = append(activeConfigDomains, metricsConfig)
		additionalTraceExporters = append(additionalTraceExporters, metricsAdditionalTraceExporters...)
	}

	// traces
	tracesEnabledInClusterCollector := slices.Contains(clusterCollectorSignals, odigoscommon.TracesObservabilitySignal)
	if len(additionalTraceExporters) > 0 || tracesEnabledInClusterCollector {
		tracesConfig := collectorconfig.TracesConfig(nodeCG, odigosNamespace, tracesProcessors, additionalTraceExporters, tracesEnabledInClusterCollector)
		activeConfigDomains = append(activeConfigDomains, tracesConfig)
	}

	mergedConfig, err := config.MergeConfigs(activeConfigDomains...)
	if err != nil {
		return "", err
	}

	allProcessors := manifestProcessosrs
	for name, processor := range mergedConfig.Processors {
		allProcessors[name] = processor
	}

	cfg := config.Config{
		Receivers:  mergedConfig.Receivers,
		Processors: allProcessors,
		Connectors: mergedConfig.Connectors,
		Exporters:  mergedConfig.Exporters,
		Extensions: mergedConfig.Extensions,
		Service:    mergedConfig.Service,
	}

	collectLogs := slices.Contains(clusterCollectorSignals, odigoscommon.LogsObservabilitySignal)
	if collectLogs {
		includes := make([]string, 0)
		for _, element := range sources.Items {
			// Paths for log files: /var/log/pods/<namespace>_<pod name>_<pod ID>/<container name>/<auto-incremented file number>.log
			// Pod specifiers
			// 	Deployment:  <namespace>_<deployment  name>-<replicaset suffix[~10]>-<pod suffix[~5]>_<pod ID>
			// 	DeamonSet:   <namespace>_<daemonset   name>-<            pod suffix[~5]            >_<pod ID>
			// 	StatefulSet: <namespace>_<statefulset name>-<        ordinal index integer        >_<pod ID>
			// The suffixes are not the same lenght always, so we cannot match the pattern reliably.
			// We expect there to exactly one OwnerReference
			if len(element.OwnerReferences) != 1 {
				log.Log.V(0).Error(
					fmt.Errorf("Unexpected number of OwnerReferences: %d", len(element.OwnerReferences)),
					"failed to compile include list for configmap",
				)
				continue
			}
			owner := element.OwnerReferences[0]
			name := owner.Name
			includes = append(includes, fmt.Sprintf("/var/log/pods/%s_%s-*_*/*/*.log", element.Namespace, name))
		}

		cfg.Receivers["filelog"] = config.GenericMap{
			"include":           includes,
			"exclude":           []string{"/var/log/pods/kube-system_*/**/*", "/var/log/pods/" + odigosNamespace + "_*/**/*"},
			"start_at":          "end",
			"include_file_path": true,
			"include_file_name": false,
			"operators": []config.GenericMap{
				{
					"id":   "container-parser",
					"type": "container",
				},
			},
			"retry_on_failure": config.GenericMap{
				// From documentation:
				// When true, the receiver will pause reading a file and attempt to resend the current batch of logs
				//  if it encounters an error from downstream components.
				//
				// filelog might get overwhelmed when it just starts and there are already a lot of logs to process in the node.
				// when downstream components (cluster collector and receiving destination) are under too much pressure,
				// they will reject the data, slowing down the filelog receiver and allowing it to retry the data and adjust to
				// downstream pressure.
				"enabled": true,
			},
		}

		// err := updateOrCreateK8sAttributesForLogs(&cfg)
		// if err != nil {
		// 	return "", err
		// }
		// // remove logs processors from CRD logsProcessors in case it is there so not to add it twice
		// for i, processor := range logsProcessors {
		// 	if processor == k8sAttributesProcessorName {
		// 		logsProcessors = append(logsProcessors[:i], logsProcessors[i+1:]...)
		// 		break
		// 	}
		// }

		// set "service.name" for logs same as the workload name.
		// note: this does not respect the override service name a user can set in sources.
		// cfg.Processors[logsServiceNameProcessorName] = config.GenericMap{
		// 	"attributes": []config.GenericMap{
		// 		{
		// 			"key":            string(semconv.ServiceNameKey),
		// 			"from_attribute": string(semconv.K8SDeploymentNameKey),
		// 			"action":         "insert", // avoid overwriting existing value
		// 		},
		// 		{
		// 			"key":            string(semconv.ServiceNameKey),
		// 			"from_attribute": string(semconv.K8SStatefulSetNameKey),
		// 			"action":         "insert", // avoid overwriting existing value
		// 		},
		// 		{
		// 			"key":            string(semconv.ServiceNameKey),
		// 			"from_attribute": string(semconv.K8SDaemonSetNameKey),
		// 			"action":         "insert", // avoid overwriting existing value
		// 		},
		// 		{
		// 			"key":            string(semconv.ServiceNameKey),
		// 			"from_attribute": string(semconv.K8SCronJobNameKey),
		// 			"action":         "insert", // avoid overwriting existing value
		// 		},
		// 	},
		// }

		cfg.Service.Pipelines["logs"] = config.Pipeline{
			Receivers:  []string{"filelog"},
			Processors: append(getFileLogPipelineProcessors(), logsProcessors...),
			Exporters:  []string{collectorconfig.ClusterCollectorExporterName},
		}
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func getConfigMap(ctx context.Context, c client.Client, namespace string) (*v1.ConfigMap, error) {
	configMap := &v1.ConfigMap{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: k8sconsts.OdigosNodeCollectorConfigMapName}, configMap); err != nil {
		return nil, err
	}

	return configMap, nil
}

func getSignalsFromOtelcolConfig(otelcolConfigContent string) ([]odigoscommon.ObservabilitySignal, error) {
	config := config.Config{}
	err := yaml.Unmarshal([]byte(otelcolConfigContent), &config)
	if err != nil {
		return nil, err
	}

	tracesEnabled := false
	metricsEnabled := false
	logsEnabled := false
	for pipelineName, pipeline := range config.Service.Pipelines {
		// only consider pipelines with `otlp` receiver
		// which are the ones that can actually receive data
		if !slices.Contains(pipeline.Receivers, collectorconfig.OTLPInReceiverName) {
			continue
		}
		if strings.HasPrefix(pipelineName, "traces") {
			tracesEnabled = true
		} else if strings.HasPrefix(pipelineName, "metrics") {
			metricsEnabled = true
		} else if strings.HasPrefix(pipelineName, "logs") {
			logsEnabled = true
		}
	}

	signals := []odigoscommon.ObservabilitySignal{}
	if tracesEnabled {
		signals = append(signals, odigoscommon.TracesObservabilitySignal)
	}
	if metricsEnabled {
		signals = append(signals, odigoscommon.MetricsObservabilitySignal)
	}
	if logsEnabled {
		signals = append(signals, odigoscommon.LogsObservabilitySignal)
	}

	return signals, nil
}

func getAgentPipelineCommonProcessors() []string {
	// for agents, we never want to upstream backpressure, as we don't want memory to build up
	// in the sending application.
	// this is why batch processor is first to always accept data and then memory limiter
	// is used to drop the data if the collector is overloaded in memory.
	// Read more about it here: https://github.com/open-telemetry/opentelemetry-collector/issues/11726
	// Also related: https://github.com/open-telemetry/opentelemetry-collector/issues/9591
	return append(
		[]string{collectorconfig.BatchProcessorName, collectorconfig.MemoryLimiterProcessorName},
		getCommonProcessors()...,
	)
}

func getFileLogPipelineProcessors() []string {
	// filelog pipeline processors
	// memory_limiter is first processor, it will reject data if the collectors memory is full.
	// no need to batch, as the stanza receiver already batches the data, and so is the gateway.
	// batch processor will also mask and hide any back-pressure from receivers which we want propagated to the source.
	return append(
		[]string{collectorconfig.MemoryLimiterProcessorName /*k8sAttributesProcessorName, logsServiceNameProcessorName*/},
		getCommonProcessors()...,
	)
}

func getCommonProcessors() []string {
	return []string{collectorconfig.NodeNameProcessorName, collectorconfig.ResourceDetectionProcessorName, collectorconfig.OdigosTrafficMetricsProcessorName}
}
