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

// metricsReadLoop handles periodic collection of metrics from HashOfMaps and AttributesMap
func (r *ebpfReceiver) metricsReadLoop(ctx context.Context, hashOfMaps *ebpf.Map, attributesMap *ebpf.Map) error {
	interval := r.config.MetricsConfig.Interval
	if interval <= 0 {
		interval = 30 * time.Second
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
			if err := r.collectMetrics(ctx, hashOfMaps, attributesMap); err != nil {
				r.logger.Error("failed to collect metrics", zap.Error(err))
			}
		}
	}
}

// collectMetrics iterates over the AttributesMap and HashOfMaps to collect metrics.
// It uses a two-phase approach:
//  1. Build a UUID -> packed attributes cache from the AttributesMap
//  2. Iterate the HashOfMaps, look up cached attributes by UUID, and process each inner map
func (r *ebpfReceiver) collectMetrics(ctx context.Context, hashOfMaps *ebpf.Map, attributesMap *ebpf.Map) error {
	// Phase 1: Build UUID -> packed attributes cache from AttributesMap
	attrCache := make(map[[64]byte]string)

	var uuidKey [64]byte
	var attrValue [1024]byte

	attrIter := attributesMap.Iterate()
	for attrIter.Next(&uuidKey, &attrValue) {
		attrStr := string(bytes.TrimRight(attrValue[:], "\x00"))
		if attrStr != "" {
			attrCache[uuidKey] = attrStr
		}
	}
	if err := attrIter.Err(); err != nil {
		r.logger.Error("attributes map iterator error", zap.Error(err))
	}

	r.logger.Debug("attributes cache built", zap.Int("entries", len(attrCache)))

	// Phase 2: Iterate HashOfMaps using UUID key
	var processKey [64]byte
	var innerMapID uint32

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

		// Look up cached attributes for this UUID
		packedAttributes, found := attrCache[processKey]
		if !found {
			uuidStr := string(bytes.TrimRight(processKey[:], "\x00"))
			r.logger.Warn("no attributes found for UUID in attributes map",
				zap.String("uuid", uuidStr))
		}

		// Process metrics from this inner map
		if err := r.processInnerMapMetrics(ctx, innerMap, packedAttributes); err != nil {
			r.logger.Error("failed to process inner map metrics",
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

// processInnerMapMetrics processes a single inner map and extracts metrics.
// packedAttributes contains the resource attributes in "key1:value1,key2:value2" format,
// looked up from the AttributesMap by UUID.
func (r *ebpfReceiver) processInnerMapMetrics(ctx context.Context, innerMap *ebpf.Map, packedAttributes string) error {
	// Process with JVM handler
	metrics, err := r.jvmHandler.ExtractJVMMetricsFromInnerMap(ctx, innerMap)
	if err != nil {
		return fmt.Errorf("JVM handler failed: %w", err)
	}

	// Add resource attributes from packedAttributes if we have metrics
	if metrics.ResourceMetrics().Len() > 0 && packedAttributes != "" {
		resourceAttrs := metrics.ResourceMetrics().At(0).Resource().Attributes()
		if err := r.parseResourceAttributes(resourceAttrs, packedAttributes, ",", ":"); err != nil {
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

// parseResourceAttributes parses a delimited string and adds resource attributes
// Format: "key1:value1,key2:value2,key3:value3"
// This handles the specific format used by Odigos for eBPF resource attribute encoding
func (r *ebpfReceiver) parseResourceAttributes(resourceAttrs pcommon.Map, attributesStr string, itemDelimiter string, keyValueSeparator string) error {
	if attributesStr == "" {
		return fmt.Errorf("empty attributes string")
	}

	parsed := false
	for _, part := range strings.Split(attributesStr, itemDelimiter) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

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
			parsed = true
		}
	}

	if !parsed {
		return fmt.Errorf("no valid attributes parsed from: %s", attributesStr)
	}

	return nil
}
