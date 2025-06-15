package graph

import (
	"context"
	"strings"
	"time"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/services"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func kindToGql(kind string) model.K8sResourceKind {
	switch strings.ToLower(kind) {
	case "deployment":
		return model.K8sResourceKindDeployment
	case "statefulset":
		return model.K8sResourceKindStatefulSet
	case "daemonset":
		return model.K8sResourceKindDaemonSet
	case "cronjob":
		return model.K8sResourceKindCronJob
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

func instrumentationConfigToActualSource(ctx context.Context, instruConfig v1alpha1.InstrumentationConfig, source *v1alpha1.Source) (*model.K8sActualSource, error) {
	selected := true
	dataStreamNames := services.GetSourceDataStreamNames(source)
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
		Kind:              kindToGql(instruConfig.OwnerReferences[0].Kind),
		Name:              instruConfig.OwnerReferences[0].Name,
		Selected:          &selected,
		DataStreamNames:   dataStreamNames,
		OtelServiceName:   &instruConfig.Spec.ServiceName,
		NumberOfInstances: nil,
		Containers:        containers,
		Conditions:        convertConditions(instruConfig.Status.Conditions),
	}, nil
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

			result = append(result, &model.Condition{
				Status:             services.TransformConditionStatus(c.Status, c.Type, reason),
				Type:               c.Type,
				Reason:             &reason,
				Message:            &message,
				LastTransitionTime: k8sLastTransitionTimeToGql(c.LastTransitionTime),
			})
		}
	}
	return result
}
