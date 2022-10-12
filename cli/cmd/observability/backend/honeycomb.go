package backend

import (
	"fmt"
	"github.com/keyval-dev/odigos/common"
	"github.com/spf13/cobra"
)

type Honeycomb struct{}

func (h *Honeycomb) Name() common.DestinationType {
	return common.HoneycombDestinationType
}

func (h *Honeycomb) ParseFlags(cmd *cobra.Command, selectedSignals []common.ObservabilitySignal) (*ObservabilityArgs, error) {
	apiKey := cmd.Flag("api-key").Value.String()
	if apiKey == "" {
		return nil, fmt.Errorf("API key required for Honeycomb backend, please specify --api-key")
	}

	return &ObservabilityArgs{
		Secret: map[string]string{
			"HONEYCOMB_API_KEY": apiKey,
		},
	}, nil
}

func (h *Honeycomb) SupportedSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
	}
}
