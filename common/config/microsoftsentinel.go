package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	sentinelEndpoint     = "MICROSOFT_SENTINEL_ENDPOINT"
	sentinelTenantID     = "MICROSOFT_SENTINEL_TENANT_ID"
	sentinelClientID     = "MICROSOFT_SENTINEL_CLIENT_ID"
	sentinelClientSecret = "MICROSOFT_SENTINEL_CLIENT_SECRET"
	sentinelRuleID       = "MICROSOFT_SENTINEL_RULE_ID"
	sentinelStreamName   = "MICROSOFT_SENTINEL_STREAM_NAME"
)

type MicrosoftSentinel struct{}

func (s *MicrosoftSentinel) DestType() common.DestinationType {
	return common.MicrosoftSentinelDestinationType
}

func (s *MicrosoftSentinel) ModifyConfig(dest ExporterConfigurer, cfg *Config) ([]string, error) {
	for _, signal := range dest.GetSignals() {
		if signal != common.LogsObservabilitySignal {
			return nil, fmt.Errorf("Microsoft Sentinel only supports logs, got %s", signal)
		}
	}
	if !isLoggingEnabled(dest) {
		return nil, fmt.Errorf("Microsoft Sentinel requires the logs signal")
	}
	values := map[string]string{}
	for _, key := range []string{sentinelEndpoint, sentinelTenantID, sentinelClientID, sentinelRuleID, sentinelStreamName} {
		value := strings.TrimSpace(dest.GetConfig()[key])
		if value == "" {
			return nil, errorMissingKey(key)
		}
		values[key] = value
	}
	endpoint, err := url.Parse(values[sentinelEndpoint])
	if err != nil || endpoint.Scheme != "https" || endpoint.Hostname() == "" || endpoint.User != nil ||
		(endpoint.Path != "" && endpoint.Path != "/") || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return nil, fmt.Errorf("%s must be an HTTPS DCR/DCE ingestion endpoint without credentials, path, query or fragment", sentinelEndpoint)
	}
	ruleID := values[sentinelRuleID]
	if !strings.HasPrefix(ruleID, "dcr-") || len(ruleID) <= len("dcr-") || strings.ContainsAny(ruleID, "/?# ") {
		return nil, fmt.Errorf("%s must be the DCR immutable ID starting with dcr-", sentinelRuleID)
	}
	stream := values[sentinelStreamName]
	if !strings.HasPrefix(stream, "Custom-") || len(stream) <= len("Custom-") || strings.ContainsAny(stream, "/?# ") {
		return nil, fmt.Errorf("%s must be the exact custom DCR input stream starting with Custom-", sentinelStreamName)
	}
	uniqueURI := "microsoftsentinel-" + dest.GetID()
	exporterName := "azureloganalytics/" + uniqueURI
	cfg.Exporters[exporterName] = GenericMap{
		"endpoint":         strings.TrimRight(values[sentinelEndpoint], "/"),
		"tenant_id":        values[sentinelTenantID],
		"client_id":        values[sentinelClientID],
		"client_secret":    SecretEnvPlaceholder(sentinelClientSecret, dest),
		"rule_id":          ruleID,
		"stream_name":      stream,
		"timeout":          "30s",
		"retry_on_failure": GenericMap{"enabled": true, "initial_interval": "5s", "max_interval": "30s", "max_elapsed_time": "300s"},
	}
	processorName := "transform/" + uniqueURI
	cfg.Processors[processorName] = GenericMap{
		"error_mode": "propagate",
		"log_statements": []GenericMap{{"context": "log", "statements": []string{
			`set(cache["raw_data"], String(body))`,
			`set(body, ParseJSON("{}"))`,
			`set(body["RawData"], cache["raw_data"])`,
			`set(body["ResourceAttributes"], resource.attributes)`,
			`set(body["LogAttributes"], attributes)`,
			`set(body["ScopeName"], instrumentation_scope.name)`,
			`set(body["ScopeVersion"], instrumentation_scope.version)`,
			`set(time, observed_time) where time_unix_nano == 0`,
			`set(time, Now()) where time_unix_nano == 0`,
			// The upstream exporter honors these attributes as routing overrides; keep this destination scoped to its configured DCR and stream.
			`delete_key(attributes, "sentinel_rule_id")`,
			`delete_key(attributes, "sentinel_stream_name")`,
			`delete_key(resource.attributes, "sentinel_rule_id")`,
			`delete_key(resource.attributes, "sentinel_stream_name")`,
		}}},
	}
	pipelineName := "logs/" + uniqueURI
	cfg.Service.Pipelines[pipelineName] = Pipeline{Processors: []string{processorName}, Exporters: []string{exporterName}}
	return []string{pipelineName}, nil
}
