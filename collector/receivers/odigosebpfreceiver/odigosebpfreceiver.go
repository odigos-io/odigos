package odigosebpfreceiver

import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/perf"
	"github.com/odigos-io/odigos/common/unixfd"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	numOfPages      = 2048
	PollingInterval = 10 * time.Second
)

type ebpfReceiver struct {
	config  *Config
	cancel  context.CancelFunc
	logger  *zap.Logger
	mapPath string

	// Consumers
	nextTraces  consumer.Traces
	nextMetrics consumer.Metrics
	nextLogs    consumer.Logs

	// WaitGroup to wait for all goroutines to finish
	wg sync.WaitGroup
}

func (r *ebpfReceiver) Start(ctx context.Context, host component.Host) error {
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	updates := make(chan *ebpf.Map)

	// Supervisor goroutine: manage readers
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		var (
			prevMap    *ebpf.Map
			readersWg  sync.WaitGroup
			cancelPrev context.CancelFunc
		)
		for {
			select {
			case <-ctx.Done():
				if cancelPrev != nil {
					cancelPrev()
					readersWg.Wait()
				}
				if prevMap != nil {
					prevMap.Close()
				}
				return
			case newMap := <-updates:
				if cancelPrev != nil {
					cancelPrev()
					readersWg.Wait()
					prevMap.Close()
				}
				readersCtx, readersCancel := context.WithCancel(ctx)
				cancelPrev = readersCancel
				prevMap = newMap
				for i := 0; i < 1; i++ {
					readersWg.Add(1)
					go func(id int, m *ebpf.Map) {
						defer readersWg.Done()
						_ = r.readLoop(readersCtx, m)
					}(i, newMap)
				}
			}
		}
	}()

	// Client goroutine: connect once and listen for FD updates
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		for {
			err := unixfd.ConnectAndListen(unixfd.DefaultSocketPath, func(fd int, msg string) {
				newMap, err := ebpf.NewMapFromFD(fd)
				if err == nil {
					updates <- newMap
				}
			})
			if err != nil {
				time.Sleep(2 * time.Second) // retry after odiglet restarts
				continue
			}
			<-ctx.Done()
			return
		}
	}()

	return nil
}

func (r *ebpfReceiver) Shutdown(ctx context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}
	done := make(chan struct{})

	// WaitGroup wait in a goroutine to support context cancellation
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		r.logger.Warn("odigos-ebpf: receiver shutdown did not finish before context was canceled",
			zap.String("mapPath", r.mapPath))
		return ctx.Err()
	case <-done:
		r.logger.Info("odigos-ebpf: receiver shutdown complete", zap.String("mapPath", r.mapPath))
		return nil // all goroutines exited gracefully
	}
}

func (r *ebpfReceiver) readLoop(ctx context.Context, m *ebpf.Map) error {
	reader, err := perf.NewReader(m, numOfPages*os.Getpagesize())
	if err != nil {
		r.logger.Error("failed to open perf reader", zap.Error(err))
		return err
	}
	defer reader.Close()

	var record perf.Record

	go func() {
		<-ctx.Done()
		// This will unblock the blocking call to reader.ReadInto()
		reader.Close()
	}()

	for {
		// This blocks until data is available or reader is closed
		err := reader.ReadInto(&record)
		if err != nil {
			if errors.Is(err, perf.ErrClosed) {
				// Closed due to shutdown signal — exit gracefully
				return nil
			}
			r.logger.Error("error reading from perf reader", zap.Error(err))
			continue
		}

		if record.LostSamples != 0 {
			r.logger.Error("lost samples", zap.Int("lost", int(record.LostSamples)))
			continue
		}

		if len(record.RawSample) < 8 {
			continue
		}

		acceptedLength := binary.NativeEndian.Uint64(record.RawSample[:8])
		if len(record.RawSample) < (8 + int(acceptedLength)) {
			continue // incomplete record
		}

		// Attempt to unmarshal into OpenTelemetry Traces format
		protoUnmarshaler := ptrace.ProtoUnmarshaler{}
		td, err := protoUnmarshaler.UnmarshalTraces(record.RawSample[8 : 8+acceptedLength])
		if err != nil {
			// Fallback: try unmarshalling legacy trace format
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

// convertResourceSpansToPdata converts a single ResourceSpans to pdata Traces
// This is here to support the old version of the agent that doesn't support the new format.
// TODO: remove this once we are sure that all the agents are updated to the new format.
func convertResourceSpansToPdata(resourceSpans *tracepb.ResourceSpans) ptrace.Traces {
	// Wrap single ResourceSpans in TracesData
	tracesData := &tracepb.TracesData{
		ResourceSpans: []*tracepb.ResourceSpans{resourceSpans},
	}

	// Marshal to bytes
	data, err := proto.Marshal(tracesData)
	if err != nil {
		return ptrace.NewTraces()
	}

	// Convert to pdata
	unmarshaler := &ptrace.ProtoUnmarshaler{}
	traces, err := unmarshaler.UnmarshalTraces(data)
	if err != nil {
		return ptrace.NewTraces()
	}

	return traces
}
