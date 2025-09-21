package collectorconfig

import (
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common/config"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

const (
	BatchProcessorName           = "batch"
	MemoryLimiterProcessorName   = "memory_limiter"
	NodeNameProcessorName        = "resource/node-name"
	ClusterCollectorExporterName = "otlp/out-cluster-collector"
	OTLPInReceiverName           = "otlp/in"
)

const (
	healthCheckExtensionName = "health_check"
	pprofExtensionName       = "pprof"
)

func commonProcessors(nodeCG *odigosv1.CollectorsGroup) config.GenericMap {

	empty := struct{}{}

	processors := config.GenericMap{}

	processors[BatchProcessorName] = empty

	memoryLimiterConfig := commonconf.GetMemoryLimiterConfig(nodeCG.Spec.ResourcesSettings)
	processors[MemoryLimiterProcessorName] = memoryLimiterConfig

	processors[NodeNameProcessorName] = config.GenericMap{
		"attributes": []config.GenericMap{{
			"key":    string(semconv.K8SNodeNameKey),
			"value":  "${NODE_NAME}",
			"action": "upsert",
		}},
	}

	return processors
}

func commonExporters(odigosNamespace string) config.GenericMap {
	exporters := config.GenericMap{}

	endpoint := fmt.Sprintf("dns:///odigos-gateway.%s:4317", odigosNamespace)
	exporters[ClusterCollectorExporterName] = config.GenericMap{
		"endpoint": endpoint,
		"tls": config.GenericMap{
			"insecure": true,
		},
		"balancer_name": "round_robin",
	}

	return exporters
}

func commonReceivers() config.GenericMap {
	receivers := config.GenericMap{}

	receivers[OTLPInReceiverName] = config.GenericMap{
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
	}

	return receivers
}

func commonExtensions() config.GenericMap {
	extensions := config.GenericMap{}

	extensions[healthCheckExtensionName] = config.GenericMap{
		"endpoint": "0.0.0.0:13133",
	}

	extensions[pprofExtensionName] = config.GenericMap{
		"endpoint": "0.0.0.0:1777",
	}

	return extensions
}

func commonService() config.Service {
	return config.Service{
		Extensions: []string{healthCheckExtensionName, pprofExtensionName},
	}
}

func CommonConfig(odigosNamespace string, nodeCG *odigosv1.CollectorsGroup) config.Config {
	return config.Config{
		Receivers:  commonReceivers(),
		Exporters:  commonExporters(odigosNamespace),
		Processors: commonProcessors(nodeCG),
		Extensions: commonExtensions(),
		Service:    commonService(),
	}
}
