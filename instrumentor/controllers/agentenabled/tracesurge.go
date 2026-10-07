package agentenabled

import (
	"context"
	"maps"
	"slices"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// traceSurgeBoosts returns, for each workload the active trace surges of the Sampling target, the
// highest percentage each of its rules is raised to: workload -> rule id -> percentage.
func traceSurgeBoosts(sampling *odigosv1.Sampling) map[k8sconsts.PodWorkload]map[string]float64 {
	var boosts map[k8sconsts.PodWorkload]map[string]float64
	for i := range sampling.Status.TraceSurges {
		surge := &sampling.Status.TraceSurges[i]
		if !surge.Active() {
			continue
		}
		for _, target := range surge.Spec.Targets {
			if boosts == nil {
				boosts = map[k8sconsts.PodWorkload]map[string]float64{}
			}
			rules, ok := boosts[target.Workload]
			if !ok {
				rules = map[string]float64{}
				boosts[target.Workload] = rules
			}
			rules[surge.Spec.RuleID] = max(rules[surge.Spec.RuleID], surge.Spec.Settings.BoostPercent)
		}
	}
	return boosts
}

// withTraceSurgeBoosts returns the sampling rules as the workload applies them: each rule an
// active trace surge of its Sampling targets the workload with sampled at the surge's percentage,
// if higher.
func withTraceSurgeBoosts(samplings *[]odigosv1.Sampling, pw k8sconsts.PodWorkload) *[]odigosv1.Sampling {
	var boosted []odigosv1.Sampling
	for i := range *samplings {
		sampling := &(*samplings)[i]
		rules := traceSurgeBoosts(sampling)[pw]
		if len(rules) == 0 {
			continue
		}
		if boosted == nil {
			boosted = slices.Clone(*samplings)
		}
		boosted[i] = *sampling.DeepCopy()
		for j := range boosted[i].Spec.NoisyOperations {
			op := &boosted[i].Spec.NoisyOperations[j]
			boost, ok := rules[odigosv1.ComputeNoisyOperationHash(op)]
			if !ok || (op.PercentageAtMost != nil && *op.PercentageAtMost >= boost) {
				continue
			}
			op.PercentageAtMost = &boost
		}
	}
	if boosted == nil {
		return samplings
	}
	return &boosted
}

// traceSurgeBoostsHandler enqueues the workloads whose raised rules differ between two versions of
// a Sampling's trace surges. The evaluator rewrites the status every few seconds while a surge is
// open; only the workloads whose percentage changes are recalculated.
func traceSurgeBoostsHandler() handler.EventHandler {
	enqueue := func(old, new *odigosv1.Sampling, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
		var before, after map[k8sconsts.PodWorkload]map[string]float64
		if old != nil {
			before = traceSurgeBoosts(old)
		}
		if new != nil {
			after = traceSurgeBoosts(new)
		}
		for _, pw := range slices.Collect(maps.Keys(before)) {
			if !maps.Equal(before[pw], after[pw]) {
				q.Add(workloadRequest(pw))
			}
		}
		for pw := range after {
			if _, ok := before[pw]; !ok {
				q.Add(workloadRequest(pw))
			}
		}
	}
	return handler.Funcs{
		CreateFunc: func(_ context.Context, e event.CreateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			if s, ok := e.Object.(*odigosv1.Sampling); ok {
				enqueue(nil, s, q)
			}
		},
		UpdateFunc: func(_ context.Context, e event.UpdateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			old, oldOk := e.ObjectOld.(*odigosv1.Sampling)
			new, newOk := e.ObjectNew.(*odigosv1.Sampling)
			if oldOk && newOk {
				enqueue(old, new, q)
			}
		},
		DeleteFunc: func(_ context.Context, e event.DeleteEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			if s, ok := e.Object.(*odigosv1.Sampling); ok {
				enqueue(s, nil, q)
			}
		},
	}
}

func workloadRequest(pw k8sconsts.PodWorkload) reconcile.Request {
	return reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: pw.Namespace,
		Name:      workload.CalculateWorkloadRuntimeObjectName(pw.Name, pw.Kind),
	}}
}
