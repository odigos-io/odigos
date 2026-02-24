package odigosebpfreceiver

import (
	"bytes"
	"context"
	"encoding/binary"
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

func (r *ebpfReceiver) logsReadLoop(ctx context.Context, m *ebpf.Map) error {
	reader, err := NewBufferReader(m, r.logger)
	if err != nil {
		return err
	}
	defer reader.Close()

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

		ld := logEventToPdata(&event)

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
