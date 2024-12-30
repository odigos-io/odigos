package watchers

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/odigos-io/odigos/frontend/endpoints/sse"
)

var (
	ErrUnmatchedMsgType            = errors.New("message type does not match the one configured for the batcher")
	DefaultBatchDuration           = 2 * time.Second
	DefaultMinBatchSize            = 4
	DefaultFailureBatchMessageFunc = func(batchSize int, crd string) string {
		return fmt.Sprintf("%s: %d failed", crd, batchSize)
	}
	DefaultSuccessBatchMessageFunc = func(batchSize int, crd string) string {
		return fmt.Sprintf("%s: %d successful", crd, batchSize)
	}
)

type EventBatcher struct {
	mu       sync.Mutex
	batch    []sse.SSEMessage
	timer    *time.Timer
	stopped  atomic.Bool
	config   EventBatcherConfig
	stopOnce sync.Once
}

type EventBatcherConfig struct {
	// Event to batch, not configuring this value (empty string) will cause all events to be batched
	Event sse.MessageEvent
	// Message type to batch, not configuring this value (empty string) will cause all messages to be batched
	MessageType sse.MessageType
	// Time to wait before sending a batch of notifications
	Duration time.Duration
	// Minimum number of notifications to batch, if the batch size is less than this, the messages will be sent individually
	MinBatchSize int
	// CRD type to batch, TODO: should we allow this to be empty?
	CRDType string
	// Function to generate a message for a batch of failed messages
	FailureBatchMessageFunc func(batchSize int, crd string) string
	// Function to generate a message for a batch of successful messages
	SuccessBatchMessageFunc func(batchSize int, crd string) string
}

func NewEventBatcher(config EventBatcherConfig) *EventBatcher {
	if config.Duration == 0 {
		config.Duration = DefaultBatchDuration
	}

	if config.MinBatchSize == 0 {
		config.MinBatchSize = DefaultMinBatchSize
	}

	if config.FailureBatchMessageFunc == nil {
		config.FailureBatchMessageFunc = DefaultFailureBatchMessageFunc
	}

	if config.SuccessBatchMessageFunc == nil {
		config.SuccessBatchMessageFunc = DefaultSuccessBatchMessageFunc
	}

	return &EventBatcher{
		config: config,
	}
}

func (eb *EventBatcher) AddEvent(msgType sse.MessageType, data, target string) error {
	if eb.stopped.Load() {
		return nil
	}

	if eb.config.MessageType != "" && msgType != eb.config.MessageType {
		return ErrUnmatchedMsgType
	}

	eb.mu.Lock()
	defer eb.mu.Unlock()

	message := sse.SSEMessage{
		Type:    msgType,
		Event:   eb.config.Event,
		Data:    data,
		CRDType: eb.config.CRDType,
		Target:  target,
	}

	eb.batch = append(eb.batch, message)

	if eb.timer == nil {
		// A new batch timer is started once the first message is added
		eb.timer = time.AfterFunc(eb.config.Duration, eb.sendBatch)
	}

	return nil
}

func (eb *EventBatcher) sendBatch() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if len(eb.batch) == 0 {
		eb.batch = nil
		eb.timer = nil
		return
	}

	if len(eb.batch) < eb.config.MinBatchSize {
		for _, message := range eb.batch {
			sse.SendMessageToClient(message)
		}
	} else {
		// currently we are grouping batches by success and error messages
		batchMessages := eb.prepareBatchMessage()
		for _, batch := range batchMessages {
			sse.SendMessageToClient(batch)
		}
	}

	eb.batch = nil
	eb.timer = nil
}

func (eb *EventBatcher) prepareBatchMessage() []sse.SSEMessage {
	successCount := 0
	failureCount := 0

	for _, message := range eb.batch {
		if message.Type == sse.MessageTypeSuccess {
			successCount++
		} else if message.Type == sse.MessageTypeError {
			failureCount++
		}
	}

	var result []sse.SSEMessage
	if successCount > 0 {
		result = append(result, sse.SSEMessage{
			Event:   eb.config.Event,
			Type:    sse.MessageTypeSuccess,
			Target:  "",
			Data:    eb.config.SuccessBatchMessageFunc(successCount, eb.config.CRDType),
			CRDType: eb.config.CRDType,
		})
	}

	if failureCount > 0 {
		result = append(result, sse.SSEMessage{
			Event:   eb.config.Event,
			Type:    sse.MessageTypeError,
			Target:  "",
			Data:    eb.config.FailureBatchMessageFunc(failureCount, eb.config.CRDType),
			CRDType: eb.config.CRDType,
		})
	}
	return result
}

func (eb *EventBatcher) Cancel() {
	eb.stopOnce.Do(func() {
		eb.stopped.Store(true)
		if eb.timer != nil {
			eb.timer.Stop()
			eb.timer = nil
		}
		eb.batch = nil
	})
}
