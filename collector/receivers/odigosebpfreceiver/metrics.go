package odigosebpfreceiver

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cilium/ebpf"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"
)

// metricsReadLoop handles periodic collection of metrics from HashOfMaps
func (r *ebpfReceiver) metricsReadLoop(ctx context.Context, m *ebpf.Map) error {
	// Config should already have defaults from createDefaultConfig(), but validate just in case
	interval := r.config.MetricsConfig.Interval
	if interval <= 0 {
		interval = 30 * time.Second // Fallback if somehow defaults weren't applied
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	r.logger.Info("starting JVM metrics collection loop", zap.Duration("interval", interval))

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
			continue
		}

		processedMaps++
	}

	r.logger.Debug("metrics collection completed",
		zap.Int("inner_maps_found", innerMapsCount),
		zap.Int("maps_processed", processedMaps))
	return nil
}

// processInnerMapMetrics processes a single inner map and extracts metrics
// each innermap represents a single process and its metrics
func (r *ebpfReceiver) processInnerMapMetrics(ctx context.Context, innerMap *ebpf.Map, processKey [512]byte) error {

	// Process with JVM handler
	metrics, err := r.jvmHandler.ExtractJVMMetricsFromInnerMap(ctx, innerMap, processKey)
	if err != nil {
		return fmt.Errorf("JVM handler failed: %w", err)
	}

	// Add resource attributes from processKey if we have metrics
	if metrics.ResourceMetrics().Len() > 0 {
		resourceAttrs := metrics.ResourceMetrics().At(0).Resource().Attributes()
		if err := r.addResourceAttributesFromProcessKey(resourceAttrs, processKey); err != nil {
			r.logger.Error("failed to add resource attributes", zap.Error(err))
			return err
		}
	}

	// Forward metrics to consumer if consumer is not nil and there are metrics to forward
	if r.nextMetrics != nil && metrics.ResourceMetrics().Len() > 0 {
		if err := r.nextMetrics.ConsumeMetrics(ctx, metrics); err != nil {
			return fmt.Errorf("failed to consume metrics: %w", err)
		}
	}

	return nil
}

// addResourceAttributesFromProcessKey parses processKey and adds resource attributes
// Format: "k8s.container.name:frontend,k8s.deployment.name:frontend,service.name:frontend,..."
// This is specific to how Odigos encodes process metadata in eBPF maps
func (r *ebpfReceiver) addResourceAttributesFromProcessKey(resourceAttrs pcommon.Map, processKey [512]byte) error {
	// Convert byte array to string
	processKeyStr := string(processKey[:])

	// Trim null bytes from the string
	processKeyStr = string(bytes.TrimRight([]byte(processKeyStr), "\x00"))

	// If empty, skip processing
	if len(processKeyStr) == 0 {
		return nil
	}

	// Parse comma-separated key:value pairs
	return r.parseResourceAttributes(resourceAttrs, processKeyStr, ",", ":")
}

// parseResourceAttributes parses a delimited string and adds resource attributes
// Format: "key1:value1,key2:value2,key3:value3"
// This handles the specific format used by Odigos for eBPF processKey encoding
func (r *ebpfReceiver) parseResourceAttributes(resourceAttrs pcommon.Map, attributesStr string, itemDelimiter string, keyValueSeparator string) error {
	if attributesStr == "" {
		return fmt.Errorf("empty attributes string")
	}

	parts := strings.Split(attributesStr, itemDelimiter)
	if len(parts) == 0 {
		return fmt.Errorf("no attributes found")
	}

	attributesAdded := 0
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Split by keyValueSeparator to get key:value
		attrParts := strings.SplitN(part, keyValueSeparator, 2)
		if len(attrParts) != 2 {
			r.logger.Warn("invalid resource attribute format, skipping",
				zap.String("attribute", part),
				zap.String("expected_format", "key"+keyValueSeparator+"value"))
			continue
		}

		attrKey := strings.TrimSpace(attrParts[0])
		attrValue := strings.TrimSpace(attrParts[1])

		if attrKey != "" && attrValue != "" {
			resourceAttrs.PutStr(attrKey, attrValue)
			attributesAdded++
		}
	}

	if attributesAdded == 0 {
		return fmt.Errorf("no valid attributes parsed from: %s", attributesStr)
	}

	return nil
}
