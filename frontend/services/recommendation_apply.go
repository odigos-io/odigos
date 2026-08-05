package services

import (
	"context"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/recommendations"
)

// ApplyRecommendationRemediation looks up the remediation in the catalog manifest and applies
// its steps (EditConfig to odigos-local-ui-config, ApplyOdigosAction from applyExamples).
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

	configSteps := recommendations.ConfigSteps(remediation.Steps)
	actionSteps := recommendations.ActionSteps(remediation.Steps)
	if len(configSteps)+len(actionSteps) != len(remediation.Steps) {
		return fmt.Errorf("remediation %q has unsupported step types", remediationType)
	}

	if len(configSteps) > 0 {
		var applyErr error
		err = upsertLocalUiConfig(ctx, c, func(cfg *common.OdigosConfiguration) {
			applyErr = recommendations.ApplyStepsToConfig(cfg, configSteps)
		})
		if err != nil {
			return err
		}
		if applyErr != nil {
			return applyErr
		}
	}

	for i := range actionSteps {
		if err := applyOdigosActionStep(ctx, remediation); err != nil {
			return fmt.Errorf("ApplyOdigosAction steps[%d]: %w", i, err)
		}
	}

	return nil
}

func applyOdigosActionStep(ctx context.Context, remediation recommendations.Remediation) error {
	example, ok := remediation.OdigosActionExample()
	if !ok {
		return fmt.Errorf("ApplyOdigosAction requires an applyExamples entry of type %q", recommendations.ApplyExampleTypeOdigosAction)
	}
	content := strings.TrimSpace(example.Content)
	if content == "" {
		return fmt.Errorf("OdigosAction applyExamples content is empty")
	}

	var action odigosv1.Action
	if err := yaml.Unmarshal([]byte(content), &action); err != nil {
		return fmt.Errorf("parse OdigosAction YAML: %w", err)
	}
	if action.Name == "" {
		return fmt.Errorf("OdigosAction YAML metadata.name is required")
	}
	action.Namespace = env.GetCurrentNamespace()

	_, err := kube.DefaultClient.OdigosClient.Actions(action.Namespace).Create(ctx, &action, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("create Action %q: %w", action.Name, err)
	}
	return nil
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
