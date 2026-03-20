package datacollectorcfg

import (
	"maps"

	"github.com/odigos-io/odigos/common/config"
)

type BasicConfigOptions struct {
	AdditionalReceivers  config.GenericMap
	AdditionalProcessors config.GenericMap
	AdditionalExtensions config.GenericMap
	ServiceExtensions    []string
	ServiceTelemetry     *config.Telemetry
}

func GetBasicConfig(opts BasicConfigOptions) *config.Config {
	receivers := config.GenericMap{
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
	}
	if opts.AdditionalReceivers != nil {
		maps.Copy(receivers, opts.AdditionalReceivers)
	}

	processors := config.GenericMap{
		"batch": config.GenericMap{},
	}
	if opts.AdditionalProcessors != nil {
		maps.Copy(processors, opts.AdditionalProcessors)
	}

	extensions := config.GenericMap{
		"health_check": config.GenericMap{
			"endpoint": "0.0.0.0:13133",
		},
		"pprof": config.GenericMap{
			"endpoint": "0.0.0.0:1777",
		},
	}
	if opts.AdditionalExtensions != nil {
		maps.Copy(extensions, opts.AdditionalExtensions)
	}

	serviceExtensions := []string{"health_check", "pprof"}
	serviceExtensions = append(serviceExtensions, opts.ServiceExtensions...)

	service := config.Service{
		Pipelines:  map[string]config.Pipeline{},
		Extensions: serviceExtensions,
	}
	if opts.ServiceTelemetry != nil {
		service.Telemetry = *opts.ServiceTelemetry
	}

	return &config.Config{
		Receivers:  receivers,
		Processors: processors,
		Extensions: extensions,
		Exporters:  config.GenericMap{},
		Connectors: config.GenericMap{},
		Service:    service,
	}
}
