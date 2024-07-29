package graph

import (
	"time"

	"github.com/odigos-io/odigos/frontend/endpoints"
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

func k8sThinSourceToGql(k8sSource *endpoints.ThinSource) *gqlmodel.K8sActualSource {

	hasInstrumentedApplication := k8sSource.IaDetails != nil

	var gqlIaDetails *gqlmodel.InstrumentedApplicationDetails
	if hasInstrumentedApplication {
		gqlIaDetails = &gqlmodel.InstrumentedApplicationDetails{
			Languages:  make([]*gqlmodel.SourceLanguage, len(k8sSource.IaDetails.Languages)),
			Conditions: make([]*gqlmodel.Condition, len(k8sSource.IaDetails.Conditions)),
		}

		for i, lang := range k8sSource.IaDetails.Languages {
			gqlIaDetails.Languages[i] = &gqlmodel.SourceLanguage{
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

func k8sSourceToGql(k8sSource *endpoints.Source) *gqlmodel.K8sActualSource {
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

func k8sApplicationItemToGql(appItem *endpoints.GetApplicationItemInNamespace) *gqlmodel.K8sActualSource {

	stringKind := string(appItem.Kind)

	return &gqlmodel.K8sActualSource{
		Kind:              k8sKindToGql(stringKind),
		Name:              appItem.Name,
		NumberOfInstances: &appItem.Instances,
	}
}
