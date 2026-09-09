package config

import (
	"maps"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every Destination Secret is mounted into the one shared gateway collector, prefixed with
// DestSecretEnvPrefix (see getSecretsFromDests in the autoscaler), so a collector config may
// only reference secrets through destination-scoped env vars. A bare ${FIELD} placeholder
// resolves to whichever destination of that type was mounted last, or to nothing at all, and
// neither failure shows up in the Destination status - the exporter just authenticates with
// the wrong credential. These tests render every registered destination and pin that
// invariant for all of them at once.

const (
	sigTraces   = common.TracesObservabilitySignal
	sigMetrics  = common.MetricsObservabilitySignal
	sigLogs     = common.LogsObservabilitySignal
	sigProfiles = common.ProfilesObservabilitySignal
)

type secretScopingDestination struct {
	destType common.DestinationType
	id       string
	config   map[string]string
	signals  []common.ObservabilitySignal
}

func (d *secretScopingDestination) GetID() string                            { return d.id }
func (d *secretScopingDestination) GetType() common.DestinationType          { return d.destType }
func (d *secretScopingDestination) GetConfig() map[string]string             { return d.config }
func (d *secretScopingDestination) GetSignals() []common.ObservabilitySignal { return d.signals }

type secretScopingCase struct {
	// set only when a destination needs more than one case, e.g. because two auth methods
	// each carry a different secret.
	name string
	// the fields a user fills in for this destination, split by the destinations catalog into
	// Destination.spec.data and the Destination Secret.
	config  map[string]string
	signals []common.ObservabilitySignal
	// Secret keys the rendered config is expected to reference, each one scoped to the destination.
	wantScopedSecrets []string
	// placeholders the rendered config is expected to reference without a destination scope.
	wantUnscopedPlaceholders []string
}

var secretScopingPlaceholderRe = regexp.MustCompile(`\$\{([^}]+)\}`)

// secretScopingCases must hold an entry for every configer in availableConfigers, so a newly
// added destination cannot skip the scoping invariant.
var secretScopingCases = map[common.DestinationType][]secretScopingCase{
	"alibabacloud": {{
		config: map[string]string{
			"ALIBABA_ENDPOINT": "alibaba.example.com:8080",
			"ALIBABA_TOKEN":    "alibaba-token-value",
		},
		signals:           []common.ObservabilitySignal{sigTraces},
		wantScopedSecrets: []string{"ALIBABA_TOKEN"},
	}},
	"appdynamics": {{
		config: map[string]string{
			"APPDYNAMICS_ACCOUNT_NAME": "appdynamics-account-name-value",
			"APPDYNAMICS_API_KEY":      "appdynamics-api-key-value",
			"APPDYNAMICS_ENDPOINT_URL": "https://appdynamics.example.com",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"APPDYNAMICS_API_KEY"},
	}},
	"axiom": {{
		config: map[string]string{
			"AXIOM_API_TOKEN": "axiom-api-token-value",
			"AXIOM_DATASET":   "axiom-dataset-value",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigLogs},
		wantScopedSecrets: []string{"AXIOM_API_TOKEN"},
	}},
	"azureblob": {{
		config: map[string]string{
			"AZURE_BLOB_ACCOUNT_NAME":   "azure-blob-account-name-value",
			"AZURE_BLOB_CONTAINER_NAME": "azure-blob-container-name-value",
		},
		signals: []common.ObservabilitySignal{sigTraces, sigLogs},
	}},
	"azuremonitor": {{
		config: map[string]string{
			"AZURE_MONITOR_CONNECTION_STRING": "InstrumentationKey=00000000-0000-0000-0000-000000000000",
		},
		signals: []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
	}},
	"betterstack": {{
		config: map[string]string{
			"BETTERSTACK_TOKEN": "betterstack-token-value",
		},
		signals:           []common.ObservabilitySignal{sigMetrics, sigLogs},
		wantScopedSecrets: []string{"BETTERSTACK_TOKEN"},
	}},
	"bonree": {{
		config: map[string]string{
			"BONREE_ACCOUNT_ID":     "bonree-account-id-value",
			"BONREE_ENDPOINT":       "https://bonree.example.com",
			"BONREE_ENVIRONMENT_ID": "bonree-environment-id-value",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics},
		wantScopedSecrets: []string{"BONREE_ACCOUNT_ID", "BONREE_ENVIRONMENT_ID"},
	}},
	"causely": {{
		config: map[string]string{
			"CAUSELY_URL": "https://causely.example.com",
		},
		signals: []common.ObservabilitySignal{sigTraces, sigMetrics},
	}},
	"checkly": {{
		config: map[string]string{
			"CHECKLY_API_KEY": "checkly-api-key-value",
			"CHECKLY_ENDOINT": "https://checkly.example.com",
		},
		signals:           []common.ObservabilitySignal{sigTraces},
		wantScopedSecrets: []string{"CHECKLY_API_KEY"},
	}},
	"chronosphere": {{
		config: map[string]string{
			"CHRONOSPHERE_API_TOKEN": "chronosphere-api-token-value",
			"CHRONOSPHERE_DOMAIN":    "https://chronosphere.example.com",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics},
		wantScopedSecrets: []string{"CHRONOSPHERE_API_TOKEN"},
	}},
	"clickhouse": {{
		config: map[string]string{
			"CLICKHOUSE_CA_PEM":                      "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
			"CLICKHOUSE_CREATE_SCHEME":               "True",
			"CLICKHOUSE_DATABASE_NAME":               "otel",
			"CLICKHOUSE_ENDPOINT":                    "https://clickhouse.example.com",
			"CLICKHOUSE_LOGS_TABLE":                  "otel_logs",
			"CLICKHOUSE_METRICS_TABLE_EXP_HISTOGRAM": "otel_metrics_exponential_histogram",
			"CLICKHOUSE_METRICS_TABLE_GAUGE":         "otel_metrics_gauge",
			"CLICKHOUSE_METRICS_TABLE_HISTOGRAM":     "otel_metrics_histogram",
			"CLICKHOUSE_METRICS_TABLE_SUM":           "otel_metrics_sum",
			"CLICKHOUSE_METRICS_TABLE_SUMMARY":       "otel_metrics_summary",
			"CLICKHOUSE_PASSWORD":                    "clickhouse-password-value",
			"CLICKHOUSE_TRACES_TABLE":                "otel_traces",
			"CLICKHOUSE_USERNAME":                    "clickhouse-user",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"CLICKHOUSE_PASSWORD"},
	}},
	"cloudwatch": {{
		config: map[string]string{
			"AWS_CLOUDWATCH_LOG_GROUP_NAME":  "aws-cloudwatch-log-group-name-value",
			"AWS_CLOUDWATCH_LOG_STREAM_NAME": "aws-cloudwatch-log-stream-name-value",
		},
		signals: []common.ObservabilitySignal{sigMetrics, sigLogs},
	}},
	"coralogix": {{
		config: map[string]string{
			"CORALOGIX_APPLICATION_NAME": "coralogix-application-name-value",
			"CORALOGIX_DOMAIN":           "coralogix.com",
			"CORALOGIX_PRIVATE_KEY":      "coralogix-private-key-value",
			"CORALOGIX_SUBSYSTEM_NAME":   "coralogix-subsystem-name-value",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"CORALOGIX_PRIVATE_KEY"},
	}},
	"dash0": {{
		config: map[string]string{
			"DASH0_ENDPOINT": "https://dash0.example.com",
			"DASH0_TOKEN":    "dash0-token-value",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"DASH0_TOKEN"},
	}},
	"datadog": {{
		config: map[string]string{
			"DATADOG_API_KEY": "datadog-api-key-value",
			"DATADOG_SITE":    "us3.datadoghq.com",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"DATADOG_API_KEY"},
	}},
	// debug, gcs, mock, nop and sentry have no entry under destinations/data, so their fields
	// are only known from their own configer.
	"debug": {{
		config: map[string]string{
			"VERBOSITY":        "basic",
			"ITEMS_PER_SECOND": "10",
		},
		signals: []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
	}},
	"gcs": {{
		config: map[string]string{
			"GCS_BUCKET": "gcs-bucket-value",
		},
		signals: []common.ObservabilitySignal{sigTraces, sigLogs},
	}},
	"mock": {{
		config: map[string]string{
			"MOCK_RESPONSE_DURATION_MS": "10",
			"MOCK_REJECT_FRACTION":      "0.1",
		},
		signals: []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
	}},
	"nop": {{
		config:  map[string]string{},
		signals: []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
	}},
	"sentry": {{
		config:            map[string]string{},
		signals:           []common.ObservabilitySignal{sigTraces},
		wantScopedSecrets: []string{"DSN"},
	}},
	"dynamic": {{
		config: map[string]string{
			"DYNAMIC_CONFIGURATION_DATA": "endpoint: https://dynamic.example.com\nheaders:\n  authorization: ${MY_DYNAMIC_TOKEN}\n",
			"DYNAMIC_DESTINATION_TYPE":   "otlphttp",
		},
		signals:                  []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs, sigProfiles},
		wantUnscopedPlaceholders: []string{"MY_DYNAMIC_TOKEN"},
	}},
	"dynatrace": {{
		config: map[string]string{
			"DYNATRACE_API_TOKEN": "dynatrace-api-token-value",
			"DYNATRACE_URL":       "https://dynatrace.example.com",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"DYNATRACE_API_TOKEN"},
	}},
	"elasticapm": {{
		config: map[string]string{
			"ELASTIC_APM_SECRET_TOKEN":    "elastic-apm-secret-token-value",
			"ELASTIC_APM_SERVER_ENDPOINT": "https://elasticapm.example.com",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"ELASTIC_APM_SECRET_TOKEN"},
	}},
	"elasticsearch": {{
		config: map[string]string{
			"ELASTICSEARCH_PASSWORD": "elasticsearch-password-value",
			"ELASTICSEARCH_URL":      "https://elasticsearch.example.com",
			"ELASTICSEARCH_USERNAME": "elastic-user",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigLogs},
		wantScopedSecrets: []string{"ELASTICSEARCH_PASSWORD"},
	}},
	"googlecloud": {{
		config: map[string]string{
			"GCP_APPLICATION_CREDENTIALS": "gcp-application-credentials-value",
		},
		signals: []common.ObservabilitySignal{sigTraces, sigLogs},
	}},
	"googlecloudotlp": {{
		config: map[string]string{
			"GCP_APPLICATION_CREDENTIALS": "gcp-application-credentials-value",
		},
		signals: []common.ObservabilitySignal{sigTraces},
	}},
	"grafanacloudloki": {{
		config: map[string]string{
			"GRAFANA_CLOUD_LOKI_ENDPOINT": "https://grafanacloudloki.example.com",
			"GRAFANA_CLOUD_LOKI_PASSWORD": "grafana-cloud-loki-password-value",
			"GRAFANA_CLOUD_LOKI_USERNAME": "grafana-cloud-loki-username-value",
		},
		signals:           []common.ObservabilitySignal{sigLogs},
		wantScopedSecrets: []string{"GRAFANA_CLOUD_LOKI_PASSWORD"},
	}},
	"grafanacloudprometheus": {{
		config: map[string]string{
			"GRAFANA_CLOUD_PROMETHEUS_PASSWORD":    "grafana-cloud-prometheus-password-value",
			"GRAFANA_CLOUD_PROMETHEUS_RW_ENDPOINT": "https://prometheus-prod.grafana.net/api/prom/push",
			"GRAFANA_CLOUD_PROMETHEUS_USERNAME":    "grafana-cloud-prometheus-username-value",
		},
		signals:           []common.ObservabilitySignal{sigMetrics},
		wantScopedSecrets: []string{"GRAFANA_CLOUD_PROMETHEUS_PASSWORD"},
	}},
	"grafanacloudtempo": {{
		config: map[string]string{
			"GRAFANA_CLOUD_TEMPO_ENDPOINT": "https://grafanacloudtempo.example.com",
			"GRAFANA_CLOUD_TEMPO_PASSWORD": "grafana-cloud-tempo-password-value",
			"GRAFANA_CLOUD_TEMPO_USERNAME": "grafana-cloud-tempo-username-value",
		},
		signals:           []common.ObservabilitySignal{sigTraces},
		wantScopedSecrets: []string{"GRAFANA_CLOUD_TEMPO_PASSWORD"},
	}},
	"greptime": {{
		config: map[string]string{
			"GREPTIME_BASIC_PASSWORD": "greptime-basic-password-value",
			"GREPTIME_BASIC_USERNAME": "greptime-basic-username-value",
			"GREPTIME_DB_NAME":        "greptime-db-name-value",
			"GREPTIME_ENDPOINT":       "https://greptime.example.com",
		},
		signals:           []common.ObservabilitySignal{sigMetrics},
		wantScopedSecrets: []string{"GREPTIME_BASIC_PASSWORD"},
	}},
	"groundcover": {{
		config: map[string]string{
			"GROUNDCOVER_API_KEY":  "groundcover-api-key-value",
			"GROUNDCOVER_ENDPOINT": "https://groundcover.example.com",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"GROUNDCOVER_API_KEY"},
	}},
	"honeycomb": {{
		config: map[string]string{
			"HONEYCOMB_API_KEY":  "honeycomb-api-key-value",
			"HONEYCOMB_ENDPOINT": "api.honeycomb.io",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"HONEYCOMB_API_KEY"},
	}},
	"hyperdx": {{
		config: map[string]string{
			"HYPERDX_API_KEY": "hyperdx-api-key-value",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"HYPERDX_API_KEY"},
	}},
	"instana": {{
		config: map[string]string{
			"INSTANA_AGENT_KEY": "instana-agent-key-value",
			"INSTANA_ENDPOINT":  "https://instana.example.com",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"INSTANA_AGENT_KEY"},
	}},
	"jaeger": {{
		config: map[string]string{
			"JAEGER_URL": "jaeger.example.com:4317",
		},
		signals: []common.ObservabilitySignal{sigTraces},
	}},
	"kafka": {{
		config: map[string]string{
			"KAFKA_AUTH_METHOD":      "plain_text",
			"KAFKA_PASSWORD":         "kafka-password-value",
			"KAFKA_PROTOCOL_VERSION": "kafka-protocol-version-value",
			"KAFKA_USERNAME":         "kafka-user",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"KAFKA_PASSWORD"},
	}},
	"kloudmate": {{
		config: map[string]string{
			"KLOUDMATE_API_KEY": "kloudmate-api-key-value",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"KLOUDMATE_API_KEY"},
	}},
	"last9": {{
		config: map[string]string{
			"LAST9_OTLP_BASIC_AUTH_HEADER": "last9-otlp-basic-auth-header-value",
			"LAST9_OTLP_ENDPOINT":          "https://last9.example.com",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"LAST9_OTLP_BASIC_AUTH_HEADER"},
	}},
	"lightstep": {{
		config: map[string]string{
			"LIGHTSTEP_ACCESS_TOKEN": "lightstep-access-token-value",
		},
		signals:           []common.ObservabilitySignal{sigTraces},
		wantScopedSecrets: []string{"LIGHTSTEP_ACCESS_TOKEN"},
	}},
	"logzio": {{
		config: map[string]string{
			"LOGZIO_LOGS_TOKEN":    "logzio-logs-token-value",
			"LOGZIO_METRICS_TOKEN": "logzio-metrics-token-value",
			"LOGZIO_REGION":        "listener.logz.io",
			"LOGZIO_TRACING_TOKEN": "logzio-tracing-token-value",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"LOGZIO_LOGS_TOKEN", "LOGZIO_METRICS_TOKEN", "LOGZIO_TRACING_TOKEN"},
	}},
	"loki": {{
		config: map[string]string{
			"LOKI_LABELS":   "[\"k8s.container.name\", \"k8s.pod.name\", \"k8s.namespace.name\"]",
			"LOKI_PASSWORD": "loki-password-value",
			"LOKI_URL":      "https://loki.example.com",
			"LOKI_USERNAME": "loki-user",
		},
		signals:           []common.ObservabilitySignal{sigLogs},
		wantScopedSecrets: []string{"LOKI_PASSWORD"},
	}},
	"lumigo": {{
		config: map[string]string{
			"LUMIGO_ENDPOINT": "https://ga-otlp.lumigo-tracer-edge.golumigo.com",
			"LUMIGO_TOKEN":    "lumigo-token-value",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"LUMIGO_TOKEN"},
	}},
	"middleware": {{
		config: map[string]string{
			"MW_API_KEY": "mw-api-key-value",
			"MW_TARGET":  "https://middleware.example.com",
		},
		signals: []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		// MW_TARGET is not a secret field in the destinations catalog, so it is stored in
		// Destination.spec.data and never lands in the Secret this placeholder reads from.
		wantScopedSecrets: []string{"MW_API_KEY", "MW_TARGET"},
	}},
	"newrelic": {{
		config: map[string]string{
			"NEWRELIC_API_KEY":  "newrelic-api-key-value",
			"NEWRELIC_ENDPOINT": "https://otlp.nr-data.net",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"NEWRELIC_API_KEY"},
	}},
	"observe": {{
		config: map[string]string{
			"OBSERVE_CUSTOMER_ID": "observe-customer-id-value",
			"OBSERVE_TOKEN":       "observe-token-value",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"OBSERVE_TOKEN"},
	}},
	"oneuptime": {{
		config: map[string]string{
			"ONEUPTIME_INGESTION_KEY": "oneuptime-ingestion-key-value",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"ONEUPTIME_INGESTION_KEY"},
	}},
	"openobserve": {{
		config: map[string]string{
			"OPEN_OBSERVE_API_KEY":     "open-observe-api-key-value",
			"OPEN_OBSERVE_ENDPOINT":    "https://openobserve.example.com",
			"OPEN_OBSERVE_STREAM_NAME": "default",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigLogs},
		wantScopedSecrets: []string{"OPEN_OBSERVE_API_KEY"},
	}},
	"oracle": {{
		config: map[string]string{
			"ORACLE_DATA_KEY":      "oracle-data-key-value",
			"ORACLE_DATA_KEY_TYPE": "private",
			"ORACLE_ENDPOINT":      "https://oracle.example.com",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics},
		wantScopedSecrets: []string{"ORACLE_DATA_KEY"},
	}},
	"otlp": {{
		config: map[string]string{
			"OTLP_GRPC_CLIENT_CERT_PEM":      "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
			"OTLP_GRPC_CLIENT_KEY_PEM":       "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
			"OTLP_GRPC_ENDPOINT":             "otlp.example.com:4317",
			"OTLP_GRPC_MTLS_ENABLED":         "true",
			"OTLP_GRPC_OAUTH2_CLIENT_ID":     "client-id",
			"OTLP_GRPC_OAUTH2_CLIENT_SECRET": "otlp-grpc-oauth2-client-secret-value",
			"OTLP_GRPC_OAUTH2_ENABLED":       "true",
			"OTLP_GRPC_OAUTH2_TOKEN_URL":     "https://otlp.example.com/token",
			"OTLP_GRPC_TLS_ENABLED":          "true",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs, sigProfiles},
		wantScopedSecrets: []string{"OTLP_GRPC_CLIENT_CERT_PEM", "OTLP_GRPC_CLIENT_KEY_PEM", "OTLP_GRPC_OAUTH2_CLIENT_SECRET"},
	}},
	"otlphttp": {{
		// OAuth2 takes precedence over basic auth, so the two credentials need a case each.
		name: "oauth2 and mtls",
		config: map[string]string{
			"OTLP_HTTP_CLIENT_CERT_PEM":      "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
			"OTLP_HTTP_CLIENT_KEY_PEM":       "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
			"OTLP_HTTP_ENDPOINT":             "https://otlphttp.example.com",
			"OTLP_HTTP_MTLS_ENABLED":         "true",
			"OTLP_HTTP_OAUTH2_CLIENT_ID":     "client-id",
			"OTLP_HTTP_OAUTH2_CLIENT_SECRET": "otlp-http-oauth2-client-secret-value",
			"OTLP_HTTP_OAUTH2_ENABLED":       "true",
			"OTLP_HTTP_OAUTH2_TOKEN_URL":     "https://otlphttp.example.com/token",
			"OTLP_HTTP_TLS_ENABLED":          "true",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs, sigProfiles},
		wantScopedSecrets: []string{"OTLP_HTTP_CLIENT_CERT_PEM", "OTLP_HTTP_CLIENT_KEY_PEM", "OTLP_HTTP_OAUTH2_CLIENT_SECRET"},
	}, {
		name: "basic auth",
		config: map[string]string{
			"OTLP_HTTP_BASIC_AUTH_PASSWORD": "otlp-http-basic-auth-password-value",
			"OTLP_HTTP_BASIC_AUTH_USERNAME": "basic-user",
			"OTLP_HTTP_ENDPOINT":            "https://otlphttp.example.com",
			"OTLP_HTTP_TLS_ENABLED":         "true",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs, sigProfiles},
		wantScopedSecrets: []string{"OTLP_HTTP_BASIC_AUTH_PASSWORD"},
	}},
	"prometheus": {{
		// a username switches the exporter from bearer token auth to basic auth, and each of
		// the two reads a different key of the same Secret.
		name: "basic auth",
		config: map[string]string{
			"PROMETHEUS_BASIC_AUTH_PASSWORD": "prometheus-basic-auth-password-value",
			"PROMETHEUS_BASIC_AUTH_USERNAME": "prom-user",
			"PROMETHEUS_REMOTEWRITE_URL":     "https://prometheus.example.com",
			"PROMETHEUS_USE_AUTHENTICATION":  "true",
		},
		signals:           []common.ObservabilitySignal{sigMetrics},
		wantScopedSecrets: []string{"PROMETHEUS_BASIC_AUTH_PASSWORD"},
	}, {
		name: "bearer token",
		config: map[string]string{
			"PROMETHEUS_BEARER_TOKEN":       "prometheus-bearer-token-value",
			"PROMETHEUS_REMOTEWRITE_URL":    "https://prometheus.example.com",
			"PROMETHEUS_USE_AUTHENTICATION": "true",
		},
		signals:           []common.ObservabilitySignal{sigMetrics},
		wantScopedSecrets: []string{"PROMETHEUS_BEARER_TOKEN"},
	}},
	"pyroscope": {{
		config: map[string]string{
			"PYROSCOPE_URL": "https://pyroscope.example.com",
		},
		signals: []common.ObservabilitySignal{sigProfiles},
	}},
	"qryn": {{
		config: map[string]string{
			"QRYN_API_KEY":    "qryn-api-key-value",
			"QRYN_API_SECRET": "qryn-api-secret-value",
			"QRYN_URL":        "https://qryn.example.com",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"QRYN_API_SECRET"},
	}},
	"qryn-oss": {{
		config: map[string]string{
			"QRYN_OSS_PASSWORD": "qryn-oss-password-value",
			"QRYN_OSS_URL":      "https://qryn-oss.example.com",
			"QRYN_OSS_USERNAME": "qryn-user",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"QRYN_OSS_PASSWORD"},
	}},
	"quickwit": {{
		config: map[string]string{
			"QUICKWIT_URL": "https://quickwit.example.com",
		},
		signals: []common.ObservabilitySignal{sigTraces, sigLogs},
	}},
	"s3": {{
		config: map[string]string{
			"S3_BUCKET":    "s3-bucket-value",
			"S3_MARSHALER": "otlp_json",
			"S3_REGION":    "af-south-1",
		},
		signals: []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
	}},
	"seq": {{
		config: map[string]string{
			"SEQ_API_KEY":  "seq-api-key-value",
			"SEQ_ENDPOINT": "https://seq.example.com",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigLogs},
		wantScopedSecrets: []string{"SEQ_API_KEY"},
	}},
	"signalfx": {{
		config: map[string]string{
			"SIGNALFX_ACCESS_TOKEN": "signalfx-access-token-value",
			"SIGNALFX_CA_PEM":       "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
			"SIGNALFX_REALM":        "signalfx-realm-value",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics},
		wantScopedSecrets: []string{"SIGNALFX_ACCESS_TOKEN"},
	}},
	"signoz": {{
		config: map[string]string{
			"SIGNOZ_URL": "http://signoz.example.com:4317",
		},
		signals: []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
	}},
	"splunk": {{
		config: map[string]string{
			"SPLUNK_ACCESS_TOKEN": "splunk-access-token-value",
			"SPLUNK_REALM":        "splunk-realm-value",
		},
		signals:           []common.ObservabilitySignal{sigTraces},
		wantScopedSecrets: []string{"SPLUNK_ACCESS_TOKEN"},
	}},
	"splunkotlp": {{
		config: map[string]string{
			"SPLUNK_ACCESS_TOKEN": "splunk-access-token-value",
			"SPLUNK_REALM":        "splunk-realm-value",
		},
		signals:           []common.ObservabilitySignal{sigTraces},
		wantScopedSecrets: []string{"SPLUNK_ACCESS_TOKEN"},
	}},
	"sumologic": {{
		config: map[string]string{
			"SUMOLOGIC_COLLECTION_URL": "https://sumologic.example.com",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"SUMOLOGIC_COLLECTION_URL"},
	}},
	"telemetryhub": {{
		config: map[string]string{
			"TELEMETRY_HUB_API_KEY": "telemetry-hub-api-key-value",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
		wantScopedSecrets: []string{"TELEMETRY_HUB_API_KEY"},
	}},
	"tempo": {{
		config: map[string]string{
			"TEMPO_URL": "http://tempo.example.com:4317",
		},
		signals: []common.ObservabilitySignal{sigTraces},
	}},
	"tingyun": {{
		config: map[string]string{
			"TINGYUN_ENDPOINT":    "https://tingyun.example.com",
			"TINGYUN_LICENSE_KEY": "tingyun-license-key-value",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics},
		wantScopedSecrets: []string{"TINGYUN_LICENSE_KEY"},
	}},
	"traceloop": {{
		config: map[string]string{
			"TRACELOOP_API_KEY":  "traceloop-api-key-value",
			"TRACELOOP_ENDPOINT": "https://api.traceloop.com",
		},
		signals:           []common.ObservabilitySignal{sigTraces, sigMetrics},
		wantScopedSecrets: []string{"TRACELOOP_API_KEY"},
	}},
	"uptrace": {{
		config: map[string]string{
			"UPTRACE_DSN": "uptrace-dsn-value",
		},
		signals: []common.ObservabilitySignal{sigTraces, sigMetrics, sigLogs},
	}},
	"victoriametricscloud": {{
		config: map[string]string{
			"VICTORIA_METRICS_CLOUD_ENDPOINT": "https://victoriametricscloud.example.com",
			"VICTORIA_METRICS_CLOUD_TOKEN":    "victoria-metrics-cloud-token-value",
		},
		signals:           []common.ObservabilitySignal{sigMetrics},
		wantScopedSecrets: []string{"VICTORIA_METRICS_CLOUD_TOKEN"},
	}},
	"xray": {{
		config:  map[string]string{},
		signals: []common.ObservabilitySignal{sigTraces},
	}},
}

func renderSecretScopingCase(t *testing.T, destType common.DestinationType, destID string, c secretScopingCase, into *Config) {
	t.Helper()

	configers, err := LoadConfigers()
	require.NoError(t, err)
	configer, exists := configers[destType]
	require.True(t, exists, "no configer registered for %s", destType)

	dest := &secretScopingDestination{
		destType: destType,
		id:       destID,
		// some destinations (QrynOSS) rewrite the config map they are given, so every render
		// gets its own copy.
		config:  maps.Clone(c.config),
		signals: c.signals,
	}

	_, err = configer.ModifyConfig(dest, into)
	require.NoError(t, err, "the case config must be complete enough to render the destination")
}

func emptyCollectorConfig() *Config {
	return &Config{
		Receivers:  GenericMap{},
		Exporters:  GenericMap{},
		Processors: GenericMap{},
		Extensions: GenericMap{},
		Connectors: GenericMap{},
		Service:    Service{Pipelines: map[string]Pipeline{}},
	}
}

// collectPlaceholders walks every value the gateway collector will expand and returns the
// name inside each ${...}. Dynamic destinations arrive as yaml.v2 maps, hence map[any]any.
func collectPlaceholders(value any, found *[]string) {
	switch v := value.(type) {
	case string:
		for _, match := range secretScopingPlaceholderRe.FindAllStringSubmatch(v, -1) {
			*found = append(*found, match[1])
		}
	case GenericMap:
		for _, nested := range v {
			collectPlaceholders(nested, found)
		}
	case map[string]any:
		for _, nested := range v {
			collectPlaceholders(nested, found)
		}
	case map[any]any:
		for _, nested := range v {
			collectPlaceholders(nested, found)
		}
	case []GenericMap:
		for _, nested := range v {
			collectPlaceholders(nested, found)
		}
	case []any:
		for _, nested := range v {
			collectPlaceholders(nested, found)
		}
	case []string:
		for _, nested := range v {
			collectPlaceholders(nested, found)
		}
	}
}

func configPlaceholders(cfg *Config) []string {
	var found []string
	for _, section := range []GenericMap{cfg.Receivers, cfg.Exporters, cfg.Processors, cfg.Extensions, cfg.Connectors} {
		collectPlaceholders(section, &found)
	}
	return uniqueSorted(found)
}

// splitByScope separates the placeholders that belong to destID from the ones that are not
// scoped to it at all, returning the scoped ones with their prefix stripped.
func splitByScope(placeholders []string, destID string) (scoped []string, unscoped []string) {
	prefix := DestSecretEnvPrefix(destID)
	for _, placeholder := range placeholders {
		if strings.HasPrefix(placeholder, prefix) {
			scoped = append(scoped, strings.TrimPrefix(placeholder, prefix))
			continue
		}
		unscoped = append(unscoped, placeholder)
	}
	return uniqueSorted(scoped), uniqueSorted(unscoped)
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func secretScopingCaseName(destType common.DestinationType, c secretScopingCase) string {
	if c.name == "" {
		return string(destType)
	}
	return string(destType) + "/" + c.name
}

func TestEveryRegisteredDestinationHasASecretScopingCase(t *testing.T) {
	configers, err := LoadConfigers()
	require.NoError(t, err)

	registered := make([]string, 0, len(configers))
	for destType := range configers {
		registered = append(registered, string(destType))
	}

	covered := make([]string, 0, len(secretScopingCases))
	for destType, cases := range secretScopingCases {
		covered = append(covered, string(destType))
		assert.NotEmpty(t, cases, "%s must have at least one case", destType)
	}

	assert.ElementsMatch(t, registered, covered,
		"every destination must be rendered by TestDestinationConfigsReferenceOnlyScopedSecrets")
}

func TestDestinationConfigsReferenceOnlyScopedSecrets(t *testing.T) {
	for destType, cases := range secretScopingCases {
		for _, c := range cases {
			t.Run(secretScopingCaseName(destType, c), func(t *testing.T) {
				destID := "odigos.io.dest." + string(destType) + "-aaaa"
				cfg := emptyCollectorConfig()
				renderSecretScopingCase(t, destType, destID, c, cfg)

				scoped, unscoped := splitByScope(configPlaceholders(cfg), destID)

				assert.Equal(t, uniqueSorted(c.wantScopedSecrets), scoped,
					"the Secret keys this destination reads from its own Secret changed")
				assert.Equal(t, uniqueSorted(c.wantUnscopedPlaceholders), unscoped,
					"a placeholder that is not scoped to this destination reads another destination's env var")
			})
		}
	}
}

func TestTwoDestinationsOfTheSameTypeShareNoSecretEnvVar(t *testing.T) {
	for destType, cases := range secretScopingCases {
		for _, c := range cases {
			if len(c.wantScopedSecrets) == 0 {
				continue
			}
			t.Run(secretScopingCaseName(destType, c), func(t *testing.T) {
				firstID := "odigos.io.dest." + string(destType) + "-aaaa"
				secondID := "odigos.io.dest." + string(destType) + "-bbbb"

				// both destinations render into one config, exactly as two destinations of the
				// same type do in a real cluster gateway.
				cfg := emptyCollectorConfig()
				renderSecretScopingCase(t, destType, firstID, c, cfg)
				firstPlaceholders := configPlaceholders(cfg)
				renderSecretScopingCase(t, destType, secondID, c, cfg)
				secondPlaceholders := placeholdersNotIn(configPlaceholders(cfg), firstPlaceholders)

				firstScoped, _ := splitByScope(firstPlaceholders, firstID)
				secondScoped, _ := splitByScope(secondPlaceholders, secondID)

				assert.Equal(t, uniqueSorted(c.wantScopedSecrets), firstScoped)
				assert.Equal(t, uniqueSorted(c.wantScopedSecrets), secondScoped,
					"the second destination must read its own env var for every secret it uses")
			})
		}
	}
}

// Dynamic destinations are the deliberate exception: their exporter yaml is written by the
// user, so their Secret is mounted unprefixed and two dynamic destinations that pick the same
// ${ENV} name do read the same value. The destination field tooltip warns about it, and
// getSecretsFromDests in the autoscaler is what keeps them unprefixed.
func TestTwoDynamicDestinationsShareTheirUserAuthoredEnvVars(t *testing.T) {
	cases := secretScopingCases[common.DynamicDestinationType]
	require.Len(t, cases, 1)

	cfg := emptyCollectorConfig()
	renderSecretScopingCase(t, common.DynamicDestinationType, "odigos.io.dest.dynamic-aaaa", cases[0], cfg)
	firstPlaceholders := configPlaceholders(cfg)
	renderSecretScopingCase(t, common.DynamicDestinationType, "odigos.io.dest.dynamic-bbbb", cases[0], cfg)

	require.NotEmpty(t, firstPlaceholders)
	assert.Empty(t, placeholdersNotIn(configPlaceholders(cfg), firstPlaceholders),
		"the second dynamic destination reuses the env vars of the first one")
}

func placeholdersNotIn(placeholders []string, exclude []string) []string {
	excluded := make(map[string]struct{}, len(exclude))
	for _, placeholder := range exclude {
		excluded[placeholder] = struct{}{}
	}
	var out []string
	for _, placeholder := range placeholders {
		if _, ok := excluded[placeholder]; !ok {
			out = append(out, placeholder)
		}
	}
	return out
}
