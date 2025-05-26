package config

import (
	"fmt"
	"strconv"

	"github.com/odigos-io/odigos/common"
)

type Debug struct{}

const (
	VERBOSITY        = "VERBOSITY"
	ITEMS_PER_SECOND = "ITEMS_PER_SECOND"
)

func (s *Debug) DestType() common.DestinationType {
	return common.DebugDestinationType
}

func (s *Debug) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	exporterName := "debug/" + dest.GetID()

	verbosity, verbosityExists := dest.GetConfig()[VERBOSITY]
	if !verbosityExists {
		// Default verbosity
		verbosity = "basic"
	}

	itemsPerSecond, itemsPerSecondExists := dest.GetConfig()[ITEMS_PER_SECOND]
	samplingInitial := 1    // log the first item each second
	samplingThereafter := 1 // log 1/1 items after that (e.g. all items)
	if itemsPerSecondExists {
		// Default items per second
		itemsPerSecondInt, err := strconv.Atoi(itemsPerSecond)
		if err != nil {
			return nil, fmt.Errorf("invalid value for %s: %s", ITEMS_PER_SECOND, itemsPerSecond)
		}
		samplingInitial = itemsPerSecondInt
		samplingThereafter = 0 // after logging the requested items each second, log 0/1 items (e.g. none)
	}

	currentConfig.Exporters[exporterName] = GenericMap{
		"verbosity":           verbosity,
		"sampling_initial":    samplingInitial,
		"sampling_thereafter": samplingThereafter,
	}

	var pipelineNames []string
	if IsTracingEnabled(dest) {
		tracesPipelineName := "traces/debug-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	if IsMetricsEnabled(dest) {
		metricsPipelineName := "metrics/debug-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, metricsPipelineName)
	}

	if IsLoggingEnabled(dest) {
		logsPipelineName := "logs/debug-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	return pipelineNames, nil
}
