package graph

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

// workloadIDFromContainerFieldContext walks
// current field -> *K8sWorkloadContainer -> []*K8sWorkloadContainer -> *K8sWorkload.
func workloadIDFromContainerFieldContext(ctx context.Context) (model.K8sWorkloadID, error) {
	fc := graphql.GetFieldContext(ctx)
	if fc == nil || fc.Parent == nil || fc.Parent.Parent == nil || fc.Parent.Parent.Parent == nil {
		return model.K8sWorkloadID{}, fmt.Errorf("missing parent resolver context")
	}
	w, ok := fc.Parent.Parent.Parent.Result.(**model.K8sWorkload)
	if !ok || w == nil || (*w).ID == nil {
		return model.K8sWorkloadID{}, fmt.Errorf("parent is not a workload")
	}
	return *(*w).ID, nil
}

// workloadContainerFromUrlTemplatizationFieldContext walks
// current field -> *K8sWorkloadContainerURLTemplatization -> *K8sWorkloadContainer
// -> []*K8sWorkloadContainer -> *K8sWorkload.
func workloadContainerFromUrlTemplatizationFieldContext(ctx context.Context) (model.K8sWorkloadID, string, error) {
	fc := graphql.GetFieldContext(ctx)
	if fc == nil || fc.Parent == nil || fc.Parent.Parent == nil || fc.Parent.Parent.Parent == nil || fc.Parent.Parent.Parent.Parent == nil {
		return model.K8sWorkloadID{}, "", fmt.Errorf("missing parent resolver context")
	}
	containerName, ok := containerNameFromFieldResult(fc.Parent.Parent.Result)
	if !ok {
		return model.K8sWorkloadID{}, "", fmt.Errorf("parent is not a workload container")
	}
	w, ok := fc.Parent.Parent.Parent.Parent.Result.(**model.K8sWorkload)
	if !ok || w == nil || (*w).ID == nil {
		return model.K8sWorkloadID{}, "", fmt.Errorf("parent is not a workload")
	}
	return *(*w).ID, containerName, nil
}

func containerNameFromFieldResult(result any) (string, bool) {
	switch c := result.(type) {
	case *model.K8sWorkloadContainer:
		if c == nil || c.ContainerName == "" {
			return "", false
		}
		return c.ContainerName, true
	case **model.K8sWorkloadContainer:
		if c == nil || *c == nil || (*c).ContainerName == "" {
			return "", false
		}
		return (*c).ContainerName, true
	default:
		return "", false
	}
}
