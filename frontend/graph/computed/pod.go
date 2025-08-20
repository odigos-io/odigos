package computed

import (
	"github.com/odigos-io/odigos/frontend/graph/model"
)

type ComputedPodContainer struct {
	ContainerName string

	// Computed values extracted from the pod manifest
	OtelDistroName                    *string
	ExpectingInstrumentationInstances bool
	OdigosInstrumentationDeviceName   *string

	// Values as found in the pod status.
	// can be nil if status is not available or when the value is empty in container status
	Started *bool
	Ready   *bool

	// same as Ready, but not nil (IsReady is true if Ready is not nil and true)
	IsReady bool

	// true if the container is in a crash loop back off
	IsCrashLoop bool

	// Values as found in the container status.
	// nil if status is not available, not relevant, or unset in the container status
	RestartCount       *int
	RunningStartedTime *string
	WaitingReasonEnum  *string
	WaitingMessage     *string
}

type CachedPod struct {

	// Pod id
	PodNamespace string
	PodName      string

	// relevant values from the pod manifest
	PodNodeName  string
	PodStartTime string

	AgentInjected       bool
	AgentInjectedStatus *model.DesiredConditionStatus
	Containers          []ComputedPodContainer
}
