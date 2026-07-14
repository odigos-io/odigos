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

	odigoscollector "github.com/odigos-io/odigos/common/collector"
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

type routerConnector struct {
	tracesConfig  tracesConfig
	metricsConfig metricsConfig
	logsConfig    logsConfig

	configExtensionID     *component.ID
	odigosConfigExtension odigoscollector.OdigosConfigExtension
}

func (r *routerConnector) Start(ctx context.Context, host component.Host) error {
	// validated as not nil in Config.Validate(), can be nil in generated tests
	if r.configExtensionID == nil {
		return nil
	}
	ext, found := host.GetExtensions()[*r.configExtensionID]
	if !found || ext == nil {
		return fmt.Errorf("odigos config extension %s not found", *r.configExtensionID)
	}
	odigosExt, ok := ext.(odigoscollector.OdigosConfigExtension)
	if !ok {
		return fmt.Errorf("extension %s is not a valid odigos config extension", *r.configExtensionID)
	}
	r.odigosConfigExtension = odigosExt
	odigosExt.WaitForCacheSync(ctx)
	return nil
}

func (r *routerConnector) Shutdown(_ context.Context) error { return nil }
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
		defaultTracesConsumer = nil
	}

	return &routerConnector{
		tracesConfig:      tracesConfig{consumers: tr, defaultCons: defaultTracesConsumer, logger: set.Logger},
		configExtensionID: config.OdigosConfigExtension,
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
		defaultMetricsConsumer = nil
	}

	return &routerConnector{
		configExtensionID: config.OdigosConfigExtension,
		metricsConfig:     metricsConfig{consumers: tr, defaultCons: defaultMetricsConsumer, logger: set.Logger},
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
		defaultLogsConsumer = nil
	}

	return &routerConnector{
		logsConfig:        logsConfig{consumers: tr, defaultCons: defaultLogsConsumer, logger: set.Logger},
		configExtensionID: config.OdigosConfigExtension,
	}, nil
}

func (r *routerConnector) resolveDataStreams(resource pcommon.Resource) []string {
	streams, found := r.odigosConfigExtension.GetDataStreamsForWorkload(resource)
	if !found || len(streams) == 0 {
		return nil
	}
	return streams
}

func (r *routerConnector) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	cfg := r.tracesConfig
	tracesByConsumer := make(map[consumer.Traces]ptrace.Traces)
	defaultTraces := ptrace.NewTraces()
	var errs error

	rSpans := td.ResourceSpans()
	for i := 0; i < rSpans.Len(); i++ {
		rs := rSpans.At(i)
		pipelines := r.resolveDataStreams(rs.Resource())

		if len(pipelines) == 0 {
			rs.CopyTo(defaultTraces.ResourceSpans().AppendEmpty())
			continue
		}

		for _, pipeline := range pipelines {
			pipelineID := collectorpipeline.NewIDWithName(collectorpipeline.SignalTraces, pipeline)
			consumer, err := cfg.consumers.Consumer(pipelineID)
			if err != nil {
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
		pipelines := r.resolveDataStreams(rm.Resource())
		// If no pipeline matched, copy the resource metrics to the default consumer
		if len(pipelines) == 0 {
			cfg.logger.Debug("no pipelines matched for resource metrics", zap.Any("resource", rm.Resource().Attributes()))
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
		pipelines := r.resolveDataStreams(rl.Resource())

		if len(pipelines) == 0 {
			rl.CopyTo(defaultLogs.ResourceLogs().AppendEmpty())
			continue
		}

		for _, pipeline := range pipelines {
			consumer, err := cfg.consumers.Consumer(
				collectorpipeline.NewIDWithName(collectorpipeline.SignalLogs, pipeline),
			)
			if err != nil {
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
