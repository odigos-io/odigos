package config

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/url"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	clickhouseEndpoint                 = "CLICKHOUSE_ENDPOINT"
	clickhouseUsername                 = "CLICKHOUSE_USERNAME"
	clickhousePassword                 = "${CLICKHOUSE_PASSWORD}"
	clickhouseCreateSchema             = "CLICKHOUSE_CREATE_SCHEME"
	clickhouseDatabaseName             = "CLICKHOUSE_DATABASE_NAME"
	clickhouseTlsEnabled               = "CLICKHOUSE_TLS_ENABLED"
	clickhouseCaPem                    = "CLICKHOUSE_CA_PEM"
	clickhouseCertPem                  = "CLICKHOUSE_CERT_PEM"
	clickhouseKeyPem                   = "${CLICKHOUSE_KEY_PEM}"
	clickhouseKeyPemConfig             = "CLICKHOUSE_KEY_PEM"
	clickhouseInsecureSkipVerify       = "CLICKHOUSE_INSECURE_SKIP_VERIFY"
	clickhouseTracesTable              = "CLICKHOUSE_TRACES_TABLE"
	clickhouseLogsTable                = "CLICKHOUSE_LOGS_TABLE"
	clickhouseMetricsTableSum          = "CLICKHOUSE_METRICS_TABLE_SUM"
	clickhouseMetricsTableGauge        = "CLICKHOUSE_METRICS_TABLE_GAUGE"
	clickhouseMetricsTableHistogram    = "CLICKHOUSE_METRICS_TABLE_HISTOGRAM"
	clickhouseMetricsTableSummary      = "CLICKHOUSE_METRICS_TABLE_SUMMARY"
	clickhouseMetricsTableExpHistogram = "CLICKHOUSE_METRICS_TABLE_EXP_HISTOGRAM"
	hyperdxLogNormalizer               = "HYPERDX_LOG_NORMALIZER"
)

type Clickhouse struct{}

func (c *Clickhouse) DestType() common.DestinationType {
	return common.ClickhouseDestinationType
}

func validateCertificatePem(pemData string) error {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return errors.New("failed to decode PEM block")
	}
	if block.Type != "CERTIFICATE" {
		return errors.New("PEM block is not a certificate")
	}
	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		return errors.New("failed to parse certificate: " + err.Error())
	}
	return nil
}

func clickhouseTlsConfig(dest ExporterConfigurer) (GenericMap, error) {
	tlsConfig := GenericMap{
		"insecure": false,
	}
	if caPem, ok := dest.GetConfig()[clickhouseCaPem]; ok && caPem != "" {
		if err := validateCertificatePem(caPem); err != nil {
			return nil, errors.New("invalid CA certificate: " + err.Error())
		}
		tlsConfig["ca_pem"] = caPem
	}

	certPem, hasCert := dest.GetConfig()[clickhouseCertPem]
	_, hasKey := dest.GetConfig()[clickhouseKeyPemConfig]
	if hasCert && certPem != "" && !hasKey {
		return nil, errors.New("clickhouse client certificate (cert_pem) requires client private key (key_pem)")
	}
	if hasKey && (!hasCert || certPem == "") {
		return nil, errors.New("clickhouse client private key (key_pem) requires client certificate (cert_pem)")
	}
	if hasCert && certPem != "" && hasKey {
		if err := validateCertificatePem(certPem); err != nil {
			return nil, errors.New("invalid client certificate: " + err.Error())
		}
		tlsConfig["cert_pem"] = certPem
		tlsConfig["key_pem"] = clickhouseKeyPem
	}

	if insecureSkipVerify, ok := dest.GetConfig()[clickhouseInsecureSkipVerify]; ok && insecureSkipVerify != "" {
		tlsConfig["insecure_skip_verify"] = parseBool(insecureSkipVerify)
	}
	return tlsConfig, nil
}

func (c *Clickhouse) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	endpoint, exists := dest.GetConfig()[clickhouseEndpoint]
	if !exists {
		return nil, errors.New("clickhouse endpoint not specified, gateway will not be configured for Clickhouse")
	}

	if !strings.Contains(endpoint, "://") {
		endpoint = "tcp://" + endpoint
	}

	parsedUrl, err := url.Parse(endpoint)
	if err != nil {
		return nil, errors.New("clickhouse endpoint is not a valid URL")
	}

	if parsedUrl.Port() == "" {
		endpoint = strings.Replace(endpoint, parsedUrl.Host, parsedUrl.Host+":9000", 1)
	}

	exporterName := "clickhouse/clickhouse-" + dest.GetID()
	exporterConfig := GenericMap{
		"endpoint": endpoint,
	}

	if username, userExists := dest.GetConfig()[clickhouseUsername]; userExists {
		exporterConfig["username"] = username
		exporterConfig["password"] = clickhousePassword
	}

	createSchema := dest.GetConfig()[clickhouseCreateSchema]
	exporterConfig["create_schema"] = getBooleanConfig(createSchema, "create")

	dbName, exists := dest.GetConfig()[clickhouseDatabaseName]
	if !exists {
		return nil, errors.New("clickhouse database name not specified, gateway will not be configured for Clickhouse")
	}
	exporterConfig["database"] = dbName

	if dest.GetConfig()[clickhouseTlsEnabled] == "true" {
		tlsConfig, err := clickhouseTlsConfig(dest)
		if err != nil {
			return nil, err
		}
		exporterConfig["tls"] = tlsConfig
	}

	if tracesTable, ok := dest.GetConfig()[clickhouseTracesTable]; ok {
		exporterConfig["traces_table_name"] = tracesTable
	}

	if logsTable, ok := dest.GetConfig()[clickhouseLogsTable]; ok {
		exporterConfig["logs_table_name"] = logsTable
	}

	// Handle each metric table separately if provided
	metricsTables := GenericMap{}
	if sum, ok := dest.GetConfig()[clickhouseMetricsTableSum]; ok {
		metricsTables["sum"] = GenericMap{"name": sum}
	}
	if gauge, ok := dest.GetConfig()[clickhouseMetricsTableGauge]; ok {
		metricsTables["gauge"] = GenericMap{"name": gauge}
	}
	if hist, ok := dest.GetConfig()[clickhouseMetricsTableHistogram]; ok {
		metricsTables["histogram"] = GenericMap{"name": hist}
	}
	if summary, ok := dest.GetConfig()[clickhouseMetricsTableSummary]; ok {
		metricsTables["summary"] = GenericMap{"name": summary}
	}
	if expHist, ok := dest.GetConfig()[clickhouseMetricsTableExpHistogram]; ok {
		metricsTables["exponential_histogram"] = GenericMap{"name": expHist}
	}

	exporterConfig["metrics_tables"] = metricsTables

	currentConfig.Exporters[exporterName] = exporterConfig

	var pipelineNames []string
	if isTracingEnabled(dest) {
		name := "traces/clickhouse-" + dest.GetID()
		currentConfig.Service.Pipelines[name] = Pipeline{Exporters: []string{exporterName}}
		pipelineNames = append(pipelineNames, name)
	}
	if isMetricsEnabled(dest) {
		name := "metrics/clickhouse-" + dest.GetID()
		currentConfig.Service.Pipelines[name] = Pipeline{Exporters: []string{exporterName}}
		pipelineNames = append(pipelineNames, name)
	}
	if isLoggingEnabled(dest) {
		name := "logs/clickhouse-" + dest.GetID()
		pipeline := Pipeline{Exporters: []string{exporterName}}

		if val, ok := dest.GetConfig()[hyperdxLogNormalizer]; ok && getBooleanConfig(val, "true") {
			processorName := "transform/hyperdx-log-normalizer-" + dest.GetID()
			currentConfig.Processors[processorName] = HyperdxLogNormalizerProcessor
			pipeline.Processors = []string{processorName}
		}

		currentConfig.Service.Pipelines[name] = pipeline
		pipelineNames = append(pipelineNames, name)
	}

	return pipelineNames, nil
}
