package mockdestinationexporter

import (
	"time"
)

// Config contains the main configuration options for the mockdestination exporter
type Config struct {

	// ResponseDuration is the amount of time the exporter will wait before returning a response.
	// It can be used to simulate loaded and slow destinations.
	ResponseDuration time.Duration `mapstructure:"response_duration"`

	// RejectFraction is the fraction of exports that will randomly be rejected.
	// Set to 0 to disable rejection, and to 1 to reject all exports.
	// Can be used to simulate destinations that are back-pressuring the collector.
	RejectFraction float64 `mapstructure:"reject_fraction"`
}
