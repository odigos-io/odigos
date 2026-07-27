package config

// Sending queue batch config for OpenTelemetry exporterhelper.
// Replaces the standalone batchprocessor when set on an exporter.
// https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/exporterhelper/README.md#sending-queue

// BatchUnit is how batch size is measured (items or bytes).
// Maps to exporterhelper sending_queue.batch.sizer.
type BatchUnit string

const (
	BatchUnitItems BatchUnit = "items"
	BatchUnitBytes BatchUnit = "bytes"

	defaultQueueSize = 1000
	defaultBatchSize = 8192

	// Same as the OpenTelemetry batchprocessor default.
	defaultBatchFlushTimeout = "200ms"
)

type QueueUnit string

const (
	QueueUnitItems    QueueUnit = "items"
	QueueUnitBytes    QueueUnit = "bytes"
	QueueUnitRequests QueueUnit = "requests"
)

type SendingQueueConfig struct {
	Unit  QueueUnit
	Size  int
	Batch BatchConfig
}

type BatchConfig struct {
	Unit BatchUnit
	Size int
}

// DefaultSendingQueueConfig is the destination-level default:
// queue = 1000 requests, batch = 8192 items.
func DefaultSendingQueueConfig() SendingQueueConfig {
	return SendingQueueConfig{
		Unit: QueueUnitRequests,
		Size: defaultQueueSize,
		Batch: BatchConfig{
			Unit: BatchUnitItems,
			Size: defaultBatchSize,
		},
	}
}

// BuildSendingQueue builds an exporterhelper sending_queue with batch enabled.
func BuildSendingQueue(queue SendingQueueConfig) GenericMap {
	return GenericMap{
		"sizer":      string(queue.Unit),
		"queue_size": queue.Size,
		"batch": GenericMap{
			"sizer":         string(queue.Batch.Unit),
			"min_size":      queue.Batch.Size,
			"flush_timeout": defaultBatchFlushTimeout,
		},
	}
}
