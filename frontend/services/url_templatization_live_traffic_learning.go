package services

import (
	"context"

	"github.com/odigos-io/odigos/frontend/graph/model"
)

// UrlTemplatizationLiveTrafficLearning resolves
// K8sWorkloadContainerUrlTemplatization.liveTrafficLearning (unmatched paths /
// recommendations for one container). Enterprise UI sets this on server.Deps;
// OSS leaves it nil so the GraphQL field resolves to null without calling an
// implementation.
type UrlTemplatizationLiveTrafficLearning interface {
	GetUrlTemplatizationLiveTrafficLearning(ctx context.Context, namespace, kind, name, containerName string) (*model.K8sWorkloadContainerLiveTrafficLearning, error)
}
