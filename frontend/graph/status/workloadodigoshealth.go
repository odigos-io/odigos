package status

const (
	WorkloadOdigosHealthStatus = "WorkloadOdigosHealth"
)

type WorkloadOdigosHealthStatusReason string

const (
	WorkloadOdigosHealthStatusReasonSuccess                     WorkloadOdigosHealthStatusReason = "Success"
	WorkloadOdigosHealthStatusReasonSuccessAndEmittingTelemetry WorkloadOdigosHealthStatusReason = "SuccessAndEmittingTelemetry"
	WorkloadOdigosHealthStatusReasonDisabled                    WorkloadOdigosHealthStatusReason = "DisabledForInstrumentation"
)
