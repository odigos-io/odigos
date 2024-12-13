package detector

import (
	detector "github.com/odigos-io/runtime-detector"
)

type ProcessEvent = detector.ProcessEvent

type Detector = detector.Detector

type DetectorOption = detector.DetectorOption

const (
	ProcessExecEvent = detector.ProcessExecEvent
	ProcessExitEvent = detector.ProcessExitEvent
)

func NewDetector(events chan<- ProcessEvent, opts ...DetectorOption) (*Detector, error) {
	return detector.NewDetector(events, opts...)
}
