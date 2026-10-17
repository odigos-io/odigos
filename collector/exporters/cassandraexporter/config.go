// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cassandraexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/cassandraexporter"
import (
	"fmt"
	"regexp"
	"time"

	"go.opentelemetry.io/collector/config/configtls"

	"go.opentelemetry.io/collector/config/configopaque"
)

type Config struct {
	TLS         *configtls.ClientConfig `mapstructure:"tls"`
	Auth        Auth                    `mapstructure:"auth"`
	DSN         string                  `mapstructure:"dsn"`
	Keyspace    string                  `mapstructure:"keyspace"`
	TraceTable  string                  `mapstructure:"trace_table"`
	LogsTable   string                  `mapstructure:"logs_table"`
	Compression Compression             `mapstructure:"compression"`
	Replication Replication             `mapstructure:"replication"`
	Port        int                     `mapstructure:"port"`
	Timeout     time.Duration           `mapstructure:"timeout"`
}

type Replication struct {
	Class             string `mapstructure:"class"`
	ReplicationFactor int    `mapstructure:"replication_factor"`
	// prevent unkeyed literal initialization
	_ struct{}
}

type Compression struct {
	Algorithm string `mapstructure:"algorithm"`
	// prevent unkeyed literal initialization
	_ struct{}
}

type Auth struct {
	UserName string              `mapstructure:"username"`
	Password configopaque.String `mapstructure:"password"`
	// prevent unkeyed literal initialization
	_ struct{}
}

func (cfg *Config) Validate() error {
	identifier := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	for _, name := range []string{cfg.Keyspace, cfg.TraceTable, cfg.LogsTable} {
		if !identifier.MatchString(name) {
			return fmt.Errorf("invalid Cassandra schema identifier %q", name)
		}
	}
	if cfg.Port < 1 || cfg.Port > 65535 || cfg.Timeout <= 0 || cfg.Replication.ReplicationFactor < 1 {
		return fmt.Errorf("invalid Cassandra port, timeout or replication factor")
	}
	if cfg.Replication.Class != "SimpleStrategy" {
		return fmt.Errorf("pre-create production keyspaces with NetworkTopologyStrategy; automatic creation uses SimpleStrategy")
	}
	switch cfg.Compression.Algorithm {
	case "LZ4Compressor", "SnappyCompressor", "DeflateCompressor", "ZstdCompressor":
	default:
		return fmt.Errorf("invalid Cassandra compression")
	}
	return nil
}
