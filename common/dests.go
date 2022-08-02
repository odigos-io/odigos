package common

type DestinationType string

const (
	GrafanaDestinationType   DestinationType = "grafana"
	DatadogDestinationType   DestinationType = "datadog"
	HoneycombDestinationType DestinationType = "honeycomb"
	NewRelicDestinationType  DestinationType = "newrelic"
	LogzioDestinationType    DestinationType = "logzio"
)
