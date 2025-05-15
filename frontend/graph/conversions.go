package graph

import (
	"time"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/services"

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

// Convert LastTransitionTime to a string pointer if it's not nil
func k8sLastTransitionTimeToGql(t v1.Time) *string {
	if t.IsZero() {
		return nil
	}
	str := t.UTC().Format(time.RFC3339)
	return &str
}

func instrumentationConfigToActualSource(instruConfig v1alpha1.InstrumentationConfig, source v1alpha1.Source) *model.K8sActualSource {
	selected := true
	streamNames := services.GetSourceStreamNames(&source)
	var containers []*model.SourceContainer

	// Map the containers runtime details
	for _, statusContainer := range instruConfig.Status.RuntimeDetailsByContainer {
		var instrumented bool
		var instrumentationMessage string
		var otelDistroName string

		for _, specContainer := range instruConfig.Spec.Containers {
			if specContainer.ContainerName == statusContainer.ContainerName {
				instrumented = specContainer.AgentEnabled
				instrumentationMessage = specContainer.AgentEnabledMessage
				if instrumentationMessage == "" {
					instrumentationMessage = string(specContainer.AgentEnabledReason)
				}
				otelDistroName = specContainer.OtelDistroName
			}
		}

		containers = append(containers, &model.SourceContainer{
			ContainerName:          statusContainer.ContainerName,
			Language:               string(statusContainer.Language),
			RuntimeVersion:         statusContainer.RuntimeVersion,
			Instrumented:           instrumented,
			InstrumentationMessage: instrumentationMessage,
			OtelDistroName:         &otelDistroName,
		})
	}

	// Return the converted K8sActualSource object
	return &model.K8sActualSource{
		Namespace:         instruConfig.Namespace,
		Kind:              k8sKindToGql(instruConfig.OwnerReferences[0].Kind),
		Name:              instruConfig.OwnerReferences[0].Name,
		Selected:          &selected,
		StreamNames:       streamNames,
		OtelServiceName:   &instruConfig.Spec.ServiceName,
		NumberOfInstances: nil,
		Containers:        containers,
		Conditions:        convertConditions(instruConfig.Status.Conditions),
	}
}

func convertConditions(conditions []v1.Condition) []*model.Condition {
	var result []*model.Condition
	for _, c := range conditions {
		if c.Type != "AppliedInstrumentationDevice" {
			reason := c.Reason
			message := c.Message
			if message == "" {
				message = string(c.Reason)
			}

			var status model.ConditionStatus

			switch c.Status {
			case v1.ConditionUnknown:
				status = model.ConditionStatusLoading
			case v1.ConditionTrue:
				status = model.ConditionStatusSuccess
			case v1.ConditionFalse:
				status = model.ConditionStatusError
			}

			// force "disabled" status ovverrides for certain "reasons"
			if v1alpha1.IsReasonStatusDisabled(reason) {
				status = model.ConditionStatusDisabled
			}

			result = append(result, &model.Condition{
				Status:             status,
				Type:               c.Type,
				Reason:             &reason,
				Message:            &message,
				LastTransitionTime: k8sLastTransitionTimeToGql(c.LastTransitionTime),
			})
		}
	}
	return result
}
