package status

import (
	"github.com/odigos-io/odigos/frontend/graph/model"
)

const (
	WorkloadOdigosHealthStatus = "WorkloadOdigosHealth"
)

// StaticPodEnterpriseFeatureHealthStatus returns an "Unsupported" health-status condition
// when the given workload kind is a StaticPod and the cluster is not on the on-prem tier.
// StaticPod instrumentation is an enterprise feature, so on community tier we surface this
// as a permanent, informational condition (mapped to a disabled-looking visual in the UI)
// instead of letting the workload look perpetually unhealthy due to missing telemetry.
// Returns nil if the workload is not a static pod or the tier is on-prem.
func StaticPodEnterpriseFeatureHealthStatus(kind model.K8sResourceKind, tier model.Tier) *model.DesiredConditionStatus {
	if kind != model.K8sResourceKindStaticPod || tier == model.TierOnprem {
		return nil
	}
	reasonStr := string(WorkloadOdigosHealthStatusReasonEnterpriseFeature)
	return &model.DesiredConditionStatus{
		Name:       WorkloadOdigosHealthStatus,
		Status:     model.DesiredStateProgressUnsupported,
		ReasonEnum: &reasonStr,
		Message:    "Static pod instrumentation is an enterprise (on-prem) feature",
	}
}

type WorkloadOdigosHealthStatusReason string

const (
	WorkloadOdigosHealthStatusReasonSuccess                     WorkloadOdigosHealthStatusReason = "Success"
	WorkloadOdigosHealthStatusReasonSuccessAndEmittingTelemetry WorkloadOdigosHealthStatusReason = "SuccessAndEmittingTelemetry"
	WorkloadOdigosHealthStatusReasonDisabled                    WorkloadOdigosHealthStatusReason = "DisabledForInstrumentation"
	// WorkloadOdigosHealthStatusReasonEnterpriseFeature indicates that the workload requires an
	// enterprise tier feature (e.g. static pod instrumentation) that is not available on the
	// current Odigos tier. The condition is permanent for this cluster until the tier is upgraded.
	WorkloadOdigosHealthStatusReasonEnterpriseFeature WorkloadOdigosHealthStatusReason = "EnterpriseFeature"
)
