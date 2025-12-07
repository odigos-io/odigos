package detector

import (
	"context"

	detector "github.com/odigos-io/runtime-detector"
)

type ProcessEvent = detector.ProcessEvent
type ExecDetails = detector.ProcessExecDetails

type DetectorOption = detector.DetectorOption

const (
	ProcessExecEvent     = detector.ProcessExecEvent
	ProcessForkEvent     = detector.ProcessForkEvent
	ProcessFileOpenEvent = detector.ProcessFileOpenEvent
	ProcessExitEvent     = detector.ProcessExitEvent
)

// Detector is used to report process events.
type Detector interface {
	Run(ctx context.Context) error
}

// NewDetector creates a new Detector.
//
// The process events are sent to the events channel.
// The channel is closed when the detector is stopped (just before returning from Run).
func NewDetector(events chan<- ProcessEvent, opts ...DetectorOption) (Detector, error) {
	return detector.NewDetector(events, opts...)
}
