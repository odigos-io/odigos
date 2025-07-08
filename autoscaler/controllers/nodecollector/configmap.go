package nodecollector

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	odigoscommon "github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	k8sAttributesProcessorName   = "k8sattributes/odigos-k8sattributes"
	logsServiceNameProcessorName = "resource/service-name"
)

func SyncConfigMap(sources *odigosv1.InstrumentationConfigList, signals []odigoscommon.ObservabilitySignal, allProcessors *odigosv1.ProcessorList,
	datacollection *odigosv1.CollectorsGroup, ctx context.Context,
	c client.Client, scheme *runtime.Scheme) error {
	logger := log.FromContext(ctx)

	processors := commonconf.FilterAndSortProcessorsByOrderHint(allProcessors, odigosv1.CollectorsGroupRoleNodeCollector)

	desired, err := getDesiredConfigMap(sources, signals, processors, datacollection, scheme)
	if err != nil {
		logger.Error(err, "failed to get desired config map")
		return err
	}

	existing := &v1.ConfigMap{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: datacollection.Namespace, Name: datacollection.Name}, existing); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(0).Info("creating config map")
			_, err := createConfigMap(desired, ctx, c)
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
	_, err = patchConfigMap(ctx, existing, desired, c)
	if err != nil {
		logger.Error(err, "failed to patch config map")
		return err
	}

	return nil
}

func patchConfigMap(ctx context.Context, existing *v1.ConfigMap, desired *v1.ConfigMap, c client.Client) (*v1.ConfigMap, error) {
	if reflect.DeepEqual(existing.Data, desired.Data) &&
		reflect.DeepEqual(existing.ObjectMeta.OwnerReferences, desired.ObjectMeta.OwnerReferences) {
		log.FromContext(ctx).V(0).Info("Config maps already match")
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

func createConfigMap(desired *v1.ConfigMap, ctx context.Context, c client.Client) (*v1.ConfigMap, error) {
	if err := c.Create(ctx, desired); err != nil {
		return nil, err
	}

	return desired, nil
}

func getDesiredConfigMap(sources *odigosv1.InstrumentationConfigList, signals []odigoscommon.ObservabilitySignal, processors []*odigosv1.Processor,
	datacollection *odigosv1.CollectorsGroup, scheme *runtime.Scheme) (*v1.ConfigMap, error) {
	cmData, err := calculateConfigMapData(datacollection, sources, signals, processors)
	if err != nil {
		return nil, err
	}

	desired := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      datacollection.Name,
			Namespace: datacollection.Namespace,
		},
		Data: map[string]string{
			k8sconsts.OdigosNodeCollectorConfigMapKey: cmData,
		},
	}

	if err := ctrl.SetControllerReference(datacollection, &desired, scheme); err != nil {
		return nil, err
	}

	return &desired, nil
}

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

func calculateConfigMapData(nodeCG *odigosv1.CollectorsGroup, sources *odigosv1.InstrumentationConfigList, signals []odigoscommon.ObservabilitySignal,
	processors []*odigosv1.Processor) (string, error) {

	ownMetricsPort := nodeCG.Spec.CollectorOwnMetricsPort

	empty := struct{}{}

	processorsCfg, tracesProcessors, metricsProcessors, logsProcessors, errs := config.GetCrdProcessorsConfigMap(commonconf.ToProcessorConfigurerArray(processors))
	for name, err := range errs {
		log.Log.V(0).Error(err, "processor", name)
	}

	memoryLimiterConfiguration := commonconf.GetMemoryLimiterConfig(nodeCG.Spec.ResourcesSettings)

	processorsCfg["batch"] = empty
	processorsCfg["memory_limiter"] = memoryLimiterConfiguration
	processorsCfg["resource"] = config.GenericMap{
		"attributes": []config.GenericMap{{
			"key":    "k8s.node.name",
			"value":  "${NODE_NAME}",
			"action": "upsert",
		}},
	}
	resourceDetectionProcessor := config.GenericMap{
		"detectors": []string{"ec2", "azure"},
		"timeout":   "2s",
	}
	// This is a workaround to avoid adding the gcp detector if not running on a gke environment
	// once https://github.com/GoogleCloudPlatform/opentelemetry-operations-go/issues/1026 is resolved, we can always put the gcp detector
	if commonconf.ControllerConfig.OnGKE {
		resourceDetectionProcessor["detectors"] = append(resourceDetectionProcessor["detectors"].([]string), "gcp")
	}
	processorsCfg["resourcedetection"] = resourceDetectionProcessor

	processorsCfg["odigostrafficmetrics"] = config.GenericMap{
		// adding the following resource attributes to the metrics allows to aggregate the metrics by source.
		"res_attributes_keys": []string{
			string(semconv.ServiceNameKey),
			string(semconv.K8SNamespaceNameKey),
			string(semconv.K8SDeploymentNameKey),
			string(semconv.K8SStatefulSetNameKey),
			string(semconv.K8SDaemonSetNameKey),
		},
	}
	processorsCfg["resource/pod-name"] = config.GenericMap{
		"attributes": []config.GenericMap{{
			"key":    "k8s.pod.name",
			"value":  "${POD_NAME}",
			"action": "upsert",
		}},
	}

	exporters := config.GenericMap{
		"otlp/gateway": config.GenericMap{
			"endpoint": fmt.Sprintf("dns:///odigos-gateway.%s:4317", env.GetCurrentNamespace()),
			"tls": config.GenericMap{
				"insecure": true,
			},
			"balancer_name": "round_robin",
		},
		"otlp/odigos-own-telemetry-ui": config.GenericMap{
			"endpoint": fmt.Sprintf("ui.%s:%d", env.GetCurrentNamespace(), consts.OTLPPort),
			"tls": config.GenericMap{
				"insecure": true,
			},
			"retry_on_failure": config.GenericMap{
				"enabled": false,
			},
		},
	}
	tracesPipelineExporter := []string{"otlp/gateway"}

	// Add loadbalancing exporter for traces to ensure consistent gateway routing.
	// This allows the servicegraph connector to properly aggregate trace data
	// by sending all traces from a node collector to the same gateway instance.
	tracesEnabled := slices.Contains(signals, odigoscommon.TracesObservabilitySignal)
	if tracesEnabled {
		exporters["loadbalancing"] = config.GenericMap{
			"protocol": config.GenericMap{"otlp": config.GenericMap{"tls": config.GenericMap{"insecure": true}}},
			"resolver": config.GenericMap{"k8s": config.GenericMap{"service": fmt.Sprintf("odigos-gateway.%s", env.GetCurrentNamespace())}},
		}
		tracesPipelineExporter = []string{"loadbalancing"}
	}

	cfg := config.Config{
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
			"prometheus/self-metrics": config.GenericMap{
				"config": config.GenericMap{
					"scrape_configs": []config.GenericMap{
						{
							"job_name":        "otelcol",
							"scrape_interval": "10s",
							"static_configs": []config.GenericMap{
								{
									"targets": []string{fmt.Sprintf("127.0.0.1:%d", ownMetricsPort)},
								},
							},
							"metric_relabel_configs": []config.GenericMap{
								{
									"source_labels": []string{"__name__"},
									"regex":         "(.*odigos.*)",
									"action":        "keep",
								},
							},
						},
					},
				},
			},
		},
		Exporters:  exporters,
		Processors: processorsCfg,
		Extensions: config.GenericMap{
			"health_check": config.GenericMap{
				"endpoint": "0.0.0.0:13133",
			},
			"pprof": config.GenericMap{
				"endpoint": "0.0.0.0:1777",
			},
		},
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				"metrics/otelcol": {
					Receivers:  []string{"prometheus/self-metrics"},
					Processors: []string{"resource/pod-name"},
					Exporters:  []string{"otlp/odigos-own-telemetry-ui"},
				},
			},
			Extensions: []string{"health_check", "pprof"},
			Telemetry: config.Telemetry{
				Metrics: config.GenericMap{
					"readers": []config.GenericMap{
						{
							"pull": config.GenericMap{
								"exporter": config.GenericMap{
									"prometheus": config.GenericMap{
										"host": "0.0.0.0",
										"port": ownMetricsPort,
									},
								},
							},
						},
					},
				},
				Resource: map[string]*string{
					// The collector add "otelcol" as a service name, so we need to remove it
					// to avoid duplication, since we are interested in the instrumented services.
					string(semconv.ServiceNameKey): nil,
					// The collector adds its own version as a service version, which is not needed currently.
					string(semconv.ServiceVersionKey): nil,
				},
			},
		},
	}

	collectLogs := slices.Contains(signals, odigoscommon.LogsObservabilitySignal)
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

		odigosSystemNamespaceName := env.GetCurrentNamespace()
		cfg.Receivers["filelog"] = config.GenericMap{
			"include":           includes,
			"exclude":           []string{"/var/log/pods/kube-system_*/**/*", "/var/log/pods/" + odigosSystemNamespaceName + "_*/**/*"},
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

		err := updateOrCreateK8sAttributesForLogs(&cfg)
		if err != nil {
			return "", err
		}
		// remove logs processors from CRD logsProcessors in case it is there so not to add it twice
		for i, processor := range logsProcessors {
			if processor == k8sAttributesProcessorName {
				logsProcessors = append(logsProcessors[:i], logsProcessors[i+1:]...)
				break
			}
		}

		// set "service.name" for logs same as the workload name.
		// note: this does not respect the override service name a user can set in sources.
		cfg.Processors[logsServiceNameProcessorName] = config.GenericMap{
			"attributes": []config.GenericMap{
				{
					"key":            string(semconv.ServiceNameKey),
					"from_attribute": string(semconv.K8SDeploymentNameKey),
					"action":         "insert", // avoid overwriting existing value
				},
				{
					"key":            string(semconv.ServiceNameKey),
					"from_attribute": string(semconv.K8SStatefulSetNameKey),
					"action":         "insert", // avoid overwriting existing value
				},
				{
					"key":            string(semconv.ServiceNameKey),
					"from_attribute": string(semconv.K8SDaemonSetNameKey),
					"action":         "insert", // avoid overwriting existing value
				},
			},
		}

		cfg.Service.Pipelines["logs"] = config.Pipeline{
			Receivers:  []string{"filelog"},
			Processors: append(getFileLogPipelineProcessors(), logsProcessors...),
			Exporters:  []string{"otlp/gateway"},
		}
	}

	collectTraces := slices.Contains(signals, odigoscommon.TracesObservabilitySignal)
	if collectTraces {
		cfg.Service.Pipelines["traces"] = config.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: append(getAgentPipelineCommonProcessors(), tracesProcessors...),
			Exporters:  tracesPipelineExporter,
		}
	}

	collectMetrics := slices.Contains(signals, odigoscommon.MetricsObservabilitySignal)
	if collectMetrics {
		cfg.Receivers["kubeletstats"] = config.GenericMap{
			"auth_type":            "serviceAccount",
			"endpoint":             "https://${env:NODE_NAME}:10250",
			"insecure_skip_verify": true,
			"collection_interval":  "10s",
		}

		cfg.Receivers["hostmetrics"] = config.GenericMap{
			"collection_interval": "10s",
			"root_path":           "/hostfs",
			"scrapers": config.GenericMap{
				"paging": config.GenericMap{
					"metrics": config.GenericMap{
						"system.paging.utilization": config.GenericMap{
							"enabled": true,
						},
					},
				},
				"cpu": config.GenericMap{
					"metrics": config.GenericMap{
						"system.cpu.utilization": config.GenericMap{
							"enabled": true,
						},
					},
				},
				"disk": struct{}{},
				"filesystem": config.GenericMap{
					"metrics": config.GenericMap{
						"system.filesystem.utilization": config.GenericMap{
							"enabled": true,
						},
					},
					"exclude_mount_points": config.GenericMap{
						"match_type":   "regexp",
						"mount_points": []string{"/var/lib/kubelet/*"},
					},
				},
				"load":      struct{}{},
				"memory":    struct{}{},
				"network":   struct{}{},
				"processes": struct{}{},
			},
		}

		cfg.Service.Pipelines["metrics"] = config.Pipeline{
			Receivers:  []string{"otlp", "kubeletstats", "hostmetrics"},
			Processors: append(getAgentPipelineCommonProcessors(), metricsProcessors...),
			Exporters:  []string{"otlp/gateway"},
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
		if !slices.Contains(pipeline.Receivers, "otlp") {
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
		[]string{"batch", "memory_limiter"},
		getCommonProcessors()...,
	)
}

func getFileLogPipelineProcessors() []string {
	// filelog pipeline processors
	// memory_limiter is first processor, it will reject data if the collectors memory is full.
	// no need to batch, as the stanza receiver already batches the data, and so is the gateway.
	// batch processor will also mask and hide any back-pressure from receivers which we want propagated to the source.
	return append(
		[]string{"memory_limiter", k8sAttributesProcessorName, logsServiceNameProcessorName},
		getCommonProcessors()...,
	)
}

func getCommonProcessors() []string {
	return []string{"resource", "resourcedetection", "odigostrafficmetrics"}
}
