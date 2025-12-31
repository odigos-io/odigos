package odigosebpfreceiver

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/cilium/ebpf"
	"go.opentelemetry.io/collector/pdata/ptrace"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	rtml "github.com/odigos-io/go-rtml"
)

func (r *ebpfReceiver) tracesReadLoop(ctx context.Context, m *ebpf.Map) error {
	reader, err := NewBufferReader(m, r.logger)
	if err != nil {
		r.logger.Error("failed to open buffer reader", zap.Error(err))
		return err
	}
	defer reader.Close()

	var record BufferRecord

	// Close the reader when context is cancelled to unblock ReadInto()
	go func() {
		<-ctx.Done()
		reader.Close()
	}()

	// Create a proto unmarshaler for the current OpenTelemetry format
	protoUnmarshaler := ptrace.ProtoUnmarshaler{}

	for {
		// Check memory pressure before each read attempt
		for rtml.IsMemLimitReached() {
			delayDuration := 20 * time.Millisecond

			// Track total wait time
			r.telemetry.EbpfMemoryPressureWaitTimeTotal.Add(ctx, delayDuration.Milliseconds())

			r.logger.Debug("memory pressure detected, sleeping", zap.Duration("duration", delayDuration))
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(delayDuration):
				// Continue checking memory pressure
			}
		}

		// Only proceed to read when memory pressure is low
		err := reader.ReadInto(&record)
		if err != nil {
			if IsClosedError(err) {
				return nil
			}
			r.logger.Error("error reading from buffer reader", zap.Error(err))
			continue
		}

		if record.LostSamples != 0 {
			// Record the lost samples metric
			r.telemetry.EbpfLostSamples.Add(ctx, int64(record.LostSamples))
			// Keep the log for debugging, but at debug level
			r.logger.Debug("lost samples", zap.Int("lost", int(record.LostSamples)))
			continue
		}

		if len(record.RawSample) < 8 {
			continue
		}

		acceptedLength := binary.NativeEndian.Uint64(record.RawSample[:8])
		if len(record.RawSample) < (8 + int(acceptedLength)) {
			continue
		}

		r.telemetry.EbpfTotalBytesRead.Add(ctx, int64(len(record.RawSample)))

		// Try to unmarshal as current OpenTelemetry format first
		td, err := protoUnmarshaler.UnmarshalTraces(record.RawSample[8 : 8+acceptedLength])
		if err != nil {
			// Fall back to legacy format for backward compatibility
			var span tracepb.ResourceSpans
			err = proto.Unmarshal(record.RawSample[8:8+acceptedLength], &span)
			if err != nil {
				r.logger.Error("error unmarshalling span", zap.Error(err))
				continue
			}
			td = convertResourceSpansToPdata(&span)
		}

		err = r.nextTraces.ConsumeTraces(ctx, td)
		if err != nil {
			r.logger.Error("err consuming traces", zap.Error(err))
			continue
		}
	}
}

// convertResourceSpansToPdata converts a single ResourceSpans to pdata Traces.
// This function exists to support older agents that send data in the legacy format.
// TODO: remove this once all agents are updated to use the current format.
func convertResourceSpansToPdata(resourceSpans *tracepb.ResourceSpans) ptrace.Traces {
	tracesData := &tracepb.TracesData{
		ResourceSpans: []*tracepb.ResourceSpans{resourceSpans},
	}

	data, err := proto.Marshal(tracesData)
	if err != nil {
		return ptrace.NewTraces()
	}

	unmarshaler := &ptrace.ProtoUnmarshaler{}
	traces, err := unmarshaler.UnmarshalTraces(data)
	if err != nil {
		return ptrace.NewTraces()
	}

	return traces
}
