package collectormetrics

import (
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

const (
	// These metrics are added to each exporter. And are provided by the collector exporterhelper.
	// These are not stable and may change in the future.
	// Using these metrics, we can estimate the amount of data sent by each exporter.
	exporterSentSpansMetricName   = "otelcol_exporter_sent_spans_total"
	exporterSentMetricsMetricName = "otelcol_exporter_sent_metric_points_total"
	exporterSentLogsMetricName    = "otelcol_exporter_sent_log_records_total"

	// This metric is added by the service graph exporter.
	// It is used to estimate the number of service graph requests and build the service graph.
	serviceGraphRequestMetricName = "servicegraph_traces_service_graph_request_total"

	// Each metrics from the exporters has this attribute which is the name of the exporter.
	exporterMetricAttributesKey = "exporter"

	// These metrics are added by the odigostrafficmetrics processor.
	// Each processor comes with `otelcol_processor_incoming_items` and `otelcol_processor_outgoing_items` metrics.
	// but since we need our processor anyway, we use our custom metrics from it to reduce the handling of breaking changes by the collector.
	// These metrics are used to estimate the average size of spans/metrics/logs.
	processorAcceptedSpansMetricName   = "otelcol_odigos_accepted_spans_total"
	processorAcceptedMetricsMetricName = "otelcol_odigos_accepted_metric_points_total"
	processorAcceptedLogsMetricName    = "otelcol_odigos_accepted_log_records_total"
)

type clusterCollectorMetrics struct {
	destinations   map[string]*singleDestinationMetrics
	destinationsMu sync.Mutex
	avgCalculator  *averageSizeCalculator
	serviceGraph   *ServiceGraph
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

func newClusterCollectorMetrics() clusterCollectorMetrics {
	return clusterCollectorMetrics{
		destinations:  make(map[string]*singleDestinationMetrics),
		avgCalculator: &averageSizeCalculator{},
		serviceGraph:  newServiceGraph(),
	}
}

func (dm *clusterCollectorMetrics) removeClusterCollector(clusterCollectorID string) {
	for _, d := range dm.destinations {
		d.mu.Lock()
		delete(d.clusterCollectorsTraffic, clusterCollectorID)
		d.mu.Unlock()
	}
}

func (dm *clusterCollectorMetrics) removeDestination(dID string) {
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

func (dm *clusterCollectorMetrics) newDestinationTrafficMetrics(metricName string, numDataPoints float64, t time.Time) *destinationTrafficMetrics {
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
func (dm *clusterCollectorMetrics) newDestinationMetrics(dp pmetric.NumberDataPoint, metricName string, clusterCollectorID string) *singleDestinationMetrics {
	dtm := dm.newDestinationTrafficMetrics(metricName, dp.DoubleValue(), dp.Timestamp().AsTime())

	sm := &singleDestinationMetrics{
		clusterCollectorsTraffic: map[string]*destinationTrafficMetrics{
			clusterCollectorID: dtm,
		},
	}

	return sm
}

func (dm *clusterCollectorMetrics) updateDestinationMetricsByExporter(dp pmetric.NumberDataPoint, metricName string, clusterCollectorID string) {
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

func (dm *clusterCollectorMetrics) updateAverageEstimates(md pmetric.Metrics) {
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
						switch m.Name() {
						case processorAcceptedSpansMetricName:
							acceptedSpans = int64(dataPoint.DoubleValue())
						case processorAcceptedMetricsMetricName:
							acceptedMetrics = int64(dataPoint.DoubleValue())
						case processorAcceptedLogsMetricName:
							acceptedLogs = int64(dataPoint.DoubleValue())
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

func (dm *clusterCollectorMetrics) handleClusterCollectorMetrics(senderPod string, md pmetric.Metrics) {
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
				case serviceGraphRequestMetricName:
					for d := 0; d < m.Sum().DataPoints().Len(); d++ {
						dp := m.Sum().DataPoints().At(d)
						dm.serviceGraph.UpdateFromDataPoint(dp)
					}
				}
			}
		}
	}
}

func (dm *clusterCollectorMetrics) metricsByID(dID string) (trafficMetrics, bool) {
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

// ServiceGraphEdges returns a snapshot of the service graph edges
// It is used to build the service graph in the UI
// example of the structure of the result:
// map[
//   coupon: map[
//     membership:        {0 2025-06-24 16:16:39.384 +0000 UTC}
//   ]

//   frontend: map[
//     coupon:            {0 2025-06-24 16:16:39.384 +0000 UTC}
//     currency:          {0 2025-06-24 16:16:39.384 +0000 UTC}
//     geolocation:       {0 2025-06-24 16:16:39.384 +0000 UTC}
//     inventory:         {0 2025-06-24 16:16:39.384 +0000 UTC}
//     pricing:           {0 2025-06-24 16:16:39.384 +0000 UTC}
//   ]

//   prometheus-server: map[
//     prometheus-kube-state-metrics:      {181 2025-06-25 06:01:07.771 +0000 UTC}
//     prometheus-prometheus-node-exporter:{181 2025-06-25 06:01:07.771 +0000 UTC}
//     prometheus-prometheus-pushgateway:  {181 2025-06-25 06:01:07.771 +0000 UTC}
//     prometheus-server:                  {181 2025-06-25 06:01:07.771 +0000 UTC}
//   ]

//   user: map[
//     frontend:                          {0 2025-06-24 16:16:39.384 +0000 UTC}
//     prometheus-server:                {2814 2025-06-25 06:01:07.771 +0000 UTC}
//   ]
// ]

func (ccm *clusterCollectorMetrics) serviceGraphEdges() map[string]map[string]ServiceGraphEdge {
	ccm.serviceGraph.mu.Lock()
	defer ccm.serviceGraph.mu.Unlock()

	result := make(map[string]map[string]ServiceGraphEdge, len(ccm.serviceGraph.edges))
	for client, servers := range ccm.serviceGraph.edges {
		result[client] = make(map[string]ServiceGraphEdge, len(servers))
		for server, edge := range servers {
			result[client][server] = *edge
		}
	}

	return result
}

func (dm *clusterCollectorMetrics) destinationsMetrics() map[string]trafficMetrics {
	dm.destinationsMu.Lock()
	defer dm.destinationsMu.Unlock()

	result := make(map[string]trafficMetrics, len(dm.destinations))
	for dID, sdm := range dm.destinations {
		result[dID] = sdm.metrics()
	}

	return result
}

type ServiceGraph struct {
	mu    sync.Mutex
	edges map[string]map[string]*ServiceGraphEdge // client → server → edge
}

type ServiceGraphEdge struct {
	RequestCount int64
	LastUpdated  time.Time
}

func newServiceGraph() *ServiceGraph {
	return &ServiceGraph{
		edges: make(map[string]map[string]*ServiceGraphEdge),
	}
}

func (sg *ServiceGraph) UpdateFromDataPoint(dp pmetric.NumberDataPoint) {
	attrs := dp.Attributes().AsRaw()

	client, ok1 := attrs["client"].(string)
	server, ok2 := attrs["server"].(string)

	if !ok1 || !ok2 || client == "unknown" || server == "unknown" {
		return
	}

	val := int64(0)
	switch dp.ValueType() {
	case pmetric.NumberDataPointValueTypeInt:
		val = dp.IntValue()
	case pmetric.NumberDataPointValueTypeDouble:
		val = int64(dp.DoubleValue())
	}

	timestamp := dp.Timestamp().AsTime()

	sg.mu.Lock()
	defer sg.mu.Unlock()

	if _, ok := sg.edges[client]; !ok {
		sg.edges[client] = make(map[string]*ServiceGraphEdge)
	}

	edge, exists := sg.edges[client][server]
	if !exists {
		sg.edges[client][server] = &ServiceGraphEdge{
			RequestCount: val,
			LastUpdated:  timestamp,
		}
	} else {
		// always overwrite because it's a cumulative counter
		edge.RequestCount = val
		edge.LastUpdated = timestamp
	}
}
