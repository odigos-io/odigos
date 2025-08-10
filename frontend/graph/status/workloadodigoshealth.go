package status

const (
	WorkloadOdigosHealthStatus = "WorkloadOdigosHealth"
)

type WorkloadOdigosHealthStatusReason string

const (
	WorkloadOdigosHealthStatusReasonSuccess WorkloadOdigosHealthStatusReason = "Success"
	WorkloadOdigosHealthStatusReasonError   WorkloadOdigosHealthStatusReason = "Error"
)
