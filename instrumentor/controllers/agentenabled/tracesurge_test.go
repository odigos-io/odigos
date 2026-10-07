package agentenabled

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func traceSurge(ruleID string, boost float64, phase odigosv1.TraceSurgePhase, targets ...k8sconsts.PodWorkload) odigosv1.TraceSurge {
	surge := odigosv1.TraceSurge{
		Spec: odigosv1.TraceSurgeSpec{
			SamplingName: "default", RuleID: ruleID,
			Settings: odigosv1.TraceSurgeSettings{BoostPercent: boost},
		},
		Status: odigosv1.TraceSurgeStatus{Phase: phase},
	}
	for _, target := range targets {
		surge.Spec.Targets = append(surge.Spec.Targets, odigosv1.TraceSurgeTarget{Workload: target})
	}
	return surge
}

func TestWithTraceSurgeBoosts(t *testing.T) {
	frontend := k8sconsts.PodWorkload{Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment, Name: "frontend"}
	browse := k8sconsts.PodWorkload{Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment, Name: "browse"}
	one, fifty := 1.0, 50.0
	surged := odigosv1.NoisyOperation{Name: "shop", PercentageAtMost: &one}
	alreadyHigher := odigosv1.NoisyOperation{Name: "high", SourceScopes: &k8sconsts.SourcesScopes{Namespaces: []string{"x"}}, PercentageAtMost: &fifty}
	surgedID := odigosv1.ComputeNoisyOperationHash(&surged)
	highID := odigosv1.ComputeNoisyOperationHash(&alreadyHigher)
	samplings := []odigosv1.Sampling{{
		ObjectMeta: metav1.ObjectMeta{Name: "default"},
		Spec:       odigosv1.SamplingSpec{NoisyOperations: []odigosv1.NoisyOperation{surged, alreadyHigher}},
		Status: odigosv1.SamplingStatus{TraceSurges: []odigosv1.TraceSurge{
			traceSurge(surgedID, 80, odigosv1.TraceSurgePhaseRecovering, frontend),
			traceSurge(surgedID, 60, odigosv1.TraceSurgePhaseBoosting, frontend),
			traceSurge(highID, 20, odigosv1.TraceSurgePhaseBoosting, frontend),
		}},
	}}

	boosted := withTraceSurgeBoosts(&samplings, frontend)
	ops := (*boosted)[0].Spec.NoisyOperations
	assert.Equal(t, 80.0, *ops[0].PercentageAtMost, "the highest boost of the active surges")
	assert.Equal(t, 50.0, *ops[1].PercentageAtMost, "a boost never lowers a rule")
	assert.Equal(t, 1.0, *samplings[0].Spec.NoisyOperations[0].PercentageAtMost, "the cached rules are not modified")
	assert.Equal(t, surgedID, odigosv1.ComputeNoisyOperationHash(&ops[0]), "the rule keeps its id")

	assert.Same(t, &samplings, withTraceSurgeBoosts(&samplings, browse), "a workload no surge targets gets the rules as they are")

	samplings[0].Status.TraceSurges = []odigosv1.TraceSurge{traceSurge(surgedID, 80, odigosv1.TraceSurgePhaseRestored, frontend)}
	assert.Equal(t, 1.0, *(*withTraceSurgeBoosts(&samplings, frontend))[0].Spec.NoisyOperations[0].PercentageAtMost)
}

func TestTraceSurgeBoostsHandlerEnqueuesChangedWorkloads(t *testing.T) {
	frontend := k8sconsts.PodWorkload{Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment, Name: "frontend"}
	payments := k8sconsts.PodWorkload{Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment, Name: "payments"}
	edge := k8sconsts.PodWorkload{Namespace: "edge", Kind: k8sconsts.WorkloadKindStatefulSet, Name: "storefront"}
	sampling := func(surges ...odigosv1.TraceSurge) *odigosv1.Sampling {
		return &odigosv1.Sampling{ObjectMeta: metav1.ObjectMeta{Name: "default"}, Status: odigosv1.SamplingStatus{TraceSurges: surges}}
	}
	update := func(old, new *odigosv1.Sampling) []reconcile.Request {
		t.Helper()
		q := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[reconcile.Request]())
		defer q.ShutDown()
		traceSurgeBoostsHandler().Update(context.Background(), event.UpdateEvent{ObjectOld: old, ObjectNew: new}, q)
		var reqs []reconcile.Request
		for q.Len() > 0 {
			req, _ := q.Get()
			reqs = append(reqs, req)
			q.Done(req)
		}
		return reqs
	}
	req := func(pw k8sconsts.PodWorkload, name string) reconcile.Request {
		r := workloadRequest(pw)
		assert.Equal(t, name, r.Name)
		return r
	}

	limited := traceSurge("r", 50, odigosv1.TraceSurgePhaseLimited, frontend, payments)
	boosting := traceSurge("r", 50, odigosv1.TraceSurgePhaseBoosting, frontend, payments)
	assert.Empty(t, update(sampling(limited), sampling(limited)), "an evaluation that changes nothing reconciles nothing")
	assert.ElementsMatch(t, []reconcile.Request{req(frontend, "deployment-frontend"), req(payments, "deployment-payments")},
		update(sampling(limited), sampling(boosting)), "a promoted surge raises its targets")

	grown := traceSurge("r", 50, odigosv1.TraceSurgePhaseBoosting, frontend, payments, edge)
	assert.ElementsMatch(t, []reconcile.Request{req(edge, "statefulset-storefront")}, update(sampling(boosting), sampling(grown)),
		"a target added to an open surge is the only workload recalculated")

	recovering := grown
	recovering.Status.Phase = odigosv1.TraceSurgePhaseRecovering
	assert.Empty(t, update(sampling(grown), sampling(recovering)), "recovering keeps the boost")

	restored := grown
	restored.Status.Phase = odigosv1.TraceSurgePhaseRestored
	assert.Len(t, update(sampling(recovering), sampling(restored)), 3, "a restored surge restores all its targets")

	other := traceSurge("r", 80, odigosv1.TraceSurgePhaseBoosting, payments)
	assert.ElementsMatch(t, []reconcile.Request{req(payments, "deployment-payments")}, update(sampling(boosting), sampling(boosting, other)),
		"a higher boost on one target of an overlapping surge")
}
