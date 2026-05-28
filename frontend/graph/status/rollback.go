package status

const (
	RollbackStatus = "Rollback"
)

type RollbackReason string

const (
	RollbackReasonRollbackTriggered RollbackReason = "RollbackTriggered"
	RollbackReasonRollbackCompleted RollbackReason = "RollbackCompleted"
	RollbackReasonRollbackFailed    RollbackReason = "RollbackFailed"
)
