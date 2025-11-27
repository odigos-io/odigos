package odigosebpfreceiver

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/cilium/ebpf"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"

	rtml "github.com/odigos-io/go-rtml"

	"go.opentelemetry.io/collector/receiver"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
	"google.golang.org/protobuf/proto"

	"github.com/odigos-io/odigos/collector/receivers/odigosebpfreceiver/internal/metadata"
	"github.com/odigos-io/odigos/common/unixfd"

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

	// Telemetry
	telemetry *metadata.TelemetryBuilder
	settings  receiver.Settings

	// Reusable unmarshaler to avoid allocating it on every read
	protoUnmarshaler ptrace.ProtoUnmarshaler

	wg sync.WaitGroup
}

func (r *ebpfReceiver) Start(ctx context.Context, host component.Host) error {
	// Initialize telemetry
	telemetryBuilder, err := metadata.NewTelemetryBuilder(r.settings.TelemetrySettings)
	if err != nil {
		return fmt.Errorf("failed to create telemetry builder: %w", err)
	}
	r.telemetry = telemetryBuilder

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
			readerWg     sync.WaitGroup // Tracks the current reader goroutine
		)

		defer func() {
			if readerCancel != nil {
				readerCancel()
				// Wait for the current reader to stop before cleanup
				readerWg.Wait()
			}
			if currentMap != nil {
				currentMap.Close()
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case newMap, ok := <-updates:
				if !ok {
					return
				}
				// Clean up the previous map and reader
				if readerCancel != nil {
					readerCancel()
					// Wait for the current reader goroutine to fully stop
					// This prevents race conditions between old and new readers
					readerWg.Wait()
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
				readerWg.Add(1)
				go func() {
					defer func() {
						r.logger.Info("reader stopped")
						readerWg.Done() // Signal that this reader has stopped
					}()
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

		err := unixfd.ConnectAndListen(ctx, unixfd.DefaultSocketPath, r.logger, func(fd int) {
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

	// Cleanup telemetry
	if r.telemetry != nil {
		r.telemetry.Shutdown()
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

	/*
	 * Batching Strategy
	 *
	 * Instead of calling ConsumeTraces() for every single span, we accumulate
	 * spans into batches. This reduces:
	 * - Function call overhead (fewer pipeline traversals)
	 * - Lock contention in downstream processors
	 * - GC pressure (fewer allocation cycles)
	 *
	 * We flush the batch when it reaches maxBatchSize (100 resource spans).
	 * The batching uses MoveAndAppendTo() which moves data without copying,
	 * minimizing CPU overhead.
	 */
	const maxBatchSize = 500

	batch := ptrace.NewTraces()
	batchSize := 0

	// Helper function to flush the current batch
	flushBatch := func() {
		if batchSize == 0 {
			return
		}

		r.logger.Info("flushing batch", zap.Int("batch_size", batchSize), zap.Int("resource_spans_count", batch.ResourceSpans().Len()))
		err := r.nextTraces.ConsumeTraces(ctx, batch)
		if err != nil {
			r.logger.Error("err consuming traces", zap.Error(err))
		}
		// Reset batch state
		batch = ptrace.NewTraces()
		batchSize = 0
	}

	// Ensure we flush any remaining spans on exit
	defer flushBatch()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// Continue to read next span
		}

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
				flushBatch()
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

		// Try to unmarshal as current OpenTelemetry format first
		td, err := r.protoUnmarshaler.UnmarshalTraces(record.RawSample[8 : 8+acceptedLength])
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

		// Add the trace to the batch by moving its ResourceSpans
		// MoveAndAppendTo() moves data without copying, minimizing CPU overhead
		td.ResourceSpans().MoveAndAppendTo(batch.ResourceSpans())
		batchSize++

		// Flush when batch is full
		if batchSize >= maxBatchSize {
			flushBatch()
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
