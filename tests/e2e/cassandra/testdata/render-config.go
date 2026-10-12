package main

import (
	"encoding/json"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"os"
)

type dest struct{}

func (dest) GetID() string                   { return "core1772" }
func (dest) GetType() common.DestinationType { return common.CassandraDestinationType }
func (dest) GetSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{common.TracesObservabilitySignal, common.LogsObservabilitySignal}
}
func (dest) GetConfig() map[string]string {
	return map[string]string{"CASSANDRA_DSN": "localhost", "CASSANDRA_PORT": "19042", "CASSANDRA_KEYSPACE": "core1772_e2e", "CASSANDRA_USERNAME": "cassandra", "CASSANDRA_TLS_ENABLED": "true", "CASSANDRA_USE_CUSTOM_CA": "true", "CASSANDRA_COMPRESSION": "ZstdCompressor"}
}
func main() {
	cfg := &config.Config{Receivers: config.GenericMap{"otlp": config.GenericMap{"protocols": config.GenericMap{"http": config.GenericMap{"endpoint": "127.0.0.1:14318"}}}}, Exporters: config.GenericMap{}, Processors: config.GenericMap{"batch": config.GenericMap{"timeout": "1s", "send_batch_size": 100}}, Service: config.Service{Pipelines: map[string]config.Pipeline{}}}
	names, err := (&config.Cassandra{}).ModifyConfig(dest{}, cfg)
	if err != nil {
		panic(err)
	}
	for _, name := range names {
		p := cfg.Service.Pipelines[name]
		p.Receivers = []string{"otlp"}
		p.Processors = []string{"batch"}
		cfg.Service.Pipelines[name] = p
	}
	json.NewEncoder(os.Stdout).Encode(cfg)
}
