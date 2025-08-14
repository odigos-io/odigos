package status

const (
	WorkloadOdigosHealthStatus = "WorkloadOdigosHealth"
)

type WorkloadOdigosHealthStatusReason string

const (
	WorkloadOdigosHealthStatusReasonSuccess  WorkloadOdigosHealthStatusReason = "Success"
	WorkloadOdigosHealthStatusReasonDisabled WorkloadOdigosHealthStatusReason = "DisabledForInstrumentation"
	WorkloadOdigosHealthStatusReasonError    WorkloadOdigosHealthStatusReason = "Error"
)
