package collectormetrics

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/endpoints/common"
	k8sconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"google.golang.org/grpc/metadata"
)

const (
	traceSizeMetricName  = "otelcol_processor_odigostrafficmetrics_trace_data_size"
	metricSizeMetricName = "otelcol_processor_odigostrafficmetrics_metric_data_size"
	logSizeMetricName    = "otelcol_processor_odigostrafficmetrics_log_data_size"
)

var (
	errNoMetadata    = errors.New("no metadata found in context")
	errUnKnownSender = errors.New("unknown OTLP sender")
)

type trafficMetrics struct {
	tracesDataSent  int64
	logsDataSent    int64
	metricsDataSent int64

	tracesThroughput  int64
	logsThroughput    int64
	metricsThroughput int64

	// lastUpdate is the time when the last data was received which is relevant for the corresponding metrics.
	// The time is taken from the metric data point timestamp.
	lastUpdate time.Time
}

type sourceMetrics struct {
	// nodeCollectorsTraffic is a map of node collector IDs to their respective traffic metrics
	// Each node collector reports the traffic metrics with source identifying attributes
	nodeCollectorsTraffic map[string]*trafficMetrics
}

type odigosMetricsConsumer struct {
	sourcesMetrics map[common.SourceID]*sourceMetrics
    sourcesMu sync.Mutex
}

// singleton instance of odigosMetricsConsumer
var odigosMetrics *odigosMetricsConsumer

var (
	ServiceNameKey        = strings.ReplaceAll(string(semconv.ServiceNameKey), ".", "_")
	K8SNamespaceNameKey   = strings.ReplaceAll(string(semconv.K8SNamespaceNameKey), ".", "_")
	K8SDeploymentNameKey  = strings.ReplaceAll(string(semconv.K8SDeploymentNameKey), ".", "_")
	K8SStatefulSetNameKey = strings.ReplaceAll(string(semconv.K8SStatefulSetNameKey), ".", "_")
	K8SDaemonSetNameKey   = strings.ReplaceAll(string(semconv.K8SDaemonSetNameKey), ".", "_")
)

func (c *odigosMetricsConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// newSourceMetric creates a new sourceMetrics object with initial traffic metrics based on the data point received
// The sourceMetrics map initialized with the node collector ID and the traffic metrics
func newSourceMetric(dp pmetric.NumberDataPoint, metricName string, nodeCollectorID string) *sourceMetrics {
	tm := &trafficMetrics{
		lastUpdate: dp.Timestamp().AsTime(),
	}

	switch metricName {
	case traceSizeMetricName:
		tm.tracesDataSent = int64(dp.DoubleValue())
	case metricSizeMetricName:
		tm.metricsDataSent = int64(dp.DoubleValue())
	case logSizeMetricName:
		tm.logsDataSent = int64(dp.DoubleValue())
	}

	sm := &sourceMetrics{
		nodeCollectorsTraffic: map[string]*trafficMetrics{
			nodeCollectorID: tm,
		},
	}

	return sm
}

func (c *odigosMetricsConsumer) updateSourceMetrics(dp pmetric.NumberDataPoint, metricName string, nodeCollectorID string) {
	sID, err := metricAttributesToSourceID(dp.Attributes())
	if err != nil {
		fmt.Printf("failed to get source ID: %v\n", err)
		return
	}

	defer func () {
		sm, ok := GetSourceTrafficMetrics(sID)
		if ok {
			fmt.Printf("######## source metrics: %v\n", sm)
		}
    } ()

	c.sourcesMu.Lock()
	defer c.sourcesMu.Unlock()
	currentVal, ok := c.sourcesMetrics[sID]
	if !ok {
		c.sourcesMetrics[sID] = newSourceMetric(dp, metricName, nodeCollectorID)
		return
	}

	// TODO: check this commented code which is used when a new node collector is discovered for an existing source
	// _, ok = currentVal.nodeCollectorsTraffic[nodeCollectorID]
	// if !ok {
	// 	currentVal.nodeCollectorsTraffic[nodeCollectorID] = &trafficMetrics{}
	// 	return
	// }

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

	timeDiff := newTime.Sub(oldTime).Seconds()

	var throughput int64
	if newVal > oldVal && timeDiff > 0 {
		// calculate throughput only if the new value is greater than the old value
		throughput = (newVal - oldVal) / int64(timeDiff)
	}

	*throughputPtr = throughput
}

func (c *odigosMetricsConsumer) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	grpcMD, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return errNoMetadata
	}

	senderPods, ok := grpcMD[k8sconsts.OdigosPodNameHeaderKey]
	if !ok {
		return errUnKnownSender
	}

	if len(senderPods) != 1 {
		return errUnKnownSender
	}

	senderPod := senderPods[0]
	if strings.HasPrefix(senderPod, k8sconsts.OdigosNodeCollectorDaemonSetName) {
		c.handleNodeCollectorMetrics(senderPod, md)
		return nil
	}

	return nil
}

func (c *odigosMetricsConsumer) handleNodeCollectorMetrics(senderPod string, md pmetric.Metrics) {
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
						c.updateSourceMetrics(dataPoint, m.Name(), senderPod)
					}
				}
			}
		}
	}
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

	var kind string
	if _, ok := attrs.Get(K8SDeploymentNameKey); ok {
		kind = "Deployment"
	} else if _, ok := attrs.Get(K8SStatefulSetNameKey); ok {
		kind = "StatefulSet"
	} else if _, ok := attrs.Get(K8SDaemonSetNameKey); ok {
		kind = "DaemonSet"
	} else {
		return common.SourceID{}, errors.New("kind not found")
	}

	return common.SourceID{
		Name:      name.Str(),
		Namespace: ns.Str(),
		Kind:      kind,
	}, nil
}

// SetupOTLPReceiver sets up an OTLP receiver to receive metrics from Collectors.
// It should only be called once.
func SetupOTLPReceiver(ctx context.Context) {
	f := otlpreceiver.NewFactory()

	cfg, ok := f.CreateDefaultConfig().(*otlpreceiver.Config)
	if !ok {
		panic("failed to cast default config to otlpreceiver.Config")
	}

	cfg.GRPC.NetAddr.Endpoint = fmt.Sprintf("0.0.0.0:%d", consts.OTLPPort)

	odigosMetrics = &odigosMetricsConsumer{
		sourcesMetrics: make(map[common.SourceID]*sourceMetrics),
	}

	r, err := f.CreateMetricsReceiver(ctx, receivertest.NewNopSettings(), cfg, odigosMetrics)
	if err != nil {
		panic("failed to create receiver")
	}

	r.Start(ctx, componenttest.NewNopHost())
	defer r.Shutdown(ctx)

	fmt.Print("OTLP Receiver is running\n")
	<-ctx.Done()

}

func GetSourceTrafficMetrics(sID common.SourceID) (trafficMetrics, bool) {
	odigosMetrics.sourcesMu.Lock()
	defer odigosMetrics.sourcesMu.Unlock()

	sm, ok := odigosMetrics.sourcesMetrics[sID]
	if !ok {
		return trafficMetrics{}, false
	}

	resultMetrics := trafficMetrics{}
	// sum the traffic metrics from all the node collectors
	for _, tm := range sm.nodeCollectorsTraffic {
		resultMetrics.tracesDataSent += tm.tracesDataSent
		resultMetrics.logsDataSent += tm.logsDataSent
		resultMetrics.metricsDataSent += tm.metricsDataSent

		resultMetrics.tracesThroughput += tm.tracesThroughput
		resultMetrics.logsThroughput += tm.logsThroughput
		resultMetrics.metricsThroughput += tm.metricsThroughput

		if tm.lastUpdate.After(resultMetrics.lastUpdate) {
			resultMetrics.lastUpdate = tm.lastUpdate
		}
	}

	return resultMetrics, true
}
