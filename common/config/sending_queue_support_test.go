package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
)

type sendingQueueTestDest struct {
	queue *SendingQueueConfig
}

func (d *sendingQueueTestDest) GetID() string                              { return "d1" }
func (d *sendingQueueTestDest) GetType() common.DestinationType            { return "" }
func (d *sendingQueueTestDest) GetConfig() map[string]string               { return nil }
func (d *sendingQueueTestDest) GetSendingQueueConfig() *SendingQueueConfig { return d.queue }
func (d *sendingQueueTestDest) GetSignals() []common.ObservabilitySignal   { return nil }

// TestUnsupportedSendingQueuePrefixesOmitQueue drives applySendingQueue with each
// prefix from unsupportedSendingQueuePrefixes (prod list) and asserts queue is skipped.
func TestUnsupportedSendingQueuePrefixesOmitQueue(t *testing.T) {
	t.Parallel()

	queue := DefaultSendingQueueConfig()
	dest := &sendingQueueTestDest{queue: &queue}

	for _, prefix := range unsupportedSendingQueuePrefixes {
		prefix := prefix
		t.Run(prefix, func(t *testing.T) {
			t.Parallel()

			assert.False(t, supportsSendingQueue(prefix+"d1"))

			exporterName := prefix + "d1"
			cfg := &Config{
				Exporters: GenericMap{exporterName: GenericMap{}},
				Service: Service{
					Pipelines: map[string]Pipeline{
						"p": {Exporters: []string{exporterName}},
					},
				},
			}

			applySendingQueue(dest, cfg, []string{"p"})

			_, hasQueue := cfg.Exporters[exporterName].(GenericMap)["sending_queue"]
			assert.False(t, hasQueue, "exporter %s must not receive sending_queue", exporterName)
		})
	}
}
