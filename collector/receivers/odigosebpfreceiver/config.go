package odigosebpfreceiver

import (
	"go.opentelemetry.io/collector/receiver"
)

type Config struct {
	receiver.Settings `mapstructure:",squash"`
}
