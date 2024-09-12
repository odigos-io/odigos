package graph

import (
	"time"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	gqlmodel "github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/services"
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

func k8sThinSourceToGql(k8sSource *services.ThinSource) *gqlmodel.K8sActualSource {

	hasInstrumentedApplication := k8sSource.IaDetails != nil

	var gqlIaDetails *gqlmodel.InstrumentedApplicationDetails
	if hasInstrumentedApplication {
		gqlIaDetails = &gqlmodel.InstrumentedApplicationDetails{
			Containers: make([]*gqlmodel.SourceContainerRuntimeDetails, len(k8sSource.IaDetails.Languages)),
			Conditions: make([]*gqlmodel.Condition, len(k8sSource.IaDetails.Conditions)),
		}

		for i, lang := range k8sSource.IaDetails.Languages {
			gqlIaDetails.Containers[i] = &gqlmodel.SourceContainerRuntimeDetails{
				ContainerName: lang.ContainerName,
				Language:      lang.Language,
			}
		}

		for i, cond := range k8sSource.IaDetails.Conditions {
			gqlIaDetails.Conditions[i] = &gqlmodel.Condition{
				Type:               cond.Type,
				Status:             k8sConditionStatusToGql(cond.Status),
				Reason:             &cond.Reason,
				LastTransitionTime: k8sLastTransitionTimeToGql(cond.LastTransitionTime),
				Message:            &cond.Message,
			}
		}
	}

	return &gqlmodel.K8sActualSource{
		Namespace:                      k8sSource.Namespace,
		Kind:                           k8sKindToGql(k8sSource.Kind),
		Name:                           k8sSource.Name,
		NumberOfInstances:              &k8sSource.NumberOfRunningInstances,
		InstrumentedApplicationDetails: gqlIaDetails,
	}
}

func k8sSourceToGql(k8sSource *services.Source) *gqlmodel.K8sActualSource {
	baseSource := k8sThinSourceToGql(&k8sSource.ThinSource)
	return &gqlmodel.K8sActualSource{
		Namespace:                      baseSource.Namespace,
		Kind:                           baseSource.Kind,
		Name:                           baseSource.Name,
		NumberOfInstances:              baseSource.NumberOfInstances,
		InstrumentedApplicationDetails: baseSource.InstrumentedApplicationDetails,
		ServiceName:                    &k8sSource.ReportedName,
	}
}

func instrumentedApplicationToActualSource(instrumentedApp v1alpha1.InstrumentedApplication) *gqlmodel.K8sActualSource {
	// Map the container runtime details
	var containers []*gqlmodel.SourceContainerRuntimeDetails
	for _, container := range instrumentedApp.Spec.RuntimeDetails {
		containers = append(containers, &gqlmodel.SourceContainerRuntimeDetails{
			ContainerName: container.ContainerName,
			Language:      string(container.Language),
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

	// Map the options for instrumentation libraries
	var instrumentationOptions []*gqlmodel.InstrumentedApplicationDetails
	for _, option := range instrumentedApp.Spec.Options {
		for _, libOptions := range option.InstrumentationLibraries {
			var libraries []*gqlmodel.InstrumentationOption
			for _, configOption := range libOptions.Options {
				libraries = append(libraries, &gqlmodel.InstrumentationOption{
					OptionKey: configOption.OptionKey,
					SpanKind:  gqlmodel.SpanKind(configOption.SpanKind),
				})
			}

			instrumentationOptions = append(instrumentationOptions, &gqlmodel.InstrumentedApplicationDetails{
				Containers: containers,
				Conditions: conditions,
			})
		}
	}

	// Return the converted K8sActualSource object
	return &gqlmodel.K8sActualSource{
		Namespace:                instrumentedApp.Namespace,
		Kind:                     k8sKindToGql(instrumentedApp.OwnerReferences[0].Kind),
		Name:                     instrumentedApp.OwnerReferences[0].Name,
		ServiceName:              &instrumentedApp.Name, // Assuming serviceName is derived from the app name
		NumberOfInstances:        nil,                   // Assuming this is handled separately; can be updated if needed
		AutoInstrumented:         instrumentedApp.Spec.Options != nil,
		AutoInstrumentedDecision: "", // Assuming this comes from another source, add if applicable
		InstrumentedApplicationDetails: &gqlmodel.InstrumentedApplicationDetails{
			Containers: containers,
			Conditions: conditions,
		},
	}
}
