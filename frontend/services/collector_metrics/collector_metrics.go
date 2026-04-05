package collectormetrics

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common/consts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/frontend/services/common"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/receiver/xreceiver"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"k8s.io/apimachinery/pkg/watch"
)

const (
	traceSizeMetricName  = "otelcol_odigos_trace_data_size_bytes_total"
	metricSizeMetricName = "otelcol_odigos_metric_data_size_bytes_total"
	logSizeMetricName    = "otelcol_odigos_log_data_size_bytes_total"
)

var (
	errNoSenderPod     = errors.New("no sender pod found in the resource attributes")
	errNoCollectorRole = errors.New("no collector role found in the resource attributes")
	errUnKnownSender   = errors.New("unknown OTLP sender")
)

type trafficMetrics struct {
	// trace data sent in bytes, cumulative
	tracesDataSent int64
	// log data sent in bytes, cumulative
	logsDataSent int64
	// metric data sent in bytes, cumulative
	metricsDataSent int64

	// trace throughput in bytes/sec
	tracesThroughput int64
	// log throughput in bytes/sec
	logsThroughput int64
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
	sources                 sourcesMetrics
	clusterCollectorMetrics clusterCollectorMetrics
	deletedChan             chan notification
}

var (
	K8SNamespaceNameKey   = string(semconv.K8SNamespaceNameKey)
	K8SDeploymentNameKey  = string(semconv.K8SDeploymentNameKey)
	K8SStatefulSetNameKey = string(semconv.K8SStatefulSetNameKey)
	K8SDaemonSetNameKey   = string(semconv.K8SDaemonSetNameKey)
	K8SCronJobNameKey     = string(semconv.K8SCronJobNameKey)
	K8SJobNameKey         = string(semconv.K8SJobNameKey)
	K8SRolloutNameKey     = k8sconsts.K8SArgoRolloutNameAttribute // Argo Rollout custom attribute - no semconv for it

	OdigosWorkloadNameAttribute = consts.OdigosWorkloadNameAttribute
	OdigosWorkloadKindAttribute = consts.OdigosWorkloadKindAttribute
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
				c.clusterCollectorMetrics.removeClusterCollector(n.object)
			case destination:
				c.clusterCollectorMetrics.removeDestination(n.object)
			case source:
				switch n.eventType {
				case watch.Deleted:
					c.sources.removeSource(n.sourceID)
				case watch.Added:
					c.sources.addSource(n.sourceID)
				default:
					fmt.Println("Unknown event type in metrics notification loop")
				}

			}
		case <-ctx.Done():
			return
		}
	}
}

func collectorRoleFromResource(md pmetric.Metrics) (k8sconsts.CollectorRole, error) {
	v, ok := md.ResourceMetrics().At(0).Resource().Attributes().Get("odigos.collector.role")
	if !ok {
		return "", errNoCollectorRole
	}

	return k8sconsts.CollectorRole(v.Str()), nil
}

func getSenderPod(md pmetric.Metrics) (string, error) {
	v, ok := md.ResourceMetrics().At(0).Resource().Attributes().Get(string(semconv.K8SPodNameKey))
	if !ok {
		return "", errNoSenderPod
	}

	return v.Str(), nil
}

func (c *OdigosMetricsConsumer) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	collectorRole, err := collectorRoleFromResource(md)
	if err != nil {
		return err
	}
	senderPod, err := getSenderPod(md)
	if err != nil {
		return err
	}

	if collectorRole == k8sconsts.CollectorsRoleNodeCollector {
		c.sources.handleNodeCollectorMetrics(senderPod, md)
		return nil
	}

	if collectorRole == k8sconsts.CollectorsRoleClusterGateway {
		c.clusterCollectorMetrics.handleClusterCollectorMetrics(senderPod, md)
		return nil
	}

	return errUnKnownSender
}

func NewOdigosMetrics() *OdigosMetricsConsumer {
	return &OdigosMetricsConsumer{
		sources:                 newSourcesMetrics(),
		clusterCollectorMetrics: newClusterCollectorMetrics(),
		deletedChan:             make(chan notification),
	}
}

// Run serves OTLP gRPC (metrics; profiles when profilingEnabled) on otlpGrpcPort, or consts.OTLPPort if unset.
// The OTLP receiver factory default HTTP server remains enabled unless configured elsewhere.
func (c *OdigosMetricsConsumer) Run(ctx context.Context, odigosNS string, nextProfiles xconsumer.Profiles, profilingEnabled bool, otlpGrpcPort int) {
	lg := commonlogger.LoggerCompat().With("subsystem", "collector-metrics", "component", "otlp-receiver")

	// In-cluster collectors send OTLP to this process; the port must match gateway/collector config (default consts.OTLPPort).
	listenPort := otlpGrpcPort
	if listenPort <= 0 {
		listenPort = consts.OTLPPort
	}
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
		err := runWatcher(ctx, &deleteWatcher{
			odigosNS:            odigosNS,
			deleteNotifications: c.deletedChan,
		})
		if err != nil {
			lg.Error("Error running delete watcher", "err", err)
		}
	}()

	// setup the OTLP receiver
	f := otlpreceiver.NewFactory()

	cfg, ok := f.CreateDefaultConfig().(*otlpreceiver.Config)
	if !ok {
		panic("failed to cast default config to otlpreceiver.Config")
	}

	// Modify the gRPC listener address
	cfg.GRPC = configoptional.Some(configgrpc.ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  fmt.Sprintf("0.0.0.0:%d", listenPort),
			Transport: confignet.TransportTypeTCP,
		},
	})

	host := componenttest.NewNopHost()
	set := receivertest.NewNopSettings(f.Type())

	// Register profiles on the shared OTLP receiver before metrics so the same otlpReceiver
	// instance has nextProfiles set before CreateMetrics; both must be registered before Start.
	var profilesReceiver xreceiver.Profiles
	if profilingEnabled && nextProfiles != nil {
		xf, xok := f.(xreceiver.Factory)
		if !xok {
			lg.Info("OTLP receiver factory does not support profiles; continuing with metrics only")
		} else {
			var profilesCreateErr error
			profilesReceiver, profilesCreateErr = xf.CreateProfiles(ctx, set, cfg, nextProfiles)
			if profilesCreateErr != nil {
				lg.Error("failed to create OTLP profiles receiver; continuing with metrics only", "err", profilesCreateErr)
				profilesReceiver = nil
			}
		}
	}

	metricsReceiver, err := f.CreateMetrics(ctx, set, cfg, c)
	if err != nil {
		panic("failed to create receiver")
	}

	if err := metricsReceiver.Start(ctx, host); err != nil {
		lg.Error("failed to start OTLP metrics receiver", "err", err)
	}
	if profilesReceiver != nil {
		if err := profilesReceiver.Start(ctx, host); err != nil {
			lg.Error("failed to start OTLP profiles receiver", "err", err)
		}
	}
	if profilingEnabled && nextProfiles != nil && profilesReceiver == nil {
		lg.Info("OTLP profiles receiver was not created; profiling ingestion disabled until fixed (gateway may log Unimplemented on ProfilesService)")
	}

	defer func() {
		if profilesReceiver != nil {
			_ = profilesReceiver.Shutdown(ctx)
		}
		_ = metricsReceiver.Shutdown(ctx)
	}()

	lg.Info("OTLP receiver listening", "grpc", fmt.Sprintf("0.0.0.0:%d", listenPort), "profilesEnabled", profilingEnabled && profilesReceiver != nil)
	<-ctx.Done()
	closeWg.Wait()
}

func (c *OdigosMetricsConsumer) GetSingleSourceMetrics(sID common.SourceID) (trafficMetrics, bool) {
	return c.sources.metricsByID(sID)
}

func (c *OdigosMetricsConsumer) GetSingleDestinationMetrics(dID string) (trafficMetrics, bool) {
	return c.clusterCollectorMetrics.metricsByID(dID)
}

func (c *OdigosMetricsConsumer) GetSourcesMetrics() map[common.SourceID]trafficMetrics {
	return c.sources.sourcesMetrics()
}

func (c *OdigosMetricsConsumer) GetDestinationsMetrics() map[string]trafficMetrics {
	return c.clusterCollectorMetrics.destinationsMetrics()
}

func (c *OdigosMetricsConsumer) GetServiceGraphEdges() map[string]map[string]ServiceGraphEdge {
	return c.clusterCollectorMetrics.serviceGraphEdges()
}
