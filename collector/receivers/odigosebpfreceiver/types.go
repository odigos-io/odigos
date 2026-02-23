package odigosebpfreceiver

const (
	numOfPages = 2048
)

// ReceiverType defines the type of receiver (traces, metrics, or logs)
type ReceiverType int

const (
	ReceiverTypeTraces ReceiverType = iota
	ReceiverTypeMetrics
	ReceiverTypeLogs
)
