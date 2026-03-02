package odigosebpfreceiver

import (
	"bytes"
	"context"
	"encoding/binary"
	"sync"
	"time"

	"github.com/cilium/ebpf"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"

	rtml "github.com/odigos-io/go-rtml"
)

const (
	logEventMaxLogSize  = 256
	logEventTraceIDSize = 16
	logEventSpanIDSize  = 8
)

// logEvent mirrors the BPF log_event struct layout from ebpf-core/pkg/instrumentors/logs/capture/bpf/probe.bpf.c.
// Fields must match the C struct exactly for binary.Read deserialization.
type logEvent struct {
	Timestamp  uint64
	PID        uint32 // Linux TID (lower 32 bits of pid_tgid)
	TGID       uint32 // Thread Group ID (process ID)
	FD         uint32 // 1=stdout, 2=stderr
	Len        uint32 // Actual data length (<= logEventMaxLogSize)
	Comm       [16]byte
	TraceID    [logEventTraceIDSize]byte
	SpanID     [logEventSpanIDSize]byte
	HasContext uint8
	Pad        [7]byte
	Data       [logEventMaxLogSize]byte
}

func (e *logEvent) logData() string {
	l := e.Len
	if l > logEventMaxLogSize {
		l = logEventMaxLogSize
	}
	return string(e.Data[:l])
}

func (e *logEvent) commString() string {
	for i, b := range e.Comm {
		if b == 0 {
			return string(e.Comm[:i])
		}
	}
	return string(e.Comm[:])
}

func (e *logEvent) streamString() string {
	switch e.FD {
	case 1:
		return "stdout"
	case 2:
		return "stderr"
	default:
		return "unknown"
	}
}

// logsAttrCache is a thread-safe cache of TGID -> packed resource attributes.
// It is built from the attributesMap at startup and refreshed on cache misses
// by doing a direct map lookup.
type logsAttrCache struct {
	mu    sync.RWMutex
	cache map[uint32]string
}

func newLogsAttrCache() *logsAttrCache {
	return &logsAttrCache{cache: make(map[uint32]string)}
}

func (c *logsAttrCache) get(tgid uint32) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.cache[tgid]
	return v, ok
}

func (c *logsAttrCache) set(tgid uint32, attrs string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[tgid] = attrs
}

// buildLogsAttrCache populates the cache by iterating over the entire attributesMap.
func buildLogsAttrCache(attributesMap *ebpf.Map, logger *zap.Logger) *logsAttrCache {
	ac := newLogsAttrCache()

	var tgidKey uint32
	var attrValue [1024]byte

	iter := attributesMap.Iterate()
	for iter.Next(&tgidKey, &attrValue) {
		attrStr := string(bytes.TrimRight(attrValue[:], "\x00"))
		if attrStr != "" {
			ac.cache[tgidKey] = attrStr
		}
	}
	if err := iter.Err(); err != nil {
		logger.Error("logs attributes map iterator error", zap.Error(err))
	}

	logger.Debug("logs attributes cache built", zap.Int("entries", len(ac.cache)))
	return ac
}

// lookupAttrs looks up attributes for a TGID, first from cache, then from the eBPF map on miss.
func lookupAttrs(ac *logsAttrCache, attributesMap *ebpf.Map, tgid uint32) string {
	if attrs, ok := ac.get(tgid); ok {
		return attrs
	}

	// Cache miss — try direct map lookup (new process registered since cache was built)
	var attrValue [1024]byte
	if err := attributesMap.Lookup(tgid, &attrValue); err == nil {
		attrStr := string(bytes.TrimRight(attrValue[:], "\x00"))
		if attrStr != "" {
			ac.set(tgid, attrStr)
			return attrStr
		}
	}

	return ""
}

func (r *ebpfReceiver) logsReadLoop(ctx context.Context, m *ebpf.Map, attributesMap *ebpf.Map) error {
	reader, err := NewBufferReader(m, r.logger)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Build initial TGID -> packed attributes cache from the ext map
	attrCache := buildLogsAttrCache(attributesMap, r.logger)

	var record BufferRecord

	// Close the reader when context is cancelled to unblock ReadInto()
	go func() {
		<-ctx.Done()
		reader.Close()
	}()

	for {
		// Check memory pressure before each read attempt
		for rtml.IsMemLimitReached() {
			delayDuration := 20 * time.Millisecond

			r.telemetry.EbpfMemoryPressureWaitTimeTotal.Add(ctx, delayDuration.Milliseconds())

			r.logger.Debug("memory pressure detected, sleeping", zap.Duration("duration", delayDuration))
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(delayDuration):
			}
		}

		err := reader.ReadInto(&record)
		if err != nil {
			if IsClosedError(err) {
				return nil
			}
			r.logger.Error("error reading from buffer reader", zap.Error(err))
			continue
		}

		if record.LostSamples != 0 {
			r.telemetry.EbpfLostSamples.Add(ctx, int64(record.LostSamples))
			r.logger.Debug("lost samples", zap.Int("lost", int(record.LostSamples)))
			continue
		}

		r.telemetry.EbpfTotalBytesRead.Add(ctx, int64(len(record.RawSample)))

		var event logEvent
		if err := binary.Read(bytes.NewReader(record.RawSample), binary.LittleEndian, &event); err != nil {
			r.logger.Error("error deserializing log event", zap.Error(err))
			continue
		}

		// Look up resource attributes for this process
		packedAttrs := lookupAttrs(attrCache, attributesMap, event.TGID)

		ld := logEventToPdata(&event)

		// Add resource attributes from packed attrs if available
		if packedAttrs != "" && ld.ResourceLogs().Len() > 0 {
			resourceAttrs := ld.ResourceLogs().At(0).Resource().Attributes()
			if err := r.parseResourceAttributes(resourceAttrs, packedAttrs, ",", ":"); err != nil {
				r.logger.Debug("failed to parse log resource attributes", zap.Error(err))
			}
		}

		err = r.nextLogs.ConsumeLogs(ctx, ld)
		if err != nil {
			r.logger.Error("err consuming logs", zap.Error(err))
			continue
		}
	}
}

// logEventToPdata converts a raw BPF log_event to plog.Logs.
func logEventToPdata(event *logEvent) plog.Logs {
	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	lr := sl.LogRecords().AppendEmpty()

	now := pcommon.NewTimestampFromTime(time.Now())
	lr.SetTimestamp(now)
	lr.SetObservedTimestamp(now)
	lr.Body().SetStr(event.logData())
	lr.SetSeverityNumber(plog.SeverityNumberInfo)
	lr.SetSeverityText("INFO")

	attrs := lr.Attributes()
	attrs.PutStr("log.iostream", event.streamString())
	attrs.PutStr("process.command", event.commString())
	attrs.PutInt("process.pid", int64(event.TGID))
	attrs.PutInt("process.tid", int64(event.PID))

	if event.HasContext == 1 {
		lr.SetTraceID(pcommon.TraceID(event.TraceID))
		lr.SetSpanID(pcommon.SpanID(event.SpanID))
	}

	return ld
}
