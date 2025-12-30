package watchers

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/odigos-io/odigos/frontend/services/sse"
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
	// If true, reset the timer on each new event (debounce mode)
	// If false, send batch when timer expires regardless of new events (batch mode)
	Debounce bool
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

	if eb.config.Debounce && eb.timer != nil {
		// Debounce mode: reset the timer on each new event
		eb.timer.Stop()
		eb.timer = time.AfterFunc(eb.config.Duration, eb.sendBatch)
	} else if eb.timer == nil {
		// Batch mode or first event: start the timer
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
	// Use maps to track unique targets for deduplication
	// This ensures that multiple events for the same source are counted only once
	successTargets := make(map[string]struct{})
	failureTargets := make(map[string]struct{})

	for _, message := range eb.batch {
		// Use target as key for deduplication, fallback to data if target is empty
		key := message.Target
		if key == "" {
			key = message.Data
		}

		if message.Type == sse.MessageTypeSuccess {
			successTargets[key] = struct{}{}
		} else if message.Type == sse.MessageTypeError {
			failureTargets[key] = struct{}{}
		}
	}

	var result []sse.SSEMessage
	if len(successTargets) > 0 {
		result = append(result, sse.SSEMessage{
			Type:    sse.MessageTypeSuccess,
			Event:   eb.config.Event,
			Data:    eb.config.SuccessBatchMessageFunc(len(successTargets), eb.config.CRDType),
			CRDType: eb.config.CRDType,
			Target:  "",
		})
	}

	if len(failureTargets) > 0 {
		result = append(result, sse.SSEMessage{
			Type:    sse.MessageTypeError,
			Event:   eb.config.Event,
			Data:    eb.config.FailureBatchMessageFunc(len(failureTargets), eb.config.CRDType),
			CRDType: eb.config.CRDType,
			Target:  "",
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
