package collectormetrics

import (
	"strings"
	"sync"
	"time"

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

type destinationsMetrics struct {
	destinations   map[string]*singleDestinationMetrics
	destinationsMu sync.Mutex
	avgCalculator  *averageSizeCalculator
}

type destinationTrafficMetrics struct {
	trafficMetrics
	// total number of spans/metrics/logs sent by the corresponding exporter
	sentSpans, sentMetrics, sentLogs int64
}

type singleDestinationMetrics struct {
	// clusterCollectorsTraffic is a map of cluster collector IDs to their respective traffic metrics
	clusterCollectorsTraffic map[string]*destinationTrafficMetrics
	// mutex to protect the clusterCollectorsTraffic map, used when a cluster collector is added or deleted
	mu sync.Mutex
}

type averageSizeCalculator struct {
	// total number of spans/metrics/logs recorded in the last update as reported by odigos processor
	acceptedSpans, acceptedMetrics, acceptedLogs int64
	// total size of spans/metrics/logs recorded in the last update as reported by odigos processor
	spansDataSize, metricsDataSize, logsDataSize int64

	lastInterval struct {
		// calculated average size of spans/metrics/logs in the last interval
		avgSpanSize, avgMetricSize, avgLogSize float64
	}
}

func (asc *averageSizeCalculator) lastCalculatedAvgSpanSize() float64 {
	return asc.lastInterval.avgSpanSize
}

func (asc *averageSizeCalculator) lastCalculatedAvgMetricSize() float64 {
	return asc.lastInterval.avgMetricSize
}

func (asc *averageSizeCalculator) lastCalculatedAvgLogSize() float64 {
	return asc.lastInterval.avgLogSize
}

func (asc *averageSizeCalculator) update(acceptedSpans, acceptedMetrics, acceptedLogs, spansDataSize, metricsDataSize, logsDataSize int64) {
	var avgSpanSizeInInterval, avgMetricSizeInInterval, avgLogSizeInInterval float64
	acceptedSpanInInterval := acceptedSpans - asc.acceptedSpans
	acceptedMetricsInInterval := acceptedMetrics - asc.acceptedMetrics
	acceptedLogsInInterval := acceptedLogs - asc.acceptedLogs
	if acceptedSpans > asc.acceptedSpans {
		avgSpanSizeInInterval = float64(spansDataSize-asc.spansDataSize) / float64(acceptedSpanInInterval)
	}
	asc.lastInterval.avgSpanSize = avgSpanSizeInInterval
	asc.spansDataSize = spansDataSize
	asc.acceptedSpans = acceptedSpans

	if acceptedMetrics > asc.acceptedMetrics {
		avgMetricSizeInInterval = float64(metricsDataSize-asc.metricsDataSize) / float64(acceptedMetricsInInterval)
	}
	asc.lastInterval.avgMetricSize = avgMetricSizeInInterval
	asc.metricsDataSize = metricsDataSize
	asc.acceptedMetrics = acceptedMetrics

	if acceptedLogs > asc.acceptedLogs {
		avgLogSizeInInterval = float64(logsDataSize-asc.logsDataSize) / float64(acceptedLogsInInterval)
	}
	asc.lastInterval.avgLogSize = avgLogSizeInInterval
	asc.logsDataSize = logsDataSize
	asc.acceptedLogs = acceptedLogs
}

func newDestinationsMetrics() destinationsMetrics {
	return destinationsMetrics{
		destinations:  make(map[string]*singleDestinationMetrics),
		avgCalculator: &averageSizeCalculator{},
	}
}

func (dm *destinationsMetrics) removeClusterCollector(clusterCollectorID string) {
	for _, d := range dm.destinations {
		d.mu.Lock()
		delete(d.clusterCollectorsTraffic, clusterCollectorID)
		d.mu.Unlock()
	}
}

func (dm *destinationsMetrics) removeDestination(dID string) {
	dm.destinationsMu.Lock()
	delete(dm.destinations, dID)
	dm.destinationsMu.Unlock()
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

func (dm *destinationsMetrics) newDestinationTrafficMetrics(metricName string, numDataPoints float64, t time.Time) *destinationTrafficMetrics {
	tm := trafficMetrics{
		lastUpdate: t,
	}

	switch metricName {
	case exporterSentSpansMetricName:
		tm.tracesDataSent = int64(numDataPoints * dm.avgCalculator.lastCalculatedAvgSpanSize())
	case exporterSentMetricsMetricName:
		tm.metricsDataSent = int64(numDataPoints * dm.avgCalculator.lastCalculatedAvgMetricSize())
	case exporterSentLogsMetricName:
		tm.logsDataSent = int64(numDataPoints * dm.avgCalculator.lastCalculatedAvgLogSize())
	}

	dtm := &destinationTrafficMetrics{
		trafficMetrics: tm,
	}

	switch metricName {
	case exporterSentSpansMetricName:
		dtm.sentSpans = int64(numDataPoints)
	case exporterSentMetricsMetricName:
		dtm.sentMetrics = int64(numDataPoints)
	case exporterSentLogsMetricName:
		dtm.sentLogs = int64(numDataPoints)
	}

	return dtm
}

// newDestinationMetrics creates a new singleDestinationMetrics object with initial traffic metrics based on the data point received
// The clusterCollectorsTraffic map initialized with the cluster collector ID and the traffic metrics
func (dm *destinationsMetrics) newDestinationMetrics(dp pmetric.NumberDataPoint, metricName string, clusterCollectorID string) *singleDestinationMetrics {
	dtm := dm.newDestinationTrafficMetrics(metricName, dp.DoubleValue(), dp.Timestamp().AsTime())

	sm := &singleDestinationMetrics{
		clusterCollectorsTraffic: map[string]*destinationTrafficMetrics{
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
		// first time we receive data for this destination, create an entry for it with hte given clusterCollectorID
		dm.destinations[dID] = dm.newDestinationMetrics(dp, metricName, clusterCollectorID)
		return
	}

	currentVal.mu.Lock()
	defer currentVal.mu.Unlock()

	if _, ok = currentVal.clusterCollectorsTraffic[clusterCollectorID]; !ok {
		// first time we receive data for this destination and from this cluster collector
		currentVal.clusterCollectorsTraffic[clusterCollectorID] = dm.newDestinationTrafficMetrics(metricName, dp.DoubleValue(), dp.Timestamp().AsTime())
		return
	}

	// From this point on, we are updating the existing destination metrics
	var throughputPtr *int64
	var dataSentInInterval float64
	dtm := currentVal.clusterCollectorsTraffic[clusterCollectorID]

	// the metric data in 'dp' represent the number of spans/metrics/logs sent by the exporter
	// we use the average size of spans/metrics/logs to calculate the total data sent
	switch metricName {
	case exporterSentSpansMetricName:
		throughputPtr = &dtm.tracesThroughput
		spansInInterval := int64(dp.DoubleValue()) - dtm.sentSpans
		dataSentInInterval = float64(spansInInterval) * dm.avgCalculator.lastCalculatedAvgSpanSize()
		dtm.tracesDataSent += int64(dataSentInInterval)
		dtm.sentSpans = int64(dp.DoubleValue())
	case exporterSentMetricsMetricName:
		throughputPtr = &dtm.metricsThroughput
		metricsInInterval := int64(dp.DoubleValue()) - dtm.sentMetrics
		dataSentInInterval = float64(metricsInInterval) * dm.avgCalculator.lastCalculatedAvgMetricSize()
		dtm.metricsDataSent += int64(dataSentInInterval)
		dtm.sentMetrics = int64(dp.DoubleValue())
	case exporterSentLogsMetricName:
		throughputPtr = &dtm.logsThroughput
		logsInInterval := int64(dp.DoubleValue()) - dtm.sentLogs
		dataSentInInterval = float64(logsInInterval) * dm.avgCalculator.lastCalculatedAvgLogSize()
		dtm.logsDataSent += int64(dataSentInInterval)
		dtm.sentLogs = int64(dp.DoubleValue())
	}

	newTime := dp.Timestamp().AsTime()
	oldTime := dtm.lastUpdate
	dtm.lastUpdate = newTime

	if oldTime.IsZero() {
		// This is the first data point received for this destination and this metric
		// avoid calculating the throughput
		return
	}

	throughput := calculateThroughput(dataSentInInterval, newTime, oldTime)

	*throughputPtr = throughput
}

func (dm *destinationsMetrics) updateAverageEstimates(md pmetric.Metrics) {
	var (
		// number of spans/metrics/logs recorded in this snapshot
		acceptedSpans, acceptedMetrics, acceptedLogs int64
		// total size of spans/metrics/logs recorded in this snapshot
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

	// calculate the average size of spans/metrics/logs
	// by taking the data size difference between the last two updates and dividing it by the number of spans/metrics/logs accepted in the same period
	dm.avgCalculator.update(acceptedSpans, acceptedMetrics, acceptedLogs, spansDataSize, metricsDataSize, logsDataSize)
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

func (dm *destinationsMetrics) metricsByID(dID string) (trafficMetrics, bool) {
	sdm, ok := dm.destinations[dID]
	if !ok {
		return trafficMetrics{}, false
	}

	return sdm.metrics(), true
}

func (sdm *singleDestinationMetrics) metrics() trafficMetrics {
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

	return resultMetrics
}

func (dm *destinationsMetrics) metrics() map[string]trafficMetrics {
	dm.destinationsMu.Lock()
	defer dm.destinationsMu.Unlock()

	result := make(map[string]trafficMetrics, len(dm.destinations))
	for dID, sdm := range dm.destinations {
		result[dID] = sdm.metrics()
	}

	return result
}
