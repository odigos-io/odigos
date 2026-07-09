package config

import (
	"fmt"

	"github.com/odigos-io/odigos/common"
)

type CrdReceiverResults struct {
	ReceiversConfig  Config
	MetricsReceivers []string
	TracesReceivers  []string
	LogsReceivers    []string
	Errs             map[string]error
}

func CrdReceiverToConfig(receivers []ReceiverConfigurer) CrdReceiverResults {
	results := CrdReceiverResults{
		ReceiversConfig: Config{
			Receivers: GenericMap{},
		},
		MetricsReceivers: []string{},
		TracesReceivers:  []string{},
		LogsReceivers:    []string{},
		Errs:             make(map[string]error),
	}

	for _, receiver := range receivers {
		if receiver.IsDisabled() {
			continue
		}

		receiverKey := receiver.GetType()
		if receiver.GetReceiverName() != "" {
			receiverKey = fmt.Sprintf("%s/%s", receiver.GetType(), receiver.GetReceiverName())
		}
		if receiverKey == "" {
			continue
		}

		receiverConfig, err := receiver.GetConfig()
		if err != nil {
			results.Errs[receiver.GetID()] = fmt.Errorf("failed to convert receiver %q to collector config: %w", receiver.GetID(), err)
			continue
		}

		results.ReceiversConfig.Receivers[receiverKey] = receiverConfig

		for _, signal := range receiver.GetSignals() {
			switch signal {
			case common.TracesObservabilitySignal:
				results.TracesReceivers = append(results.TracesReceivers, receiverKey)
			case common.MetricsObservabilitySignal:
				results.MetricsReceivers = append(results.MetricsReceivers, receiverKey)
			case common.LogsObservabilitySignal:
				results.LogsReceivers = append(results.LogsReceivers, receiverKey)
			}
		}
	}

	return results
}
