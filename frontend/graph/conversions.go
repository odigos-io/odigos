package graph

import (
	"time"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	gqlmodel "github.com/odigos-io/odigos/frontend/graph/model"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func k8sKindToGql(k8sResourceKind string) gqlmodel.K8sResourceKind {
	switch k8sResourceKind {
	case "Deployment":
		return gqlmodel.K8sResourceKindDeployment
	case "StatefulSet":
		return gqlmodel.K8sResourceKindStatefulSet
	case "DaemonSet":
		return gqlmodel.K8sResourceKindDaemonSet
	}
	return ""
}

func k8sConditionStatusToGql(status v1.ConditionStatus) gqlmodel.ConditionStatus {
	switch status {
	case v1.ConditionTrue:
		return gqlmodel.ConditionStatusTrue
	case v1.ConditionFalse:
		return gqlmodel.ConditionStatusFalse
	case v1.ConditionUnknown:
		return gqlmodel.ConditionStatusUnknown
	}
	return gqlmodel.ConditionStatusUnknown

}

func k8sLastTransitionTimeToGql(t v1.Time) *string {
	if t.IsZero() {
		return nil
	}
	str := t.UTC().Format(time.RFC3339)
	return &str
}

func instrumentedApplicationToActualSource(instrumentedApp v1alpha1.InstrumentedApplication) *gqlmodel.K8sActualSource {
	// Map the container runtime details
	var containers []*gqlmodel.SourceContainerRuntimeDetails
	for _, container := range instrumentedApp.Spec.RuntimeDetails {
		var otherAgentName *string
		if container.OtherAgent != nil {
			otherAgentName = &container.OtherAgent.Name
		}

		containers = append(containers, &gqlmodel.SourceContainerRuntimeDetails{
			ContainerName:  container.ContainerName,
			Language:       string(container.Language),
			RuntimeVersion: container.RuntimeVersion,
			OtherAgent:     otherAgentName,
		})
	}

	// Map the conditions of the application
	var conditions []*gqlmodel.Condition
	for _, condition := range instrumentedApp.Status.Conditions {
		conditions = append(conditions, &gqlmodel.Condition{
			Type:               condition.Type,
			Status:             k8sConditionStatusToGql(condition.Status),
			Reason:             &condition.Reason,
			LastTransitionTime: k8sLastTransitionTimeToGql(condition.LastTransitionTime),
			Message:            &condition.Message,
		})
	}

	// Return the converted K8sActualSource object
	return &gqlmodel.K8sActualSource{
		Namespace:         instrumentedApp.Namespace,
		Kind:              k8sKindToGql(instrumentedApp.OwnerReferences[0].Kind),
		Name:              instrumentedApp.OwnerReferences[0].Name,
		ServiceName:       &instrumentedApp.Name,
		NumberOfInstances: nil,
		AutoInstrumented:  instrumentedApp.Spec.Options != nil,
		InstrumentedApplicationDetails: &gqlmodel.InstrumentedApplicationDetails{
			Containers: containers,
			Conditions: conditions,
		},
	}
}
