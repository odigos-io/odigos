package collectormetrics

import (
	"errors"
	"sync"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/services/common"
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

func (sm *sourcesMetrics) updateSourceMetrics(dp pmetric.NumberDataPoint, metricName string, nodeCollectorID string) {
	sID, err := metricAttributesToSourceID(dp.Attributes())
	if err != nil {
		return
	}

	sm.sourcesMu.Lock()
	defer sm.sourcesMu.Unlock()
	currentVal, ok := sm.sourcesMap[sID]
	if !ok {
		// this source is not tracked, this can happen if:
		// 1) a source has been deleted, and the collectors keep reporting its metrics (those metrics refer to the old deleted source and won't update)
		// or
		// 2) this is a new source and we didn't receive the watch event for it's creation yet
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

func (sourcesMetrics *sourcesMetrics) addSource(sID common.SourceID) {
	sourcesMetrics.sourcesMu.Lock()
	defer sourcesMetrics.sourcesMu.Unlock()

	sourcesMetrics.sourcesMap[sID] = &singleSourceMetrics{
		nodeCollectorsTraffic: make(map[string]*trafficMetrics),
	}
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
	ns, ok := attrs.Get(K8SNamespaceNameKey)
	if !ok {
		return common.SourceID{}, errors.New("namespace not found")
	}

	var kind k8sconsts.WorkloadKind
	var name pcommon.Value
	var found bool

	// Check for workload name by odigos custom resource attribute if present
	if odigosWorkloadName, ok := attrs.Get(OdigosWorkloadNameAttribute); ok {
		name, found = odigosWorkloadName, true
	}

	// Check for Odigos-specific workload kind attribute first
	// This is needed to distinguish between workloads that share the same semconv key
	// (e.g., DeploymentConfig uses k8s.deployment.name)
	if odigosKind, ok := attrs.Get(OdigosWorkloadKindAttribute); ok {
		kind = k8sconsts.WorkloadKind(odigosKind.Str())

		if !found {
			switch kind {
			case k8sconsts.WorkloadKindDeployment:
				name, found = attrs.Get(K8SDeploymentNameKey)
			case k8sconsts.WorkloadKindStatefulSet:
				name, found = attrs.Get(K8SStatefulSetNameKey)
			case k8sconsts.WorkloadKindDaemonSet:
				name, found = attrs.Get(K8SDaemonSetNameKey)
			case k8sconsts.WorkloadKindCronJob:
				name, found = attrs.Get(K8SCronJobNameKey)
			case k8sconsts.WorkloadKindJob:
				name, found = attrs.Get(K8SJobNameKey)
			case k8sconsts.WorkloadKindArgoRollout:
				name, found = attrs.Get(K8SRolloutNameKey)
			}
		}

		if !found {
			return common.SourceID{}, errors.New("workload name not found")
		}
	} else {
		// Fallback to legacy behavior for backwards compatibility
		if depName, ok := attrs.Get(K8SDeploymentNameKey); ok {
			kind = k8sconsts.WorkloadKindDeployment
			name = depName
		} else if ssName, ok := attrs.Get(K8SStatefulSetNameKey); ok {
			kind = k8sconsts.WorkloadKindStatefulSet
			name = ssName
		} else if dsName, ok := attrs.Get(K8SDaemonSetNameKey); ok {
			kind = k8sconsts.WorkloadKindDaemonSet
			name = dsName
		} else if cjName, ok := attrs.Get(K8SCronJobNameKey); ok {
			kind = k8sconsts.WorkloadKindCronJob
			name = cjName
		} else if jobName, ok := attrs.Get(K8SJobNameKey); ok {
			kind = k8sconsts.WorkloadKindJob
			name = jobName
		} else if rolloutName, ok := attrs.Get(K8SRolloutNameKey); ok {
			kind = k8sconsts.WorkloadKindArgoRollout
			name = rolloutName
		} else {
			return common.SourceID{}, errors.New("kind not found")
		}
	}

	return common.SourceID{
		Name:      name.Str(),
		Namespace: ns.Str(),
		Kind:      kind,
	}, nil
}
