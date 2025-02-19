package common

type DestinationType string

const (
	AppDynamicsDestinationType            DestinationType = "appdynamics"
	AWSCloudWatchDestinationType          DestinationType = "cloudwatch"
	AWSS3DestinationType                  DestinationType = "s3"
	AxiomDestinationType                  DestinationType = "axiom"
	AzureBlobDestinationType              DestinationType = "azureblob"
	BetterStackDestinationType            DestinationType = "betterstack"
	CauselyDestinationType                DestinationType = "causely"
	ChronosphereDestinationType           DestinationType = "chronosphere"
	ClickhouseDestinationType             DestinationType = "clickhouse"
	CoralogixDestinationType              DestinationType = "coralogix"
	Dash0DestinationType                  DestinationType = "dash0"
	DatadogDestinationType                DestinationType = "datadog"
	DebugDestinationType                  DestinationType = "debug"
	DynatraceDestinationType              DestinationType = "dynatrace"
	ElasticAPMDestinationType             DestinationType = "elasticapm"
	ElasticsearchDestinationType          DestinationType = "elasticsearch"
	GCSDestinationType                    DestinationType = "gcs"
	GenericOTLPDestinationType            DestinationType = "otlp"
	GoogleCloudDestinationType            DestinationType = "googlecloud"
	GrafanaCloudLokiDestinationType       DestinationType = "grafanacloudloki"
	GrafanaCloudPrometheusDestinationType DestinationType = "grafanacloudprometheus"
	GrafanaCloudTempoDestinationType      DestinationType = "grafanacloudtempo"
	GroundcoverDestinationType            DestinationType = "groundcover"
	HoneycombDestinationType              DestinationType = "honeycomb"
	HyperDxDestinationType                DestinationType = "hyperdx"
	InstanaDestinationType                DestinationType = "instana"
	JaegerDestinationType                 DestinationType = "jaeger"
	KafkaDestinationType                  DestinationType = "kafka"
	KloudMateDestinationType              DestinationType = "kloudmate"
	Last9DestinationType                  DestinationType = "last9"
	LightstepDestinationType              DestinationType = "lightstep"
	LogzioDestinationType                 DestinationType = "logzio"
	LokiDestinationType                   DestinationType = "loki"
	LumigoDestinationType                 DestinationType = "lumigo"
	MiddlewareDestinationType             DestinationType = "middleware"
	MockDestinationType                   DestinationType = "mock"
	NewRelicDestinationType               DestinationType = "newrelic"
	NopDestinationType                    DestinationType = "nop"
	OpsVerseDestinationType               DestinationType = "opsverse"
	OtlpHttpDestinationType               DestinationType = "otlphttp"
	PrometheusDestinationType             DestinationType = "prometheus"
	QrynDestinationType                   DestinationType = "qryn"
	QrynOSSDestinationType                DestinationType = "qryn-oss"
	QuickwitDestinationType               DestinationType = "quickwit"
	SentryDestinationType                 DestinationType = "sentry"
	SignozDestinationType                 DestinationType = "signoz"
	SplunkDestinationType                 DestinationType = "splunk"
	SumoLogicDestinationType              DestinationType = "sumologic"
	TempoDestinationType                  DestinationType = "tempo"
	TraceloopDestinationType              DestinationType = "traceloop"
	UptraceDestinationType                DestinationType = "uptrace"
)
