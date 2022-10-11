package backend

import (
	"fmt"
	"github.com/keyval-dev/odigos/common"
	"github.com/spf13/cobra"
	"net/url"
	"strings"
)

type Datadog struct{}

func (d *Datadog) Name() common.DestinationType {
	return common.DatadogDestinationType
}

func (d *Datadog) ParseFlags(cmd *cobra.Command, selectedSignals []common.ObservabilitySignal) (*ObservabilityArgs, error) {
	apiKey := cmd.Flag("api-key").Value.String()
	if apiKey == "" {
		return nil, fmt.Errorf("API key required for Datadog backend, please specify --api-key")
	}

	targetUrl := cmd.Flag("url").Value.String()
	if targetUrl == "" {
		return nil, fmt.Errorf("URL required for Datadog backend, please specify --url")
	}

	_, err := url.Parse(targetUrl)
	if err != nil {
		return nil, fmt.Errorf("invalud url specified: %s", err)
	}

	if !strings.Contains(targetUrl, "datadoghq.com") {
		return nil, fmt.Errorf("%s is not a valid datadog url", targetUrl)
	}

	return &ObservabilityArgs{
		Data: map[string]string{
			"DATADOG_SITE": targetUrl,
		},
		Secret: map[string]string{
			"DATADOG_API_KEY": apiKey,
		},
	}, nil
}

func (d *Datadog) SupportedSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
		common.MetricsObservabilitySignal,
	}
}
