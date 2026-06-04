package testconnection

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
)

// builds config.ExporterConfigurer from primitive values so any caller can build one without depending on a specific destination input model.
type TestConnectionConfig struct {
	DestinationType string
	ID              string
	Config          map[string]string
	Signals         []common.ObservabilitySignal
}

var _ config.ExporterConfigurer = (*TestConnectionConfig)(nil)

func (c *TestConnectionConfig) GetSignals() []common.ObservabilitySignal {
	return c.Signals
}

func (c *TestConnectionConfig) GetType() common.DestinationType {
	return common.DestinationType(c.DestinationType)
}

func (c *TestConnectionConfig) GetID() string {
	if c.ID != "" {
		return c.ID
	}
	return c.DestinationType
}

func (c *TestConnectionConfig) GetConfig() map[string]string {
	if c.Config == nil {
		return map[string]string{}
	}
	return c.Config
}
