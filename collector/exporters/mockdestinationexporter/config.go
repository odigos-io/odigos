package mockdestinationexporter

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// EncodingType controls how the exporter serializes telemetry before "sending" it.
type EncodingType string

const (
	// EncodingProto serializes telemetry to OTLP protobuf bytes, like most real destinations. Default.
	EncodingProto EncodingType = "proto"
	// EncodingNone skips serialization entirely
	EncodingNone EncodingType = "none"
	// EncodingJSON serializes telemetry to OTLP JSON bytes.
	EncodingJSON EncodingType = "json"
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

	// Encoding controls whether the exporter serializes telemetry before discarding it.
	// Real destinations marshal pdata into a wire format, which costs CPU proportional to
	// the payload size. Use "proto" or "json" to simulate that cost, or "none" to skip it.
	Encoding EncodingType `mapstructure:"encoding"`

	// these configs controls configures the various export options.
	// default values (when not set) are used just like the otlp exporter.
	TimeoutConfig exporterhelper.TimeoutConfig                             `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	QueueConfig   configoptional.Optional[exporterhelper.QueueBatchConfig] `mapstructure:"sending_queue"`
	RetryConfig   configretry.BackOffConfig                                `mapstructure:"retry_on_failure"`
}

func (c *Config) Validate() error {
	if c.ResponseDuration < 0 {
		return fmt.Errorf("response_duration must be a non-negative duration")
	}
	if c.RejectFraction < 0 || c.RejectFraction > 1 {
		return fmt.Errorf("reject_fraction must be a fraction between 0 and 1")
	}
	switch c.Encoding {
	case EncodingNone, EncodingProto, EncodingJSON:
	default:
		return fmt.Errorf("encoding must be one of %q, %q or %q", EncodingNone, EncodingProto, EncodingJSON)
	}
	return nil
}

var _ component.Config = (*Config)(nil)
