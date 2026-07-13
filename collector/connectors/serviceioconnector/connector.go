package serviceioconnector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"

	odigoscollector "github.com/odigos-io/odigos/common/collector"
)

type serviceioConnector struct {
	config               *Config
	inputSpanAttributes  []string
	outputSpanAttributes []string
	metricsFlushInterval time.Duration
	logger               *zap.Logger
	metricsConsumer      consumer.Metrics
	collectorInstanceID  string

	configMutex              sync.RWMutex
	odigosConfig             odigoscollector.OdigosConfigExtension
	workloadIdentityResolver WorkloadIdentityResolver

	startTime time.Time

	seriesMutex     sync.Mutex
	keyToMetric     map[uint64]metricSeries
	maxMetricSeries int
	seriesLimitOnce sync.Once

	shutdownCh chan struct{}
}

func newConnector(set component.TelemetrySettings, cfg component.Config, next consumer.Metrics) (*serviceioConnector, error) {
	typedCfg := cfg.(*Config)

	if typedCfg.MetricsFlushInterval == nil {
		interval := defaultMetricsFlushInterval
		typedCfg.MetricsFlushInterval = &interval
	} else if typedCfg.MetricsFlushInterval.Nanoseconds() <= 0 {
		set.Logger.Warn("metrics_flush_interval is set to 0, metrics will be flushed on every received batch of traces")
	}

	collectorInstanceID := collectorInstanceIDFromResource(set.Resource)
	if collectorInstanceID == "" {
		set.Logger.Warn("service.instance.id not found on collector telemetry resource; metrics will omit collector instance ID")
	}

	return &serviceioConnector{
		config:               typedCfg,
		inputSpanAttributes:  normalizeSpanAttributes(typedCfg.InputSpanAttributes),
		outputSpanAttributes: normalizeSpanAttributes(typedCfg.OutputSpanAttributes),
		metricsFlushInterval: typedCfg.resolvedMetricsFlushInterval(),
		logger:               set.Logger,
		metricsConsumer:      next,
		collectorInstanceID:  collectorInstanceID,
		startTime:            time.Now(),
		keyToMetric:          make(map[uint64]metricSeries),
		maxMetricSeries:      defaultMaxMetricSeries,
		shutdownCh:           make(chan struct{}),
	}, nil
}

func (c *serviceioConnector) Start(ctx context.Context, host component.Host) error {
	c.startTime = time.Now()
	c.keyToMetric = make(map[uint64]metricSeries)

	if c.config.OdigosConfigExtension == nil {
		c.logger.Warn("odigos_config_extension unset; service I/O metrics will not be computed")
		return nil
	}

	if err := c.registerOdigosConfigExtension(ctx, host); err != nil {
		return err
	}

	go c.metricFlushLoop(c.metricsFlushInterval)
	c.logger.Info("serviceio connector started",
		zap.String("collector_instance", c.collectorInstanceID),
		zap.Duration("metrics_flush_interval", c.metricsFlushInterval),
		zap.Int("input_span_attributes_count", len(c.inputSpanAttributes)),
		zap.Int("output_span_attributes_count", len(c.outputSpanAttributes)),
	)
	return nil
}

func (c *serviceioConnector) registerOdigosConfigExtension(ctx context.Context, host component.Host) error {
	extID := c.config.OdigosConfigExtension
	ext, ok := host.GetExtensions()[*extID]
	if !ok || ext == nil {
		return fmt.Errorf("odigos config extension %q not found", extID.String())
	}
	odigosExt, ok := ext.(odigoscollector.OdigosConfigExtension)
	if !ok {
		return fmt.Errorf("extension %q is not an OdigosConfigExtension (got %T)", extID.String(), ext)
	}
	if !odigosExt.WaitForCacheSync(ctx) {
		c.logger.Warn("odigos config extension cache sync did not complete; active-source filtering may be incomplete briefly on startup")
	}
	c.configMutex.Lock()
	defer c.configMutex.Unlock()
	c.odigosConfig = odigosExt
	c.workloadIdentityResolver = func(res pcommon.Resource) (string, pcommon.Map, bool) {
		cacheKey, attrs, err := odigosExt.GetWorkloadIdentityFromResource(res)
		if err != nil {
			c.logger.Error("failed to get workload identity from resource", zap.Error(err))
			return "", pcommon.NewMap(), false
		}
		return cacheKey, attrs, true
	}
	return nil
}

func (c *serviceioConnector) metricFlushLoop(flushInterval time.Duration) {
	if flushInterval <= 0 {
		return
	}

	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.flushMetrics(context.Background()); err != nil {
				c.logger.Error("failed to flush metrics", zap.Error(err))
			}
		case <-c.shutdownCh:
			return
		}
	}
}

func (c *serviceioConnector) Shutdown(_ context.Context) error {
	c.configMutex.Lock()
	c.odigosConfig = nil
	c.workloadIdentityResolver = nil
	c.configMutex.Unlock()
	close(c.shutdownCh)
	return nil
}

func (c *serviceioConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeTraces receives complete traces (e.g. from groupbytrace) and emits service I/O metrics.
func (c *serviceioConnector) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	if err := validateCompleteTraceBatch(td); err != nil {
		c.logger.Error("invalid complete trace batch", zap.Error(err))
		return nil
	}
	c.configMutex.RLock()
	odigosConfig := c.odigosConfig
	workloadIdentityResolver := c.workloadIdentityResolver
	c.configMutex.RUnlock()
	if odigosConfig == nil || workloadIdentityResolver == nil {
		return nil
	}

	tree, err := BuildTraceTree(td, workloadIdentityResolver)
	if err != nil {
		c.logger.Error("failed to build trace tree", zap.Error(err))
		return nil
	}

	if !c.aggregateConnectionsFromTree(tree, odigosConfig) {
		return nil
	}

	if c.metricsFlushInterval <= 0 {
		if err := c.flushMetrics(ctx); err != nil {
			c.logger.Error("failed to flush metrics", zap.Error(err))
		}
	}

	return nil
}

func (c *serviceioConnector) isActiveSourceInstance(instance *ServiceInstance, odigosConfig odigoscollector.OdigosConfigExtension) bool {
	return odigosConfig.IsActiveSource(instance.Root.Resource)
}
