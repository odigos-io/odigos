package recommendations

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	odigosapply "github.com/odigos-io/odigos/api/generated/odigos/applyconfiguration/odigos/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	recommendationcatalog "github.com/odigos-io/odigos/recommendations"
)

const (
	fieldOwner                 = "odigos-autoscaler"
	golangEnterpriseDistroName = "golang-enterprise"
)

func syncRecommendations(ctx context.Context, c client.Client) error {
	ns := env.GetCurrentNamespace()
	for _, rec := range recommendationcatalog.Get() {
		if !isRelevantForTier(rec) {
			continue
		}
		if err := syncRecommendation(ctx, c, rec, ns); err != nil {
			return err
		}
	}
	return nil
}

func syncRecommendation(ctx context.Context, c client.Client, rec recommendationcatalog.Recommendation, ns string) error {
	met, err := evaluateConditions(ctx, c, rec)
	if err != nil {
		return err
	}

	var actionsList odigosv1.ActionList
	if err := c.List(ctx, &actionsList); err != nil {
		return err
	}

	cfgData, err := getEffectiveConfigData(ctx, c)
	if err != nil {
		return err
	}

	applied, err := evaluateAppliedWhen(cfgData, actionsList.Items, rec)
	if err != nil {
		return err
	}

	desired := odigosapply.Recommendation(rec.K8sObjectName, ns).
		WithSpec(odigosapply.RecommendationSpec().
			WithType(rec.Type).
			WithConditionsMet(met).
			WithApplied(applied))
	return c.Apply(ctx, desired, client.ForceOwnership, client.FieldOwner(fieldOwner))
}

func evaluateConditions(ctx context.Context, c client.Client, rec recommendationcatalog.Recommendation) (bool, error) {
	for _, condition := range rec.Conditions {
		holds, err := conditionHolds(ctx, c, condition)
		if err != nil {
			return false, err
		}
		if !holds {
			return false, nil
		}
	}
	return true, nil
}

func conditionHolds(ctx context.Context, c client.Client, condition recommendationcatalog.Condition) (bool, error) {
	switch condition.Type {
	case recommendationcatalog.ConditionTypeGoEnterpriseSources:
		return hasGoEnterpriseSource(ctx, c)
	default:
		commonlogger.FromContext(ctx).WithName("recommendations").
			Info("unknown recommendation condition type", "type", condition.Type)
		return false, nil
	}
}

func hasGoEnterpriseSource(ctx context.Context, c client.Client) (bool, error) {
	var configs odigosv1.InstrumentationConfigList
	if err := c.List(ctx, &configs); err != nil {
		return false, err
	}
	for i := range configs.Items {
		for _, container := range configs.Items[i].Spec.Containers {
			if container.OtelDistroName == golangEnterpriseDistroName {
				return true, nil
			}
		}
	}
	return false, nil
}
