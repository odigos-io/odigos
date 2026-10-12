package config

import (
	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCassandraConfig(t *testing.T) {
	dest := &mockDestination{id: "odigos.io.dest.cassandra-a", config: map[string]string{"CASSANDRA_DSN": "cassandra.local", "CASSANDRA_PORT": "9042", "CASSANDRA_USERNAME": "writer", "CASSANDRA_TLS_ENABLED": "true", "CASSANDRA_USE_CUSTOM_CA": "true", "CASSANDRA_REPLICATION_FACTOR": "3"}, signals: []common.ObservabilitySignal{common.TracesObservabilitySignal, common.LogsObservabilitySignal}}
	cfg := newKafkaConfig()
	pipes, err := (&Cassandra{}).ModifyConfig(dest, cfg)
	require.NoError(t, err)
	require.Len(t, pipes, 2)
	exporter := cfg.Exporters["cassandra/"+dest.id].(GenericMap)
	require.Equal(t, 9042, exporter["port"])
	require.Equal(t, GenericMap{"class": "SimpleStrategy", "replication_factor": 3}, exporter["replication"])
	require.Equal(t, SecretEnvPlaceholder("CASSANDRA_PASSWORD", dest), exporter["auth"].(GenericMap)["password"])
	require.Equal(t, SecretEnvPlaceholder("CASSANDRA_CA_PEM", dest), exporter["tls"].(GenericMap)["ca_pem"])
	for _, signal := range []common.ObservabilitySignal{common.MetricsObservabilitySignal, common.ProfilesObservabilitySignal} {
		dest.signals = []common.ObservabilitySignal{signal}
		_, err = (&Cassandra{}).ModifyConfig(dest, newKafkaConfig())
		require.Error(t, err)
	}
}
func TestCassandraRejectsInvalidConfig(t *testing.T) {
	for _, values := range []map[string]string{{}, {"CASSANDRA_DSN": "http://host"}, {"CASSANDRA_DSN": "host:9042"}, {"CASSANDRA_DSN": "host", "CASSANDRA_PORT": "0"}, {"CASSANDRA_DSN": "host", "CASSANDRA_KEYSPACE": "otel; DROP KEYSPACE system"}, {"CASSANDRA_DSN": "host", "CASSANDRA_COMPRESSION": "bad"}} {
		dest := &mockDestination{config: values, signals: []common.ObservabilitySignal{common.TracesObservabilitySignal}}
		cfg := newKafkaConfig()
		_, err := (&Cassandra{}).ModifyConfig(dest, cfg)
		require.Error(t, err)
		require.Empty(t, cfg.Exporters)
		require.Empty(t, cfg.Service.Pipelines)
	}
}
