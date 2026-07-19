package status

import (
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/computed"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/status"
	podsManifestInjectionStatus "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
)

func CalculatePodsManifestInjectionStatus(ic *v1alpha1.InstrumentationConfig, pods []computed.CachedPod) *model.DesiredConditionStatus {
	if ic == nil {
		return calculateUnmarkedPodsManifestInjectionStatus(pods)
	}

	for _, c := range ic.Status.Conditions {
		if c.Type == podsManifestInjectionStatus.PodsManifestInjectionType {
			r, ok := podsManifestInjectionStatus.PodsManifestInjectionReasonByName(c.Reason)
			if !ok {
				return &model.DesiredConditionStatus{
					Name:    podsManifestInjectionStatus.PodsManifestInjectionType,
					Status:  model.DesiredStateProgressUnknown,
					Message: c.Message,
				}
			}

			return podsManifestInjectionReasonToStatus(r, c.Message)
		}
	}

	return podsManifestInjectionReasonToStatus(podsManifestInjectionStatus.PodsManifestInjectionNotYetReconciled, "")
}

func calculateUnmarkedPodsManifestInjectionStatus(pods []computed.CachedPod) *model.DesiredConditionStatus {
	if len(pods) == 0 {
		return podsManifestInjectionReasonToStatus(podsManifestInjectionStatus.PodsManifestInjectionNoPods, "")
	}
	for _, pod := range pods {
		if pod.AgentInjected {
			return podsManifestInjectionReasonToStatus(podsManifestInjectionStatus.PodsManifestInjectionUnmarkedFromOdigos_Disabled, "")
		}
	}
	return podsManifestInjectionReasonToStatus(podsManifestInjectionStatus.PodsManifestInjectionPodsAppliedSuccessfully_Disabled, "")
}

func podsManifestInjectionReasonToStatus(r status.Reason, message string) *model.DesiredConditionStatus {
	if message == "" {
		message = r.Message
	}
	if message == "" {
		message = r.Summary
	}

	actionItems := make([]*model.DesiredConditionActionItem, 0, len(r.ActionItems))
	for _, actionItem := range r.ActionItems {
		actionItems = append(actionItems, &model.DesiredConditionActionItem{
			Type:       model.DesiredConditionActionItemType(actionItem.Type),
			ButtonText: actionItem.ButtonText,
		})
	}

	return &model.DesiredConditionStatus{
		Name:        podsManifestInjectionStatus.PodsManifestInjectionType,
		Status:      model.DesiredStateProgress(r.OdigosSeverity),
		ReasonEnum:  &r.Title,
		Message:     message,
		ActionItems: actionItems,
	}
}

func CalculatePodsManifestInjectionOverview(ic *v1alpha1.InstrumentationConfig, pods []computed.CachedPod) *model.K8sWorkloadPodsManifestInjectionOverview {
	totalPods := len(pods)
	var totalAgentNotAppliedPods, totalAgentAppliedPods, totalAgentOutOfDatePods int

	desiredHash := ""
	if ic != nil {
		desiredHash = ic.Spec.AgentsMetaHash
	}
	podManifestInjectionOptional := ic != nil && ic.Spec.AgentInjectionEnabled && ic.Spec.PodManifestInjectionOptional

	for _, pod := range pods {
		// injected via webhook pod mutation
		if pod.AgentsMetaHash != "" {
			if desiredHash != "" && pod.AgentsMetaHash != desiredHash {
				totalAgentOutOfDatePods++
			} else {
				totalAgentAppliedPods++
			}
		} else { // pod not injected via webhook pod mutation
			if podManifestInjectionOptional {
				// treat "no restart" distros as applied
				totalAgentAppliedPods++
			} else {
				totalAgentNotAppliedPods++
			}
		}
	}

	agentEnabled := ic != nil && ic.Spec.AgentInjectionEnabled
	var agentNotAppliedOk, agentAppliedOk, agentOutOfDateOk bool
	if agentEnabled {
		agentNotAppliedOk = totalAgentNotAppliedPods == 0
		agentAppliedOk = true
		agentOutOfDateOk = totalAgentOutOfDatePods == 0
	} else {
		agentNotAppliedOk = true
		agentAppliedOk = totalAgentAppliedPods == 0
		agentOutOfDateOk = totalAgentOutOfDatePods == 0
	}

	return &model.K8sWorkloadPodsManifestInjectionOverview{
		TotalPods:                totalPods,
		TotalAgentNotAppliedPods: totalAgentNotAppliedPods,
		AgentNotAppliedOk:        agentNotAppliedOk,
		TotalAgentAppliedPods:    totalAgentAppliedPods,
		AgentAppliedOk:           agentAppliedOk,
		TotalAgentOutOfDatePods:  totalAgentOutOfDatePods,
		AgentOutOfDateOk:         agentOutOfDateOk,
	}
}
