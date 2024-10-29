package services

import (
	"context"
	"fmt"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListInstrumentationRules fetches all instrumentation rules
func ListInstrumentationRules(ctx context.Context) ([]*model.InstrumentationRule, error) {
	odigosns := consts.DefaultOdigosNamespace
	instrumentationRules, err := kube.DefaultClient.OdigosClient.InstrumentationRules(odigosns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting instrumentation rules: %w", err)
	}

	var gqlRules []*model.InstrumentationRule
	for _, rule := range instrumentationRules.Items {
		gqlRules = append(gqlRules, &model.InstrumentationRule{
			RuleId:                   rule.Name,
			RuleName:                 rule.Spec.RuleName,
			Notes:                    rule.Spec.Notes,
			Disabled:                 rule.Spec.Disabled,
			Workloads:                rule.Spec.Workloads,
			InstrumentationLibraries: rule.Spec.InstrumentationLibraries,
			PayloadCollection:        rule.Spec.PayloadCollection,
			OtelSdks:                 rule.Spec.OtelSdks,
		})
	}
	return gqlRules, nil
}

func GetInstrumentationRule(ctx context.Context, id string) (*model.InstrumentationRule, error) {
	odigosns := consts.DefaultOdigosNamespace

	rule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(odigosns).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("instrumentation rule with ID %s not found", id)
		}
		return nil, fmt.Errorf("error getting instrumentation rule: %w", err)
	}

	return &model.InstrumentationRule{
		RuleId:                   rule.Name,
		RuleName:                 rule.Spec.RuleName,
		Notes:                    rule.Spec.Notes,
		Disabled:                 rule.Spec.Disabled,
		Workloads:                rule.Spec.Workloads,
		InstrumentationLibraries: rule.Spec.InstrumentationLibraries,
		PayloadCollection:        rule.Spec.PayloadCollection,
		OtelSdks:                 rule.Spec.OtelSdks,
	}, nil
}

func UpdateInstrumentationRule(ctx context.Context, id string, input model.InstrumentationRuleInput) (*model.InstrumentationRule, error) {
	odigosns := consts.DefaultOdigosNamespace

	// Retrieve existing rule
	existingRule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(odigosns).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("instrumentation rule with ID %s not found", id)
		}
		return nil, fmt.Errorf("error getting instrumentation rule: %w", err)
	}

	// Update the existing rule's specification
	existingRule.Spec.RuleName = input.RuleName
	existingRule.Spec.Notes = input.Notes
	existingRule.Spec.Disabled = input.Disabled
	existingRule.Spec.Workloads = input.Workloads
	existingRule.Spec.InstrumentationLibraries = input.InstrumentationLibraries
	existingRule.Spec.PayloadCollection = input.PayloadCollection
	existingRule.Spec.OtelSdks = input.OtelSdks

	// Update rule in Kubernetes
	updatedRule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(odigosns).Update(ctx, existingRule, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error updating instrumentation rule: %w", err)
	}

	return &model.InstrumentationRule{
		RuleId:                   updatedRule.Name,
		RuleName:                 updatedRule.Spec.RuleName,
		Notes:                    updatedRule.Spec.Notes,
		Disabled:                 updatedRule.Spec.Disabled,
		Workloads:                updatedRule.Spec.Workloads,
		InstrumentationLibraries: updatedRule.Spec.InstrumentationLibraries,
		PayloadCollection:        updatedRule.Spec.PayloadCollection,
		OtelSdks:                 updatedRule.Spec.OtelSdks,
	}, nil
}

func DeleteInstrumentationRule(ctx context.Context, id string) (bool, error) {
	odigosns := consts.DefaultOdigosNamespace

	err := kube.DefaultClient.OdigosClient.InstrumentationRules(odigosns).Delete(ctx, id, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, fmt.Errorf("instrumentation rule with ID %s not found", id)
		}
		return false, fmt.Errorf("error deleting instrumentation rule: %w", err)
	}

	return true, nil
}

func CreateInstrumentationRule(ctx context.Context, input model.InstrumentationRuleInput) (*model.InstrumentationRule, error) {
	odigosns := consts.DefaultOdigosNamespace

	// Define the new rule spec based on the input
	newRule := &odigosv1alpha1.InstrumentationRule{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "ui-instrumentation-rule-", // Generate a unique name
		},
		Spec: odigosv1alpha1.InstrumentationRuleSpec{
			RuleName:                 input.RuleName,
			Notes:                    input.Notes,
			Disabled:                 input.Disabled,
			Workloads:                input.Workloads,
			InstrumentationLibraries: input.InstrumentationLibraries,
			PayloadCollection:        input.PayloadCollection,
			OtelSdks:                 input.OtelSdks,
		},
	}

	// Create the rule in Kubernetes
	createdRule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(odigosns).Create(ctx, newRule, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error creating instrumentation rule: %w", err)
	}

	// Convert to GraphQL model and return
	return &model.InstrumentationRule{
		RuleId:                   createdRule.Name,
		RuleName:                 createdRule.Spec.RuleName,
		Notes:                    createdRule.Spec.Notes,
		Disabled:                 createdRule.Spec.Disabled,
		Workloads:                createdRule.Spec.Workloads,
		InstrumentationLibraries: createdRule.Spec.InstrumentationLibraries,
		PayloadCollection:        createdRule.Spec.PayloadCollection,
		OtelSdks:                 createdRule.Spec.OtelSdks,
	}, nil
}
