package odigossourcetodestinationfilterprocessor

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type filterProcessor struct {
	logger *zap.Logger
	config *Config
}

func (fp *filterProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	rspans := td.ResourceSpans()

	for i := 0; i < rspans.Len(); i++ {
		resourceSpan := rspans.At(i)
		ilSpans := resourceSpan.ScopeSpans()

		for j := 0; j < ilSpans.Len(); j++ {
			scopeSpan := ilSpans.At(j)
			spans := scopeSpan.Spans()

			spans.RemoveIf(func(span ptrace.Span) bool {
				return !fp.matches(span, resourceSpan)
			})
		}
	}

	return td, nil
}

func (fp *filterProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	rMetrics := md.ResourceMetrics()

	for i := 0; i < rMetrics.Len(); i++ {
		resourceMetric := rMetrics.At(i)
		resourceAttributes := resourceMetric.Resource().Attributes()
		ilMetrics := resourceMetric.ScopeMetrics()

		for j := 0; j < ilMetrics.Len(); j++ {
			scopeMetric := ilMetrics.At(j)
			metrics := scopeMetric.Metrics()

			metrics.RemoveIf(func(metric pmetric.Metric) bool {
				return !fp.metricMatches(metric, resourceAttributes)
			})
		}
	}

	return md, nil
}

func (fp *filterProcessor) metricMatches(metric pmetric.Metric, resourceAttributes pcommon.Map) bool {
	for _, condition := range fp.config.MatchConditions {
		name, _ := resourceAttributes.Get("name")
		namespace, _ := resourceAttributes.Get("namespace")
		kind, _ := resourceAttributes.Get("kind")

		if name.AsString() == condition.Name &&
			namespace.AsString() == condition.Namespace &&
			kind.AsString() == condition.Kind {
			return true
		}
	}

	return false
}

func (fp *filterProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	rLogs := ld.ResourceLogs()

	for i := 0; i < rLogs.Len(); i++ {
		resourceLog := rLogs.At(i)
		resourceAttributes := resourceLog.Resource().Attributes()
		ilLogs := resourceLog.ScopeLogs()

		for j := 0; j < ilLogs.Len(); j++ {
			scopeLog := ilLogs.At(j)
			logRecords := scopeLog.LogRecords()

			logRecords.RemoveIf(func(log plog.LogRecord) bool {
				return !fp.logMatches(log.Attributes(), resourceAttributes)
			})
		}
	}

	return ld, nil
}

func (fp *filterProcessor) logMatches(logAttributes, resourceAttributes pcommon.Map) bool {

	name, _ := resourceAttributes.Get("name")
	namespace, _ := resourceAttributes.Get("namespace")
	kind, _ := resourceAttributes.Get("kind")

	for _, condition := range fp.config.MatchConditions {
		if name.AsString() == condition.Name &&
			namespace.AsString() == condition.Namespace &&
			kind.AsString() == condition.Kind {
			return true
		}
	}

	return false
}

func (fp *filterProcessor) matches(span ptrace.Span, resourceSpan ptrace.ResourceSpans) bool {
	attributes := resourceSpan.Resource().Attributes()

	namespace := getAttribute(attributes, "k8s.namespace.name")
	if namespace == "" {
		return false
	}

	name, kind := getDynamicNameAndKind(attributes)
	if name == "" || kind == "" {
		return false
	}

	for _, condition := range fp.config.MatchConditions {
		if name == condition.Name &&
			namespace == condition.Namespace &&
			kind == condition.Kind {
			return true
		}
	}

	return false
}

func getDynamicNameAndKind(attributes pcommon.Map) (name string, kind string) {

	resourceTypes := []struct {
		kind string
		key  string
	}{
		{"deployment", "k8s.deployment.name"},
		{"statefulSet", "k8s.statefulset.name"},
		{"daemonSet", "k8s.daemonset.name"},
	}

	for _, resourceType := range resourceTypes {
		if value, exists := attributes.Get(resourceType.key); exists {
			return value.AsString(), resourceType.kind
		}
	}

	return "", ""
}

func getAttribute(attributes pcommon.Map, key string) string {
	if value, exists := attributes.Get(key); exists {
		return value.AsString()
	}
	return ""
}
