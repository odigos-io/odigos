package collectormetrics

import (
	"fmt"
	"strings"
	"sync"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

const (
	exporterSentSpansMetricName   = "otelcol_exporter_sent_spans"
	exporterSentMetricsMetricName = "otelcol_exporter_sent_metric_points"
	exporterSentLogsMetricName    = "otelcol_exporter_sent_log_records"

	processorAcceptedSpansMetricName   = "otelcol_processor_accepted_spans"
	processorAcceptedMetricsMetricName = "otelcol_processor_accepted_metric_points"
	processorAcceptedLogsMetricName    = "otelcol_processor_accepted_log_records"

	exporterMetricAttributesKey  = "exporter"
	processorMetricAttributesKey = "processor"
)

type singleDestinationMetrics struct {
	// clusterCollectorsTraffic is a map of cluster collector IDs to their respective traffic metrics
	clusterCollectorsTraffic map[string]*trafficMetrics
	// mutex to protect the clusterCollectorsTraffic map, used when a cluster collector is added or deleted
	mu sync.Mutex
}

type destinationsMetrics struct {
	destinations   map[string]*singleDestinationMetrics
	destinationsMu sync.Mutex

	avgSpanSize   float64
	avgMetricSize float64
	avgLogSize    float64
}

func newDestinationsMetrics() destinationsMetrics {
	return destinationsMetrics{
		destinations: make(map[string]*singleDestinationMetrics),
	}
}

func metricAttributesToDestinationID(attrs pcommon.Map) string {
	exporterName, ok := attrs.Get(exporterMetricAttributesKey)
	if !ok {
		return ""
	}

	exporterNameStr := exporterName.Str()

	inedx := strings.Index(exporterNameStr, "odigos.io.dest")
	if inedx == -1 {
		return ""
	}

	return exporterNameStr[inedx:]
}

func (dm *destinationsMetrics) newDestinationTrafficMetrics(metricName string, sentDp pmetric.NumberDataPoint) *trafficMetrics {
	tm := &trafficMetrics{
		lastUpdate: sentDp.Timestamp().AsTime(),
	}

	switch metricName {
	case exporterSentSpansMetricName:
		tm.tracesDataSent = int64(sentDp.DoubleValue() * dm.avgSpanSize)
	case exporterSentMetricsMetricName:
		tm.metricsDataSent = int64(sentDp.DoubleValue() * dm.avgMetricSize)
	case exporterSentLogsMetricName:
		tm.logsDataSent = int64(sentDp.DoubleValue() * dm.avgLogSize)
	}

	return tm
}

// newDestinationMetrics creates a new singleDestinationMetrics object with initial traffic metrics based on the data point received
// The clusterCollectorsTraffic map initialized with the cluster collector ID and the traffic metrics
func (dm *destinationsMetrics) newDestinationMetrics(dp pmetric.NumberDataPoint, metricName string, clusterCollectorID string) *singleDestinationMetrics {
	dtm := dm.newDestinationTrafficMetrics(metricName, dp)

	sm := &singleDestinationMetrics{
		clusterCollectorsTraffic: map[string]*trafficMetrics{
			clusterCollectorID: dtm,
		},
	}

	return sm
}

func (dm *destinationsMetrics) updateDestinationMetricsByExporter(dp pmetric.NumberDataPoint, metricName string, clusterCollectorID string) {
	dID := metricAttributesToDestinationID(dp.Attributes())
	if dID == "" {
		return
	}

	dm.destinationsMu.Lock()
	defer dm.destinationsMu.Unlock()
	currentVal, ok := dm.destinations[dID]
	if !ok {
		fmt.Printf("Creating new destination metrics for destination %s\n", dID)
		// first time we receive data for this destination, create an entry for it with hte given clusterCollectorID
		dm.destinations[dID] = dm.newDestinationMetrics(dp, metricName, clusterCollectorID)
		return
	}

	currentVal.mu.Lock()
	defer currentVal.mu.Unlock()

	if _, ok = currentVal.clusterCollectorsTraffic[clusterCollectorID]; !ok {
		// first time we receive data for this destination and from this cluster collector
		currentVal.clusterCollectorsTraffic[clusterCollectorID] = dm.newDestinationTrafficMetrics(metricName, dp)
		return
	}

	// From this point on, we are updating the existing destination metrics
	var dataSentPtr, throughputPtr *int64
	var newDataSent int64
	// the metric data in 'dp' represent the number of spans/metrics/logs sent by the exporter
	// we use the average size of spans/metrics/logs to calculate the total data sent
	switch metricName {
	case exporterSentSpansMetricName:
		dataSentPtr = &currentVal.clusterCollectorsTraffic[clusterCollectorID].tracesDataSent
		throughputPtr = &currentVal.clusterCollectorsTraffic[clusterCollectorID].tracesThroughput
		newDataSent = int64(dp.DoubleValue() * dm.avgSpanSize)
	case exporterSentMetricsMetricName:
		dataSentPtr = &currentVal.clusterCollectorsTraffic[clusterCollectorID].metricsDataSent
		throughputPtr = &currentVal.clusterCollectorsTraffic[clusterCollectorID].metricsThroughput
		newDataSent = int64(dp.DoubleValue() * dm.avgMetricSize)
	case exporterSentLogsMetricName:
		dataSentPtr = &currentVal.clusterCollectorsTraffic[clusterCollectorID].logsDataSent
		throughputPtr = &currentVal.clusterCollectorsTraffic[clusterCollectorID].logsThroughput
		newDataSent = int64(dp.DoubleValue() * dm.avgLogSize)
	}

	newTime := dp.Timestamp().AsTime()
	oldTime := currentVal.clusterCollectorsTraffic[clusterCollectorID].lastUpdate
	oldDataSent := *dataSentPtr

	*dataSentPtr = newDataSent
	currentVal.clusterCollectorsTraffic[clusterCollectorID].lastUpdate = newTime

	if oldTime.IsZero() {
		// This is the first data point received for this source and this metric
		// avoid calculating the throughput
		return
	}

	timeDiff := newTime.Sub(oldTime).Seconds()

	var throughput int64
	// calculate throughput only if the new value is greater than the old value and the time difference is positive
	// otherwise, the throughput is set to 0
	if newDataSent > oldDataSent && timeDiff > 0 {
		fmt.Printf("Updating throughput for destination %s. oldDataSent: %d, newDataSent: %d, timeDiff: %f\n", dID, oldDataSent, newDataSent, timeDiff)
		throughput = (newDataSent - oldDataSent) / int64(timeDiff)
		fmt.Printf("Throughput: %d\n", throughput)
	}

	*throughputPtr = throughput
}

func (dm *destinationsMetrics) updateAverageEstimates(md pmetric.Metrics) {
	var (
		acceptedSpans, acceptedMetrics, acceptedLogs int64
		spansDataSize, metricsDataSize, logsDataSize int64
	)

	rm := md.ResourceMetrics()
	for i := 0; i < rm.Len(); i++ {
		smSlice := rm.At(i).ScopeMetrics()
		for j := 0; j < smSlice.Len(); j++ {
			sm := smSlice.At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				m := sm.Metrics().At(k)
				switch m.Name() {
				// our processor is recording the number of spans/metrics/logs it accepted
				case processorAcceptedSpansMetricName, processorAcceptedMetricsMetricName, processorAcceptedLogsMetricName:
					for dataPointIndex := 0; dataPointIndex < m.Sum().DataPoints().Len(); dataPointIndex++ {
						dataPoint := m.Sum().DataPoints().At(dataPointIndex)
						processorName, ok := dataPoint.Attributes().Get(processorMetricAttributesKey)
						if ok && processorName.Str() == "odigostrafficmetrics" {
							switch m.Name() {
							case processorAcceptedSpansMetricName:
								acceptedSpans = int64(dataPoint.DoubleValue())
							case processorAcceptedMetricsMetricName:
								acceptedMetrics = int64(dataPoint.DoubleValue())
							case processorAcceptedLogsMetricName:
								acceptedLogs = int64(dataPoint.DoubleValue())
							}
						}
					}
				case traceSizeMetricName, metricSizeMetricName, logSizeMetricName:
					for dataPointIndex := 0; dataPointIndex < m.Sum().DataPoints().Len(); dataPointIndex++ {
						dataPoint := m.Sum().DataPoints().At(dataPointIndex)
						switch m.Name() {
						case traceSizeMetricName:
							spansDataSize = int64(dataPoint.DoubleValue())
						case metricSizeMetricName:
							metricsDataSize = int64(dataPoint.DoubleValue())
						case logSizeMetricName:
							logsDataSize = int64(dataPoint.DoubleValue())
						}
					}
				}
			}
		}
	}

	if acceptedSpans != 0 {
		dm.avgSpanSize = float64(spansDataSize / acceptedSpans)
		fmt.Printf("Updating average span size. spanDataSize: %d, acceptedSpans: %d, avgSpanSize: %f\n", spansDataSize, acceptedSpans, dm.avgSpanSize)
	}
	if acceptedMetrics != 0 {
		dm.avgMetricSize = float64(metricsDataSize / acceptedMetrics)
	}
	if acceptedLogs != 0 {
		dm.avgLogSize = float64(logsDataSize / acceptedLogs)
	}
}

func (dm *destinationsMetrics) handleClusterCollectorMetrics(senderPod string, md pmetric.Metrics) {
	dm.updateAverageEstimates(md)

	rm := md.ResourceMetrics()
	for i := 0; i < rm.Len(); i++ {
		smSlice := rm.At(i).ScopeMetrics()
		for j := 0; j < smSlice.Len(); j++ {
			sm := smSlice.At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				m := sm.Metrics().At(k)
				switch m.Name() {
				case exporterSentSpansMetricName, exporterSentLogsMetricName, exporterSentMetricsMetricName:
					for dataPointIndex := 0; dataPointIndex < m.Sum().DataPoints().Len(); dataPointIndex++ {
						dataPoint := m.Sum().DataPoints().At(dataPointIndex)
						dm.updateDestinationMetricsByExporter(dataPoint, m.Name(), senderPod)
					}
				}
			}
		}
	}
}

func (dm *destinationsMetrics) getDestinationTrafficMetrics(dID string) (trafficMetrics, bool) {
	sdm, ok := dm.destinations[dID]
	if !ok {
		return trafficMetrics{}, false
	}

	sdm.mu.Lock()
	defer sdm.mu.Unlock()

	resultMetrics := trafficMetrics{}
	// sum the traffic metrics from all the cluster collectors
	for _, tm := range sdm.clusterCollectorsTraffic {
		resultMetrics.tracesDataSent += tm.tracesDataSent
		resultMetrics.logsDataSent += tm.logsDataSent
		resultMetrics.metricsDataSent += tm.metricsDataSent

		resultMetrics.tracesThroughput += tm.tracesThroughput
		resultMetrics.logsThroughput += tm.logsThroughput
		resultMetrics.metricsThroughput += tm.metricsThroughput

		// use the latest update time among all the node collectors
		if tm.lastUpdate.After(resultMetrics.lastUpdate) {
			resultMetrics.lastUpdate = tm.lastUpdate
		}
	}

	return resultMetrics, true
}
