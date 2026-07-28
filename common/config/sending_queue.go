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

	// OpenTelemetry exporterhelper defaults when sending_queue is enabled with batch: {}.
	defaultQueueEnabled      = true
	defaultQueueSize         = 1000
	defaultBatchMinSize      = 8192
	defaultBatchMaxSize      = 0
	defaultBatchFlushTimeout = "200ms"
	defaultBatchSizer        = BatchUnitItems
)

type QueueUnit string

const (
	QueueUnitItems    QueueUnit = "items"
	QueueUnitBytes    QueueUnit = "bytes"
	QueueUnitRequests QueueUnit = "requests"

	defaultQueueSizer = QueueUnitRequests
)

// SendingQueueConfig holds optional overrides for exporterhelper sending_queue.
// Zero values mean OTel defaults: queue_size 1000 requests, batch min_size 8192 items.
type SendingQueueConfig struct {
	Unit  QueueUnit
	Size  int
	Batch BatchConfig
}

type BatchConfig struct {
	Unit         BatchUnit
	Size         int
	MaxSize      *int // nil = omit; 0 = explicit no max
	FlushTimeout string
}

// DefaultSendingQueueConfig enables sending_queue with OTel defaults via batch: {}
// (queue_size 1000 requests, batch min_size 8192 items, flush_timeout 200ms).
func DefaultSendingQueueConfig() SendingQueueConfig {
	return SendingQueueConfig{}
}

// BuildSendingQueue builds an exporterhelper sending_queue.
// Always sets enabled: true and batch (at least batch: {} so batching is on).
// Zero-value fields omit overrides and keep OTel defaults (1000 requests / 8192 items / 200ms).
func BuildSendingQueue(queue SendingQueueConfig) GenericMap {
	batch := GenericMap{}
	if queue.Batch.Unit != "" {
		batch["sizer"] = string(queue.Batch.Unit)
	}
	if queue.Batch.Size > 0 {
		batch["min_size"] = queue.Batch.Size
	}
	if queue.Batch.MaxSize != nil {
		batch["max_size"] = *queue.Batch.MaxSize
	}
	if queue.Batch.FlushTimeout != "" {
		batch["flush_timeout"] = queue.Batch.FlushTimeout
	}

	sendingQueue := GenericMap{
		"enabled": defaultQueueEnabled,
		"batch":   batch,
	}
	if queue.Unit != "" {
		sendingQueue["sizer"] = string(queue.Unit)
	}
	if queue.Size > 0 {
		sendingQueue["queue_size"] = queue.Size
	}
	return sendingQueue
}
