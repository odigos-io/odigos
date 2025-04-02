package config

import (
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common"

	corev1 "k8s.io/api/core/v1"
)

const (
	GREPTIME_ENDPOINT    = "GREPTIME_ENDPOINT"
	GREPTIME_DB_NAME     = "GREPTIME_DB_NAME"
	GREPTIME_BASIC_TOKEN = "GREPTIME_BASIC_TOKEN"
)

type Greptime struct{}

func (j *Greptime) DestType() common.DestinationType {
	return common.GreptimeDestinationType
}

func (j *Greptime) ModifyConfig(dest ExporterConfigurer, cfg *Config) ([]string, error) {
	config := dest.GetConfig()
	uniqueUri := "greptime-" + dest.GetID()
	var pipelineNames []string

	endpoint, exists := config[GREPTIME_ENDPOINT]
	if !exists {
		return nil, errorMissingKey(GREPTIME_ENDPOINT)
	}
	endpoint, err := parseOtlpHttpEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	endpoint += "/v1/otlp"

	dbName, exists := config[GREPTIME_DB_NAME]
	if !exists {
		return nil, errorMissingKey(GREPTIME_DB_NAME)
	}

	secret, err := getSecret(dest.GetSecretName())
	if err != nil {
		return nil, err
	}
	err = verifyGreptimeToken(secret)
	if err != nil {
		return nil, err
	}
	token := encodeBase64(string(secret.Data[GREPTIME_BASIC_TOKEN]))

	exporterName := "otlphttp/" + uniqueUri
	cfg.Exporters[exporterName] = GenericMap{
		"endpoint": endpoint,
		"headers": GenericMap{
			"X-Greptime-DB-Name": dbName,
			"Authorization":      "Basic " + token,
		},
	}

	if isMetricsEnabled(dest) {
		pipeName := "metrics/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}

func verifyGreptimeToken(secret *corev1.Secret) error {
	data, exists := secret.Data[GREPTIME_BASIC_TOKEN]
	if !exists {
		return fmt.Errorf("secret does not contain key GREPTIME_BASIC_TOKEN")
	}

	rawToken := string(data)
	parts := strings.SplitN(rawToken, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("GREPTIME_BASIC_TOKEN in secret must be in 'username:password' format")
	}

	return nil
}
