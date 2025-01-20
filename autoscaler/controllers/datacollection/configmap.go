package datacollection

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/ghodss/yaml"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/autoscaler/controllers/common"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/autoscaler/controllers/datacollection/custom"
	odigoscommon "github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/consts"
	constsK8s "github.com/odigos-io/odigos/k8sutils/pkg/consts"
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

func SyncConfigMap(sources *odigosv1.InstrumentationConfigList, dests *odigosv1.DestinationList, allProcessors *odigosv1.ProcessorList,
	datacollection *odigosv1.CollectorsGroup, ctx context.Context,
	c client.Client, scheme *runtime.Scheme, disableNameProcessor bool) error {
	logger := log.FromContext(ctx)

	processors := commonconf.FilterAndSortProcessorsByOrderHint(allProcessors, odigosv1.CollectorsGroupRoleNodeCollector)

	// If sampling configured, load balancing exporter should be added to the data collection config
	SamplingExists := commonconf.FindFirstProcessorByType(allProcessors, "odigossampling")
	setTracesLoadBalancer := SamplingExists != nil

	desired, err := getDesiredConfigMap(sources, dests, processors, datacollection, scheme, setTracesLoadBalancer, disableNameProcessor)
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

func getDesiredConfigMap(sources *odigosv1.InstrumentationConfigList, dests *odigosv1.DestinationList, processors []*odigosv1.Processor,
	datacollection *odigosv1.CollectorsGroup, scheme *runtime.Scheme, setTracesLoadBalancer bool, disableNameProcessor bool) (*v1.ConfigMap, error) {
	cmData, err := calculateConfigMapData(datacollection, sources, dests, processors, setTracesLoadBalancer, disableNameProcessor)
	if err != nil {
		return nil, err
	}

	desired := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      datacollection.Name,
			Namespace: datacollection.Namespace,
		},
		Data: map[string]string{
			constsK8s.OdigosNodeCollectorConfigMapKey: cmData,
		},
	}

	if custom.ShouldApplyCustomDataCollection(dests) {
		custom.AddCustomConfigMap(dests, &desired)
	}

	if err := ctrl.SetControllerReference(datacollection, &desired, scheme); err != nil {
		return nil, err
	}

	return &desired, nil
}

func calculateConfigMapData(nodeCG *odigosv1.CollectorsGroup, sources *odigosv1.InstrumentationConfigList, dests *odigosv1.DestinationList, processors []*odigosv1.Processor,
	setTracesLoadBalancer bool, disableNameProcessor bool) (string, error) {

	ownMetricsPort := nodeCG.Spec.CollectorOwnMetricsPort

	empty := struct{}{}

	processorsCfg, tracesProcessors, metricsProcessors, logsProcessors, errs := config.GetCrdProcessorsConfigMap(commonconf.ToProcessorConfigurerArray(processors))
	for name, err := range errs {
		log.Log.V(0).Error(err, "processor", name)
	}

	if !disableNameProcessor {
		processorsCfg["odigosresourcename"] = empty
	}

	memoryLimiterConfiguration := common.GetMemoryLimiterConfig(nodeCG.Spec.ResourcesSettings)

	processorsCfg["batch"] = empty
	processorsCfg["memory_limiter"] = memoryLimiterConfiguration
	processorsCfg["resource"] = config.GenericMap{
		"attributes": []config.GenericMap{{
			"key":    "k8s.node.name",
			"value":  "${NODE_NAME}",
			"action": "upsert",
		}},
	}
	processorsCfg["resourcedetection"] = config.GenericMap{"detectors": []string{"ec2", "gcp", "azure"}}
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

	if setTracesLoadBalancer {
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
									"regex":         "(.*odigos.*|^otelcol_processor_accepted.*)",
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
					"address": fmt.Sprintf("0.0.0.0:%d", ownMetricsPort),
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

	collectTraces := false
	collectMetrics := false
	collectLogs := false
	for _, dst := range dests.Items {
		for _, s := range dst.Spec.Signals {
			if s == odigoscommon.LogsObservabilitySignal && !custom.DestRequiresCustom(dst.Spec.Type) {
				collectLogs = true
			}
			if s == odigoscommon.TracesObservabilitySignal || dst.Spec.Type == odigoscommon.PrometheusDestinationType {
				collectTraces = true
			}
			if s == odigoscommon.MetricsObservabilitySignal && !custom.DestRequiresCustom(dst.Spec.Type) {
				collectMetrics = true
			}
		}
	}

	commonProcessors := getCommonProcessors(disableNameProcessor)

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
			"start_at":          "beginning",
			"include_file_path": true,
			"include_file_name": false,
			"operators": []config.GenericMap{
				{
					"id":   "container-parser",
					"type": "container",
				},
			},
		}

		cfg.Service.Pipelines["logs"] = config.Pipeline{
			Receivers:  []string{"filelog"},
			Processors: append(commonProcessors, logsProcessors...),
			Exporters:  []string{"otlp/gateway"},
		}
	}

	if collectTraces {
		cfg.Service.Pipelines["traces"] = config.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: append(commonProcessors, tracesProcessors...),
			Exporters:  tracesPipelineExporter,
		}
	}

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
			Processors: append(commonProcessors, metricsProcessors...),
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
	if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: constsK8s.OdigosNodeCollectorConfigMapName}, configMap); err != nil {
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

func getCommonProcessors(disableNameProcessor bool) []string {
	// memory limiter is placed right after batch processor an not the first processor in pipeline
	// this is so that instrumented application always succeeds in sending data to the collector
	// (on it being added to a batch) and checking the memory limit later after the batch
	// where memory rejection would drop the data instead of backpressuring the application.
	// Read more about it here: https://github.com/open-telemetry/opentelemetry-collector/issues/11726
	// Also related: https://github.com/open-telemetry/opentelemetry-collector/issues/9591
	processors := []string{"batch", "memory_limiter"}
	if !disableNameProcessor {
		processors = append(processors, "odigosresourcename")
	}
	processors = append(processors, "resource", "resourcedetection", "odigostrafficmetrics")
	return processors
}
