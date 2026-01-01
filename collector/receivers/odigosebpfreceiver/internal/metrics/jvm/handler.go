package jvm

import (
	"context"
	"fmt"
	"time"

	"github.com/cilium/ebpf"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	semconv1_26 "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
)

const (
	// Metric names
	metricGCDuration        = semconv1_26.JvmGcDurationName
	metricMemoryUsedAfterGC = semconv1_26.JvmMemoryUsedAfterLastGcName
	metricClassesLoaded     = semconv1_26.JvmClassLoadedName
	metricClassesUnloaded   = semconv1_26.JvmClassUnloadedName
	metricThreadCount       = semconv1_26.JvmThreadCountName

	// OTel attribute keys
	attrGCAction       = semconv1_26.JvmGcActionKey
	attrGCName         = semconv1_26.JvmGcNameKey
	attrMemoryType     = semconv1_26.JvmMemoryTypeKey
	attrMemoryPoolName = semconv1_26.JvmMemoryPoolNameKey
	attrThreadDaemon   = semconv1_26.JvmThreadDaemonKey
	attrThreadState    = semconv1_26.JvmThreadStateKey

	// Metric descriptions
	DescClassLoaded   = "Number of classes loaded since JVM start"
	DescClassUnloaded = "Number of classes unloaded since JVM start"
	DescMemoryUsed    = "Measure of memory used after the most recent garbage collection event"
	DescThreadCount   = "Number of executing platform threads"
	DescGCDuration    = "Duration of JVM garbage collection actions"
)

// JVMMetricsHandler processes JVM metrics from eBPF maps and converts them to OpenTelemetry pdata format
type JVMMetricsHandler struct {
	logger *zap.Logger
}

// NewJVMMetricsHandler creates a new JVM metrics handler
func NewJVMMetricsHandler(logger *zap.Logger) *JVMMetricsHandler {
	return &JVMMetricsHandler{
		logger: logger,
	}
}

// ExtractJVMMetricsFromInnerMap extracts JVM metrics from a process inner map and converts them to OpenTelemetry format
func (h *JVMMetricsHandler) ExtractJVMMetricsFromInnerMap(ctx context.Context, innerMap *ebpf.Map, processKey [512]byte) (pmetric.Metrics, error) {
	metrics := pmetric.NewMetrics()
	resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
	scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()

	// Add scope information
	scopeMetrics.Scope().SetName("jvm-ebpf-metrics")
	scopeMetrics.Scope().SetVersion("1.0.0")

	var key MetricKey
	var value MetricValue

	iter := innerMap.Iterate()
	defer func() {
		if err := iter.Err(); err != nil {
			h.logger.Error("Error iterating inner map", zap.Error(err))
		}
	}()

	entriesFound := 0
	metricsAdded := 0

	for iter.Next(&key, &value) {
		entriesFound++
		metricType := key.MetricType()

		switch metricType {
		case MetricClassLoaded:
			h.addClassLoadedMetric(scopeMetrics, value.AsCounter())
		case MetricClassUnloaded:
			h.addClassUnloadedMetric(scopeMetrics, value.AsCounter())
		case MetricMemoryUsedAfterGC:
			memType := MemoryType(key.Attr1())
			poolName := MemoryPoolName(key.Attr2())
			h.addMemoryUsedMetric(scopeMetrics, value.AsGauge(), memType, poolName)
		case MetricGCDuration:
			gcAction := GCAction(key.Attr1())
			gcName := GCName(key.Attr2())
			h.addGCHistogramMetric(scopeMetrics, value.AsHistogram(), gcAction, gcName)
		case MetricThreadCount:
			daemon := ThreadDaemon(key.Attr1())
			state := ThreadState(key.Attr2())
			h.addThreadCountMetric(scopeMetrics, value.AsGauge(), daemon, state)
		default:
			h.logger.Warn("Unknown metric type", zap.Uint32("type", uint32(metricType)))
		}

		metricsAdded++

		// Reset counters/histogram after read (delta reporting)
		// Don't reset gauges - they represent current state
		if metricType != MetricMemoryUsedAfterGC && metricType != MetricThreadCount {
			var zeroValue MetricValue
			if err := innerMap.Update(&key, &zeroValue, ebpf.UpdateExist); err != nil {
				h.logger.Debug("Failed to reset metric", zap.String("key", fmt.Sprintf("%d", key)), zap.Error(err))
			}
		}
	}

	h.logger.Debug("JVM metrics extraction completed",
		zap.Int("ebpf_entries_found", entriesFound),
		zap.Int("metrics_added", metricsAdded),
		zap.Int("total_metrics_in_scope", scopeMetrics.Metrics().Len()))

	return metrics, nil
}

func (h *JVMMetricsHandler) addClassLoadedMetric(scopeMetrics pmetric.ScopeMetrics, counter CounterValue) {
	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(metricClassesLoaded)
	metric.SetDescription(DescClassLoaded)
	metric.SetUnit(semconv1_26.JvmClassLoadedUnit)

	sum := metric.SetEmptySum()
	sum.SetIsMonotonic(true)
	// Set cumulative temporality - values represent total since measurement started, not delta changes
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)

	dataPoint := sum.DataPoints().AppendEmpty()
	dataPoint.SetIntValue(int64(counter.Count))
	now := pcommon.NewTimestampFromTime(time.Now())
	dataPoint.SetTimestamp(now)
	dataPoint.SetStartTimestamp(now)
}

func (h *JVMMetricsHandler) addClassUnloadedMetric(scopeMetrics pmetric.ScopeMetrics, counter CounterValue) {
	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(metricClassesUnloaded)
	metric.SetDescription(DescClassUnloaded)
	metric.SetUnit(semconv1_26.JvmClassUnloadedUnit)

	sum := metric.SetEmptySum()
	sum.SetIsMonotonic(true)
	// Set cumulative temporality - values represent total since measurement started, not delta changes
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)

	dataPoint := sum.DataPoints().AppendEmpty()
	dataPoint.SetIntValue(int64(counter.Count))
	now := pcommon.NewTimestampFromTime(time.Now())
	dataPoint.SetTimestamp(now)
	dataPoint.SetStartTimestamp(now)
}

func (h *JVMMetricsHandler) addMemoryUsedMetric(scopeMetrics pmetric.ScopeMetrics, gauge GaugeValue, memType MemoryType, poolName MemoryPoolName) {
	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(metricMemoryUsedAfterGC)
	metric.SetDescription(DescMemoryUsed)
	metric.SetUnit(semconv1_26.JvmMemoryUsedAfterLastGcUnit)

	gaugeMetric := metric.SetEmptyGauge()
	dataPoint := gaugeMetric.DataPoints().AppendEmpty()
	dataPoint.SetIntValue(int64(gauge.Value))
	now := pcommon.NewTimestampFromTime(time.Now())
	dataPoint.SetTimestamp(now)

	// Add attributes
	attrs := dataPoint.Attributes()
	if memType != MemoryTypeUnknown {
		attrs.PutStr(string(attrMemoryType), memType.String())
	}
	if poolName != PoolNameUnknown {
		attrs.PutStr(string(attrMemoryPoolName), poolName.String())
	}
}

func (h *JVMMetricsHandler) addThreadCountMetric(scopeMetrics pmetric.ScopeMetrics, gauge GaugeValue, daemon ThreadDaemon, state ThreadState) {
	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(metricThreadCount)
	metric.SetDescription(DescThreadCount)
	metric.SetUnit(semconv1_26.JvmThreadCountUnit)

	gaugeMetric := metric.SetEmptyGauge()
	dataPoint := gaugeMetric.DataPoints().AppendEmpty()
	dataPoint.SetIntValue(int64(gauge.Value))
	now := pcommon.NewTimestampFromTime(time.Now())
	dataPoint.SetTimestamp(now)

	// Add attributes
	attrs := dataPoint.Attributes()
	if daemon != ThreadDaemonUnknown {
		attrs.PutStr(string(attrThreadDaemon), daemon.String())
	}
	if state != ThreadStateUnknown {
		attrs.PutStr(string(attrThreadState), state.String())
	}
}

func (h *JVMMetricsHandler) addGCHistogramMetric(scopeMetrics pmetric.ScopeMetrics, hist HistogramValue, gcAction GCAction, gcName GCName) {
	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(metricGCDuration)
	metric.SetDescription(DescGCDuration)
	metric.SetUnit(semconv1_26.JvmGcDurationUnit)

	histogramMetric := metric.SetEmptyHistogram()
	// Set cumulative temporality - histogram represents total observations since measurement started
	histogramMetric.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)

	dataPoint := histogramMetric.DataPoints().AppendEmpty()
	now := pcommon.NewTimestampFromTime(time.Now())
	dataPoint.SetTimestamp(now)
	dataPoint.SetStartTimestamp(now)

	// Set count and sum
	dataPoint.SetCount(uint64(hist.TotalCount))
	dataPoint.SetSum(float64(hist.SumNs) / 1e9) // Convert nanoseconds to seconds

	// Set bucket boundaries (in seconds) and counts
	// Bucket boundaries: 0.001s, 0.01s, 0.1s, 1.0s, +Inf
	bucketBounds := []float64{0.001, 0.01, 0.1, 1.0}
	bucketCounts := []uint64{
		uint64(hist.Bucket1ms),
		uint64(hist.Bucket1ms + hist.Bucket10ms),
		uint64(hist.Bucket1ms + hist.Bucket10ms + hist.Bucket100ms),
		uint64(hist.Bucket1ms + hist.Bucket10ms + hist.Bucket100ms + hist.Bucket1s),
		uint64(hist.TotalCount), // Total includes all buckets
	}

	dataPoint.ExplicitBounds().FromRaw(bucketBounds)
	dataPoint.BucketCounts().FromRaw(bucketCounts)

	// Add attributes
	attrs := dataPoint.Attributes()
	if gcAction != GCActionUnknown {
		attrs.PutStr(string(attrGCAction), gcAction.String())
	}
	if gcName != GCNameUnknown {
		attrs.PutStr(string(attrGCName), gcName.String())
	}

	h.logger.Debug("GC histogram recorded",
		zap.Uint32("total_count", hist.TotalCount),
		zap.Float64("sum_ms", float64(hist.SumNs)/1e6),
		zap.String("gc_action", gcAction.String()),
		zap.String("gc_name", gcName.String()),
	)
}
