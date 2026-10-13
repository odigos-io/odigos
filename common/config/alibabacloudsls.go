package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	SLS_ENDPOINT          = "SLS_ENDPOINT"
	SLS_PROJECT           = "SLS_PROJECT"
	SLS_LOGSTORE          = "SLS_LOGSTORE"
	SLS_ACCESS_KEY_ID     = "SLS_ACCESS_KEY_ID"
	SLS_ACCESS_KEY_SECRET = "SLS_ACCESS_KEY_SECRET"
)

type AlibabaCloudSLS struct{}

func (s *AlibabaCloudSLS) DestType() common.DestinationType {
	return common.AlibabaCloudSLSDestinationType
}

func (s *AlibabaCloudSLS) ModifyConfig(dest ExporterConfigurer, cfg *Config) ([]string, error) {
	config := dest.GetConfig()
	for _, key := range []string{SLS_ENDPOINT, SLS_PROJECT, SLS_LOGSTORE} {
		if strings.TrimSpace(config[key]) == "" {
			return nil, errorMissingKey(key)
		}
	}
	endpoint := strings.TrimSpace(config[SLS_ENDPOINT])
	if !strings.Contains(endpoint, "://") {
		endpoint = "https://" + endpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid SLS endpoint: %w", err)
	}
	if parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil ||
		(parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("SLS endpoint must be an HTTPS service endpoint without a path, credentials, query, or fragment")
	}
	if !isLoggingEnabled(dest) {
		return nil, nil
	}
	name := "alibabacloudsls-" + dest.GetID()
	exporterName := "alibabacloud_logservice/" + name
	cfg.Exporters[exporterName] = GenericMap{
		"endpoint":          "https://" + strings.TrimPrefix(parsed.Host, strings.TrimSpace(config[SLS_PROJECT])+"."),
		"project":           strings.TrimSpace(config[SLS_PROJECT]),
		"logstore":          strings.TrimSpace(config[SLS_LOGSTORE]),
		"access_key_id":     SecretEnvPlaceholder(SLS_ACCESS_KEY_ID, dest),
		"access_key_secret": SecretEnvPlaceholder(SLS_ACCESS_KEY_SECRET, dest),
	}
	processorName := "batch/" + name
	if cfg.Processors == nil {
		cfg.Processors = GenericMap{}
	}
	cfg.Processors[processorName] = GenericMap{
		"send_batch_size":     512,
		"send_batch_max_size": 512,
		"timeout":             "1s",
	}
	pipelineName := "logs/" + name
	cfg.Service.Pipelines[pipelineName] = Pipeline{Exporters: []string{exporterName}, Processors: []string{processorName}}
	return []string{pipelineName}, nil
}
