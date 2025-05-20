package odigosrouterconnector

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv1_21 "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// routerConnector is the main struct for all signal types.
type routerConnector struct {
	tracesConsumers  map[string]consumer.Traces
	metricsConsumers map[string]consumer.Metrics
	logsConsumers    map[string]consumer.Logs
	routingTable     *SignalRoutingMap
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
	config := cfg.(*Config)

	routeMap := BuildSignalRoutingMap(config.Groups)

	// Extract unique pipeline names from the routeMap for traces
	tracesConsumers := buildConsumerMap(routeMap, "traces", next)

	fmt.Println("Registered tracesConsumers:", tracesConsumers)

	return &routerConnector{
		routingTable:    &routeMap,
		tracesConsumers: tracesConsumers,
	}, nil
}

func createMetricsConnector(
	ctx context.Context,
	set connector.Settings,
	cfg component.Config,
	next consumer.Metrics,
) (connector.Metrics, error) {
	config := cfg.(*Config)

	routeMap := BuildSignalRoutingMap(config.Groups)

	metricsConsumers := buildConsumerMap(routeMap, "metrics", next)

	fmt.Println("Registered metricsConsumers:", metricsConsumers)

	return &routerConnector{
		routingTable:     &routeMap,
		metricsConsumers: metricsConsumers,
	}, nil
}

func createLogsConnector(
	ctx context.Context,
	set connector.Settings,
	cfg component.Config,
	next consumer.Logs,
) (connector.Logs, error) {
	config := cfg.(*Config)

	routeMap := BuildSignalRoutingMap(config.Groups)

	logsConsumers := buildConsumerMap(routeMap, "logs", next)

	fmt.Println("Registered logsConsumers:", logsConsumers)
	return &routerConnector{
		routingTable:  &routeMap,
		logsConsumers: logsConsumers,
	}, nil
}

func determineRoutingPipelines(attrs pcommon.Map, m SignalRoutingMap, signal string) []string {
	nsAttr, ok := attrs.Get(string(semconv1_21.K8SNamespaceNameKey))
	if !ok {
		return nil
	}
	ns := nsAttr.Str()

	name, kind := getDynamicNameAndKind(attrs)
	if name == "" || kind == "" {
		return nil
	}

	key := fmt.Sprintf("%s/%s/%s", ns, NormalizeKind(kind), name)

	routingIndex, ok := m[key]
	if !ok {
		return nil
	}

	pipelines, ok := routingIndex[signal]
	if !ok {
		return nil
	}

	return pipelines
}

func (r *routerConnector) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	rSpans := td.ResourceSpans()
	pipelineTraces := make(map[string]ptrace.Traces)

	fmt.Println("routingTable", *r.routingTable)
	for i := 0; i < rSpans.Len(); i++ {
		fmt.Println("rSpans", rSpans.At(i))
		rs := rSpans.At(i)

		// Determine pipelines this span belongs to, based on its resource attributes
		pipelines := determineRoutingPipelines(rs.Resource().Attributes(), *r.routingTable, "traces")

		fmt.Println("routing pipelines", pipelines)

		for _, pipeline := range pipelines {
			fmt.Println("pipeline in consume traces", pipeline)
			// Skip if pipeline isn't wired (e.g., not in tracesConsumers)
			if _, allowed := r.tracesConsumers[pipeline]; !allowed {
				fmt.Println("pipeline not allowed in consume traces", pipeline)
				continue
			}

			// Initialize the batch container if not done already
			if _, exists := pipelineTraces[pipeline]; !exists {
				fmt.Println("initializing pipeline traces", pipeline)
				pipelineTraces[pipeline] = ptrace.NewTraces()
			}

			// Append resource span to the relevant pipeline batch
			rs.CopyTo(pipelineTraces[pipeline].ResourceSpans().AppendEmpty())
		}
	}

	// Forward each batch to the configured downstream consumer
	for pipeline, batch := range pipelineTraces {
		fmt.Println("forwarding traces to", pipeline)
		if err := r.tracesConsumers[pipeline].ConsumeTraces(ctx, batch); err != nil {
			return err
		}
	}

	return nil
}

func (r *routerConnector) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	rMetrics := md.ResourceMetrics()
	pipelineMetrics := make(map[string]pmetric.Metrics)

	for i := 0; i < rMetrics.Len(); i++ {
		rm := rMetrics.At(i)

		// Determine destination pipelines based on workload metadata
		pipelines := determineRoutingPipelines(rm.Resource().Attributes(), *r.routingTable, "metrics")

		for _, pipeline := range pipelines {
			if _, allowed := r.metricsConsumers[pipeline]; !allowed {
				continue
			}
			if _, exists := pipelineMetrics[pipeline]; !exists {
				pipelineMetrics[pipeline] = pmetric.NewMetrics()
			}
			rm.CopyTo(pipelineMetrics[pipeline].ResourceMetrics().AppendEmpty())
		}
	}

	// Send routed metrics to relevant consumers
	for pipeline, batch := range pipelineMetrics {
		if err := r.metricsConsumers[pipeline].ConsumeMetrics(ctx, batch); err != nil {
			return err
		}
	}

	return nil
}

func (r *routerConnector) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	rLogs := ld.ResourceLogs()
	pipelineLogs := make(map[string]plog.Logs)

	for i := 0; i < rLogs.Len(); i++ {
		rl := rLogs.At(i)

		// Extract routing info from log's resource attributes
		pipelines := determineRoutingPipelines(rl.Resource().Attributes(), *r.routingTable, "logs")

		for _, pipeline := range pipelines {
			if _, allowed := r.logsConsumers[pipeline]; !allowed {
				continue
			}
			if _, exists := pipelineLogs[pipeline]; !exists {
				pipelineLogs[pipeline] = plog.NewLogs()
			}
			rl.CopyTo(pipelineLogs[pipeline].ResourceLogs().AppendEmpty())
		}
	}

	// Emit logs per matched pipeline
	for pipeline, batch := range pipelineLogs {
		if err := r.logsConsumers[pipeline].ConsumeLogs(ctx, batch); err != nil {
			return err
		}
	}

	return nil
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

// Each pipeline (e.g., traces/B) expects a dedicated consumer entry.
// This map allows the connector to forward data to the correct downstream group.
func buildConsumerMap[T any](
	routeMap SignalRoutingMap,
	signal string,
	next T,
) map[string]T {
	consumers := make(map[string]T)
	seen := make(map[string]struct{})

	for _, signalMap := range routeMap {
		for _, pipeline := range signalMap[signal] {
			if _, exists := seen[pipeline]; !exists {
				consumers[pipeline] = next
				seen[pipeline] = struct{}{}
			}
		}
	}

	return consumers
}
