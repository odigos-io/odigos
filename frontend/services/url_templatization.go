package services

import (
	"context"

	"github.com/odigos-io/odigos/frontend/graph/model"
)

// WorkloadUrlTemplatizationFunc resolves URL templatization for a workload
// (existingConfigs + liveTrafficLearning). Enterprise UI sets this on server.Deps;
// OSS leaves it nil so the GraphQL field returns empty data.
type WorkloadUrlTemplatizationFunc func(ctx context.Context, namespace, kind, name string) (*model.K8sWorkloadURLTemplatization, error)

// EmptyWorkloadUrlTemplatization is the OSS / unset-hook response for
// K8sWorkload.urlTemplatization.
func EmptyWorkloadUrlTemplatization() *model.K8sWorkloadURLTemplatization {
	return &model.K8sWorkloadURLTemplatization{
		ExistingConfigs: []*model.URLTemplatizationExistingConfig{},
		LiveTrafficLearning: &model.URLTemplatizationLiveTrafficLearning{
			Server:                 []*model.UnmatchedURLPath{},
			Client:                 []*model.UnmatchedURLPath{},
			ServerRecommendedRules: []*model.URLTemplatizationRecommendedRule{},
			ClientRecommendedRules: []*model.URLTemplatizationRecommendedRule{},
		},
	}
}
