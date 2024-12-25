package graph

import (
	"time"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/destinations"
	"github.com/odigos-io/odigos/frontend/graph/model"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func k8sKindToGql(k8sResourceKind string) model.K8sResourceKind {
	switch k8sResourceKind {
	case "Deployment":
		return model.K8sResourceKindDeployment
	case "StatefulSet":
		return model.K8sResourceKindStatefulSet
	case "DaemonSet":
		return model.K8sResourceKindDaemonSet
	}
	return ""
}

func k8sConditionStatusToGql(status v1.ConditionStatus) model.ConditionStatus {
	switch status {
	case v1.ConditionTrue:
		return model.ConditionStatusTrue
	case v1.ConditionFalse:
		return model.ConditionStatusFalse
	case v1.ConditionUnknown:
		return model.ConditionStatusUnknown
	}
	return model.ConditionStatusUnknown

}

func k8sLastTransitionTimeToGql(t v1.Time) *string {
	if t.IsZero() {
		return nil
	}
	str := t.UTC().Format(time.RFC3339)
	return &str
}

func instrumentedApplicationToActualSource(instrumentedApp v1alpha1.InstrumentedApplication) *model.K8sActualSource {
	// Map the container runtime details
	var containers []*model.SourceContainerRuntimeDetails
	for _, container := range instrumentedApp.Spec.RuntimeDetails {
		var otherAgentName *string
		if container.OtherAgent != nil {
			otherAgentName = &container.OtherAgent.Name
		}

		containers = append(containers, &model.SourceContainerRuntimeDetails{
			ContainerName:  container.ContainerName,
			Language:       string(container.Language),
			RuntimeVersion: container.RuntimeVersion,
			OtherAgent:     otherAgentName,
		})
	}

	// Map the conditions of the application
	var conditions []*model.Condition
	for _, condition := range instrumentedApp.Status.Conditions {
		conditions = append(conditions, &model.Condition{
			Type:               condition.Type,
			Status:             k8sConditionStatusToGql(condition.Status),
			Reason:             &condition.Reason,
			LastTransitionTime: k8sLastTransitionTimeToGql(condition.LastTransitionTime),
			Message:            &condition.Message,
		})
	}

	// Return the converted K8sActualSource object
	return &model.K8sActualSource{
		Namespace:         instrumentedApp.Namespace,
		Kind:              k8sKindToGql(instrumentedApp.OwnerReferences[0].Kind),
		Name:              instrumentedApp.OwnerReferences[0].Name,
		ServiceName:       &instrumentedApp.Name,
		NumberOfInstances: nil,
		AutoInstrumented:  instrumentedApp.Spec.Options != nil,
		InstrumentedApplicationDetails: &model.InstrumentedApplicationDetails{
			Containers: containers,
			Conditions: conditions,
		},
	}
}

func convertCustomReadDataLabels(labels []*destinations.CustomReadDataLabel) []*model.CustomReadDataLabel {
	var result []*model.CustomReadDataLabel
	for _, label := range labels {
		result = append(result, &model.CustomReadDataLabel{
			Condition: label.Condition,
			Title:     label.Title,
			Value:     label.Value,
		})
	}
	return result
}

func convertConditions(conditions []v1.Condition) []*model.Condition {
	var result []*model.Condition
	for _, c := range conditions {
		result = append(result, &model.Condition{
			Status:             model.ConditionStatus(c.Status),
			Type:               c.Type,
			Reason:             &c.Reason,
			Message:            &c.Message,
			LastTransitionTime: convertLastTransitionTime(c.LastTransitionTime),
		})
	}
	return result
}

// Convert LastTransitionTime to a string pointer if it's not nil
func convertLastTransitionTime(transitionTime v1.Time) *string {
	var lastTransitionTime *string

	if !transitionTime.IsZero() {
		t := transitionTime.Format(time.RFC3339)
		lastTransitionTime = &t
	}

	return lastTransitionTime
}
