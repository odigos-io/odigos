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
	config *Config
	cancel context.CancelFunc
	logger *zap.Logger

	// Pipeline consumers for forwarding telemetry data
	nextTraces  consumer.Traces
	nextMetrics consumer.Metrics
	nextLogs    consumer.Logs

	wg sync.WaitGroup
}

func (r *ebpfReceiver) Start(ctx context.Context, host component.Host) error {
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	updates := make(chan *ebpf.Map, 1)

	/*
	 * eBPF Receiver Architecture
	 *
	 * This receiver operates with two main goroutines that work together to handle
	 * eBPF maps from odiglet and process trace data:
	 *
	 * 1. FD Client Goroutine: Connects to odiglet via Unix socket and receives file
	 *    descriptors for eBPF maps. When odiglet restarts, it creates a new map and
	 *    sends us the FD. This goroutine converts FDs to map objects and forwards
	 *    them to the map manager.
	 *
	 * 2. Map Manager Goroutine: Handles the lifecycle of eBPF maps. It receives new
	 *    maps from the FD client, stops any existing perf readers, closes old maps,
	 *    and starts new perf readers for incoming maps. This ensures seamless
	 *    switching between maps during odiglet restarts.
	 *
	 * The two goroutines communicate via the 'updates' channel, creating a pipeline
	 * that maintains continuous trace data flow even when odiglet restarts.
	 */

	/*
	 * Map Manager Goroutine
	 *
	 * Manages eBPF map lifecycle as described in the architecture overview above.
	 * Receives new maps via the updates channel and handles the switching process.
	 */
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()

		var (
			currentMap   *ebpf.Map
			readerCancel context.CancelFunc
		)

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
				// Clean up the previous map and reader
				if readerCancel != nil {
					readerCancel()
				}
				if currentMap != nil {
					currentMap.Close()
				}

				// Switch to the new map
				currentMap = newMap
				readerCtx, cancel := context.WithCancel(ctx)
				readerCancel = cancel

				r.logger.Info("switched to new eBPF map", zap.Int("fd", newMap.FD()))

				// Start reading from the new map
				go func() {
					defer r.logger.Info("reader stopped")
					if err := r.readLoop(readerCtx, newMap); err != nil {
						r.logger.Error("readLoop failed", zap.Error(err))
					}
				}()
			}
		}
	}()

	/*
	 * FD Client Goroutine
	 *
	 * Connects to odiglet as described in the architecture overview above.
	 * Receives file descriptors for new eBPF maps and forwards them to the map manager.
	 */
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer close(updates)

		r.logger.Info("starting FD client")

		err := unixfd.ConnectAndListen(ctx, unixfd.DefaultSocketPath, func(fd int) {
			r.logger.Info("received new FD from odiglet", zap.Int("fd", fd))

			// Convert the file descriptor into an eBPF map object
			newMap, err := ebpf.NewMapFromFD(fd)
			if err != nil {
				r.logger.Error("failed to create map from FD", zap.Error(err), zap.Int("fd", fd))
				unix.Close(fd)
				return
			}

			// Send the new map to the map manager for processing
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

func (r *ebpfReceiver) Shutdown(ctx context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}
	done := make(chan struct{})

	// Wait for all goroutines to finish in a separate goroutine
	// so we can respect the shutdown context timeout
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		r.logger.Warn("odigos-ebpf: receiver shutdown did not finish before context was canceled")
		return ctx.Err()
	case <-done:
		r.logger.Info("odigos-ebpf: receiver shutdown complete")
		return nil
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

	// Close the reader when context is cancelled to unblock ReadInto()
	go func() {
		<-ctx.Done()
		reader.Close()
	}()

	for {
		err := reader.ReadInto(&record)
		if err != nil {
			if errors.Is(err, perf.ErrClosed) {
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
			continue
		}

		// Try to unmarshal as current OpenTelemetry format first
		protoUnmarshaler := ptrace.ProtoUnmarshaler{}
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
