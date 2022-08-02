package common

type DestinationType string

const (
	GrafanaDestinationType    DestinationType = "grafana"
	DatadogDestinationType    DestinationType = "datadog"
	HoneycombDestinationType  DestinationType = "honeycomb"
	NewRelicDestinationType   DestinationType = "newrelic"
	PrometheusDestinationType DestinationType = "prometheus"
	LokiDestinationType       DestinationType = "loki"
	TempoDestinationType      DestinationType = "tempo"
)
