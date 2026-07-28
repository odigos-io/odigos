package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
)

type applySendingQueueDest struct {
	queue *SendingQueueConfig
}

func (d *applySendingQueueDest) GetID() string                              { return "d1" }
func (d *applySendingQueueDest) GetType() common.DestinationType            { return "" }
func (d *applySendingQueueDest) GetConfig() map[string]string               { return nil }
func (d *applySendingQueueDest) GetSendingQueueConfig() *SendingQueueConfig { return d.queue }
func (d *applySendingQueueDest) GetSignals() []common.ObservabilitySignal   { return nil }

func TestBuildSendingQueue_Defaults(t *testing.T) {
	t.Parallel()

	got := BuildSendingQueue(DefaultSendingQueueConfig())
	assert.Equal(t, GenericMap{
		"enabled": true,
		"batch":   GenericMap{},
	}, got)
}

func TestBuildSendingQueue_ExplicitOverrides(t *testing.T) {
	t.Parallel()

	maxSize := 4096
	got := BuildSendingQueue(SendingQueueConfig{
		Unit: QueueUnitItems,
		Size: 500,
		Batch: BatchConfig{
			Unit:         BatchUnitBytes,
			Size:         1024,
			MaxSize:      &maxSize,
			FlushTimeout: "1s",
		},
	})
	assert.Equal(t, GenericMap{
		"enabled":    true,
		"sizer":      string(QueueUnitItems),
		"queue_size": 500,
		"batch": GenericMap{
			"sizer":         string(BatchUnitBytes),
			"min_size":      1024,
			"max_size":      4096,
			"flush_timeout": "1s",
		},
	}, got)
}

func TestApplySendingQueue(t *testing.T) {
	t.Parallel()

	queue := DefaultSendingQueueConfig()
	dest := &applySendingQueueDest{queue: &queue}

	t.Run("supported", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{
			Exporters: GenericMap{"otlp_grpc/d1": GenericMap{}},
			Service:   Service{Pipelines: map[string]Pipeline{"p": {Exporters: []string{"otlp_grpc/d1"}}}},
		}
		assert.True(t, applySendingQueue(dest, cfg, []string{"p"}))
		_, hasQueue := cfg.Exporters["otlp_grpc/d1"].(GenericMap)["sending_queue"]
		assert.True(t, hasQueue)
	})

	t.Run("unsupported prefixes", func(t *testing.T) {
		t.Parallel()
		for _, prefix := range unsupportedSendingQueuePrefixes {
			prefix := prefix
			t.Run(prefix, func(t *testing.T) {
				t.Parallel()
				assert.False(t, supportsSendingQueue(prefix+"d1"))

				exporterName := prefix + "d1"
				cfg := &Config{
					Exporters: GenericMap{exporterName: GenericMap{}},
					Service:   Service{Pipelines: map[string]Pipeline{"p": {Exporters: []string{exporterName}}}},
				}
				assert.False(t, applySendingQueue(dest, cfg, []string{"p"}))
				_, hasQueue := cfg.Exporters[exporterName].(GenericMap)["sending_queue"]
				assert.False(t, hasQueue, "exporter %s must not receive sending_queue", exporterName)
			})
		}
	})

	t.Run("mixed", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{
			Exporters: GenericMap{
				"otlp_grpc/d1": GenericMap{},
				"awsemf/d1":    GenericMap{},
			},
			Service: Service{Pipelines: map[string]Pipeline{
				"p": {Exporters: []string{"otlp_grpc/d1", "awsemf/d1"}},
			}},
		}
		assert.False(t, applySendingQueue(dest, cfg, []string{"p"}))
		_, hasQueue := cfg.Exporters["otlp_grpc/d1"].(GenericMap)["sending_queue"]
		assert.True(t, hasQueue)
		_, hasQueue = cfg.Exporters["awsemf/d1"].(GenericMap)["sending_queue"]
		assert.False(t, hasQueue)
	})
}
