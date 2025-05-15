package graph

import (
	"context"
	"fmt"
	"time"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func instrumentationConfigToActualSource(ctx context.Context, instruConfig v1alpha1.InstrumentationConfig) (*model.K8sActualSource, error) {
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

	workloadConditions, err := instrumentationConfigWorkloadConditions(ctx, instruConfig)
	if err != nil {
		return nil, err
	}

	conditions := append(instruConfig.Status.Conditions, workloadConditions...)

	// Return the converted K8sActualSource object
	return &model.K8sActualSource{
		Namespace:         instruConfig.Namespace,
		Kind:              k8sKindToGql(instruConfig.OwnerReferences[0].Kind),
		Name:              instruConfig.OwnerReferences[0].Name,
		NumberOfInstances: nil,
		OtelServiceName:   &instruConfig.Spec.ServiceName,
		Containers:        containers,
		Conditions:        convertConditions(conditions),
	}, nil
}

func instrumentationConfigWorkloadConditions(ctx context.Context, ic v1alpha1.InstrumentationConfig) ([]v1.Condition, error) {
	conditions := make([]v1.Condition, 0)
	kind := k8sKindToGql(ic.OwnerReferences[0].Kind)
	ns := ic.Namespace
	name := ic.OwnerReferences[0].Name
	switch kind {
	case model.K8sResourceKindDeployment:
		dep, err := kube.DefaultClient.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get Deployment: %w", err)
		}
		for _, c := range dep.Status.Conditions {
			conditions = append(conditions, v1.Condition{
				Type:               string(c.Type),
				Status:             v1.ConditionStatus(c.Status),
				Reason:             c.Reason,
				Message:            c.Message,
				LastTransitionTime: c.LastTransitionTime,
			})
		}
	case model.K8sResourceKindDaemonSet:
		ds, err := kube.DefaultClient.AppsV1().DaemonSets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get DaemonSet: %w", err)
		}
		for _, c := range ds.Status.Conditions {
			conditions = append(conditions, v1.Condition{
				Type:               string(c.Type),
				Status:             v1.ConditionStatus(c.Status),
				Reason:             c.Reason,
				Message:            c.Message,
				LastTransitionTime: c.LastTransitionTime,
			})
		}
	case model.K8sResourceKindStatefulSet:
		ss, err := kube.DefaultClient.AppsV1().StatefulSets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get StatefulSet: %w", err)
		}
		for _, c := range ss.Status.Conditions {
			conditions = append(conditions, v1.Condition{
				Type:               string(c.Type),
				Status:             v1.ConditionStatus(c.Status),
				Reason:             c.Reason,
				Message:            c.Message,
				LastTransitionTime: c.LastTransitionTime,
			})
		}
	default:
		return nil, fmt.Errorf("unknown workload kind: %+v", kind)
	}
	return conditions, nil
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
