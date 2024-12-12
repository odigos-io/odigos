package detector

import (
	"context"

	detector "github.com/odigos-io/runtime-detector"
)

type ProcessEvent = detector.ProcessEvent

type Detector = detector.Detector

type DetectorOption = detector.DetectorOption

const (
	ProcessExecEvent = detector.ProcessExecEvent
	ProcessExitEvent = detector.ProcessExitEvent
)

func NewDetector(ctx context.Context, events chan<- ProcessEvent, opts ...DetectorOption) (*Detector, error) {
	return detector.NewDetector(ctx, events, opts...)
}