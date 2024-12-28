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

func instrumentationConfigToActualSource(instruConfig v1alpha1.InstrumentationConfig) *model.K8sActualSource {
	// Map the containers runtime details
	var containers []*model.SourceContainerRuntimeDetails
	for _, container := range instruConfig.Status.RuntimeDetailsByContainer {
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

	// Map the conditions
	var conditions []*model.Condition
	for _, condition := range instruConfig.Status.Conditions {
		conditions = append(conditions, &model.Condition{
			Status:             k8sConditionStatusToGql(condition.Status),
			Type:               condition.Type,
			Reason:             &condition.Reason,
			Message:            &condition.Message,
			LastTransitionTime: k8sLastTransitionTimeToGql(condition.LastTransitionTime),
		})
	}

	// Return the converted K8sActualSource object
	return &model.K8sActualSource{
		Namespace:         instruConfig.Namespace,
		Kind:              k8sKindToGql(instruConfig.OwnerReferences[0].Kind),
		Name:              instruConfig.OwnerReferences[0].Name,
		NumberOfInstances: nil,
		ServiceName:       &instruConfig.Name,
		ReportedName:      &instruConfig.Spec.ServiceName,
		Containers:        containers,
		Conditions:        conditions,
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
		// Convert LastTransitionTime to a string pointer if it's not nil
		var lastTransitionTime *string
		if !c.LastTransitionTime.IsZero() {
			t := c.LastTransitionTime.Format(time.RFC3339)
			lastTransitionTime = &t
		}

		result = append(result, &model.Condition{
			Status:             model.ConditionStatus(c.Status),
			Type:               c.Type,
			Reason:             &c.Reason,
			Message:            &c.Message,
			LastTransitionTime: lastTransitionTime,
		})
	}
	return result
}
