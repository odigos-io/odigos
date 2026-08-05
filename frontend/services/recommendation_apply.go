package services

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/recommendations"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ApplyRecommendationRemediation looks up the remediation in the catalog manifest and applies
// its steps to odigos-local-ui-config.
func ApplyRecommendationRemediation(ctx context.Context, c client.Client, recommendationType model.RecommendationType, remediationType string) error {
	catalogType, err := toCommonRecommendationType(recommendationType)
	if err != nil {
		return err
	}

	catalog, ok := recommendations.GetByType(catalogType)
	if !ok {
		return fmt.Errorf("recommendation type %q not found in catalog", recommendationType)
	}

	remediation, ok := catalog.RemediationByType(remediationType)
	if !ok {
		return fmt.Errorf("remediation type %q not found for recommendation %q", remediationType, recommendationType)
	}
	if len(remediation.Steps) == 0 {
		return fmt.Errorf("remediation %q for recommendation %q has no steps", remediationType, recommendationType)
	}

	var applyErr error
	err = upsertLocalUiConfig(ctx, c, func(cfg *common.OdigosConfiguration) {
		applyErr = recommendations.ApplyStepsToConfig(cfg, remediation.Steps)
	})
	if err != nil {
		return err
	}
	return applyErr
}

func toCommonRecommendationType(t model.RecommendationType) (common.RecommendationType, error) {
	switch t {
	case model.RecommendationTypeInferDBAttributes:
		return common.RecommendationTypeInferDBAttributes, nil
	case model.RecommendationTypeAutoGoOffsetUpdater:
		return common.RecommendationTypeAutoGoOffsetUpdater, nil
	case model.RecommendationTypeEnableOwnMetrics:
		return common.RecommendationTypeEnableOwnMetrics, nil
	case model.RecommendationTypeSampleHealthProbes:
		return common.RecommendationTypeSampleHealthProbes, nil
	case model.RecommendationTypeURLTemplatization:
		return common.RecommendationTypeUrlTemplatization, nil
	default:
		return "", fmt.Errorf("unknown recommendation type %q", t)
	}
}
