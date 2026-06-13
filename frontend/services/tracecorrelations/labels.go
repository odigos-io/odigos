package tracecorrelations

import (
	"sort"
	"strings"

	prommodel "github.com/prometheus/common/model"

	"github.com/odigos-io/odigos/common/consts"
)

const (
	inputAttributePrefix  = "input."
	outputAttributePrefix = "output."
)

var (
	collectorInstanceLabelKeys = []string{
		"odigos_collector_instance_id",
		"odigos.collector.instance.id",
		"odigos_collector_instance",
		"odigos.collector.instance",
	}
	k8sWorkloadNameLabelKeys = []struct {
		label string
		kind  string
	}{
		{"k8s_deployment_name", "Deployment"},
		{"k8s.deployment.name", "Deployment"},
		{"k8s_statefulset_name", "StatefulSet"},
		{"k8s.statefulset.name", "StatefulSet"},
		{"k8s_daemonset_name", "DaemonSet"},
		{"k8s.daemonset.name", "DaemonSet"},
		{"k8s_job_name", "Job"},
		{"k8s.job.name", "Job"},
		{"k8s_cronjob_name", "CronJob"},
		{"k8s.cronjob.name", "CronJob"},
		{"k8s_argoproj_rollout_name", "Rollout"},
		{"k8s.argoproj.rollout.name", "Rollout"},
	}
)

func workloadFromMetric(labels prommodel.Metric) (workloadKey, bool) {
	namespace := labelValue(labels, "k8s_namespace_name", "k8s.namespace.name")
	container := labelValue(labels, "k8s_container_name", "k8s.container.name")
	if namespace == "" || container == "" {
		return workloadKey{}, false
	}

	kind := labelValue(labels, "odigos_workload_kind", consts.OdigosWorkloadKindAttribute)
	name := labelValue(labels, "odigos_workload_name", consts.OdigosWorkloadNameAttribute)

	if kind == "" || name == "" {
		for _, candidate := range k8sWorkloadNameLabelKeys {
			if value := string(labels[prommodel.LabelName(candidate.label)]); value != "" {
				kind = candidate.kind
				name = value
				break
			}
		}
	}

	if kind == "" || name == "" {
		return workloadKey{}, false
	}

	return workloadKey{
		namespace: namespace,
		kind:      kind,
		name:      name,
		container: container,
	}, true
}

func attributeGroupFromMetric(labels prommodel.Metric, prefix string) attributeGroup {
	attrs := make(map[string]string)
	underscorePrefix := strings.ReplaceAll(prefix, ".", "_")

	for name, value := range labels {
		key := string(name)
		if key == prommodel.MetricNameLabel || isCollectorInstanceLabel(key) || isWorkloadIdentityLabel(key) {
			continue
		}
		if key == "service_name" || key == "service.name" {
			continue
		}

		attrKey, ok := attributeKeyFromLabel(key, prefix, underscorePrefix)
		if !ok {
			continue
		}
		attrs[attrKey] = string(value)
	}

	return attributeGroup{
		attrs: attrs,
		sig:   attributeSignature(attrs),
	}
}

func attributeKeyFromLabel(label, dotPrefix, underscorePrefix string) (string, bool) {
	switch {
	case strings.HasPrefix(label, dotPrefix):
		return strings.TrimPrefix(label, dotPrefix), true
	case strings.HasPrefix(label, underscorePrefix):
		suffix := strings.TrimPrefix(label, underscorePrefix)
		return strings.ReplaceAll(suffix, "_", "."), true
	default:
		return "", false
	}
}

func attributeSignature(attrs map[string]string) string {
	if len(attrs) == 0 {
		return ""
	}
	keys := make([]string, 0, len(attrs))
	for key := range attrs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+attrs[key])
	}
	return strings.Join(parts, "\x00")
}

func labelValue(labels prommodel.Metric, keys ...string) string {
	for _, key := range keys {
		if value, ok := labels[prommodel.LabelName(key)]; ok {
			return string(value)
		}
	}
	return ""
}

func isCollectorInstanceLabel(key string) bool {
	for _, candidate := range collectorInstanceLabelKeys {
		if key == candidate {
			return true
		}
	}
	return false
}

func isWorkloadIdentityLabel(key string) bool {
	switch key {
	case "k8s_namespace_name", "k8s.namespace.name",
		"k8s_container_name", "k8s.container.name",
		"odigos_workload_kind", consts.OdigosWorkloadKindAttribute,
		"odigos_workload_name", consts.OdigosWorkloadNameAttribute,
		"k8s_deployment_name", "k8s.deployment.name",
		"k8s_statefulset_name", "k8s.statefulset.name",
		"k8s_daemonset_name", "k8s.daemonset.name",
		"k8s_job_name", "k8s.job.name",
		"k8s_cronjob_name", "k8s.cronjob.name",
		"k8s_argoproj_rollout_name", "k8s.argoproj.rollout.name":
		return true
	default:
		return false
	}
}
