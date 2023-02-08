package common

type DestinationType string

const (
	GrafanaDestinationType       DestinationType = "grafana"
	DatadogDestinationType       DestinationType = "datadog"
	HoneycombDestinationType     DestinationType = "honeycomb"
	NewRelicDestinationType      DestinationType = "newrelic"
	LogzioDestinationType        DestinationType = "logzio"
	PrometheusDestinationType    DestinationType = "prometheus"
	LokiDestinationType          DestinationType = "loki"
	TempoDestinationType         DestinationType = "tempo"
	JaegerDestinationType        DestinationType = "jaeger"
	ElasticsearchDestinationType DestinationType = "elasticsearch"
	GenericOTLPDestinationType   DestinationType = "otlp"
	SignozDestinationType        DestinationType = "signoz"
	QrynDestinationType          DestinationType = "qryn"
	OpsVerseDestinationType      DestinationType = "opsverse"
	SplunkDestinationType        DestinationType = "splunk"
	LightstepDestinationType     DestinationType = "lightstep"
	GoogleCloudDestinationType   DestinationType = "googlecloud"
	SentryDestinationType        DestinationType = "sentry"
)
