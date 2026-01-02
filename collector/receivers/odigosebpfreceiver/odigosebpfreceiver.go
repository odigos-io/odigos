package odigosebpfreceiver

import (
	"context"
	"fmt"
	"sync"

	"github.com/cilium/ebpf"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"

	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"

	"github.com/odigos-io/odigos/collector/receivers/odigosebpfreceiver/internal/metadata"
	"github.com/odigos-io/odigos/collector/receivers/odigosebpfreceiver/internal/metrics/jvm"
	"github.com/odigos-io/odigos/common/unixfd"
)

type ebpfReceiver struct {
	config       *Config
	cancel       context.CancelFunc
	logger       *zap.Logger
	receiverType ReceiverType

	// Pipeline consumers for forwarding telemetry data
	nextTraces  consumer.Traces
	nextMetrics consumer.Metrics
	nextLogs    consumer.Logs

	// Telemetry
	telemetry *metadata.TelemetryBuilder
	settings  receiver.Settings

	// Metrics handlers - only used for metrics receivers
	jvmHandler *jvm.JVMMetricsHandler

	wg sync.WaitGroup
}

func (r *ebpfReceiver) Start(ctx context.Context, host component.Host) error {
	// Initialize telemetry
	telemetryBuilder, err := metadata.NewTelemetryBuilder(r.settings.TelemetrySettings)
	if err != nil {
		return fmt.Errorf("failed to create telemetry builder: %w", err)
	}
	r.telemetry = telemetryBuilder

	// Initialize metrics handlers - only for metrics receivers
	if r.receiverType == ReceiverTypeMetrics {
		r.jvmHandler = jvm.NewJVMMetricsHandler(r.logger)
	}

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

					var err error
					switch r.receiverType {
					case ReceiverTypeTraces:
						err = r.tracesReadLoop(readerCtx, newMap)
					case ReceiverTypeMetrics:
						err = r.metricsReadLoop(readerCtx, newMap)
					}

					if err != nil {
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

		var requestType string
		switch r.receiverType {
		case ReceiverTypeTraces:
			requestType = unixfd.ReqGetTracesFD
		case ReceiverTypeMetrics:
			requestType = unixfd.ReqGetMetricsFD
		}

		err := unixfd.ConnectAndListen(ctx, unixfd.DefaultSocketPath, requestType, r.logger, func(fd int) {
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
