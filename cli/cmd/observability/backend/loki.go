package backend

import (
	"fmt"
	"github.com/keyval-dev/odigos/common"
	"github.com/spf13/cobra"
	"net/url"
)

type Loki struct{}

func (l *Loki) Name() common.DestinationType {
	return common.LokiDestinationType
}

func (l *Loki) ParseFlags(cmd *cobra.Command, selectedSignals []common.ObservabilitySignal) (*ObservabilityArgs, error) {
	lokiUrl := cmd.Flag("url").Value.String()
	if lokiUrl == "" {
		return nil, fmt.Errorf("loki URL required when using Loki backend. pleease specify --url")
	}

	_, err := url.Parse(lokiUrl)
	if err != nil {
		return nil, fmt.Errorf("invalud loki URL specified: %s", err)
	}

	return &ObservabilityArgs{
		Data: map[string]string{
			"LOKI_URL": lokiUrl,
		},
		Secret: make(map[string]string),
	}, nil
}

func (l *Loki) SupportedSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{
		common.LogsObservabilitySignal,
	}
}
