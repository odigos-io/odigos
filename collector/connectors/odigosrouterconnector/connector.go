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
	"go.uber.org/zap"

	"github.com/odigos-io/odigos/collector/extension/odigosk8sresourcesexention"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
)

// k8sResourcesProvider is an interface that provides access to k8s resources for workload routing
type k8sResourcesProvider interface {
	GetDatastreamsForWorkload(odigosk8sresourcesexention.WorkloadKey) ([]odigosk8sresourcesexention.DatastreamName, bool)
}

type tracesConfig struct {
	consumers   connector.TracesRouterAndConsumer
	datastreams map[odigosk8sresourcesexention.DatastreamName]struct{}
	defaultCons consumer.Traces
	logger      *zap.Logger
}

type metricsConfig struct {
	consumers   connector.MetricsRouterAndConsumer
	datastreams map[odigosk8sresourcesexention.DatastreamName]struct{}
	defaultCons consumer.Metrics
	logger      *zap.Logger
}

type logsConfig struct {
	consumers   connector.LogsRouterAndConsumer
	datastreams map[odigosk8sresourcesexention.DatastreamName]struct{}
	defaultCons consumer.Logs
	logger      *zap.Logger
}

// routerConnector is the main struct for all signal types.
type routerConnector struct {
	tracesConfig                 tracesConfig
	metricsConfig                metricsConfig
	logsConfig                   logsConfig
	odigosKsResourcesExtensionID component.ID
	odigosKsResources            k8sResourcesProvider
}

func (r *routerConnector) Start(_ context.Context, host component.Host) error {
	extensions := host.GetExtensions()
	odigosKsResourcesExtensionComponent, ok := extensions[r.odigosKsResourcesExtensionID]
	if !ok {
		return fmt.Errorf("odigos k8s resources extension not found")
	}
	odigosKsResourcesExtension, ok := odigosKsResourcesExtensionComponent.(k8sResourcesProvider)
	if !ok {
		return fmt.Errorf("odigos k8s resources extension does not implement k8sResourcesProvider interface")
	}
	r.odigosKsResources = odigosKsResourcesExtension
	return nil
}

func (r *routerConnector) Shutdown(_ context.Context) error {
	return nil
}

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

	datastreams := calculateDatastreamsForSignals(config, common.TracesObservabilitySignal)

	return &routerConnector{
		odigosKsResourcesExtensionID: config.OdigosK8sResourcesExtensionID,
		tracesConfig: tracesConfig{
			consumers:   tr,
			defaultCons: defaultTracesConsumer,
			logger:      set.Logger,
			datastreams: datastreams,
		},
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

	datastreams := calculateDatastreamsForSignals(config, common.MetricsObservabilitySignal)

	return &routerConnector{
		odigosKsResourcesExtensionID: config.OdigosK8sResourcesExtensionID,
		metricsConfig: metricsConfig{
			consumers:   tr,
			defaultCons: defaultMetricsConsumer,
			logger:      set.Logger,
			datastreams: datastreams,
		},
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
	datastreams := calculateDatastreamsForSignals(config, common.LogsObservabilitySignal)

	return &routerConnector{
		odigosKsResourcesExtensionID: config.OdigosK8sResourcesExtensionID,
		logsConfig: logsConfig{
			consumers:   tr,
			defaultCons: defaultLogsConsumer,
			logger:      set.Logger,
			datastreams: datastreams,
		},
	}, nil
}

func (r *routerConnector) determineWorkloadDataStreams(attrs pcommon.Map, availableDataStreams map[odigosk8sresourcesexention.DatastreamName]struct{}) ([]string, string) {
	wk := odigosk8sresourcesexention.ResourceAttributesToWorkloadKey(attrs)
	if wk == nil {
		return nil, ""
	}
	ds, ok := r.odigosKsResources.GetDatastreamsForWorkload(*wk)
	if !ok {
		return nil, string(*wk)
	}

	effectiveDataStreams := []string{}
	for _, datastream := range ds {
		if _, ok := availableDataStreams[datastream]; ok {
			effectiveDataStreams = append(effectiveDataStreams, string(datastream))
		}
	}

	return effectiveDataStreams, string(*wk)
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

		// Determine dataStreams for this resource
		dataStreams, workloadKey := r.determineWorkloadDataStreams(rs.Resource().Attributes(), cfg.datastreams)

		// if no pipelines matched, copy the resource span to the default consumer
		if len(dataStreams) == 0 {
			cfg.logger.Debug("no pipelines matched for", zap.Any("key", workloadKey))
			rs.CopyTo(defaultTraces.ResourceSpans().AppendEmpty())
			continue
		}

		for _, dataStream := range dataStreams {

			pipelineID := collectorpipeline.NewIDWithName(collectorpipeline.SignalTraces, dataStream)
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
		pipelines, workloadKey := r.determineWorkloadDataStreams(rm.Resource().Attributes(), cfg.datastreams)

		// If no pipeline matched, copy the resource metrics to the default consumer
		if len(pipelines) == 0 {
			cfg.logger.Debug("no pipelines matched for", zap.Any("key", workloadKey))
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
		pipelines, workloadKey := r.determineWorkloadDataStreams(rl.Resource().Attributes(), cfg.datastreams)

		// If no pipeline matched, copy the resource logs to the default consumer
		if len(pipelines) == 0 {
			cfg.logger.Debug("no pipelines matched for", zap.Any("key", workloadKey))
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
