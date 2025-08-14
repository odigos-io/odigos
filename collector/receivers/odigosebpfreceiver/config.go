package odigosebpfreceiver

import (
	"time"

	"go.opentelemetry.io/collector/receiver"
)

type Config struct {
	receiver.Settings `mapstructure:",squash"`

	// Maximum number of records to read per syscall [e.g., 1024 records per syscall].
	MaxReadBatchSize int `mapstructure:"max_read_batch_size"`

	// Polling interval in milliseconds [e.g., 5-10ms for latency/CPU balance]
	PollInterval time.Duration `mapstructure:"poll_interval"`

	// Max number of goroutines per ring buffer [e.g., 1-2 goroutines per buffer]
	MaxGoroutinesPerBuffer int `mapstructure:"max_goroutines_per_buffer"`
}
