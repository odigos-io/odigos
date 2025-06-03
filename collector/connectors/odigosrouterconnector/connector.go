package odigosrouterconnector

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	collectorpipeline "go.opentelemetry.io/collector/pipeline"
	semconv1_21 "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.uber.org/zap"

	"github.com/odigos-io/odigos/common/consts"
)

type tracesConfig struct {
	consumers   connector.TracesRouterAndConsumer
	defaultCons consumer.Traces
	logger      *zap.Logger
}

type metricsConfig struct {
	consumers   connector.MetricsRouterAndConsumer
	defaultCons consumer.Metrics
	logger      *zap.Logger
}

type logsConfig struct {
	consumers   connector.LogsRouterAndConsumer
	defaultCons consumer.Logs
	logger      *zap.Logger
}

// routerConnector is the main struct for all signal types.
type routerConnector struct {
	tracesConfig  tracesConfig
	metricsConfig metricsConfig
	logsConfig    logsConfig
	routingTable  *SignalRoutingMap
}

func (r *routerConnector) Start(_ context.Context, _ component.Host) error { return nil }
func (r *routerConnector) Shutdown(_ context.Context) error                { return nil }
func (r *routerConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func createTracesConnector(
	ctx context.Context,
	set connector.Settings,
	cfg component.Config,
	next consumer.Traces,
) (connector.Traces, error) {

	tr, ok := next.(connector.TracesRouterAndConsumer)
	if !ok {
		return nil, errors.New("expected consumer to be a connector router")
	}

	config := cfg.(*Config)

	defaultTracesConsumer, err := tr.Consumer(
		collectorpipeline.NewIDWithName(collectorpipeline.SignalTraces, consts.DefaultDataStream),
	)
	if err != nil {
		set.Logger.Warn("failed to get default traces consumer")
		// Do not return the error — just continue with nil fallback
		defaultTracesConsumer = nil
	}

	routeMap := BuildSignalRoutingMap(config.DataStreams)

	return &routerConnector{
		routingTable: &routeMap,
		tracesConfig: tracesConfig{consumers: tr, defaultCons: defaultTracesConsumer, logger: set.Logger},
	}, nil
}

func createMetricsConnector(
	ctx context.Context,
	set connector.Settings,
	cfg component.Config,
	next consumer.Metrics,
) (connector.Metrics, error) {

	tr, ok := next.(connector.MetricsRouterAndConsumer)
	if !ok {
		return nil, errors.New("expected consumer to be a connector router")
	}

	config := cfg.(*Config)

	defaultMetricsConsumer, err := tr.Consumer(
		collectorpipeline.NewIDWithName(collectorpipeline.SignalMetrics, consts.DefaultDataStream),
	)
	if err != nil {
		set.Logger.Warn("failed to get default metrics consumer")
		// Do not return the error — just continue with nil fallback
		defaultMetricsConsumer = nil
	}

	routeMap := BuildSignalRoutingMap(config.DataStreams)

	return &routerConnector{
		routingTable:  &routeMap,
		metricsConfig: metricsConfig{consumers: tr, defaultCons: defaultMetricsConsumer, logger: set.Logger},
	}, nil
}

func createLogsConnector(
	ctx context.Context,
	set connector.Settings,
	cfg component.Config,
	next consumer.Logs,
) (connector.Logs, error) {

	tr, ok := next.(connector.LogsRouterAndConsumer)
	if !ok {
		return nil, errors.New("expected consumer to be a connector router")
	}

	config := cfg.(*Config)

	defaultLogsConsumer, err := tr.Consumer(
		collectorpipeline.NewIDWithName(collectorpipeline.SignalLogs, consts.DefaultDataStream),
	)
	if err != nil {
		set.Logger.Warn("failed to get default logs consumer")
		// Do not return the error — just continue with nil fallback
		// This can happen if the default pipeline is not configured (Sources and Destinations)
		defaultLogsConsumer = nil
	}
	routeMap := BuildSignalRoutingMap(config.DataStreams)

	return &routerConnector{
		routingTable: &routeMap,
		logsConfig:   logsConfig{consumers: tr, defaultCons: defaultLogsConsumer, logger: set.Logger},
	}, nil
}

func determineRoutingPipelines(attrs pcommon.Map, m SignalRoutingMap, signal string) ([]string, string) {
	nsAttr, ok := attrs.Get(string(semconv1_21.K8SNamespaceNameKey))
	if !ok {
		return nil, ""
	}
	ns := nsAttr.Str()

	name, kind := getDynamicNameAndKind(attrs)
	if name == "" || kind == "" {
		return nil, ""
	}

	key := fmt.Sprintf("%s/%s/%s", ns, NormalizeKind(kind), name)

	routingIndex, ok := m[key]
	if !ok {
		// still need to check for namespaces (future select) e.g. ns1/*/*
		// this is done for the case where a namespace is selected as "future select"
		// in that case a single source will be created for the namespace.
		key = fmt.Sprintf("%s/*/*", ns)
		routingIndex, ok = m[key]
		if !ok {
			return nil, ""
		}
	}

	pipelines, ok := routingIndex[signal]
	if !ok {
		return nil, ""
	}

	return pipelines, key
}

func (r *routerConnector) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	cfg := r.tracesConfig
	tracesByConsumer := make(map[consumer.Traces]ptrace.Traces)

	// fallback to default traces consumer if no pipelines are matched
	defaultTraces := ptrace.NewTraces()

	var errs error

	rSpans := td.ResourceSpans()

	for i := 0; i < rSpans.Len(); i++ {
		rs := rSpans.At(i)

		// Determine pipelines for this resource
		pipelines, key := determineRoutingPipelines(rs.Resource().Attributes(), *r.routingTable, "traces")

		// if no pipelines matched, copy the resource span to the default consumer
		if len(pipelines) == 0 {
			cfg.logger.Debug("no pipelines matched for", zap.Any("key", key))
			rs.CopyTo(defaultTraces.ResourceSpans().AppendEmpty())
			continue
		}

		for _, pipeline := range pipelines {

			pipelineID := collectorpipeline.NewIDWithName(collectorpipeline.SignalTraces, pipeline)
			consumer, err := cfg.consumers.Consumer(pipelineID)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to get consumer for pipeline %s: %w", pipelineID, err))
				continue
			}

			batch, ok := tracesByConsumer[consumer]
			if !ok {
				batch = ptrace.NewTraces()
			}
			rs.CopyTo(batch.ResourceSpans().AppendEmpty())
			tracesByConsumer[consumer] = batch
		}
	}

	// Forward all grouped batches to their respective consumers
	for cons, batch := range tracesByConsumer {
		if batch.ResourceSpans().Len() == 0 {
			continue
		}
		if err := cons.ConsumeTraces(ctx, batch); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	// Fallback, if any spans unmatched
	if defaultTraces.ResourceSpans().Len() > 0 {
		if cfg.defaultCons != nil {
			if err := cfg.defaultCons.ConsumeTraces(ctx, defaultTraces); err != nil {
				cfg.logger.Debug("failed to send traces to the default pipeline", zap.Error(err))
			}
		}
	}

	return errs
}

func (r *routerConnector) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	cfg := r.metricsConfig
	metricsByConsumer := make(map[consumer.Metrics]pmetric.Metrics)

	defaultMetrics := pmetric.NewMetrics()
	var errs error

	rMetrics := md.ResourceMetrics()
	for i := 0; i < rMetrics.Len(); i++ {
		rm := rMetrics.At(i)
		pipelines, key := determineRoutingPipelines(rm.Resource().Attributes(), *r.routingTable, "metrics")

		// If no pipeline matched, copy the resource metrics to the default consumer
		if len(pipelines) == 0 {
			cfg.logger.Debug("no pipelines matched for", zap.Any("key", key))
			rm.CopyTo(defaultMetrics.ResourceMetrics().AppendEmpty())
			continue
		}

		for _, pipeline := range pipelines {
			consumer, err := cfg.consumers.Consumer(collectorpipeline.NewIDWithName(collectorpipeline.SignalMetrics, pipeline))
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to get metrics consumer for pipeline %s: %w", pipeline, err))
				continue
			}

			batch, ok := metricsByConsumer[consumer]
			if !ok {
				batch = pmetric.NewMetrics()
			}
			rm.CopyTo(batch.ResourceMetrics().AppendEmpty())
			metricsByConsumer[consumer] = batch
		}
	}

	for cons, batch := range metricsByConsumer {
		if batch.ResourceMetrics().Len() == 0 {
			continue
		}
		if err := cons.ConsumeMetrics(ctx, batch); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	if defaultMetrics.ResourceMetrics().Len() > 0 {
		if cfg.defaultCons != nil {
			if err := cfg.defaultCons.ConsumeMetrics(ctx, defaultMetrics); err != nil {
				cfg.logger.Debug("failed to send metrics to the default pipeline", zap.Error(err))
			}
		}
	}

	return errs
}

func (r *routerConnector) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	cfg := r.logsConfig
	// Grouped batches by consumer
	logsByConsumer := make(map[consumer.Logs]plog.Logs)
	// Fallback batch for unmatched spans
	defaultLogs := plog.NewLogs()
	var errs error

	rLogs := ld.ResourceLogs()
	for i := 0; i < rLogs.Len(); i++ {
		rl := rLogs.At(i)
		// Determine destination pipelines based on resource metadata
		pipelines, key := determineRoutingPipelines(rl.Resource().Attributes(), *r.routingTable, "logs")

		// If no pipeline matched, copy the resource logs to the default consumer
		if len(pipelines) == 0 {
			cfg.logger.Debug("no pipelines matched for", zap.Any("key", key))
			rl.CopyTo(defaultLogs.ResourceLogs().AppendEmpty())
			continue
		}

		for _, pipeline := range pipelines {
			consumer, err := cfg.consumers.Consumer(
				collectorpipeline.NewIDWithName(collectorpipeline.SignalLogs, pipeline),
			)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to get logs consumer for pipeline %s: %w", pipeline, err))
				continue
			}

			batch, ok := logsByConsumer[consumer]
			if !ok {
				batch = plog.NewLogs()
			}
			rl.CopyTo(batch.ResourceLogs().AppendEmpty())
			logsByConsumer[consumer] = batch
		}
	}

	// Send each grouped batch
	for cons, batch := range logsByConsumer {
		if batch.ResourceLogs().Len() == 0 {
			continue
		}
		if err := cons.ConsumeLogs(ctx, batch); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	// Handle fallback for unmatched logs
	if defaultLogs.ResourceLogs().Len() > 0 {
		if cfg.defaultCons != nil {
			if err := cfg.defaultCons.ConsumeLogs(ctx, defaultLogs); err != nil {
				cfg.logger.Debug("failed to send logs to the default pipeline", zap.Error(err))
			}
		}
	}

	return errs
}

// getDynamicNameAndKind extracts the workload name and kind from a resource's attributes.
// It searches for known Kubernetes keys such as deployment, statefulset, and daemonset,
// and returns the first matched workload name and its corresponding kind.
// If none are found, it returns empty strings for both.

var kindKeyMap = map[string]string{
	string(semconv1_21.K8SDeploymentNameKey):  "Deployment",
	string(semconv1_21.K8SStatefulSetNameKey): "StatefulSet",
	string(semconv1_21.K8SDaemonSetNameKey):   "DaemonSet",
}

func getDynamicNameAndKind(attrs pcommon.Map) (name string, kind string) {
	for key, kindType := range kindKeyMap {
		if val, ok := attrs.Get(key); ok {
			return val.Str(), kindType
		}
	}
	return "", ""
}
