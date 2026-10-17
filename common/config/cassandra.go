package config

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/odigos-io/odigos/common"
)

type Cassandra struct{}

func (c *Cassandra) DestType() common.DestinationType { return common.CassandraDestinationType }

func (c *Cassandra) ModifyConfig(dest ExporterConfigurer, cfg *Config) ([]string, error) {
	values := dest.GetConfig()
	dsn := strings.TrimSpace(values["CASSANDRA_DSN"])
	if dsn == "" {
		return nil, errorMissingKey("CASSANDRA_DSN")
	}
	if strings.ContainsAny(dsn, "/, \t\n") || (strings.Contains(dsn, ":") && net.ParseIP(dsn) == nil) {
		return nil, fmt.Errorf("CASSANDRA_DSN must be a single Cassandra hostname or IP address")
	}
	if isMetricsEnabled(dest) || isProfilingEnabled(dest) {
		return nil, fmt.Errorf("Cassandra supports only traces and logs")
	}
	exporter := GenericMap{"dsn": dsn}
	for key, field := range map[string]string{"CASSANDRA_PORT": "port", "CASSANDRA_REPLICATION_FACTOR": "replication_factor"} {
		if value := values[key]; value != "" {
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 || (field == "port" && n > 65535) {
				return nil, fmt.Errorf("invalid %s", key)
			}
			if field == "port" {
				exporter[field] = n
			} else {
				exporter["replication"] = GenericMap{"class": "SimpleStrategy", field: n}
			}
		}
	}
	for key, field := range map[string]string{
		"CASSANDRA_KEYSPACE":    "keyspace",
		"CASSANDRA_TRACE_TABLE": "trace_table",
		"CASSANDRA_LOGS_TABLE":  "logs_table",
	} {
		if value := values[key]; value != "" {
			if !regexp.MustCompile(`^[a-z][a-z0-9_]*$`).MatchString(value) {
				return nil, fmt.Errorf("%s must be an unquoted lowercase CQL identifier", key)
			}
			exporter[field] = value
		}
	}
	if value := values["CASSANDRA_COMPRESSION"]; value != "" {
		switch value {
		case "LZ4Compressor", "SnappyCompressor", "DeflateCompressor", "ZstdCompressor":
		default:
			return nil, fmt.Errorf("unsupported Cassandra compression algorithm")
		}
		exporter["compression"] = GenericMap{"algorithm": value}
	}
	if username := values["CASSANDRA_USERNAME"]; username != "" {
		exporter["auth"] = GenericMap{"username": username, "password": SecretEnvPlaceholder("CASSANDRA_PASSWORD", dest)}
	}
	if values["CASSANDRA_TLS_ENABLED"] == "true" {
		tls := GenericMap{"insecure": false}
		if values["CASSANDRA_USE_CUSTOM_CA"] == "true" {
			tls["ca_pem"] = SecretEnvPlaceholder("CASSANDRA_CA_PEM", dest)
		}
		exporter["tls"] = tls
	}
	name := "cassandra/" + dest.GetID()
	cfg.Exporters[name] = exporter
	var pipelines []string
	for _, signal := range []struct {
		name    string
		enabled bool
	}{{"traces", isTracingEnabled(dest)}, {"logs", isLoggingEnabled(dest)}} {
		if signal.enabled {
			pipeline := signal.name + "/cassandra-" + dest.GetID()
			cfg.Service.Pipelines[pipeline] = Pipeline{Exporters: []string{name}}
			pipelines = append(pipelines, pipeline)
		}
	}
	return pipelines, nil
}
