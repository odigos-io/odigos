package odigosebpfreceiver

import (
	"context"
	"fmt"
	"sync"

	"github.com/cilium/ebpf"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"

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
	jvmHandler        *jvm.JVMMetricsHandler
	processStartTimes map[[64]byte]pcommon.Timestamp

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
		r.processStartTimes = make(map[[64]byte]pcommon.Timestamp)
	}

	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	switch r.receiverType {
	case ReceiverTypeTraces:
		r.startTracesReceiver(ctx)
	case ReceiverTypeMetrics:
		r.startMetricsReceiver(ctx)
	}

	return nil
}

// startTracesReceiver sets up a single FD client and map manager for traces.
func (r *ebpfReceiver) startTracesReceiver(ctx context.Context) {
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
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()

		var (
			currentMap   *ebpf.Map
			readerCancel context.CancelFunc
			readerWg     sync.WaitGroup
		)

		defer func() {
			if readerCancel != nil {
				readerCancel()
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
				if readerCancel != nil {
					readerCancel()
					readerWg.Wait()
				}
				if currentMap != nil {
					currentMap.Close()
				}

				currentMap = newMap
				readerCtx, cancel := context.WithCancel(ctx)
				readerCancel = cancel

				r.logger.Info("switched to new eBPF traces map", zap.Int("fd", newMap.FD()))

				readerWg.Add(1)
				go func() {
					defer func() {
						r.logger.Info("traces reader stopped")
						readerWg.Done()
					}()

					if err := r.tracesReadLoop(readerCtx, newMap); err != nil {
						r.logger.Error("tracesReadLoop failed", zap.Error(err))
					}
				}()
			}
		}
	}()

	// FD client: connects to odiglet and receives FDs for traces eBPF maps.
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer close(updates)

		r.logger.Info("starting traces FD client")

		err := unixfd.ConnectAndListen(ctx, unixfd.DefaultSocketPath, unixfd.ReqGetTracesFD, r.logger, func(fd int) {
			r.logger.Info("received new traces FD from odiglet", zap.Int("fd", fd))

			newMap, err := ebpf.NewMapFromFD(fd)
			if err != nil {
				r.logger.Error("failed to create map from FD", zap.Error(err), zap.Int("fd", fd))
				unix.Close(fd)
				return
			}

			select {
			case updates <- newMap:
				r.logger.Info("queued new traces map for processing")
			case <-ctx.Done():
				newMap.Close()
			}
		})

		if err != nil && ctx.Err() == nil {
			r.logger.Error("traces FD client failed", zap.Error(err))
		}
	}()
}

// metricsMapPair holds both metrics eBPF maps received atomically from a single FD exchange.
type metricsMapPair struct {
	hashOfMaps    *ebpf.Map
	attributesMap *ebpf.Map
}

// startMetricsReceiver sets up a single FD client that receives both the HashOfMaps and
// AttributesMap file descriptors in a single message, and a map manager that restarts
// the metrics read loop when new maps arrive.
func (r *ebpfReceiver) startMetricsReceiver(ctx context.Context) {
	updates := make(chan metricsMapPair, 1)

	// Map manager: manages the lifecycle of both HashOfMaps and AttributesMap,
	// which arrive atomically via a single FD exchange.
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()

		var (
			currentHashOfMaps    *ebpf.Map
			currentAttributesMap *ebpf.Map
			readerCancel         context.CancelFunc
			readerWg             sync.WaitGroup
		)

		defer func() {
			if readerCancel != nil {
				readerCancel()
				readerWg.Wait()
			}
			if currentHashOfMaps != nil {
				currentHashOfMaps.Close()
			}
			if currentAttributesMap != nil {
				currentAttributesMap.Close()
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case pair, ok := <-updates:
				if !ok {
					return
				}

				if readerCancel != nil {
					readerCancel()
					readerWg.Wait()
				}
				if currentHashOfMaps != nil {
					currentHashOfMaps.Close()
				}
				if currentAttributesMap != nil {
					currentAttributesMap.Close()
				}

				currentHashOfMaps = pair.hashOfMaps
				currentAttributesMap = pair.attributesMap

				r.logger.Info("received new metrics maps",
					zap.Int("hashOfMaps_fd", currentHashOfMaps.FD()),
					zap.Int("attributesMap_fd", currentAttributesMap.FD()))

				readerCtx, cancel := context.WithCancel(ctx)
				readerCancel = cancel

				hashOfMaps := currentHashOfMaps
				attributesMap := currentAttributesMap
				readerWg.Add(1)
				go func() {
					defer func() {
						r.logger.Info("metrics reader stopped")
						readerWg.Done()
					}()

					if err := r.metricsReadLoop(readerCtx, hashOfMaps, attributesMap); err != nil {
						r.logger.Error("metricsReadLoop failed", zap.Error(err))
					}
				}()
			}
		}
	}()

	// FD client: connects to odiglet and receives both metrics FDs via ConnectAndListenMulti.
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer close(updates)

		r.logger.Info("starting metrics FD client")

		err := unixfd.ConnectAndListenMulti(ctx, unixfd.DefaultSocketPath, unixfd.ReqGetMetricsFD, r.logger, func(fds []int) {
			if len(fds) != 2 {
				r.logger.Error("expected 2 metrics FDs, closing all",
					zap.Int("received", len(fds)))
				for _, fd := range fds {
					unix.Close(fd)
				}
				return
			}

			r.logger.Info("received metrics FDs from odiglet",
				zap.Int("hashOfMaps_fd", fds[0]),
				zap.Int("attributesMap_fd", fds[1]))

			hashMap, err := ebpf.NewMapFromFD(fds[0])
			if err != nil {
				r.logger.Error("failed to create HashOfMaps from FD", zap.Error(err), zap.Int("fd", fds[0]))
				unix.Close(fds[0])
				unix.Close(fds[1])
				return
			}

			attrMap, err := ebpf.NewMapFromFD(fds[1])
			if err != nil {
				r.logger.Error("failed to create AttributesMap from FD", zap.Error(err), zap.Int("fd", fds[1]))
				hashMap.Close()
				unix.Close(fds[1])
				return
			}

			select {
			case updates <- metricsMapPair{hashOfMaps: hashMap, attributesMap: attrMap}:
				r.logger.Info("queued new metrics maps for processing")
			case <-ctx.Done():
				hashMap.Close()
				attrMap.Close()
			}
		})

		if err != nil && ctx.Err() == nil {
			r.logger.Error("metrics FD client failed", zap.Error(err))
		}
	}()
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
