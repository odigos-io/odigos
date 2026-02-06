package jvm

import (
	"context"
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
	metricMemoryUsed        = semconv1_26.JvmMemoryUsedName
	metricMemoryCommitted   = semconv1_26.JvmMemoryCommittedName
	metricMemoryLimit       = semconv1_26.JvmMemoryLimitName
	metricMemoryUsedAfterGC = semconv1_26.JvmMemoryUsedAfterLastGcName
	metricClassesLoaded     = semconv1_26.JvmClassLoadedName
	metricClassesUnloaded   = semconv1_26.JvmClassUnloadedName
	metricClassesCount      = semconv1_26.JvmClassCountName
	metricThreadCount       = semconv1_26.JvmThreadCountName
	metricCPUTime           = semconv1_26.JvmCPUTimeName
	metricCPUCount          = semconv1_26.JvmCPUCountName
	metricCPUUtilization    = semconv1_26.JvmCPURecentUtilizationName

	// New Relic metric names (process.runtime.jvm.* prefix) for multi-vendor compatibility
	processRuntimeJVMMetricMemoryUsage     = "process.runtime.jvm.memory.usage"
	processRuntimeJVMMetricMemoryCommitted = "process.runtime.jvm.memory.committed"
	processRuntimeJVMMetricMemoryLimit     = "process.runtime.jvm.memory.limit"
	processRuntimeJVMMetricMemoryMax       = "process.runtime.jvm.memory.max"

	// Grafana metric names (jvm.classes.* prefix) for multi-vendor compatibility
	grafanaMetricClassesLoaded   = "jvm.classes.loaded"
	grafanaMetricClassesUnloaded = "jvm.classes.unloaded"

	// OTel attribute keys
	attrGCAction       = semconv1_26.JvmGcActionKey
	attrGCName         = semconv1_26.JvmGcNameKey
	attrMemoryType     = semconv1_26.JvmMemoryTypeKey
	attrMemoryPoolName = semconv1_26.JvmMemoryPoolNameKey
	attrThreadDaemon   = semconv1_26.JvmThreadDaemonKey
	attrThreadState    = semconv1_26.JvmThreadStateKey

	// Metric descriptions
	descClassLoaded       = semconv1_26.JvmClassLoadedDescription
	descClassUnloaded     = semconv1_26.JvmClassUnloadedDescription
	descClassCount        = semconv1_26.JvmClassCountDescription
	descMemoryUsed        = semconv1_26.JvmMemoryUsedDescription
	descMemoryCommitted   = semconv1_26.JvmMemoryCommittedDescription
	descMemoryLimit       = semconv1_26.JvmMemoryLimitDescription
	descMemoryUsedAfterGC = semconv1_26.JvmMemoryUsedAfterLastGcDescription
	descThreadCount       = semconv1_26.JvmThreadCountDescription
	descGCDuration        = semconv1_26.JvmGcDurationDescription
	descCPUTime           = semconv1_26.JvmCPUTimeDescription
	descCPUCount          = semconv1_26.JvmCPUCountDescription
	descCPUUtilization    = semconv1_26.JvmCPURecentUtilizationDescription
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

// emitGaugeMetric is a helper that creates a gauge metric with the given parameters.
func (h *JVMMetricsHandler) emitGaugeMetric(
	scopeMetrics pmetric.ScopeMetrics,
	name string,
	description string,
	unit string,
	value int64,
	attrSetter func(pcommon.Map),
) {
	if value == 0 {
		return
	}

	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(name)
	metric.SetDescription(description)
	metric.SetUnit(unit)

	gaugeMetric := metric.SetEmptyGauge()
	dataPoint := gaugeMetric.DataPoints().AppendEmpty()
	dataPoint.SetIntValue(value)
	now := pcommon.NewTimestampFromTime(time.Now())
	dataPoint.SetTimestamp(now)

	if attrSetter != nil {
		attrSetter(dataPoint.Attributes())
	}
}

// emitHistogramMetric is a helper that creates a histogram metric with the given parameters.
func (h *JVMMetricsHandler) emitHistogramMetric(
	scopeMetrics pmetric.ScopeMetrics,
	name string,
	description string,
	unit string,
	hist HistogramValue,
	startTime pcommon.Timestamp,
	attrSetter func(pcommon.Map),
) {
	if hist.TotalCount == 0 {
		return
	}

	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(name)
	metric.SetDescription(description)
	metric.SetUnit(unit)

	histogramMetric := metric.SetEmptyHistogram()
	histogramMetric.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)

	dataPoint := histogramMetric.DataPoints().AppendEmpty()
	now := pcommon.NewTimestampFromTime(time.Now())
	dataPoint.SetTimestamp(now)
	dataPoint.SetStartTimestamp(startTime)

	dataPoint.SetCount(uint64(hist.TotalCount))
	dataPoint.SetSum(float64(hist.SumNs) / 1e9) // Convert nanoseconds to seconds

	// OTLP bucket_counts are per-bucket (non-cumulative).
	// The sum of bucket_counts must equal the Count field.
	bucketBounds := []float64{0.001, 0.01, 0.1, 1.0}
	bucketCounts := []uint64{
		uint64(hist.Bucket1ms),
		uint64(hist.Bucket10ms),
		uint64(hist.Bucket100ms),
		uint64(hist.Bucket1s),
		uint64(hist.BucketInf),
	}

	dataPoint.ExplicitBounds().FromRaw(bucketBounds)
	dataPoint.BucketCounts().FromRaw(bucketCounts)

	if attrSetter != nil {
		attrSetter(dataPoint.Attributes())
	}
}

// ExtractJVMMetricsFromInnerMap extracts JVM metrics from a process inner map and converts them to OpenTelemetry format.
// startTime is the timestamp when this process was first observed, used as StartTimestamp for cumulative metrics.
func (h *JVMMetricsHandler) ExtractJVMMetricsFromInnerMap(ctx context.Context, innerMap *ebpf.Map, startTime pcommon.Timestamp) (pmetric.Metrics, error) {
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
			h.addClassLoadedMetric(scopeMetrics, value.AsGauge())
		case MetricClassUnloaded:
			h.addClassUnloadedMetric(scopeMetrics, value.AsGauge())
		case MetricClassCount:
			h.addClassCountMetric(scopeMetrics, value.AsGauge())

		// Memory metrics - extract common attributes once
		case MetricMemoryUsed, MetricMemoryCommitted, MetricMemoryLimit, MetricMemoryUsedAfterGC:
			memType := MemoryType(key.Attr1())
			poolName := MemoryPoolName(key.Attr2())
			gauge := value.AsGauge()
			switch metricType {
			case MetricMemoryUsed:
				h.addMemoryUsedMetric(scopeMetrics, gauge, memType, poolName)
			case MetricMemoryCommitted:
				h.addMemoryCommittedMetric(scopeMetrics, gauge, memType, poolName)
			case MetricMemoryLimit:
				h.addMemoryLimitMetric(scopeMetrics, gauge, memType, poolName)
			case MetricMemoryUsedAfterGC:
				h.addMemoryUsedAfterGCMetric(scopeMetrics, gauge, memType, poolName)
			}

		case MetricGCDuration:
			gcAction := GCAction(key.Attr1())
			gcName := GCName(key.Attr2())
			h.addGCHistogramMetric(scopeMetrics, value.AsHistogram(), gcAction, gcName, startTime)
		case MetricThreadCount:
			daemon := ThreadDaemon(key.Attr1())
			state := ThreadState(key.Attr2())
			h.addThreadCountMetric(scopeMetrics, value.AsGauge(), daemon, state)

		// CPU metrics
		case MetricCPUTime:
			h.addCPUTimeMetric(scopeMetrics, value.AsCounter(), startTime)
		case MetricCPUCount:
			h.addCPUCountMetric(scopeMetrics, value.AsGauge())
		case MetricCPURecentUtilization:
			h.addCPUUtilizationMetric(scopeMetrics, value.AsGauge())
		default:
			h.logger.Warn("Unknown metric type", zap.Uint32("type", uint32(metricType)))
		}

		metricsAdded++
	}

	h.logger.Debug("JVM metrics extraction completed",
		zap.Int("ebpf_entries_found", entriesFound),
		zap.Int("metrics_added", metricsAdded),
		zap.Int("total_metrics_in_scope", scopeMetrics.Metrics().Len()))

	return metrics, nil
}

// setMemoryAttributes adds memory type and pool name attributes to the given pcommon.Map
func setMemoryAttributes(attrs pcommon.Map, memType MemoryType, poolName MemoryPoolName) {
	if memType != MemoryTypeUnknown {
		attrs.PutStr(string(attrMemoryType), memType.String())
	}
	if poolName != PoolNameUnknown {
		attrs.PutStr(string(attrMemoryPoolName), poolName.String())
	}
}

// setThreadAttributes adds thread daemon and state attributes to the given pcommon.Map
func setThreadAttributes(attrs pcommon.Map, daemon ThreadDaemon, state ThreadState) {
	if daemon != ThreadDaemonUnknown {
		attrs.PutStr(string(attrThreadDaemon), daemon.String())
	}
	if state != ThreadStateUnknown {
		attrs.PutStr(string(attrThreadState), state.String())
	}
}

// setGCAttributes adds GC action and name attributes to the given pcommon.Map
func setGCAttributes(attrs pcommon.Map, gcAction GCAction, gcName GCName) {
	if gcAction != GCActionUnknown {
		attrs.PutStr(string(attrGCAction), gcAction.String())
	}
	if gcName != GCNameUnknown {
		attrs.PutStr(string(attrGCName), gcName.String())
	}
}

func (h *JVMMetricsHandler) addClassLoadedMetric(scopeMetrics pmetric.ScopeMetrics, gauge GaugeValue) {
	h.emitGaugeMetric(scopeMetrics, metricClassesLoaded, descClassLoaded,
		semconv1_26.JvmClassLoadedUnit, int64(gauge.Value), nil)
	h.emitGaugeMetric(scopeMetrics, grafanaMetricClassesLoaded, descClassLoaded,
		semconv1_26.JvmClassLoadedUnit, int64(gauge.Value), nil)
}

func (h *JVMMetricsHandler) addClassUnloadedMetric(scopeMetrics pmetric.ScopeMetrics, gauge GaugeValue) {
	h.emitGaugeMetric(scopeMetrics, metricClassesUnloaded, descClassUnloaded,
		semconv1_26.JvmClassUnloadedUnit, int64(gauge.Value), nil)
	h.emitGaugeMetric(scopeMetrics, grafanaMetricClassesUnloaded, descClassUnloaded,
		semconv1_26.JvmClassUnloadedUnit, int64(gauge.Value), nil)
}

func (h *JVMMetricsHandler) addClassCountMetric(scopeMetrics pmetric.ScopeMetrics, gauge GaugeValue) {
	h.emitGaugeMetric(scopeMetrics, metricClassesCount, descClassCount,
		semconv1_26.JvmClassCountUnit, int64(gauge.Value), nil)
}

func (h *JVMMetricsHandler) addMemoryUsedMetric(scopeMetrics pmetric.ScopeMetrics, gauge GaugeValue, memType MemoryType, poolName MemoryPoolName) {
	if gauge.Value == 0 {
		return
	}

	attrSetter := func(attrs pcommon.Map) {
		setMemoryAttributes(attrs, memType, poolName)
	}

	h.emitGaugeMetric(scopeMetrics, metricMemoryUsed, descMemoryUsed, semconv1_26.JvmMemoryUsedUnit, int64(gauge.Value), attrSetter)

	h.emitGaugeMetric(scopeMetrics, processRuntimeJVMMetricMemoryUsage, descMemoryUsed, semconv1_26.JvmMemoryUsedUnit, int64(gauge.Value), attrSetter)
}

func (h *JVMMetricsHandler) addMemoryCommittedMetric(scopeMetrics pmetric.ScopeMetrics, gauge GaugeValue, memType MemoryType, poolName MemoryPoolName) {
	if gauge.Value == 0 {
		return
	}

	attrSetter := func(attrs pcommon.Map) {
		setMemoryAttributes(attrs, memType, poolName)
	}

	h.emitGaugeMetric(scopeMetrics, metricMemoryCommitted, descMemoryCommitted, semconv1_26.JvmMemoryCommittedUnit, int64(gauge.Value), attrSetter)

	h.emitGaugeMetric(scopeMetrics, processRuntimeJVMMetricMemoryCommitted, descMemoryCommitted, semconv1_26.JvmMemoryCommittedUnit, int64(gauge.Value), attrSetter)
}

func (h *JVMMetricsHandler) addMemoryLimitMetric(scopeMetrics pmetric.ScopeMetrics, gauge GaugeValue, memType MemoryType, poolName MemoryPoolName) {
	if gauge.Value == 0 {
		return
	}

	attrSetter := func(attrs pcommon.Map) {
		setMemoryAttributes(attrs, memType, poolName)
	}

	h.emitGaugeMetric(scopeMetrics, metricMemoryLimit, descMemoryLimit, semconv1_26.JvmMemoryLimitUnit, int64(gauge.Value), attrSetter)

	h.emitGaugeMetric(scopeMetrics, processRuntimeJVMMetricMemoryLimit, descMemoryLimit, semconv1_26.JvmMemoryLimitUnit, int64(gauge.Value), attrSetter)
	h.emitGaugeMetric(scopeMetrics, processRuntimeJVMMetricMemoryMax, descMemoryLimit, semconv1_26.JvmMemoryLimitUnit, int64(gauge.Value), attrSetter)
}

func (h *JVMMetricsHandler) addMemoryUsedAfterGCMetric(scopeMetrics pmetric.ScopeMetrics, gauge GaugeValue, memType MemoryType, poolName MemoryPoolName) {
	if gauge.Value == 0 {
		return
	}

	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(metricMemoryUsedAfterGC)
	metric.SetDescription(descMemoryUsedAfterGC)
	metric.SetUnit(semconv1_26.JvmMemoryUsedAfterLastGcUnit)

	gaugeMetric := metric.SetEmptyGauge()
	dataPoint := gaugeMetric.DataPoints().AppendEmpty()
	dataPoint.SetIntValue(int64(gauge.Value))
	now := pcommon.NewTimestampFromTime(time.Now())
	dataPoint.SetTimestamp(now)

	setMemoryAttributes(dataPoint.Attributes(), memType, poolName)
}

func (h *JVMMetricsHandler) addThreadCountMetric(scopeMetrics pmetric.ScopeMetrics, gauge GaugeValue, daemon ThreadDaemon, state ThreadState) {
	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(metricThreadCount)
	metric.SetDescription(descThreadCount)
	metric.SetUnit(semconv1_26.JvmThreadCountUnit)

	gaugeMetric := metric.SetEmptyGauge()
	dataPoint := gaugeMetric.DataPoints().AppendEmpty()
	dataPoint.SetIntValue(int64(gauge.Value))
	now := pcommon.NewTimestampFromTime(time.Now())
	dataPoint.SetTimestamp(now)

	setThreadAttributes(dataPoint.Attributes(), daemon, state)
}

func (h *JVMMetricsHandler) addGCHistogramMetric(scopeMetrics pmetric.ScopeMetrics, hist HistogramValue, gcAction GCAction, gcName GCName, startTime pcommon.Timestamp) {
	attrSetter := func(attrs pcommon.Map) {
		setGCAttributes(attrs, gcAction, gcName)
	}

	h.emitHistogramMetric(scopeMetrics, metricGCDuration, descGCDuration, semconv1_26.JvmGcDurationUnit, hist, startTime, attrSetter)

	h.logger.Debug("GC histogram recorded",
		zap.Uint32("total_count", hist.TotalCount),
		zap.Float64("sum_ms", float64(hist.SumNs)/1e6),
		zap.String("gc_action", gcAction.String()),
		zap.String("gc_name", gcName.String()),
	)
}

func (h *JVMMetricsHandler) addCPUTimeMetric(scopeMetrics pmetric.ScopeMetrics, counter CounterValue, startTime pcommon.Timestamp) {
	if counter.Count == 0 {
		return
	}

	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(metricCPUTime)
	metric.SetDescription(descCPUTime)
	metric.SetUnit(semconv1_26.JvmCPUTimeUnit)

	sum := metric.SetEmptySum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)

	dataPoint := sum.DataPoints().AppendEmpty()
	seconds := float64(counter.Count) / 1e9
	dataPoint.SetDoubleValue(seconds)
	now := pcommon.NewTimestampFromTime(time.Now())
	dataPoint.SetTimestamp(now)
	dataPoint.SetStartTimestamp(startTime)
}

func (h *JVMMetricsHandler) addCPUCountMetric(scopeMetrics pmetric.ScopeMetrics, gauge GaugeValue) {
	if gauge.Value == 0 {
		return
	}

	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(metricCPUCount)
	metric.SetDescription(descCPUCount)
	metric.SetUnit(semconv1_26.JvmCPUCountUnit)

	gaugeMetric := metric.SetEmptyGauge()
	dataPoint := gaugeMetric.DataPoints().AppendEmpty()
	dataPoint.SetIntValue(int64(gauge.Value))
	now := pcommon.NewTimestampFromTime(time.Now())
	dataPoint.SetTimestamp(now)
}

func (h *JVMMetricsHandler) addCPUUtilizationMetric(scopeMetrics pmetric.ScopeMetrics, gauge GaugeValue) {
	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(metricCPUUtilization)
	metric.SetDescription(descCPUUtilization)
	metric.SetUnit(semconv1_26.JvmCPURecentUtilizationUnit)

	gaugeMetric := metric.SetEmptyGauge()
	dataPoint := gaugeMetric.DataPoints().AppendEmpty()
	utilization := float64(gauge.Value) / 1e7 // normalize values as they were scaled up to store float values as uint in ebpf maps
	dataPoint.SetDoubleValue(utilization)
	now := pcommon.NewTimestampFromTime(time.Now())
	dataPoint.SetTimestamp(now)
}
