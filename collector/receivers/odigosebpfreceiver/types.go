package odigosebpfreceiver

const (
	numOfPages = 2048
)

// ReceiverType defines the type of receiver (traces or metrics)
type ReceiverType int

const (
	ReceiverTypeTraces ReceiverType = iota
	ReceiverTypeMetrics
)
