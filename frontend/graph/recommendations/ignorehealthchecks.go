package recommendations

import (
	"context"

	actionsv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IgnoreHealthChecksReason string

const (
	IgnoreHealthChecksReasonActionExists   IgnoreHealthChecksReason = "ActionExists"
	IgnoreHealthChecksReasonActionNotFound IgnoreHealthChecksReason = "ActionNotFound"
)

const AtuoIgnoreHealthChecksRecommendationActionName = "ignore-health-checks"

func IgnoreHealthChecksRecommendation(ctx context.Context, cacheClient client.Client, odigosNamespace string) (*model.Recommendation, error) {

	// check if there is any "ignore health checks" action in the cluster
	actionList := &odigosv1.ActionList{}
	err := cacheClient.List(ctx, actionList, client.InNamespace(odigosNamespace))
	if err != nil {
		return nil, err
	}

	foundIgnoreHealthChecksAction := false
	for _, action := range actionList.Items {
		if action.Spec.Disabled {
			continue
		}
		if action.Spec.Samplers != nil && action.Spec.Samplers.IgnoreHealthChecks != nil {
			foundIgnoreHealthChecksAction = true
			break
		}
	}

	if !foundIgnoreHealthChecksAction {
		return ignoredHealthCheckActionNotFound(), nil
	}

	return ignoredHealthCheckActionFound(), nil
}

func ignoredHealthCheckActionFound() *model.Recommendation {
	return &model.Recommendation{
		Type:       model.RecommendationTypeIgnoreHealthChecks,
		Status:     model.RecommendationStatusRecommended,
		ReasonEnum: string(IgnoreHealthChecksReasonActionExists),
		Message:    "health-check traces are being ignored (recommended)",
	}
}

func ignoredHealthCheckActionNotFound() *model.Recommendation {
	return &model.Recommendation{
		Type:       model.RecommendationTypeIgnoreHealthChecks,
		Status:     model.RecommendationStatusRecommedationSuggestion,
		ReasonEnum: string(IgnoreHealthChecksReasonActionNotFound),
		Message:    "health-check traces are not being ignored (all collected)",
		ActionItems: []string{
			"add an action of type `sampler/IgnoreHealthChecks`",
			"ignore this recommendation and collect all health check traces",
		},
	}
}

func ApplyIgnoreHealthChecksRecommendation(ctx context.Context, cacheClient client.Client, odigosNamespace string, fractionToRecord *float64) error {
	// Default to 0 if fractionToRecord is not provided
	fraction := 0.0
	if fractionToRecord != nil {
		fraction = *fractionToRecord
		// Ensure fraction is within valid range [0, 1]
		if fraction < 0 {
			fraction = 0
		} else if fraction > 1 {
			fraction = 1
		}
	}

	// create an action of type `sampler/IgnoreHealthChecks`
	action := &odigosv1.Action{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Action",
			APIVersion: "odigos.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      AtuoIgnoreHealthChecksRecommendationActionName,
			Namespace: odigosNamespace,
			Labels: map[string]string{
				"odigos.io/recommendation": "ignore-health-checks",
			},
		},
		Spec: odigosv1.ActionSpec{
			ActionName: "ignore-health-checks",
			Signals: []common.ObservabilitySignal{
				common.TracesObservabilitySignal,
			},
			Samplers: &actionsv1.SamplersConfig{
				IgnoreHealthChecks: &actionsv1.IgnoreHealthChecksConfig{
					FractionToRecord: fraction,
				},
			},
		},
	}
	return cacheClient.Patch(ctx, action, client.Apply, client.ForceOwnership, client.FieldOwner("odigos-recommendations"))
}
