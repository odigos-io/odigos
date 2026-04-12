package workload

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
)

// OpenTelemetry semantic conventions v1.26.0: resource attribute keys for Kubernetes workload names.
// Kept as string literals so this package does not depend on go.opentelemetry.io/otel/semconv.
const (
	OTLPK8SDeploymentNameKey  = "k8s.deployment.name"
	OTLPK8SStatefulSetNameKey = "k8s.statefulset.name"
	OTLPK8SDaemonSetNameKey   = "k8s.daemonset.name"
	OTLPK8SCronJobNameKey     = "k8s.cronjob.name"
	OTLPK8SJobNameKey         = "k8s.job.name"
)

// OTLPWorkloadNameAttrKindPairs is the precedence order for resolving workload identity from
// standard OTLP resource attributes (k8s.*.name keys) plus Argo Rollout.
//
// Order matches collectormetrics.metricAttributesToSourceID when only semconv keys are present
// (CronJob before Job). Argo Rollout uses k8sconsts.K8SArgoRolloutNameAttribute — there is no semconv key.
var OTLPWorkloadNameAttrKindPairs = []struct {
	Key  string
	Kind k8sconsts.WorkloadKind
}{
	{Key: OTLPK8SDeploymentNameKey, Kind: k8sconsts.WorkloadKindDeployment},
	{Key: OTLPK8SStatefulSetNameKey, Kind: k8sconsts.WorkloadKindStatefulSet},
	{Key: OTLPK8SDaemonSetNameKey, Kind: k8sconsts.WorkloadKindDaemonSet},
	{Key: OTLPK8SCronJobNameKey, Kind: k8sconsts.WorkloadKindCronJob},
	{Key: OTLPK8SJobNameKey, Kind: k8sconsts.WorkloadKindJob},
	{Key: k8sconsts.K8SArgoRolloutNameAttribute, Kind: k8sconsts.WorkloadKindArgoRollout},
}
