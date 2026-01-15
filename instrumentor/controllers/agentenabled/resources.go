package agentenabled

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/instrumentor/controllers/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// fetches the relevant resources for reconciliation of the current workload object.
// if err is returned, the reconciliation should be retried.
// the function can return without error, but some of the resources might be nil.
// this indicates that they were not found, but it is valid.
func getRelevantResources(ctx context.Context, c client.Client, pw k8sconsts.PodWorkload) (
	*odigosv1.CollectorsGroup,
	*[]odigosv1.InstrumentationRule,
	*[]odigosv1.Action,
	workload.Workload,
	error) {

	// fetch the workload object, so we can extract the health check paths from it.
	obj := workload.ClientObjectFromWorkloadKind(pw.Kind)
	err := c.Get(ctx, client.ObjectKey{Name: pw.Name, Namespace: pw.Namespace}, obj)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, nil, nil, nil, err
	}

	workloadObj, err := workload.ObjectToWorkload(obj)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	cg, err := getCollectorsGroup(ctx, c)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	irls, err := getRelevantInstrumentationRules(ctx, c, pw)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	actions, err := getAgentLevelRelatedActions(ctx, c)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return cg, irls, actions, workloadObj, nil
}

func getAgentLevelRelatedActions(ctx context.Context, c client.Client) (*[]odigosv1.Action, error) {
	actionList := &odigosv1.ActionList{}
	err := c.List(ctx, actionList, &client.ListOptions{Namespace: env.GetCurrentNamespace()})
	if err != nil {
		return nil, err
	}

	// Filter only actions that have URLTemplatization config
	agentLevelActions := []odigosv1.Action{}
	for _, action := range actionList.Items {
		if action.Spec.Disabled {
			continue
		}
		if action.Spec.URLTemplatization != nil ||
			(action.Spec.Samplers != nil && action.Spec.Samplers.IgnoreHealthChecks != nil) ||
			action.Spec.SpanRenamer != nil {
			agentLevelActions = append(agentLevelActions, action)
		}
	}

	return &agentLevelActions, nil
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

		// ignore disabled rules
		if ir.Spec.Disabled {
			continue
		}

		if !utils.IsWorkloadParticipatingInRule(pw, ir) {
			continue
		}

		// filter only rules that are relevant to the agent enabled logic
		if (ir.Spec.OtelSdks != nil || ir.Spec.OtelDistros != nil) ||
			(ir.Spec.TraceConfig != nil && ir.Spec.TraceConfig.Disabled != nil) ||
			(ir.Spec.HeadersCollection != nil ||
				ir.Spec.HeadSamplingFallbackFraction != nil) {

			relevantIr = append(relevantIr, *ir)
		}
	}

	return &relevantIr, nil
}
