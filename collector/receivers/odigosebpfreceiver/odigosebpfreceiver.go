package odigosebpfreceiver

import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"sync"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/perf"
	"github.com/odigos-io/odigos/common/unixfd"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	numOfPages = 2048
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

	updates := make(chan *ebpf.Map, 1)

	// Map manager: handles switching to new maps
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()

		var (
			currentMap   *ebpf.Map
			readerCancel context.CancelFunc
		)

		// Cleanup on exit
		defer func() {
			if readerCancel != nil {
				readerCancel()
			}
			if currentMap != nil {
				currentMap.Close()
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case newMap := <-updates:
				// Stop old reader if running
				if readerCancel != nil {
					readerCancel()
				}
				if currentMap != nil {
					currentMap.Close()
				}

				// Start new reader with new map
				currentMap = newMap
				readerCtx, cancel := context.WithCancel(ctx)
				readerCancel = cancel

				r.logger.Info("switched to new eBPF map",
					zap.Int("fd", newMap.FD()),
					zap.String("mapPath", r.mapPath),
				)

				// Start reading from new map
				go func() {
					defer r.logger.Info("reader stopped")
					if err := r.readLoop(readerCtx, newMap); err != nil {
						r.logger.Error("readLoop failed", zap.Error(err))
					}
				}()
			}
		}
	}()

	// FD client: gets new FDs when odiglet restarts
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer close(updates)

		r.logger.Info("starting FD client")

		err := unixfd.ConnectAndListen(ctx, unixfd.DefaultSocketPath, func(fd int) {
			r.logger.Info("received new FD from odiglet",
				zap.Int("fd", fd),
			)

			// Create map from FD
			newMap, err := ebpf.NewMapFromFD(fd)
			if err != nil {
				r.logger.Error("failed to create map from FD",
					zap.Error(err),
					zap.Int("fd", fd),
				)
				unix.Close(fd)
				return
			}

			// Send to map manager
			select {
			case updates <- newMap:
				r.logger.Info("queued new map for processing")
			case <-ctx.Done():
				newMap.Close()
			}
		})

		if err != nil && ctx.Err() == nil {
			r.logger.Error("FD client failed", zap.Error(err))
		}
	}()

	return nil
}

// This will only create a new map when odiglet actually restarts
// Not on every polling cycle!

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
