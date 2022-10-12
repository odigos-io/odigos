package backend

import (
	"encoding/base64"
	"fmt"
	"github.com/keyval-dev/odigos/common"
	"github.com/spf13/cobra"
	"net/url"
)

const (
	GrafanaTempoUrlFlag  = "grafana-tempo-url"
	GrafanaTempoUserFlag = "grafana-tempo-user"
	GrafanaPromUrlFlag   = "grafana-prom-url"
	GrafanaPromUserFlag  = "grafana-prom-user"
	GrafanaLokiUrlFlag   = "grafana-loki-url"
	GrafanaLokiUserFlag  = "grafana-loki-user"
)

type Grafana struct{}

func (g *Grafana) Name() common.DestinationType {
	return common.GrafanaDestinationType
}

func (g *Grafana) ParseFlags(cmd *cobra.Command, selectedSignals []common.ObservabilitySignal) (*ObservabilityArgs, error) {
	apiKey := cmd.Flag("api-key").Value.String()
	if apiKey == "" {
		return nil, fmt.Errorf("API key required for Grafana Cloud backend, please specify --api-key")
	}

	args := &ObservabilityArgs{
		Data: make(map[string]string),
		Secret: map[string]string{
			"GRAFANA_API_KEY": apiKey,
		},
	}

	for _, s := range selectedSignals {
		switch s {
		case common.TracesObservabilitySignal:
			tempoUrl := cmd.Flag(GrafanaTempoUrlFlag).Value.String()
			if tempoUrl == "" {
				return nil, fmt.Errorf("tempo URL required when using Grafana Cloud backend with traces signal. pleease specify --%s", GrafanaTempoUrlFlag)
			}

			_, err := url.Parse(tempoUrl)
			if err != nil {
				return nil, fmt.Errorf("invalud tempo url specified: %s", err)
			}

			tempoUser := cmd.Flag(GrafanaTempoUserFlag).Value.String()
			if tempoUser == "" {
				return nil, fmt.Errorf("tempo user required when using Grafana Cloud backend with traces signal. pleease specify --%s", GrafanaTempoUserFlag)
			}

			args.Data["GRAFANA_TEMPO_URL"] = tempoUrl
			args.Secret["GRAFANA_TEMPO_AUTH_TOKEN"] = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", tempoUser, apiKey)))
		case common.MetricsObservabilitySignal:
			rwUrl := cmd.Flag(GrafanaPromUrlFlag).Value.String()
			if rwUrl == "" {
				return nil, fmt.Errorf("remotewrite URL required when using Grafana Cloud backend with metrics signal. pleease specify --%s", GrafanaPromUrlFlag)
			}

			_, err := url.Parse(rwUrl)
			if err != nil {
				return nil, fmt.Errorf("invalud remotewrite url specified: %s", err)
			}

			promUser := cmd.Flag(GrafanaPromUserFlag).Value.String()
			if promUser == "" {
				return nil, fmt.Errorf("prometheus user required when using Grafana Cloud backend with metrics signal. pleease specify --%s", GrafanaPromUserFlag)
			}
			args.Data["GRAFANA_REMOTEWRITE_URL"] = rwUrl
			args.Data["GRAFANA_METRICS_USER"] = promUser
		case common.LogsObservabilitySignal:
			lokiUrl := cmd.Flag(GrafanaLokiUrlFlag).Value.String()
			if lokiUrl == "" {
				return nil, fmt.Errorf("loki URL required when using Grafana Cloud backend with logs signal. pleease specify --%s", GrafanaLokiUrlFlag)
			}

			_, err := url.Parse(lokiUrl)
			if err != nil {
				return nil, fmt.Errorf("invalud loki url specified: %s", err)
			}

			lokiUser := cmd.Flag(GrafanaLokiUserFlag).Value.String()
			if lokiUser == "" {
				return nil, fmt.Errorf("loki user required when using Grafana Cloud backend with logs signal. pleease specify --%s", GrafanaLokiUserFlag)
			}

			args.Data["GRAFANA_LOKI_USER"] = lokiUser
			args.Data["GRAFANA_LOKI_URL"] = lokiUrl
		}
	}

	return args, nil
}

func (g *Grafana) SupportedSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
		common.MetricsObservabilitySignal,
		common.LogsObservabilitySignal,
	}
}
