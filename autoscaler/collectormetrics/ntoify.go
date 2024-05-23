package collectormetrics

type NotificationReason string

const (
	NewIPDiscovered NotificationReason = "NewIPDiscovered"
	IPRemoved       NotificationReason = "IPRemoved"
)

type Notification struct {
	Reason  NotificationReason
	PodName string
	IP      string
}
