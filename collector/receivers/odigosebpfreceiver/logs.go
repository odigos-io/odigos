package odigosebpfreceiver

import (
	"context"
	"sync"
	"time"
	"unsafe"

	"github.com/cilium/ebpf"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"

	rtml "github.com/odigos-io/go-rtml"
)

const (
	logEventMaxLogSize  = 4096
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
		// unreachable: the eBPF probe only emits events with FD 1 (stdout) or 2 (stderr)
		return ""
	}
}

// logsAttrCache is a thread-safe TGID → packed resource attributes cache.
type logsAttrCache struct {
	mu    sync.RWMutex
	cache map[uint32]string
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

func (c *logsAttrCache) delete(tgid uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, tgid)
}

func (c *logsAttrCache) size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

func (r *ebpfReceiver) logsReadLoop(ctx context.Context, m *ebpf.Map, attrCache *logsAttrCache) error {
	reader, err := NewBufferReader(m, r.logger)
	if err != nil {
		return err
	}
	defer reader.Close()

	r.logger.Debug("logs read loop started, waiting for eBPF log events",
		zap.Int("ringBuf_fd", m.FD()))

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

		r.telemetry.EbpfTotalBytesRead.Add(ctx, int64(len(record.RawSample)))

		// The BPF ring buffer sends variable-size events: the fixed header
		// (everything up to Data) plus only the actual log bytes.
		// Minimum valid size is the header (offset of Data field).
		headerSize := int(unsafe.Offsetof(logEvent{}.Data))
		if len(record.RawSample) < headerSize {
			r.telemetry.EbpfLostSamples.Add(ctx, 1)
			r.logger.Error("short sample", zap.Int("size", len(record.RawSample)), zap.Int("headerSize", headerSize))
			continue
		}
		// Pad the sample to full struct size so unsafe cast is safe.
		if len(record.RawSample) < int(unsafe.Sizeof(logEvent{})) {
			padded := make([]byte, unsafe.Sizeof(logEvent{}))
			copy(padded, record.RawSample)
			record.RawSample = padded
		}
		event := (*logEvent)(unsafe.Pointer(&record.RawSample[0]))

		// Look up resource attributes for this process
		packedAttrs, _ := attrCache.get(event.TGID)
		r.telemetry.EbpfLogsAttrCacheSize.Record(ctx, int64(attrCache.size()))

		ld := logEventToPdata(event)

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

	// bpf_ktime_get_ns() returns boot-time nanoseconds (monotonic clock),
	// not wall-clock time. Use time.Now() for both timestamps since we
	// are processing events as they arrive.
	now := pcommon.NewTimestampFromTime(time.Now())
	lr.SetTimestamp(now)
	lr.SetObservedTimestamp(now)
	lr.Body().SetStr(event.logData())
	lr.SetSeverityNumber(plog.SeverityNumberInfo)
	lr.SetSeverityText("INFO")

	attrs := lr.Attributes()
	attrs.PutStr(string(semconv.LogIostreamKey), event.streamString())
	attrs.PutStr(string(semconv.ProcessCommandKey), event.commString())
	attrs.PutInt(string(semconv.ProcessPIDKey), int64(event.TGID))
	attrs.PutInt(string(semconv.ThreadIDKey), int64(event.PID))

	if event.HasContext == 1 {
		lr.SetTraceID(pcommon.TraceID(event.TraceID))
		lr.SetSpanID(pcommon.SpanID(event.SpanID))
	}

	return ld
}
