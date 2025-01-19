package odigossourcesfilter

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

const (
	k8sNamespaceNameAttr   = "k8s.namespace.name"
	k8sDeploymentNameAttr  = "k8s.deployment.name"
	k8sStatefulSetNameAttr = "k8s.statefulset.name"
	k8sDaemonSetNameAttr   = "k8s.daemonset.name"
	kindDeployment         = "Deployment"
	kindStatefulSet        = "StatefulSet"
	kindDaemonSet          = "DaemonSet"
)

type filterProcessor struct {
	logger   *zap.Logger
	config   *Config
	matchMap map[string]struct{}
}

func newFilterProcessor(logger *zap.Logger, cfg *Config) *filterProcessor {
	return &filterProcessor{
		logger:   logger,
		config:   cfg,
		matchMap: initMatchMap(cfg.MatchConditions),
	}
}

func initMatchMap(conditions []string) map[string]struct{} {
	matchMap := make(map[string]struct{}, len(conditions))
	for _, condition := range conditions {
		matchMap[condition] = struct{}{}
	}
	return matchMap
}

func (fp *filterProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	rspans := td.ResourceSpans()

	rspans.RemoveIf(func(resourceSpan ptrace.ResourceSpans) bool {
		resourceAttributes := resourceSpan.Resource().Attributes()
		namespace, name, kind, found := extractResourceDetails(resourceAttributes)
		if found {
			return !fp.matches(name, namespace, kind)
		}
		return false
	})

	return td, nil
}

func (fp *filterProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	rMetrics := md.ResourceMetrics()

	rMetrics.RemoveIf(func(resourceMetric pmetric.ResourceMetrics) bool {
		resourceAttributes := resourceMetric.Resource().Attributes()
		namespace, name, kind, found := extractResourceDetails(resourceAttributes)
		if found {
			return !fp.matches(name, namespace, kind)
		}
		return false
	})

	return md, nil
}

func (fp *filterProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	rLogs := ld.ResourceLogs()

	rLogs.RemoveIf(func(resourceLog plog.ResourceLogs) bool {
		resourceAttributes := resourceLog.Resource().Attributes()
		namespace, name, kind, found := extractResourceDetails(resourceAttributes)
		if found {
			return !fp.matches(name, namespace, kind)
		}
		return false
	})

	return ld, nil
}

func (fp *filterProcessor) matches(name, namespace, kind string) bool {
	if namespace == "" {
		return false
	}

	namespaceSelectorKey := fmt.Sprintf("%s/*/*", namespace)
	if _, exists := fp.matchMap[namespaceSelectorKey]; exists {
		return true
	}

	if name != "" && kind != "" {
		key := fmt.Sprintf("%s/%s/%s", namespace, kind, name)
		if _, exists := fp.matchMap[key]; exists {
			return true
		}
	}

	return false
}

func extractResourceDetails(attributes pcommon.Map) (namespace, name, kind string, found bool) {
	if namespace, found = getAttribute(attributes, k8sNamespaceNameAttr); !found {
		return "", "", "", false
	}

	if name, kind, found := getDynamicNameAndKind(attributes); found {
		return namespace, name, kind, true
	}

	return namespace, name, kind, true
}

func getDynamicNameAndKind(attributes pcommon.Map) (name string, kind string, found bool) {
	resourceTypes := []struct {
		kind string
		key  string
	}{
		{kindDeployment, k8sDeploymentNameAttr},
		{kindStatefulSet, k8sStatefulSetNameAttr},
		{kindDaemonSet, k8sDaemonSetNameAttr},
	}

	for _, resourceType := range resourceTypes {
		if value, exists := attributes.Get(resourceType.key); exists {
			return value.AsString(), resourceType.kind, true
		}
	}

	return "", "", false
}

func getAttribute(attributes pcommon.Map, key string) (string, bool) {
	if value, exists := attributes.Get(key); exists {
		return value.AsString(), true
	}
	return "", false
}
