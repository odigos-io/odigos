package odigosvmprofileattrsprocessor

import (
	"errors"

	"github.com/odigos-io/odigos/common/unixfd"
)

// Config configures the VM profile resource attributes processor.
type Config struct {
	// SocketPath is the unix socket where the VM agent streams PID attribute events.
	SocketPath string `mapstructure:"socket_path"`
}

func createDefaultConfig() *Config {
	return &Config{
		SocketPath: unixfd.DefaultSocketPath,
	}
}

func (cfg *Config) Validate() error {
	if cfg.SocketPath == "" {
		return errors.New("socket_path must not be empty")
	}
	return nil
}
