package backend

import (
	"fmt"
	"github.com/keyval-dev/odigos/common"
	"github.com/spf13/cobra"
	"net/url"
)

type Tempo struct{}

func (t *Tempo) Name() common.DestinationType {
	return common.TempoDestinationType
}

func (t *Tempo) ParseFlags(cmd *cobra.Command, selectedSignals []common.ObservabilitySignal) (*ObservabilityArgs, error) {
	tempoUrl := cmd.Flag("url").Value.String()
	if tempoUrl == "" {
		return nil, fmt.Errorf("tempo URL required when using Tempo backend. pleease specify --url")
	}

	_, err := url.Parse(tempoUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid tempo URL specified: %s", err)
	}

	return &ObservabilityArgs{
		Data: map[string]string{
			"TEMPO_URL": tempoUrl,
		},
		Secret: make(map[string]string),
	}, nil
}

func (t *Tempo) SupportedSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
	}
}
