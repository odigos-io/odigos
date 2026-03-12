package odigosebpfreceiver

const (
	numOfPages       = 2048
	attrValueMaxSize = 1024
)

// ReceiverType defines the type of receiver (traces, metrics, or logs)
type ReceiverType int

const (
	ReceiverTypeTraces ReceiverType = iota
	ReceiverTypeMetrics
	ReceiverTypeLogs
)
