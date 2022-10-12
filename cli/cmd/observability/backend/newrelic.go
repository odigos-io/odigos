package backend

import (
	"fmt"
	"github.com/keyval-dev/odigos/common"
	"github.com/spf13/cobra"
)

type NewRelic struct{}

func (n *NewRelic) Name() common.DestinationType {
	return common.NewRelicDestinationType
}

func (n *NewRelic) ParseFlags(cmd *cobra.Command, selectedSignals []common.ObservabilitySignal) (*ObservabilityArgs, error) {
	apiKey := cmd.Flag("api-key").Value.String()
	if apiKey == "" {
		return nil, fmt.Errorf("license key required for New Relic backend, please specify --api-key")
	}

	return &ObservabilityArgs{
		Secret: map[string]string{
			"NEWRELIC_API_KEY": apiKey,
		},
	}, nil
}

func (n *NewRelic) SupportedSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
		common.MetricsObservabilitySignal,
		common.LogsObservabilitySignal,
	}
}
