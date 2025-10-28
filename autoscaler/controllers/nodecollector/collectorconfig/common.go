package collectorconfig

import (
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

const (
	OTLPInReceiverName = "otlp/in"
)

const (
	healthCheckExtensionName       = "health_check"
	pprofExtensionName             = "pprof"
	batchProcessorName             = "batch"
	memoryLimiterProcessorName     = "memory_limiter"
	nodeNameProcessorName          = "resource/node-name"
	clusterCollectorExporterName   = "otlp/out-cluster-collector"
	resourceDetectionProcessorName = "resourcedetection"
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
		detectors = []string{"ec2", "azure", "aks"}
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

func getCommonExporters(enableDataCompression *bool) config.GenericMap {

	odigosNamespace := env.GetCurrentNamespace()

	compression := "none"
	if enableDataCompression != nil && *enableDataCompression {
		compression = "gzip"
	}

	return config.GenericMap{
		clusterCollectorExporterName: config.GenericMap{
			"endpoint": fmt.Sprintf("dns:///%s.%s:4317", k8sconsts.OdigosClusterCollectorDeploymentName, odigosNamespace),
			"tls": config.GenericMap{
				"insecure": true,
			},
			"compression": compression,
		},
	}
}

func init() {

	staticProcessors = config.GenericMap{
		batchProcessorName: config.GenericMap{},
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

func CommonConfig(nodeCG *odigosv1.CollectorsGroup, runningOnGKE bool) config.Config {
	return config.Config{
		Receivers:  commonReceivers,
		Exporters:  getCommonExporters(nodeCG.Spec.EnableDataCompression),
		Processors: commonProcessors(nodeCG, runningOnGKE),
		Extensions: commonExtensions,
		Service:    commonService,
	}
}
