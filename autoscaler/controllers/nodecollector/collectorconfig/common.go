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
	pprofExtensionName                  = "pprof"
	batchProcessorName                  = "batch"
	memoryLimiterProcessorName          = "memory_limiter"
	balancerName                        = "round_robin"
	nodeNameProcessorName               = "resource/node-name"
	clusterCollectorTracesExporterName  = "otlp_grpc/out-cluster-collector-traces"
	clusterCollectorMetricsExporterName = "otlp_grpc/out-cluster-collector-metrics"
	clusterCollectorLogsExporterName    = "otlp_grpc/out-cluster-collector-logs"
	resourceDetectionProcessorName      = "resourcedetection"
)

func isDetectorEnabled(cfg *common.ResourceDetectorConfig) bool {
	return cfg != nil && cfg.Enabled != nil && *cfg.Enabled
}

func buildResourceDetectors(cfg *common.ResourceDetectorsConfiguration, runningOnGKE bool) []string {
	if cfg == nil {
		return nil
	}

	var detectors []string

	// GCP detector is gated behind the GKE flag due to
	// https://github.com/GoogleCloudPlatform/opentelemetry-operations-go/issues/1026
	// When running on GKE the only safe detector is "gcp" unless the user explicitly
	// enabled others. When NOT on GKE, the non-GCP detectors are considered.
	if runningOnGKE {
		if isDetectorEnabled(cfg.GCP) {
			detectors = append(detectors, "gcp")
		}
	} else {
		if isDetectorEnabled(cfg.EC2) {
			detectors = append(detectors, "ec2")
		}
		if isDetectorEnabled(cfg.EKS) {
			detectors = append(detectors, "eks")
		}
		if isDetectorEnabled(cfg.Azure) {
			detectors = append(detectors, "azure")
		}
		if isDetectorEnabled(cfg.AKS) {
			detectors = append(detectors, "aks")
		}
	}

	return detectors
}

func commonProcessors(nodeCG *odigosv1.CollectorsGroup, runningOnGKE bool) config.GenericMap {

	allProcessors := config.GenericMap{}
	for k, v := range staticProcessors {
		allProcessors[k] = v
	}

	memoryLimiterConfig := commonconf.GetMemoryLimiterConfig(nodeCG.Spec.ResourcesSettings)
	allProcessors[memoryLimiterProcessorName] = memoryLimiterConfig

	detectors := buildResourceDetectors(nodeCG.Spec.ResourceDetectors, runningOnGKE)
	if len(detectors) > 0 {
		allProcessors[resourceDetectionProcessorName] = config.GenericMap{
			"detectors": detectors,
			"timeout":   "2s",
		}
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

	// Build the common exporter configuration (used by metrics and logs)
	commonExporterConfig := buildBaseExporterConfig(odigosNamespace, compression)

	// Build the trace exporter configuration with the same base config
	traceExporterConfig := buildBaseExporterConfig(odigosNamespace, compression)

	if otlpExporterConfiguration != nil && otlpExporterConfiguration.Timeout != "" {
		traceExporterConfig["timeout"] = otlpExporterConfiguration.Timeout
	}

	// Add retry_on_failure configuration if present
	if otlpExporterConfiguration != nil && otlpExporterConfiguration.RetryOnFailure != nil {

		retryConfig := config.GenericMap{}
		// Only set enabled if not nil to avoid possible nil pointer dereference
		if otlpExporterConfiguration.RetryOnFailure.Enabled != nil {
			retryConfig["enabled"] = *otlpExporterConfiguration.RetryOnFailure.Enabled
		} else {
			// by default, retry on failure is enabled
			retryConfig["enabled"] = true
		}

		// Only add the interval fields if they are not empty
		if otlpExporterConfiguration.RetryOnFailure.InitialInterval != "" {
			retryConfig["initial_interval"] = otlpExporterConfiguration.RetryOnFailure.InitialInterval
		}
		if otlpExporterConfiguration.RetryOnFailure.MaxInterval != "" {
			retryConfig["max_interval"] = otlpExporterConfiguration.RetryOnFailure.MaxInterval
		}
		if otlpExporterConfiguration.RetryOnFailure.MaxElapsedTime != "" {
			retryConfig["max_elapsed_time"] = otlpExporterConfiguration.RetryOnFailure.MaxElapsedTime
		}

		traceExporterConfig["retry_on_failure"] = retryConfig
	}

	return config.GenericMap{
		clusterCollectorTracesExporterName:  traceExporterConfig,
		clusterCollectorMetricsExporterName: commonExporterConfig,
		clusterCollectorLogsExporterName:    commonExporterConfig,
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
