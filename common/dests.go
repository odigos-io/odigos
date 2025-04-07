package common

type DestinationType string

const (
	AlibabaCloudDestinationType           DestinationType = "alibabacloud"
	AppDynamicsDestinationType            DestinationType = "appdynamics"
	AWSCloudWatchDestinationType          DestinationType = "cloudwatch"
	AWSS3DestinationType                  DestinationType = "s3"
	AWSXRayDestinationType                DestinationType = "xray"
	AxiomDestinationType                  DestinationType = "axiom"
	AzureBlobDestinationType              DestinationType = "azureblob"
	BetterStackDestinationType            DestinationType = "betterstack"
	BonreeDestinationType                 DestinationType = "bonree"
	CauselyDestinationType                DestinationType = "causely"
	ChecklyDestinationType                DestinationType = "checkly"
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
	GreptimeDestinationType               DestinationType = "greptime"
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
	ObserveDestinationType                DestinationType = "observe"
	OneUptimeDestinationType              DestinationType = "oneuptime"
	OpenObserveDestinationType            DestinationType = "openobserve"
	OpsVerseDestinationType               DestinationType = "opsverse"
	OracleDestinationType                 DestinationType = "oracle"
	OtlpHttpDestinationType               DestinationType = "otlphttp"
	PrometheusDestinationType             DestinationType = "prometheus"
	QrynDestinationType                   DestinationType = "qryn"
	QrynOSSDestinationType                DestinationType = "qryn-oss"
	QuickwitDestinationType               DestinationType = "quickwit"
	SentryDestinationType                 DestinationType = "sentry"
	SeqDestinationType                    DestinationType = "seq"
	SignozDestinationType                 DestinationType = "signoz"
	SplunkDestinationType                 DestinationType = "splunk"
	SumoLogicDestinationType              DestinationType = "sumologic"
	TelemetryHubDestinationType           DestinationType = "telemetryhub"
	TempoDestinationType                  DestinationType = "tempo"
	TingyunDestinationType                DestinationType = "tingyun"
	TraceloopDestinationType              DestinationType = "traceloop"
	UptraceDestinationType                DestinationType = "uptrace"
	VictoriaMetricsCloudDestinationType   DestinationType = "victoriametricscloud"
)
