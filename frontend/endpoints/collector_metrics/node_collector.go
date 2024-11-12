package collectormetrics

import (
	"errors"
	"sync"

	"github.com/odigos-io/odigos/frontend/endpoints/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

type singleSourceMetrics struct {
	// nodeCollectorsTraffic is a map of node collector IDs to their respective traffic metrics
	// Each node collector reports the traffic metrics with source identifying attributes
	nodeCollectorsTraffic map[string]*trafficMetrics
	// mutex to protect the nodeCollectorsTraffic map, used when a node collector is added or deleted
	mu sync.Mutex
}

type sourcesMetrics struct {
	sourcesMap map[common.SourceID]*singleSourceMetrics
	sourcesMu  sync.Mutex
}

func newSourcesMetrics() sourcesMetrics {
	return sourcesMetrics{
		sourcesMap: make(map[common.SourceID]*singleSourceMetrics),
	}
}

// newSourceMetric creates a new singleSourceMetrics object with initial traffic metrics based on the data point received
// The sourceMetrics map initialized with the node collector ID and the traffic metrics
func newSourceMetric(dp pmetric.NumberDataPoint, metricName string, nodeCollectorID string) *singleSourceMetrics {
	tm := newTrafficMetrics(metricName, dp)

	sm := &singleSourceMetrics{
		nodeCollectorsTraffic: map[string]*trafficMetrics{
			nodeCollectorID: tm,
		},
	}

	return sm
}

func (sm *sourcesMetrics) updateSourceMetrics(dp pmetric.NumberDataPoint, metricName string, nodeCollectorID string) {
	sID, err := metricAttributesToSourceID(dp.Attributes())
	if err != nil {
		return
	}

	sm.sourcesMu.Lock()
	defer sm.sourcesMu.Unlock()
	currentVal, ok := sm.sourcesMap[sID]
	if !ok {
		// first time we receive data for this source, create an entry for it with hte given nodeCollectorID
		sm.sourcesMap[sID] = newSourceMetric(dp, metricName, nodeCollectorID)
		return
	}

	currentVal.mu.Lock()
	defer currentVal.mu.Unlock()

	if _, ok = currentVal.nodeCollectorsTraffic[nodeCollectorID]; !ok {
		// first time we receive data for this source and from this node collector
		currentVal.nodeCollectorsTraffic[nodeCollectorID] = newTrafficMetrics(metricName, dp)
		return
	}

	// From this point on, we are updating the existing source metrics
	var dataSentPtr, throughputPtr *int64
	switch metricName {
	case traceSizeMetricName:
		dataSentPtr = &currentVal.nodeCollectorsTraffic[nodeCollectorID].tracesDataSent
		throughputPtr = &currentVal.nodeCollectorsTraffic[nodeCollectorID].tracesThroughput
	case metricSizeMetricName:
		dataSentPtr = &currentVal.nodeCollectorsTraffic[nodeCollectorID].metricsDataSent
		throughputPtr = &currentVal.nodeCollectorsTraffic[nodeCollectorID].metricsThroughput
	case logSizeMetricName:
		dataSentPtr = &currentVal.nodeCollectorsTraffic[nodeCollectorID].logsDataSent
		throughputPtr = &currentVal.nodeCollectorsTraffic[nodeCollectorID].logsThroughput
	}

	newVal := int64(dp.DoubleValue())
	newTime := dp.Timestamp().AsTime()
	oldTime := currentVal.nodeCollectorsTraffic[nodeCollectorID].lastUpdate
	oldVal := *dataSentPtr

	*dataSentPtr = newVal
	currentVal.nodeCollectorsTraffic[nodeCollectorID].lastUpdate = newTime

	if oldTime.IsZero() {
		// This is the first data point received for this source and this metric
		// avoid calculating the throughput
		return
	}

	throughput := calculateThroughput(float64(newVal-oldVal), newTime, oldTime)

	*throughputPtr = throughput
}

func (sourceMetrics *sourcesMetrics) handleNodeCollectorMetrics(senderPod string, md pmetric.Metrics) {
	rm := md.ResourceMetrics()
	for i := 0; i < rm.Len(); i++ {
		smSlice := rm.At(i).ScopeMetrics()
		for j := 0; j < smSlice.Len(); j++ {
			sm := smSlice.At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				m := sm.Metrics().At(k)
				switch m.Name() {
				case traceSizeMetricName, metricSizeMetricName, logSizeMetricName:
					for dataPointIndex := 0; dataPointIndex < m.Sum().DataPoints().Len(); dataPointIndex++ {
						dataPoint := m.Sum().DataPoints().At(dataPointIndex)
						sourceMetrics.updateSourceMetrics(dataPoint, m.Name(), senderPod)
					}
				}
			}
		}
	}
}

func (sourceMetrics *sourcesMetrics) removeNodeCollector(nodeCollectorID string) {
	for _, sm := range sourceMetrics.sourcesMap {
		sm.mu.Lock()
		delete(sm.nodeCollectorsTraffic, nodeCollectorID)
		sm.mu.Unlock()
	}
}

func (sourceMetrics *sourcesMetrics) removeSource(sID common.SourceID) {
	sourceMetrics.sourcesMu.Lock()
	delete(sourceMetrics.sourcesMap, sID)
	sourceMetrics.sourcesMu.Unlock()
}

func (sourceMetrics *sourcesMetrics) metricsByID(sID common.SourceID) (trafficMetrics, bool) {
	sm, ok := sourceMetrics.sourcesMap[sID]
	if !ok {
		return trafficMetrics{}, false
	}

	return sm.metrics(), true
}

func (ssm *singleSourceMetrics) metrics() trafficMetrics {
	ssm.mu.Lock()
	defer ssm.mu.Unlock()

	resultMetrics := trafficMetrics{}
	// sum the traffic metrics from all the node collectors
	for _, tm := range ssm.nodeCollectorsTraffic {
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

func (sourcesMetrics *sourcesMetrics) sourcesMetrics() map[common.SourceID]trafficMetrics {
	sourcesMetrics.sourcesMu.Lock()
	defer sourcesMetrics.sourcesMu.Unlock()

	result := make(map[common.SourceID]trafficMetrics)
	for sID, ssm := range sourcesMetrics.sourcesMap {
		result[sID] = ssm.metrics()
	}

	return result
}

func metricAttributesToSourceID(attrs pcommon.Map) (common.SourceID, error) {
	name, ok := attrs.Get(ServiceNameKey)
	if !ok {
		return common.SourceID{}, errors.New("service name not found")
	}

	ns, ok := attrs.Get(K8SNamespaceNameKey)
	if !ok {
		return common.SourceID{}, errors.New("namespace not found")
	}

	var kind workload.WorkloadKind
	if _, ok := attrs.Get(K8SDeploymentNameKey); ok {
		kind = workload.WorkloadKindDeployment
	} else if _, ok := attrs.Get(K8SStatefulSetNameKey); ok {
		kind = workload.WorkloadKindStatefulSet
	} else if _, ok := attrs.Get(K8SDaemonSetNameKey); ok {
		kind = workload.WorkloadKindDaemonSet
	} else {
		return common.SourceID{}, errors.New("kind not found")
	}

	return common.SourceID{
		Name:      name.Str(),
		Namespace: ns.Str(),
		Kind:      kind,
	}, nil
}
