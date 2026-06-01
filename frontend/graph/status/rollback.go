package status

import (
	"fmt"
	"time"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/computed"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

const (
	RollbackStatus = "Rollback"
)

type AutoRollbackReason string

const (
	AutoRollbackReasonDisabled          AutoRollbackReason = "Disabled"
	AutoRollbackReasonAgentNotEnabled   AutoRollbackReason = "AgentNotEnabled"
	AutoRollbackReasonRollbackOccurred  AutoRollbackReason = "RollbackOccurred"
	AutoRollbackReasonStable            AutoRollbackReason = "Stable"
	AutoRollbackReasonEvaluating        AutoRollbackReason = "Evaluating"
	AutoRollbackReasonWaitingForRollout AutoRollbackReason = "WaitingForRollout"
)

func createAutoRollbackStatus(reason AutoRollbackReason, message string, status model.DesiredStateProgress) *model.DesiredConditionStatus {
	reasonStr := string(reason)
	return &model.DesiredConditionStatus{
		Name:       RollbackStatus,
		Status:     status,
		ReasonEnum: &reasonStr,
		Message:    message,
	}
}

func CalculateAutoRollbackStatus(ic *odigosv1alpha1.InstrumentationConfig, autoRollbackConfig *computed.AutoRollbackConfig) *model.DesiredConditionStatus {

	// if the workload is not marked for instrumentation, the auto rollback status is not applicable.
	if ic == nil {
		return nil
	}

	// disabled in config
	if !autoRollbackConfig.Enabled {
		return createAutoRollbackStatus(AutoRollbackReasonDisabled, "Auto rollback is disabled in the odigos configuration", model.DesiredStateProgressIrrelevant)
	}

	// agent injection is not enabled (e.g. no agent, other agent, etc.)
	// in this case the auto rollback is not applicable.
	if !ic.Spec.AgentInjectionEnabled {
		return createAutoRollbackStatus(AutoRollbackReasonAgentNotEnabled, "odigos agent is not set to run with this source, auto rollback is not applicable", model.DesiredStateProgressIrrelevant)
	}

	// if we know it was rolled back due to auto-heal
	if ic.Status.RollbackOccurred {
		return createAutoRollbackStatus(AutoRollbackReasonRollbackOccurred, "odigos detected a crash and rolled back the source to protect your application", model.DesiredStateProgressNotice)
	}

	// rollback is only checked after the workload is rolled out.
	if ic.Status.InstrumentationTime == nil {
		return createAutoRollbackStatus(AutoRollbackReasonWaitingForRollout, "source stability will be checked after the source is rolled out by odigos", model.DesiredStateProgressWaiting)
	}

	// check if we reached the stability window
	timeSinceInstrumentation := time.Since(ic.Status.InstrumentationTime.Time)
	if timeSinceInstrumentation < autoRollbackConfig.StabilityWindow {
		// if we are within the stability window, and did not mark the rollback occurred, we are still evaluating
		return createAutoRollbackStatus(AutoRollbackReasonEvaluating, fmt.Sprintf("evaluating pods stability for %s", autoRollbackConfig.StabilityWindow), model.DesiredStateProgressSuccess)
	}

	return createAutoRollbackStatus(AutoRollbackReasonStable, "pods are stable after instrumentation", model.DesiredStateProgressSuccess)
}
