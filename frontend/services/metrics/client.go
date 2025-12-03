package metrics

import (
	"fmt"

	promapi "github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

const (
	VictoriaMetricsServiceName = "odigos-victoriametrics"
	// DefaultMetricsWindow is the default lookback window used for rate calculations
	DefaultMetricsWindow = "5m"
)

func NewAPIFromURL(baseURL string) (v1.API, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("own-metrics base URL is empty")
	}
	client, err := promapi.NewClient(promapi.Config{
		Address:      baseURL,
		RoundTripper: promapi.DefaultRoundTripper,
	})
	if err != nil {
		return nil, err
	}
	return v1.NewAPI(client), nil
}
