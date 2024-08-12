package collectormetrics

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/endpoints/common"
	k8sconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	traceSizeMetricName  = "processor_odigostrafficmetrics_trace_data_size"
	metricSizeMetricName = "processor_odigostrafficmetrics_metric_data_size"
	logSizeMetricName    = "processor_odigostrafficmetrics_log_data_size"
)

var (
	errNoSenderPod    = errors.New("no sender pod found in the resource attributes")
	errUnKnownSender = errors.New("unknown OTLP sender")
)

type trafficMetrics struct {
	// trace data sent in bytes, cumulative
	tracesDataSent  int64
	// log data sent in bytes, cumulative
	logsDataSent    int64
	// metric data sent in bytes, cumulative
	metricsDataSent int64

	// trace throughput in bytes/sec
	tracesThroughput  int64
	// log throughput in bytes/sec
	logsThroughput    int64
	// metric throughput in bytes/sec
	metricsThroughput int64

	// lastUpdate is the time when the last data was received which is relevant for the corresponding metrics.
	// The time is taken from the metric data point timestamp.
	lastUpdate time.Time
}

func (tm *trafficMetrics) TotalDataSent() int64 {
	return tm.tracesDataSent + tm.logsDataSent + tm.metricsDataSent
}

func (tm *trafficMetrics) TotalThroughput() int64 {
	return tm.tracesThroughput + tm.logsThroughput + tm.metricsThroughput
}

func (tm *trafficMetrics) String() string {
	return fmt.Sprintf("tracesDataSent: %d, logsDataSent: %d, metricsDataSent: %d, tracesThroughput: %d, logsThroughput: %d, metricsThroughput: %d, lastUpdate: %s",
		tm.tracesDataSent, tm.logsDataSent, tm.metricsDataSent, tm.tracesThroughput, tm.logsThroughput, tm.metricsThroughput, tm.lastUpdate.String())
}

type OdigosMetricsConsumer struct {
	sources      sourcesMetrics
	destinations destinationsMetrics
	deletedChan  chan deleteNotification
}

var (
	ServiceNameKey        = strings.ReplaceAll(string(semconv.ServiceNameKey), ".", "_")
	K8SNamespaceNameKey   = strings.ReplaceAll(string(semconv.K8SNamespaceNameKey), ".", "_")
	K8SDeploymentNameKey  = strings.ReplaceAll(string(semconv.K8SDeploymentNameKey), ".", "_")
	K8SStatefulSetNameKey = strings.ReplaceAll(string(semconv.K8SStatefulSetNameKey), ".", "_")
	K8SDaemonSetNameKey   = strings.ReplaceAll(string(semconv.K8SDaemonSetNameKey), ".", "_")
)

func (c *OdigosMetricsConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func newTrafficMetrics(metricName string, dp pmetric.NumberDataPoint) *trafficMetrics {
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

	return tm
}

func (c *OdigosMetricsConsumer) runNotificationsLoop(ctx context.Context) {
	for {
		select {
		case n, ok := <-c.deletedChan:
			if !ok {
				return
			}
			switch n.notificationType {
			case nodeCollector:
				c.sources.removeNodeCollector(n.object)
			case clusterCollector:
				c.destinations.removeClusterCollector(n.object)
			case destination:
				c.destinations.removeDestination(n.object)
			case source:
				c.sources.removeSource(n.sourceID)
			}
		case <-ctx.Done():
			return
		}
	}
}

func senderPodFromResource(md pmetric.Metrics) (string, error) {
	v, ok := md.ResourceMetrics().At(0).Resource().Attributes().Get(string(semconv.K8SPodNameKey))
	if !ok {
		return "", errNoSenderPod
	}

	return v.Str(), nil
}

func (c *OdigosMetricsConsumer) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	senderPod, err := senderPodFromResource(md)
	if err != nil {
		return err
	}
	
	if strings.HasPrefix(senderPod, k8sconsts.OdigosNodeCollectorDaemonSetName) {
		c.sources.handleNodeCollectorMetrics(senderPod, md)
		return nil
	}

	if strings.HasPrefix(senderPod, k8sconsts.OdigosClusterCollectorDeploymentName) {
		c.destinations.handleClusterCollectorMetrics(senderPod, md)
		return nil
	}

	return errUnKnownSender
}

func NewOdigosMetrics() *OdigosMetricsConsumer {
	return &OdigosMetricsConsumer{
		sources:      newSourcesMetrics(),
		destinations: newDestinationsMetrics(),
		deletedChan:  make(chan deleteNotification),
	}
}

// Run starts the OTLP receiver and the notifications loop for receiving and processing the metrics from different Odigos collectors
func (c *OdigosMetricsConsumer) Run(ctx context.Context, odigosNS string) {
	var closeWg sync.WaitGroup
	// launch the notifications loop
	closeWg.Add(1)
	go func() {
		defer closeWg.Done()
		c.runNotificationsLoop(ctx)
	}()

	// run a watcher for deletion detection
	closeWg.Add(1)
	go func() {
		defer closeWg.Done()
		err := runDeleteWatcher(ctx, &deleteWatcher{
			odigosNS:            odigosNS,
			deleteNotifications: c.deletedChan})
		if err != nil {
			log.Printf("Collector metrics: Error running delete watcher: %v\n", err)
		}
	}()

	// setup the OTLP receiver
	f := otlpreceiver.NewFactory()

	cfg, ok := f.CreateDefaultConfig().(*otlpreceiver.Config)
	if !ok {
		panic("failed to cast default config to otlpreceiver.Config")
	}

	cfg.GRPC.NetAddr.Endpoint = fmt.Sprintf("0.0.0.0:%d", consts.OTLPPort)

	r, err := f.CreateMetricsReceiver(ctx, receivertest.NewNopSettings(), cfg, c)
	if err != nil {
		panic("failed to create receiver")
	}

	r.Start(ctx, componenttest.NewNopHost())
	defer r.Shutdown(ctx)

	log.Println("OTLP receiver is running")
	<-ctx.Done()
	closeWg.Wait()
}

func (c *OdigosMetricsConsumer) GetSingleSourceMetrics(sID common.SourceID) (trafficMetrics, bool) {
	return c.sources.metricsByID(sID)
}

func (c *OdigosMetricsConsumer) GetSingleDestinationMetrics(dID string) (trafficMetrics, bool) {
	return c.destinations.metricsByID(dID)
}

func (c *OdigosMetricsConsumer) GetSourcesMetrics() map[common.SourceID]trafficMetrics {
	return c.sources.sourcesMetrics()
}

func (c *OdigosMetricsConsumer) GetDestinationsMetrics() map[string]trafficMetrics {
	return c.destinations.metrics()
}
