package tracesurge

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	cacheutils "github.com/odigos-io/odigos/k8sutils/pkg/cache"
	instance "github.com/odigos-io/odigos/k8sutils/pkg/instrumentation_instance"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// processes listed in the status of each target of a surge, so that it stays small for large workloads.
	maxListedInstances = 10

	instanceConfirmed = "confirmed"
	instanceUnknown   = "unknown"
	instanceFailed    = "failed"
)

// confirm reads, from the instrumentation instances of the surge's targets, which processes
// apply the percentage the surge requires now, and reports whether that changed.
func (e *Evaluator) confirm(ctx context.Context, surge *odigosv1.TraceSurge, now time.Time) bool {

	wasConfirmed := len(surge.Status.Targets) > 0
	for _, previous := range surge.Status.Targets {
		wasConfirmed = wasConfirmed && previous.Total > 0 && previous.Confirmed == previous.Total
	}

	changed := false
	allConfirmed, total, atNormal := true, 0, true
	statuses := make([]odigosv1.TraceSurgeTargetStatus, 0, len(surge.Spec.Targets))
	for _, target := range surge.Spec.Targets {
		previous := findTarget(surge.Status.Targets, target)
		required := e.requiredPercent(surge, target.Workload)
		atNormal = atNormal && required == surge.Spec.NormalPercent
		status, err := e.targetStatus(ctx, surge.Spec.RuleID, target, required, previous, now)
		if err != nil {
			e.Logger.Error(err, "failed to read trace surge target instances", "surge", surge.Name, "target", target.Workload)
			if previous != nil {
				status = *previous
			}
		}
		if previous == nil || previous.Confirmed != status.Confirmed || previous.Total != status.Total || !slices.Equal(instanceStates(previous.Instances), instanceStates(status.Instances)) {
			changed = true
		}
		total += status.Total
		allConfirmed = allConfirmed && status.Total > 0 && status.Confirmed == status.Total
		statuses = append(statuses, status)
	}
	surge.Status.Targets = statuses

	if allConfirmed && !wasConfirmed {
		switch {
		case surge.Active():
			addEvent(surge, now, fmt.Sprintf("All %d processes apply %s", total, formatPercent(surge.Spec.Settings.BoostPercent)),
				fmt.Sprintf("Every instrumented process of %s reports the rule's percentage applied.", targetNames(surge.Spec.Targets)))
		case atNormal:
			addEvent(surge, now, fmt.Sprintf("All %d processes back at %s", total, formatPercent(surge.Spec.NormalPercent)),
				fmt.Sprintf("Every instrumented process of %s reports the rule's normal percentage applied.", targetNames(surge.Spec.Targets)))
		default:
			addEvent(surge, now, fmt.Sprintf("All %d processes apply the rule's current percentage", total),
				fmt.Sprintf("Back at %s, except where another active surge of the rule raises it.", formatPercent(surge.Spec.NormalPercent)))
		}
	}
	return changed
}

func (e *Evaluator) targetStatus(ctx context.Context, ruleID string, target odigosv1.TraceSurgeTarget, required float64, previous *odigosv1.TraceSurgeTargetStatus, now time.Time) (odigosv1.TraceSurgeTargetStatus, error) {
	status := odigosv1.TraceSurgeTargetStatus{Workload: target.Workload}
	instances := &odigosv1.InstrumentationInstanceList{}
	err := e.Client.List(ctx, instances, client.InNamespace(target.Workload.Namespace), client.MatchingLabels{
		consts.InstrumentedAppNameLabel: workload.CalculateWorkloadRuntimeObjectName(target.Workload.Name, target.Workload.Kind),
	})
	if err != nil {
		return status, err
	}

	for _, ii := range instances.Items {
		receipt := odigosv1.TraceSurgeInstanceStatus{
			Name:  ii.Name,
			Pod:   ii.Labels[odigosv1.OwnerPodNameLabel],
			State: instanceUnknown,
			At:    &ii.Status.LastStatusTime,
		}
		for _, attr := range ii.Status.NonIdentifyingAttributes {
			switch attr.Key {
			case instance.HeadSamplingAppliedAttribute:
				if applied, ok := instance.ParseHeadSamplingApplied(attr.Value)[ruleID]; ok {
					receipt.AppliedPercent = &applied
					if math.Abs(applied-required) < 1e-9 {
						receipt.State = instanceConfirmed
					} else {
						receipt.Message = fmt.Sprintf("Applies %s, not %s yet.", formatPercent(applied), formatPercent(required))
					}
				}
			case instance.ConfigErrorAttribute:
				receipt.State = instanceFailed
				receipt.Message = attr.Value
			}
		}
		if receipt.State == instanceUnknown && receipt.Message == "" {
			receipt.Message = "The process has not reported applying the rule."
		}
		if receipt.State == instanceConfirmed {
			status.Confirmed++
		}
		status.Instances = append(status.Instances, receipt)
	}
	status.Total = len(status.Instances)
	// the status lists a few processes, the ones that did not confirm first; the counts cover all.
	slices.SortFunc(status.Instances, func(a, b odigosv1.TraceSurgeInstanceStatus) int {
		if ac, bc := a.State == instanceConfirmed, b.State == instanceConfirmed; ac != bc {
			if bc {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Name, b.Name)
	})
	if len(status.Instances) > maxListedInstances {
		status.Instances = status.Instances[:maxListedInstances]
	}

	if status.Total > 0 && status.Confirmed == status.Total {
		at := metav1.NewTime(now)
		status.ConfirmedAt = &at
		if previous != nil && previous.ConfirmedAt != nil {
			status.ConfirmedAt = previous.ConfirmedAt
		}
	}
	return status, nil
}

// requiredPercent is the rule's percentage the workload should apply now: the highest boost of
// the rule's active surges that target it, or the rule's normal percentage.
func (e *Evaluator) requiredPercent(surge *odigosv1.TraceSurge, pw k8sconsts.PodWorkload) float64 {
	required := surge.Spec.NormalPercent
	for _, other := range e.active {
		if other.Active() && other.Spec.SamplingName == surge.Spec.SamplingName && other.Spec.RuleID == surge.Spec.RuleID &&
			slices.ContainsFunc(other.Spec.Targets, func(t odigosv1.TraceSurgeTarget) bool { return t.Workload == pw }) {
			required = max(required, other.Spec.Settings.BoostPercent)
		}
	}
	return required
}

// confirmed reports whether every target of the surge has been confirmed.
func confirmed(surge *odigosv1.TraceSurge) bool {
	if len(surge.Status.Targets) != len(surge.Spec.Targets) {
		return false
	}
	for _, target := range surge.Status.Targets {
		if target.ConfirmedAt == nil {
			return false
		}
	}
	return true
}

func findTarget(statuses []odigosv1.TraceSurgeTargetStatus, target odigosv1.TraceSurgeTarget) *odigosv1.TraceSurgeTargetStatus {
	for i := range statuses {
		if statuses[i].Workload == target.Workload {
			return &statuses[i]
		}
	}
	return nil
}

func instanceStates(instances []odigosv1.TraceSurgeInstanceStatus) []string {
	states := make([]string, len(instances))
	for i, ii := range instances {
		states[i] = ii.Name + "=" + ii.State
	}
	return states
}

// InstanceTransform strips an InstrumentationInstance, for the instrumentor's cache, to what the
// confirmations read: its labels, and the attributes in which its process reports the config it
// applied. There is one per instrumented process, so the cache keeps as little as it can.
func InstanceTransform(obj interface{}) (interface{}, error) {
	ii, ok := obj.(*odigosv1.InstrumentationInstance)
	if !ok {
		return nil, fmt.Errorf("expected an InstrumentationInstance, got %T", obj)
	}
	if cacheutils.IsObjectTransformed(ii) {
		return ii, nil
	}
	stripped := &odigosv1.InstrumentationInstance{
		TypeMeta: ii.TypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:            ii.Name,
			Namespace:       ii.Namespace,
			UID:             ii.UID,
			ResourceVersion: ii.ResourceVersion,
			Labels:          ii.Labels,
		},
		Status: odigosv1.InstrumentationInstanceStatus{LastStatusTime: ii.Status.LastStatusTime},
	}
	for _, attr := range ii.Status.NonIdentifyingAttributes {
		if attr.Key == instance.HeadSamplingAppliedAttribute || attr.Key == instance.ConfigErrorAttribute {
			stripped.Status.NonIdentifyingAttributes = append(stripped.Status.NonIdentifyingAttributes, attr)
		}
	}
	cacheutils.MarkObjectAsTransformed(stripped)
	return stripped, nil
}
