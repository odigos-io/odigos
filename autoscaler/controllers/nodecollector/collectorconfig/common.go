package collectorconfig

import (
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

const (
	OTLPInReceiverName = "otlp/in"
)

const (
	healthCheckExtensionName            = "health_check"
	odigosEbpfReceiverName              = "odigosebpf"
	profilingReceiverName              = "profiling" // eBPF profiling receiver (go.opentelemetry.io/ebpf-profiler)
	pprofExtensionName                  = "pprof"
	batchProcessorName                  = "batch"
	memoryLimiterProcessorName          = "memory_limiter"
	balancerName                        = "round_robin"
	nodeNameProcessorName               = "resource/node-name"
	clusterCollectorTracesExporterName  = "otlp/out-cluster-collector-traces"
	clusterCollectorMetricsExporterName = "otlp/out-cluster-collector-metrics"
	clusterCollectorLogsExporterName     = "otlp/out-cluster-collector-logs"
	clusterCollectorProfilesExporterName = "otlp/out-cluster-collector-profiles"
	resourceDetectionProcessorName      = "resourcedetection"
	k8sattributesProcessorName          = "k8sattributes"
)

func commonProcessors(nodeCG *odigosv1.CollectorsGroup, runningOnGKE bool) config.GenericMap {

	allProcessors := config.GenericMap{}
	for k, v := range staticProcessors {
		allProcessors[k] = v
	}

	memoryLimiterConfig := commonconf.GetMemoryLimiterConfig(nodeCG.Spec.ResourcesSettings)
	allProcessors[memoryLimiterProcessorName] = memoryLimiterConfig

	var detectors []string
	// This is a workaround to avoid adding the gcp detector if not running on a gke environment
	// once https://github.com/GoogleCloudPlatform/opentelemetry-operations-go/issues/1026 is resolved, we can always put the gcp detector
	if runningOnGKE {
		detectors = []string{"gcp"}
	} else {
		detectors = []string{"ec2", "eks", "azure", "aks"}
	}
	allProcessors[resourceDetectionProcessorName] = config.GenericMap{
		"detectors": detectors,
		"timeout":   "2s",
	}

	return allProcessors
}

var staticProcessors config.GenericMap
var commonReceivers config.GenericMap
var commonExtensions config.GenericMap
var commonService config.Service

func getCommonExporters(otlpExporterConfiguration *common.OtlpExporterConfiguration, odigosNamespace string) config.GenericMap {

	compression := "none"
	if otlpExporterConfiguration != nil && otlpExporterConfiguration.EnableDataCompression != nil && *otlpExporterConfiguration.EnableDataCompression {
		compression = "gzip"
	}

	// Build the common exporter configuration (used by metrics, logs, and profiles)
	commonExporterConfig := buildBaseExporterConfig(odigosNamespace, compression)

	// Build the trace exporter configuration with the same base config
	traceExporterConfig := buildBaseExporterConfig(odigosNamespace, compression)

	if otlpExporterConfiguration != nil && otlpExporterConfiguration.Timeout != "" {
		traceExporterConfig["timeout"] = otlpExporterConfiguration.Timeout
	}

	// Retry config: from CRD when present, otherwise default so metrics/logs/profiles retry on gateway errors instead of failing.
	retryConfig := config.GenericMap{"enabled": true, "initial_interval": "5s", "max_interval": "30s", "max_elapsed_time": "5m"}
	if otlpExporterConfiguration != nil && otlpExporterConfiguration.RetryOnFailure != nil {
		if otlpExporterConfiguration.RetryOnFailure.Enabled != nil {
			retryConfig["enabled"] = *otlpExporterConfiguration.RetryOnFailure.Enabled
		}
		if otlpExporterConfiguration.RetryOnFailure.InitialInterval != "" {
			retryConfig["initial_interval"] = otlpExporterConfiguration.RetryOnFailure.InitialInterval
		}
		if otlpExporterConfiguration.RetryOnFailure.MaxInterval != "" {
			retryConfig["max_interval"] = otlpExporterConfiguration.RetryOnFailure.MaxInterval
		}
		if otlpExporterConfiguration.RetryOnFailure.MaxElapsedTime != "" {
			retryConfig["max_elapsed_time"] = otlpExporterConfiguration.RetryOnFailure.MaxElapsedTime
		}
	}

	traceExporterConfig["retry_on_failure"] = retryConfig
	commonExporterConfig["retry_on_failure"] = retryConfig

	return config.GenericMap{
		clusterCollectorTracesExporterName:   traceExporterConfig,
		clusterCollectorMetricsExporterName: commonExporterConfig,
		clusterCollectorLogsExporterName:     commonExporterConfig,
		clusterCollectorProfilesExporterName: commonExporterConfig,
	}
}

func init() {

	staticProcessors = config.GenericMap{
		batchProcessorName:                   config.GenericMap{},
		odigosLogsResourceAttrsProcessorName: config.GenericMap{},
		nodeNameProcessorName: config.GenericMap{
			"attributes": []config.GenericMap{{
				"key":    string(semconv.K8SNodeNameKey),
				"value":  "${NODE_NAME}",
				"action": "upsert",
			}},
		},
	}

	commonReceivers = config.GenericMap{
		OTLPInReceiverName: config.GenericMap{
			"protocols": config.GenericMap{
				"grpc": config.GenericMap{
					"endpoint": "0.0.0.0:4317",
					// data collection collectors will drop data instead of backpressuring the senders (odiglet or agents),
					// we don't want the applications to build up memory in the runtime if the pipeline is overloaded.
					"drop_on_overload": true,
				},
				"http": config.GenericMap{
					"endpoint": "0.0.0.0:4318",
				},
			},
		},
		odigosEbpfReceiverName: config.GenericMap{},
		profilingReceiverName:  config.GenericMap{}, // in-node eBPF CPU profiling; requires Linux, privileged
	}

	commonExtensions = config.GenericMap{
		healthCheckExtensionName: config.GenericMap{
			"endpoint": "0.0.0.0:13133",
		},
		pprofExtensionName: config.GenericMap{
			"endpoint": "0.0.0.0:1777",
		},
	}

	commonService = config.Service{
		Extensions: []string{healthCheckExtensionName, pprofExtensionName},
	}
}

func CommonApplicationTelemetryConfig(nodeCG *odigosv1.CollectorsGroup, onGKE bool, odigosNamespace string) config.Config {
	return config.Config{
		Receivers:  commonReceivers,
		Exporters:  getCommonExporters(nodeCG.Spec.OtlpExporterConfiguration, odigosNamespace),
		Processors: commonProcessors(nodeCG, onGKE),
	}
}

func CommonConfig() config.Config {
	return config.Config{
		Extensions: commonExtensions,
		Service:    commonService,
	}
}

// buildBaseExporterConfig creates a new base exporter configuration
func buildBaseExporterConfig(odigosNamespace string, compression string) config.GenericMap {
	return config.GenericMap{
		"endpoint": fmt.Sprintf("dns:///%s.%s:4317", k8sconsts.OdigosClusterCollectorServiceName, odigosNamespace),
		"tls": config.GenericMap{
			"insecure": true,
		},
		"compression":   compression,
		"balancer_name": balancerName,
	}
}
