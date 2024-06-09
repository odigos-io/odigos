package collectormetrics

type NotificationReason string

const (
	NewIPDiscovered     NotificationReason = "NewIPDiscovered"
	IPRemoved           NotificationReason = "IPRemoved"
	OdigosConfigUpdated NotificationReason = "OdigosConfigUpdated"
)

type Notification struct {
	Reason  NotificationReason
	PodName string
	IP      string
}
