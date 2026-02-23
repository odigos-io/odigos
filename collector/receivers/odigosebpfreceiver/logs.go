package odigosebpfreceiver

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/cilium/ebpf"
	"go.opentelemetry.io/collector/pdata/plog"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	rtml "github.com/odigos-io/go-rtml"
)

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

	protoUnmarshaler := plog.ProtoUnmarshaler{}

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

		if len(record.RawSample) < 8 {
			continue
		}

		acceptedLength := binary.NativeEndian.Uint64(record.RawSample[:8])
		if len(record.RawSample) < (8 + int(acceptedLength)) {
			continue
		}

		r.telemetry.EbpfTotalBytesRead.Add(ctx, int64(len(record.RawSample)))

		// Try to unmarshal as current OpenTelemetry format first
		ld, err := protoUnmarshaler.UnmarshalLogs(record.RawSample[8 : 8+acceptedLength])
		if err != nil {
			// Fall back to legacy format for backward compatibility
			var resourceLogs logspb.ResourceLogs
			err = proto.Unmarshal(record.RawSample[8:8+acceptedLength], &resourceLogs)
			if err != nil {
				r.logger.Error("error unmarshalling log record", zap.Error(err))
				continue
			}
			ld = convertResourceLogsToPdata(&resourceLogs)
		}

		err = r.nextLogs.ConsumeLogs(ctx, ld)
		if err != nil {
			r.logger.Error("err consuming logs", zap.Error(err))
			continue
		}
	}
}

// convertResourceLogsToPdata converts a single ResourceLogs to pdata Logs.
// This function exists to support older agents that send data in the legacy format.
func convertResourceLogsToPdata(resourceLogs *logspb.ResourceLogs) plog.Logs {
	logsData := &logspb.LogsData{
		ResourceLogs: []*logspb.ResourceLogs{resourceLogs},
	}

	data, err := proto.Marshal(logsData)
	if err != nil {
		return plog.NewLogs()
	}

	unmarshaler := &plog.ProtoUnmarshaler{}
	logs, err := unmarshaler.UnmarshalLogs(data)
	if err != nil {
		return plog.NewLogs()
	}

	return logs
}
