package services

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/recommendations"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetRecommendations(ctx context.Context) ([]*model.Recommendation, error) {
	odigosNs := env.GetCurrentNamespace()

	list, err := kube.DefaultClient.OdigosClient.Recommendations(odigosNs).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get recommendations: %v", err)
	}

	response := make([]*model.Recommendation, 0, len(list.Items))
	for i := range list.Items {
		converted, err := convertRecommendationToModel(&list.Items[i])
		if err != nil {
			return nil, err
		}
		response = append(response, converted)
	}

	return response, nil
}

func SetRecommendationDismissed(ctx context.Context, name string, dismissed bool) (*model.Recommendation, error) {
	odigosNs := env.GetCurrentNamespace()

	rec, err := kube.DefaultClient.OdigosClient.Recommendations(odigosNs).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get recommendation %q: %v", name, err)
	}

	if dismissed {
		if rec.Labels == nil {
			rec.Labels = map[string]string{}
		}
		rec.Labels[k8sconsts.RecommendationDismissedLabel] = "true"
	} else if rec.Labels != nil {
		delete(rec.Labels, k8sconsts.RecommendationDismissedLabel)
	}

	updated, err := kube.DefaultClient.OdigosClient.Recommendations(odigosNs).Update(ctx, rec, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update recommendation %q dismissed state: %v", name, err)
	}

	return convertRecommendationToModel(updated)
}

func convertRecommendationToModel(rec *v1alpha1.Recommendation) (*model.Recommendation, error) {
	recType, err := toGraphRecommendationType(rec.Spec.Type)
	if err != nil {
		return nil, err
	}

	catalog, ok := recommendations.GetByType(rec.Spec.Type)
	if !ok {
		catalog, ok = recommendations.GetByK8sObjectName(rec.Name)
	}
	if !ok {
		return nil, fmt.Errorf("recommendation catalog entry not found for type %q (name %q)", rec.Spec.Type, rec.Name)
	}

	return &model.Recommendation{
		Name:                    rec.Name,
		Type:                    recType,
		Applied:                 rec.Spec.Applied,
		ConditionsMet:           rec.Spec.ConditionsMet,
		Dismissed:               isRecommendationDismissed(rec),
		Oss:                     catalog.OSS,
		RequireOdigosDeployment: catalog.RequireOdigosDeployment,
		CatalogConditions:       toCatalogConditions(catalog.Conditions),
		AppliedWhen:             toAppliedWhen(catalog.AppliedWhen),
		Title:                   catalog.Title,
		Summary:                 catalog.Summary,
		Description:             catalog.Description,
		DocsURL:                 optionalString(catalog.Docs),
		Pros:                    nonNilStrings(catalog.Pros),
		Cons:                    nonNilStrings(catalog.Cons),
		Actions:                 toCatalogActions(catalog.Actions),
	}, nil
}

func isRecommendationDismissed(rec *v1alpha1.Recommendation) bool {
	return rec.Labels[k8sconsts.RecommendationDismissedLabel] == "true"
}

func toCatalogConditions(conditions []recommendations.Condition) []*model.RecommendationCatalogCondition {
	result := make([]*model.RecommendationCatalogCondition, 0, len(conditions))
	for _, c := range conditions {
		result = append(result, &model.RecommendationCatalogCondition{
			Type: c.Type,
		})
	}
	return result
}

func toAppliedWhen(checks []recommendations.AppliedWhenCheck) []*model.RecommendationAppliedWhen {
	result := make([]*model.RecommendationAppliedWhen, 0, len(checks))
	for _, a := range checks {
		item := &model.RecommendationAppliedWhen{
			Type: a.Type,
		}
		if a.Expression != "" {
			expr := a.Expression
			item.Expression = &expr
		}
		if a.ActionType != "" {
			actionType := a.ActionType
			item.ActionType = &actionType
		}
		result = append(result, item)
	}
	return result
}

func toCatalogActions(actions []recommendations.Action) []*model.RecommendationCatalogAction {
	result := make([]*model.RecommendationCatalogAction, 0, len(actions))
	for _, a := range actions {
		result = append(result, &model.RecommendationCatalogAction{
			Type:        a.Type,
			Description: a.Description,
		})
	}
	return result
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func toGraphRecommendationType(t common.RecommendationType) (model.RecommendationType, error) {
	switch t {
	case common.RecommendationTypeInferDBAttributes:
		return model.RecommendationTypeInferDBAttributes, nil
	case common.RecommendationTypeAutoGoOffsetUpdater:
		return model.RecommendationTypeAutoGoOffsetUpdater, nil
	case common.RecommendationTypeEnableOwnMetrics:
		return model.RecommendationTypeEnableOwnMetrics, nil
	case common.RecommendationTypeSampleHealthProbes:
		return model.RecommendationTypeSampleHealthProbes, nil
	case common.RecommendationTypeUrlTemplatization:
		return model.RecommendationTypeURLTemplatization, nil
	default:
		return "", fmt.Errorf("unknown recommendation type %q", t)
	}
}
