package collectorconfig

import (
	"fmt"

	"github.com/go-logr/logr"
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

// CommonSignalConfig holds configuration fields shared across all signal pipelines (traces, metrics, logs).
type CommonSignalConfig struct {
	Logger                   logr.Logger
	OdigosNamespace          string
	ManifestProcessorNames   []string
	ResourceDetectionEnabled bool
	Tier                     common.OdigosTier
}

// WithProcessors returns a copy of the config with the given manifest processor names set.
func (c CommonSignalConfig) WithProcessors(names []string) CommonSignalConfig {
	c.ManifestProcessorNames = names
	return c
}

const (
	healthCheckExtensionName = "health_check"
	// odigosebpf and odigos_enterprise_auth are only compiled into the enterprise collector image.
	// The collector rejects unknown component types when it loads the config, even if no pipeline
	// references them, so both the definition and the pipeline wiring must be gated on tier.
	odigosEbpfReceiverName              = "odigosebpf"
	pprofExtensionName                  = "pprof"
	odigosEnterpriseAuthExtensionName   = "odigos_enterprise_auth"
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

func ResourceDetectionEnabled(detectors []string) bool {
	return len(detectors) > 0
}

func BuildResourceDetectors(cfg *common.ResourceDetectorsConfiguration, runningOnGKE bool) []string {
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

func commonProcessors(nodeCG *odigosv1.CollectorsGroup, runningOnGKE bool, detectors []string) config.GenericMap {

	allProcessors := config.GenericMap{}
	for k, v := range staticProcessors {
		allProcessors[k] = v
	}

	memoryLimiterConfig := commonconf.GetMemoryLimiterConfig(nodeCG.Spec.ResourcesSettings)
	allProcessors[memoryLimiterProcessorName] = memoryLimiterConfig

	if ResourceDetectionEnabled(detectors) {
		allProcessors[resourceDetectionProcessorName] = config.GenericMap{
			"detectors": detectors,
			"timeout":   "2s",
		}
	}

	return allProcessors
}

var staticProcessors config.GenericMap

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
}

// commonReceivers returns the receivers every signal pipeline shares.
func commonReceivers(tier common.OdigosTier) config.GenericMap {
	receivers := config.GenericMap{
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
	}

	if tier.IsEnterprise() {
		receivers[odigosEbpfReceiverName] = config.GenericMap{}
	}

	return receivers
}

// commonExtensions returns the extension definitions for the given tier.
func commonExtensions(tier common.OdigosTier) config.GenericMap {
	extensions := config.GenericMap{
		healthCheckExtensionName: config.GenericMap{
			"endpoint": "0.0.0.0:13133",
		},
		pprofExtensionName: config.GenericMap{
			"endpoint": "0.0.0.0:1777",
		},
	}

	if tier.IsEnterprise() {
		extensions[odigosEnterpriseAuthExtensionName] = config.GenericMap{}
	}

	return extensions
}

// commonService returns the service block listing the extensions commonExtensions defined.
func commonService(tier common.OdigosTier) config.Service {
	extensions := []string{healthCheckExtensionName, pprofExtensionName}

	if tier.IsEnterprise() {
		extensions = append(extensions, odigosEnterpriseAuthExtensionName)
	}

	return config.Service{Extensions: extensions}
}

func CommonApplicationTelemetryConfig(nodeCG *odigosv1.CollectorsGroup, onGKE bool, odigosNamespace string, detectors []string, tier common.OdigosTier) config.Config {
	return config.Config{
		Receivers:  commonReceivers(tier),
		Exporters:  getCommonExporters(nodeCG.Spec.OtlpExporterConfiguration, odigosNamespace),
		Processors: commonProcessors(nodeCG, onGKE, detectors),
	}
}

func CommonConfig(tier common.OdigosTier) config.Config {
	return config.Config{
		Extensions: commonExtensions(tier),
		Service:    commonService(tier),
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
