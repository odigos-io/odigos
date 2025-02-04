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

// Convert LastTransitionTime to a string pointer if it's not nil
func k8sLastTransitionTimeToGql(t v1.Time) *string {
	if t.IsZero() {
		return nil
	}
	str := t.UTC().Format(time.RFC3339)
	return &str
}

func instrumentationConfigToActualSource(instruConfig v1alpha1.InstrumentationConfig) *model.K8sActualSource {
	var containers []*model.SourceContainer

	// Map the containers runtime details
	for _, statusContainer := range instruConfig.Status.RuntimeDetailsByContainer {
		var instrumented bool
		var instrumentationMessage string
		var otherAgentName *string

		for _, specContainer := range instruConfig.Spec.Containers {
			if specContainer.ContainerName == statusContainer.ContainerName {
				instrumented = specContainer.Instrumented
				instrumentationMessage = specContainer.InstrumentationMessage
				if instrumentationMessage == "" {
					instrumentationMessage = string(specContainer.InstrumentationReason)
				}
			}
		}

		if statusContainer.OtherAgent != nil {
			otherAgentName = &statusContainer.OtherAgent.Name
		}

		containers = append(containers, &model.SourceContainer{
			ContainerName:          statusContainer.ContainerName,
			Language:               string(statusContainer.Language),
			RuntimeVersion:         statusContainer.RuntimeVersion,
			Instrumented:           instrumented,
			InstrumentationMessage: instrumentationMessage,
			OtherAgent:             otherAgentName,
		})
	}

	// Return the converted K8sActualSource object
	return &model.K8sActualSource{
		Namespace:         instruConfig.Namespace,
		Kind:              k8sKindToGql(instruConfig.OwnerReferences[0].Kind),
		Name:              instruConfig.OwnerReferences[0].Name,
		NumberOfInstances: nil,
		OtelServiceName:   &instruConfig.Spec.ServiceName,
		Containers:        containers,
		Conditions:        convertConditions(instruConfig.Status.Conditions),
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
		if c.Type != "AppliedInstrumentationDevice" {
			message := c.Message
			if message == "" {
				message = string(c.Reason)
			}

			result = append(result, &model.Condition{
				Status:             model.ConditionStatus(c.Status),
				Message:            &message,
				LastTransitionTime: k8sLastTransitionTimeToGql(c.LastTransitionTime),
			})
		}
	}
	return result
}
