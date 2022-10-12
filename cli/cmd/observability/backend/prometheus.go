package backend

import (
	"fmt"
	"github.com/keyval-dev/odigos/common"
	"github.com/spf13/cobra"
	"net/url"
)

type Prometheus struct{}

func (p *Prometheus) Name() common.DestinationType {
	return common.PrometheusDestinationType
}

func (p *Prometheus) ParseFlags(cmd *cobra.Command, selectedSignals []common.ObservabilitySignal) (*ObservabilityArgs, error) {
	rwUrl := cmd.Flag("url").Value.String()
	if rwUrl == "" {
		return nil, fmt.Errorf("prometheus remote write URL required when using Prometheus backend. pleease specify --url")
	}

	_, err := url.Parse(rwUrl)
	if err != nil {
		return nil, fmt.Errorf("invalud prometheus remote write URL specified: %s", err)
	}

	return &ObservabilityArgs{
		Data: map[string]string{
			"PROMETHEUS_REMOTEWRITE_URL": rwUrl,
		},
		Secret: make(map[string]string),
	}, nil
}

func (p *Prometheus) SupportedSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{
		common.MetricsObservabilitySignal,
	}
}
