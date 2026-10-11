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
	timer    *time.Timer // throttle timer: starts on first event, fires after Duration to flush the batch
	maxTimer *time.Timer // hard-deadline timer: starts on first event, fires after MaxDelay regardless of throttle resets
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
	// When the batch reaches this size, flush immediately regardless of timers.
	// 0 means no cap (unlimited accumulation).
	MaxBatchSize int
	// Hard-deadline safety net: maximum wall-clock time from the first event in a
	// batch to its flush, regardless of how often new events arrive. Prevents
	// indefinite accumulation under sustained load. Contrast with Duration, which
	// is the throttle window (restarted per batch, not per event).
	// 0 means no limit.
	MaxDelay time.Duration
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
		Targets: []string{target},
	}

	eb.batch = append(eb.batch, message)

	if eb.config.MaxBatchSize > 0 && len(eb.batch) >= eb.config.MaxBatchSize {
		eb.flushLocked()
		return nil
	}

	if eb.timer == nil {
		eb.timer = time.AfterFunc(eb.config.Duration, eb.sendBatch)
	}

	if eb.maxTimer == nil && eb.config.MaxDelay > 0 {
		eb.maxTimer = time.AfterFunc(eb.config.MaxDelay, eb.sendBatch)
	}

	return nil
}

func (eb *EventBatcher) sendBatch() {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.flushLocked()
}

// flushLocked sends the current batch. Caller must hold eb.mu.
func (eb *EventBatcher) flushLocked() {
	if eb.timer != nil {
		eb.timer.Stop()
		eb.timer = nil
	}
	if eb.maxTimer != nil {
		eb.maxTimer.Stop()
		eb.maxTimer = nil
	}

	if len(eb.batch) == 0 {
		eb.batch = nil
		return
	}

	if len(eb.batch) < eb.config.MinBatchSize {
		for _, message := range eb.batch {
			sse.SendMessageToClient(message)
		}
	} else {
		batchMessages := eb.prepareBatchMessage()
		for _, batch := range batchMessages {
			sse.SendMessageToClient(batch)
		}
	}

	eb.batch = nil
}

func (eb *EventBatcher) prepareBatchMessage() []sse.SSEMessage {
	successTargets := make(map[string]struct{})
	failureTargets := make(map[string]struct{})

	for _, message := range eb.batch {
		key := message.Data
		if len(message.Targets) > 0 {
			key = message.Targets[0]
		}

		if message.Type == sse.MessageTypeSuccess {
			successTargets[key] = struct{}{}
		} else if message.Type == sse.MessageTypeError {
			failureTargets[key] = struct{}{}
		}
	}

	var result []sse.SSEMessage
	if len(successTargets) > 0 {
		targets := make([]string, 0, len(successTargets))
		for t := range successTargets {
			targets = append(targets, t)
		}
		result = append(result, sse.SSEMessage{
			Type:    sse.MessageTypeSuccess,
			Event:   eb.config.Event,
			Data:    eb.config.SuccessBatchMessageFunc(len(successTargets), eb.config.CRDType),
			CRDType: eb.config.CRDType,
			Targets: targets,
		})
	}

	if len(failureTargets) > 0 {
		targets := make([]string, 0, len(failureTargets))
		for t := range failureTargets {
			targets = append(targets, t)
		}
		result = append(result, sse.SSEMessage{
			Type:    sse.MessageTypeError,
			Event:   eb.config.Event,
			Data:    eb.config.FailureBatchMessageFunc(len(failureTargets), eb.config.CRDType),
			CRDType: eb.config.CRDType,
			Targets: targets,
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
		if eb.maxTimer != nil {
			eb.maxTimer.Stop()
			eb.maxTimer = nil
		}
		eb.batch = nil
	})
}
