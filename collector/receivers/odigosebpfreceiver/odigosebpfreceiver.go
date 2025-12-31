package odigosebpfreceiver

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"strings"
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

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	numOfPages = 2048
)

// ReceiverType defines the type of receiver (traces or metrics)
type ReceiverType int

const (
	ReceiverTypeTraces ReceiverType = iota
	ReceiverTypeMetrics
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

// metricsReadLoop handles periodic collection of metrics from HashOfMaps
func (r *ebpfReceiver) metricsReadLoop(ctx context.Context, m *ebpf.Map) error {
	// Config should already have defaults from createDefaultConfig(), but validate just in case
	interval := r.config.MetricsConfig.Interval
	if interval <= 0 {
		interval = 30 * time.Second // Fallback if somehow defaults weren't applied
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	r.logger.Info("starting metrics collection loop", zap.Duration("interval", interval))

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("metrics collection loop stopping")
			return nil
		case <-ticker.C:
			if err := r.collectMetrics(ctx, m); err != nil {
				r.logger.Error("failed to collect metrics", zap.Error(err))
				// Continue collecting even if one iteration fails
			}
		}
	}
}

// collectMetrics iterates over the HashOfMaps and processes each inner map for metrics
func (r *ebpfReceiver) collectMetrics(ctx context.Context, hashOfMaps *ebpf.Map) error {
	r.logger.Debug("starting metrics collection iteration")

	var processKey [512]byte // Key representing process ID or similar identifier
	var innerMapID uint32    // ID of the inner map

	// Iterate over all entries in the hash of maps
	iter := hashOfMaps.Iterate()
	defer func() {
		if err := iter.Err(); err != nil {
			r.logger.Error("iterator error", zap.Error(err))
		}
	}()

	innerMapsCount := 0
	processedMaps := 0

	for iter.Next(&processKey, &innerMapID) {
		innerMapsCount++
		r.logger.Debug("found inner map",
			zap.String("process_key", string(processKey[:])),
			zap.Uint32("inner_map_id", innerMapID))

		// Get the inner map from the ID
		innerMap, err := ebpf.NewMapFromID(ebpf.MapID(innerMapID))
		if err != nil {
			r.logger.Error("failed to get inner map",
				zap.Uint32("inner_map_id", innerMapID),
				zap.Error(err))
			continue
		}

		// Process metrics from this inner map
		if err := r.processInnerMapMetrics(ctx, innerMap, processKey); err != nil {
			r.logger.Error("failed to process inner map metrics",
				zap.String("process_key", string(processKey[:])),
				zap.Uint32("inner_map_id", innerMapID),
				zap.Error(err))
			innerMap.Close()
			continue
		}

		innerMap.Close()
		processedMaps++
	}

	r.logger.Info("metrics collection completed",
		zap.Int("inner_maps_found", innerMapsCount),
		zap.Int("maps_processed", processedMaps))
	return nil
}

// processInnerMapMetrics processes a single inner map and extracts metrics
// each innermap represents a single process and its metrics
func (r *ebpfReceiver) processInnerMapMetrics(ctx context.Context, innerMap *ebpf.Map, processKey [512]byte) error {
	r.logger.Debug("processing inner map for metrics", zap.String("process_key", string(processKey[:])))

	// Create metrics data structure
	metrics := pmetric.NewMetrics()
	resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
	scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()

	// Parse and add resource attributes from processKey
	resourceAttrs := resourceMetrics.Resource().Attributes()
	if err := r.addResourceAttributesFromProcessKey(resourceAttrs, processKey); err != nil {
		r.logger.Error("failed to add resource attributes from process key", zap.Error(err))
		return err
	}

	// Iterate over inner map entries
	var metricKey [4]byte    // key size from eBPF map spec
	var metricValue [40]byte // value size from eBPF map spec

	iter := innerMap.Iterate()
	defer func() {
		if err := iter.Err(); err != nil {
			r.logger.Error("inner map iterator error", zap.Error(err))
		}
	}()

	metricsFound := 0
	for iter.Next(&metricKey, &metricValue) {
		// Convert key bytes to string (assuming null-terminated)
		keyStr := string(metricKey[:])
		if nullIndex := strings.IndexByte(keyStr, 0); nullIndex != -1 {
			keyStr = keyStr[:nullIndex]
		}

		// Convert value bytes to uint64 (simple case for now)
		// TODO: Handle different value types based on your eBPF implementation
		value := parseMetricValue(metricValue[:])

		if err := r.addMetricToCollection(scopeMetrics, keyStr, value); err != nil {
			r.logger.Error("failed to add metric",
				zap.String("key", keyStr),
				zap.Uint64("value", value),
				zap.Error(err))
			continue
		}

		metricsFound++
		r.logger.Debug("processed metric",
			zap.String("key", keyStr),
			zap.Uint64("value", value))
	}

	// Forward metrics to next consumer if we have any
	if metricsFound > 0 && r.nextMetrics != nil {
		if err := r.nextMetrics.ConsumeMetrics(ctx, metrics); err != nil {
			return fmt.Errorf("failed to consume metrics: %w", err)
		}
		r.logger.Debug("forwarded metrics to consumer",
			zap.Int("metrics_count", metricsFound))
	}

	return nil
}

// addMetricToCollection adds a single metric to the metrics collection
func (r *ebpfReceiver) addMetricToCollection(scopeMetrics pmetric.ScopeMetrics, keyStr string, value uint64) error {
	// Parse the key: "metric.name%attr1=value1%attr2=value2"
	metricName, attributes, err := r.parseMetricKey(keyStr)
	if err != nil {
		return fmt.Errorf("failed to parse metric key %s: %w", keyStr, err)
	}

	// Determine metric type based on metric name
	metricType := r.determineMetricType(metricName)

	// Create metric
	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(metricName)
	metric.SetDescription(fmt.Sprintf("eBPF metric: %s", metricName))

	// Set timestamp
	now := pcommon.NewTimestampFromTime(time.Now())

	switch metricType {
	case "counter":
		sum := metric.SetEmptySum()
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)

		dataPoint := sum.DataPoints().AppendEmpty()
		dataPoint.SetIntValue(int64(value))
		dataPoint.SetTimestamp(now)
		dataPoint.SetStartTimestamp(now)

		// Add attributes
		attrs := dataPoint.Attributes()
		for key, val := range attributes {
			attrs.PutStr(key, val)
		}

	case "gauge":
		gauge := metric.SetEmptyGauge()
		dataPoint := gauge.DataPoints().AppendEmpty()

		dataPoint.SetIntValue(int64(value))
		dataPoint.SetTimestamp(now)
		dataPoint.SetStartTimestamp(now)

		// Add attributes
		attrs := dataPoint.Attributes()
		for key, val := range attributes {
			attrs.PutStr(key, val)
		}

	default:
		return fmt.Errorf("unsupported metric type: %s", metricType)
	}

	return nil
}

func (r *ebpfReceiver) determineMetricType(metricName string) string {
	// TODO: Implement logic based on eBPF map data or metric name patterns
	return "counter"
}

// parseMetricKey parses a key like "jvm.threads.count%service=myapp%pod=pod123"
// Returns: metricName, attributes map, error
func (r *ebpfReceiver) parseMetricKey(keyStr string) (string, map[string]string, error) {
	if keyStr == "" {
		return "", nil, fmt.Errorf("empty key")
	}

	parts := strings.Split(keyStr, "%")
	if len(parts) == 0 {
		return "", nil, fmt.Errorf("invalid key format")
	}

	// First part is the metric name
	metricName := parts[0]
	if metricName == "" {
		return "", nil, fmt.Errorf("empty metric name")
	}

	// Parse attributes from remaining parts
	attributes := make(map[string]string)
	for i := 1; i < len(parts); i++ {
		attrPart := parts[i]
		if attrPart == "" {
			continue // Skip empty parts
		}

		// Split by '=' to get key=value
		attrParts := strings.SplitN(attrPart, "=", 2)
		if len(attrParts) != 2 {
			r.logger.Warn("invalid attribute format, skipping",
				zap.String("attribute", attrPart))
			continue
		}

		attrKey := strings.TrimSpace(attrParts[0])
		attrValue := strings.TrimSpace(attrParts[1])

		if attrKey != "" && attrValue != "" {
			attributes[attrKey] = attrValue
		}
	}

	return metricName, attributes, nil
}

// parseMetricValue parses metric value from bytes
// TODO: Extend this based on your actual value encoding
func parseMetricValue(valueBytes []byte) uint64 {
	// Simple implementation: assume first 8 bytes are uint64 in native endian
	if len(valueBytes) >= 8 {
		return binary.NativeEndian.Uint64(valueBytes[:8])
	}
	// Fallback: treat as single uint32
	if len(valueBytes) >= 4 {
		return uint64(binary.NativeEndian.Uint32(valueBytes[:4]))
	}
	return 0
}

// addResourceAttributesFromProcessKey parses processKey and adds resource attributes
// TODO: The delimiter character and encoding method are not yet decided
func (r *ebpfReceiver) addResourceAttributesFromProcessKey(resourceAttrs pcommon.Map, processKey [512]byte) error {
	// TODO: Replace this with the actual encoding method used for processKey
	// For now, converting the byte array to string
	// This is a temporary implementation - you'll need to replace this with the actual decoding logic
	processKeyStr := string(processKey[:])

	// Trim null bytes from the string
	processKeyStr = string(bytes.TrimRight([]byte(processKeyStr), "\x00"))

	// If empty or very short, it might just be a process ID
	if len(processKeyStr) == 0 {
		return nil
	}

	// If you have a different encoding method (e.g., the key represents an index to a string table,
	// or it's encoded differently), replace this section
	return r.parseResourceAttributes(resourceAttrs, processKeyStr, "|") // Using "|" as delimiter for now
}

// parseResourceAttributes parses a delimited string and adds resource attributes
// Format: "key1=value1|key2=value2|key3=value3"
func (r *ebpfReceiver) parseResourceAttributes(resourceAttrs pcommon.Map, attributesStr string, delimiter string) error {
	if attributesStr == "" {
		return fmt.Errorf("empty attributes string")
	}

	parts := strings.Split(attributesStr, delimiter)
	if len(parts) == 0 {
		return fmt.Errorf("no attributes found")
	}

	attributesAdded := 0
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Split by '=' to get key=value
		attrParts := strings.SplitN(part, "=", 2)
		if len(attrParts) != 2 {
			r.logger.Warn("invalid resource attribute format, skipping",
				zap.String("attribute", part))
			continue
		}

		attrKey := strings.TrimSpace(attrParts[0])
		attrValue := strings.TrimSpace(attrParts[1])

		if attrKey != "" && attrValue != "" {
			resourceAttrs.PutStr(attrKey, attrValue)
			attributesAdded++
			r.logger.Debug("added resource attribute",
				zap.String("key", attrKey),
				zap.String("value", attrValue))
		}
	}

	if attributesAdded == 0 {
		return fmt.Errorf("no valid attributes parsed from: %s", attributesStr)
	}

	return nil
}
