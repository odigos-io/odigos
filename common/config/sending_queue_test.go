package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSendingQueue_Defaults(t *testing.T) {
	t.Parallel()

	got := BuildSendingQueue(DefaultSendingQueueConfig())
	assert.Equal(t, GenericMap{
		"enabled": defaultQueueEnabled,
		"batch":   GenericMap{},
	}, got)
}

func TestSupportsSendingQueue(t *testing.T) {
	t.Parallel()

	assert.True(t, supportsSendingQueue("datadog/d1"))
	assert.True(t, supportsSendingQueue("otlp_grpc/jaeger-d1"))
	for _, prefix := range unsupportedSendingQueuePrefixes {
		assert.False(t, supportsSendingQueue(prefix+"d1"), prefix)
	}
}

func TestApplySendingQueue(t *testing.T) {
	t.Parallel()

	queue := DefaultSendingQueueConfig()
	dest := &sendingQueueTestDest{queue: &queue}

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

	t.Run("unsupported", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{
			Exporters: GenericMap{"awsemf/d1": GenericMap{}},
			Service:   Service{Pipelines: map[string]Pipeline{"p": {Exporters: []string{"awsemf/d1"}}}},
		}
		assert.False(t, applySendingQueue(dest, cfg, []string{"p"}))
		_, hasQueue := cfg.Exporters["awsemf/d1"].(GenericMap)["sending_queue"]
		assert.False(t, hasQueue)
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

func TestBuildSendingQueue_ExplicitDefaults(t *testing.T) {
	t.Parallel()

	maxSize := defaultBatchMaxSize
	// Same numbers OTel applies when batch: {} is set.
	got := BuildSendingQueue(SendingQueueConfig{
		Unit: defaultQueueSizer,
		Size: defaultQueueSize,
		Batch: BatchConfig{
			Unit:         defaultBatchSizer,
			Size:         defaultBatchMinSize,
			MaxSize:      &maxSize,
			FlushTimeout: defaultBatchFlushTimeout,
		},
	})
	assert.Equal(t, GenericMap{
		"enabled":    defaultQueueEnabled,
		"sizer":      string(defaultQueueSizer),
		"queue_size": defaultQueueSize,
		"batch": GenericMap{
			"sizer":         string(defaultBatchSizer),
			"min_size":      defaultBatchMinSize,
			"max_size":      defaultBatchMaxSize,
			"flush_timeout": defaultBatchFlushTimeout,
		},
	}, got)
}
