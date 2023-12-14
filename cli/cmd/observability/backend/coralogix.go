package backend

import (
	"fmt"
	"github.com/keyval-dev/odigos/common"
	"github.com/spf13/cobra"
	"net/url"
	"strings"
)

type Coralogix struct{}

func (c *Coralogix) Name() common.DestinationType {
	return common.CoralogixDestinationType
}

func (c *Coralogix) SupportedSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
		common.MetricsObservabilitySignal,
		common.LogsObservabilitySignal,
	}
}