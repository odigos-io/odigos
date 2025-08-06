package odigosebpfreceiver

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/perf"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/pdata/ptrace"
)

const numOfPages = 2048

const tracesMapPath = "/sys/fs/bpf/odigos/traces_map"

type ebpfReceiver struct {
	config *Config
	cancel context.CancelFunc
	logger *zap.Logger

	// Consumers
	nextTraces  consumer.Traces
	nextMetrics consumer.Metrics
	nextLogs    consumer.Logs
}

func (r *ebpfReceiver) Start(ctx context.Context, host component.Host) error {
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	r.logger.Info("odigos-ebpf: trace receiver active, listening on ringbuffer")

	go func() {
		if err := r.readLoop(ctx); err != nil {
			r.logger.Error("read loop failed", zap.String("component", TypeStr), zap.Error(err))
		}
	}()
	return nil
}

func (r *ebpfReceiver) Shutdown(ctx context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}
	return nil
}

func (r *ebpfReceiver) readLoop(ctx context.Context) error {
	m, err := ebpf.LoadPinnedMap(tracesMapPath, nil)
	if err != nil {
		r.logger.Error("failed to load pinned map", zap.Error(err))
		return err
	}
	defer m.Close()

	reader, err := perf.NewReader(m, numOfPages*os.Getpagesize())
	if err != nil {
		r.logger.Error("failed to open perf reader", zap.Error(err))
		return err
	}
	defer reader.Close()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			record, err := reader.Read()
			if err != nil {
				if errors.Is(err, perf.ErrClosed) {
					return nil
				}
				r.logger.Error("error reading from perf reader", zap.Error(err))
				continue
			}

			if record.LostSamples != 0 {
				// later we can consider to expose this as a metric
				r.logger.Error("lost samples", zap.Int("lost", int(record.LostSamples)))
				continue
			}

			if len(record.RawSample) < 8 {
				continue
			}

			// The first 8 bytes of the record contain the length of the span encoded as a uint64
			acceptedLength := binary.NativeEndian.Uint64(record.RawSample[:8])

			if len(record.RawSample) < (8 + int(acceptedLength)) {
				continue
			}

			protoUnmarshaler := ptrace.ProtoUnmarshaler{}

			td, err := protoUnmarshaler.UnmarshalTraces(record.RawSample[8 : 8+acceptedLength])
			if err != nil {
				r.logger.Error("err unmarshling traces", zap.Error(err))
				// if we fail to unmarshal to Traces, it can happen because the agent running old version. we are trying the default fallback
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
}

// convertResourceSpansToPdata converts a single ResourceSpans to pdata Traces
func convertResourceSpansToPdata(resourceSpans *tracepb.ResourceSpans) ptrace.Traces {
	// Wrap single ResourceSpans in TracesData
	tracesData := &tracepb.TracesData{
		ResourceSpans: []*tracepb.ResourceSpans{resourceSpans},
	}

	// Marshal to bytes
	data, err := proto.Marshal(tracesData)
	if err != nil {
		fmt.Println("Marshal tracesData")
		return ptrace.NewTraces()
	}

	// Convert to pdata
	unmarshaler := &ptrace.ProtoUnmarshaler{}
	traces, err := unmarshaler.UnmarshalTraces(data)
	if err != nil {
		fmt.Println("unmashling traces", err)
		return ptrace.NewTraces()
	}

	return traces
}
