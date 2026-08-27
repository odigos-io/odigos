package config

import "strings"

// Sending queue batch config for OpenTelemetry exporterhelper.
// Replaces the standalone batchprocessor when set on an exporter.
// https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/exporterhelper/README.md#sending-queue

// BatchUnit is how batch size is measured (items or bytes).
// Maps to exporterhelper sending_queue.batch.sizer.
type BatchUnit string

const (
	BatchUnitItems BatchUnit = "items"
	BatchUnitBytes BatchUnit = "bytes"
)

// QueueUnit is how queue size is measured.
// Maps to exporterhelper sending_queue.sizer.
type QueueUnit string

const (
	QueueUnitItems    QueueUnit = "items"
	QueueUnitBytes    QueueUnit = "bytes"
	QueueUnitRequests QueueUnit = "requests"
)

// SendingQueueConfig holds optional overrides for exporterhelper sending_queue.
// Zero values mean OTel defaults (see DefaultSendingQueueConfig).
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

// DefaultSendingQueueConfig enables sending_queue with OTel defaults via batch: {}.
// Empty fields are omitted from collector config; exporterhelper then applies:
//   - enabled: true
//   - sizer: requests, queue_size: 1000
//   - batch.sizer: items, batch.min_size: 8192, batch.max_size: 0, batch.flush_timeout: 200ms
// https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/exporterhelper/README.md#sending-queue
func DefaultSendingQueueConfig() SendingQueueConfig {
	return SendingQueueConfig{}
}

// BuildSendingQueue builds an exporterhelper sending_queue.
// Always sets enabled: true and batch (at least batch: {} so batching is on).
// Zero-value fields omit overrides and keep OTel defaults (see DefaultSendingQueueConfig).
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
		"enabled": true,
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

// ConfigureDestination runs the destination configer, then applies destination-level
// sending_queue to every exporter referenced by the returned pipelines.
// sendingQueueApplied is true when sending_queue was applied to every exporter on every
// pipeline; false means it could not (keep the legacy batch processor).
func ConfigureDestination(configer Configer, dest ExporterConfigurer, cfg *Config) (pipelineNames []string, sendingQueueApplied bool, err error) {
	pipelineNames, err = configer.ModifyConfig(dest, cfg)
	if err != nil {
		return nil, false, err
	}
	sendingQueueApplied = applySendingQueue(dest, cfg, pipelineNames)
	return pipelineNames, sendingQueueApplied, nil
}

// applySendingQueue stamps sending_queue onto supported exporters.
// Returns true when sending_queue was applied to every exporter on every pipeline;
// false if any exporter could not receive it.
func applySendingQueue(dest ExporterConfigurer, cfg *Config, pipelineNames []string) bool {
	q := dest.GetSendingQueueConfig()
	if q == nil {
		return false
	}
	queue := BuildSendingQueue(*q)
	applied := len(pipelineNames) > 0

	for _, pipelineName := range pipelineNames {
		pipeline, ok := cfg.Service.Pipelines[pipelineName]
		if !ok {
			applied = false
			continue
		}
		if len(pipeline.Exporters) == 0 {
			applied = false
			continue
		}
		for _, exporterName := range pipeline.Exporters {
			if !supportsSendingQueue(exporterName) {
				applied = false
				continue
			}
			exporterConfig, ok := cfg.Exporters[exporterName]
			if !ok {
				applied = false
				continue
			}
			switch m := exporterConfig.(type) {
			case GenericMap:
				m["sending_queue"] = queue
			case map[string]interface{}:
				m["sending_queue"] = queue
			default:
				applied = false
			}
		}
	}
	return applied
}

// unsupportedSendingQueuePrefixes are collector exporter type prefixes that lack
// exporterhelper sending_queue (checked against collector v0.151.0) and keep the
// pipeline batch processor until fixed upstream / in our exporters:
//   - nop/, debug/ — no exporterhelper.WithQueue
//   - awsemf/, awsxray/ — no exporterhelper.WithQueue
//   - azureblobstorage/, googlecloudstorage/ — helper without WithQueue
//   - prometheusremotewrite/ — uses remote_write_queue, not sending_queue
//
// TODO(2026-07-27): revisit when collector exporters gain sending_queue support.
var unsupportedSendingQueuePrefixes = []string{
	"nop/",
	"debug/",
	"awsemf/",
	"awsxray/",
	"azureblobstorage/",
	"googlecloudstorage/",
	"prometheusremotewrite/",
}

// supportsSendingQueue reports whether the collector exporter type accepts
// exporterhelper sending_queue.
func supportsSendingQueue(exporterName string) bool {
	for _, prefix := range unsupportedSendingQueuePrefixes {
		if strings.HasPrefix(exporterName, prefix) {
			return false
		}
	}
	return true
}
