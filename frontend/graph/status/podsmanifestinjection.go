package status

import (
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/computed"
	"github.com/odigos-io/odigos/frontend/graph/model"
	podsManifestInjectionStatus "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
)

func CalculatePodsManifestInjectionStatus(ic *v1alpha1.InstrumentationConfig) *model.DesiredConditionStatus {
	if ic == nil {
		return nil
	}

	for _, c := range ic.Status.Conditions {
		if c.Type == podsManifestInjectionStatus.PodsManifestInjectionType {
			r, ok := podsManifestInjectionStatus.PodsManifestInjectionReasonByName(c.Reason)
			if !ok {
				return nil
			}

			actionItems := make([]*model.DesiredConditionActionItem, 0, len(r.ActionItems))
			for _, actionItem := range r.ActionItems {
				actionItems = append(actionItems, &model.DesiredConditionActionItem{
					Type:           model.DesiredConditionActionItemType(actionItem.Type),
					UserFacingText: actionItem.ButtonText,
				})
			}

			return &model.DesiredConditionStatus{
				Name:        c.Type,
				Status:      model.DesiredStateProgress(r.OdigosSeverity),
				ReasonEnum:  &r.Title,
				Message:     c.Message,
				ActionItems: actionItems,
			}
		}
	}

	return nil
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
