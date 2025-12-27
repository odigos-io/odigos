package recommendations

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
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
	if len(actionList.Items) > 0 {
		for _, action := range actionList.Items {
			if action.Spec.IgnoreHealthChecks != nil {
				foundIgnoreHealthChecksAction = true
				break
			}
		}
	}

	// check if the action already exists
	action := &odigosv1.Action{}
	err := cacheClient.Get(ctx, types.NamespacedName{Name: AtuoIgnoreHealthChecksRecommendationActionName, Namespace: metav1.NamespaceAll}, action)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	recommendation := model.Recommendation{
		Type:        model.RecommendationTypeIgnoreHealthChecks,
		Status:      model.RecommendationStatusRecommended,
		ReasonEnum:  string(IgnoreHealthChecksReasonActionExists),
		Message:     "Ignore health checks",
		ActionItems: []string{"Ignore health checks"},
	}
	return &recommendation, nil
}

func ignoredHealthCheckActionNotFound() model.Recommendation {
	return model.Recommendation{
		Type:        model.RecommendationTypeIgnoreHealthChecks,
		Status:      model.RecommendationStatusRecommended,
		ReasonEnum:  string(IgnoreHealthChecksReasonActionNotFound),
		Message:     "the cluster is not configured to ignore health checks",
		ActionItems: []string{"Ignore health checks"},
	}
}
