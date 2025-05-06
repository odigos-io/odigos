package agentenabled

import (
	"context"
	"sort"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/instrumentor/controllers/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getInstrumentationRulePriority(ir *odigosv1.InstrumentationRule) int {
	switch ir.Annotations[k8sconsts.OdigosProfileSourceAnnotation] {
	case k8sconsts.OdigosProfileSourceOnPremToken:
		return 1
	case k8sconsts.OdigosProfileSourceConfig:
		return 2
	default:
		return 3
	}
}

// fetches the relevant resources for reconciliation of the current workload object.
// if err is returned, the reconciliation should be retried.
// the function can return without error, but some of the resources might be nil.
// this indicates that they were not found, but it is valid.
func getRelevantResources(ctx context.Context, c client.Client, pw k8sconsts.PodWorkload) (
	*odigosv1.CollectorsGroup,
	*[]odigosv1.InstrumentationRule,
	*common.OdigosConfiguration,
	error) {

	cg, err := getCollectorsGroup(ctx, c)
	if err != nil {
		return nil, nil, nil, err
	}

	irls, err := getRelevantInstrumentationRules(ctx, c, pw)
	if err != nil {
		return nil, nil, nil, err
	}

	// TODO: we are yaml unmarshalling the configmap data for every workload, this is not efficient
	// can we cache the configmap data in the controller?
	effectiveConfig, err := k8sutils.GetCurrentOdigosConfig(ctx, c)
	if err != nil {
		return nil, nil, nil, err
	}

	return cg, irls, &effectiveConfig, nil
}

func getCollectorsGroup(ctx context.Context, c client.Client) (*odigosv1.CollectorsGroup, error) {
	cg := odigosv1.CollectorsGroup{}
	err := c.Get(ctx, client.ObjectKey{Namespace: env.GetCurrentNamespace(), Name: k8sconsts.OdigosNodeCollectorCollectorGroupName}, &cg)
	if err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return &cg, nil
}

func getRelevantInstrumentationRules(ctx context.Context, c client.Client, pw k8sconsts.PodWorkload) (*[]odigosv1.InstrumentationRule, error) {
	relevantIr := []odigosv1.InstrumentationRule{}
	irList := odigosv1.InstrumentationRuleList{}
	err := c.List(ctx, &irList)
	if err != nil {
		return nil, err
	}

	for i := range irList.Items {
		ir := &irList.Items[i]

		if !utils.IsWorkloadParticipatingInRule(pw, ir) {
			continue
		}

		if ir.Spec.OtelDistros == nil {
			// we only care about otel sdks rules at the moment.
			// no need to process other rules.
			continue
		}

		relevantIr = append(relevantIr, *ir)
	}

	// sort rules according to priority: onprem token rules first, then other rules which will override them.
	// if both rules have the same priority, use creation timestamp to sort them.
	sort.Slice(relevantIr, func(i, j int) bool {
		priorityI := getInstrumentationRulePriority(&relevantIr[i])
		priorityJ := getInstrumentationRulePriority(&relevantIr[j])
		if priorityI == priorityJ {
			return relevantIr[i].CreationTimestamp.Before(&relevantIr[j].CreationTimestamp)
		}
		return priorityI < priorityJ
	})

	return &relevantIr, nil
}
