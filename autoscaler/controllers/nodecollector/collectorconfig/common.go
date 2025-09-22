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
	BatchProcessorName             = "batch"
	MemoryLimiterProcessorName     = "memory_limiter"
	NodeNameProcessorName          = "resource/node-name"
	ClusterCollectorExporterName   = "otlp/out-cluster-collector"
	OTLPInReceiverName             = "otlp/in"
	ResourceDetectionProcessorName = "resourcedetection"
)

const (
	healthCheckExtensionName = "health_check"
	pprofExtensionName       = "pprof"
)

func commonProcessors(nodeCG *odigosv1.CollectorsGroup, runningOnGKE bool) config.GenericMap {

	allProcessors := config.GenericMap{}
	for k, v := range staticProcessors {
		allProcessors[k] = v
	}

	memoryLimiterConfig := commonconf.GetMemoryLimiterConfig(nodeCG.Spec.ResourcesSettings)
	allProcessors[MemoryLimiterProcessorName] = memoryLimiterConfig

	var detectors []string
	// This is a workaround to avoid adding the gcp detector if not running on a gke environment
	// once https://github.com/GoogleCloudPlatform/opentelemetry-operations-go/issues/1026 is resolved, we can always put the gcp detector
	if runningOnGKE {
		detectors = []string{"gcp"}
	} else {
		detectors = []string{"ec2", "azure"}
	}
	allProcessors[ResourceDetectionProcessorName] = config.GenericMap{
		"detectors": detectors,
		"timeout":   "2s",
	}

	return allProcessors
}

var staticProcessors config.GenericMap
var commonExporters config.GenericMap
var commonReceivers config.GenericMap
var commonExtensions config.GenericMap
var commonService config.Service

func init() {
	odigosNamespace := env.GetCurrentNamespace()

	staticProcessors = config.GenericMap{
		BatchProcessorName: config.GenericMap{},
		NodeNameProcessorName: config.GenericMap{
			"attributes": []config.GenericMap{{
				"key":    string(semconv.K8SNodeNameKey),
				"value":  "${NODE_NAME}",
				"action": "upsert",
			}},
		},
	}

	commonExporters = config.GenericMap{
		ClusterCollectorExporterName: config.GenericMap{
			"endpoint": fmt.Sprintf("dns:///%s.%s:4317", k8sconsts.OdigosClusterCollectorDeploymentName, odigosNamespace),
			"tls": config.GenericMap{
				"insecure": true,
			},
			"balancer_name": "round_robin",
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
		Exporters:  commonExporters,
		Processors: commonProcessors(nodeCG, runningOnGKE),
		Extensions: commonExtensions,
		Service:    commonService,
	}
}
