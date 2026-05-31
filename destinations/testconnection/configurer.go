package testconnection

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
)

// builds config.ExporterConfigurer from primitive values so any caller can build one without depending on a specific destination input model.
type Configurer struct {
	DestinationType string
	ID              string
	Config          map[string]string
	Signals         []common.ObservabilitySignal
}

var _ config.ExporterConfigurer = (*Configurer)(nil)

func (c *Configurer) GetSignals() []common.ObservabilitySignal {
	return c.Signals
}

func (c *Configurer) GetType() common.DestinationType {
	return common.DestinationType(c.DestinationType)
}

func (c *Configurer) GetID() string {
	if c.ID != "" {
		return c.ID
	}
	return c.DestinationType
}

func (c *Configurer) GetConfig() map[string]string {
	if c.Config == nil {
		return map[string]string{}
	}
	return c.Config
}

// SignalsFromStrings converts signal names to the odigos signal type, ignoring unknown names.
func SignalsFromStrings(signals []string) []common.ObservabilitySignal {
	converted_signals := make([]common.ObservabilitySignal, 0, len(signals))
	for _, signal := range signals {
		switch signal {
		case string(common.TracesObservabilitySignal):
			converted_signals = append(converted_signals, common.TracesObservabilitySignal)
		case string(common.MetricsObservabilitySignal):
			converted_signals = append(converted_signals, common.MetricsObservabilitySignal)
		case string(common.LogsObservabilitySignal):
			converted_signals = append(converted_signals, common.LogsObservabilitySignal)
		case string(common.ProfilesObservabilitySignal):
			converted_signals = append(converted_signals, common.ProfilesObservabilitySignal)
		}
	}
	return converted_signals
}
