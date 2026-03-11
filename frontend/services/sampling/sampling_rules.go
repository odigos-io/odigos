package sampling

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getOrCreateSamplingCR returns the single Sampling CR, creating it if it doesn't exist.
func getOrCreateSamplingCR(ctx context.Context) (*v1alpha1.Sampling, error) {
	odigosNs := env.GetCurrentNamespace()

	list, err := kube.DefaultClient.OdigosClient.Samplings(odigosNs).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list sampling CRs: %w", err)
	}

	// Currently we assume a single Sampling CR for all rules.
	// When grouping is introduced, this will need to accept a group identifier.
	if len(list.Items) > 0 {
		return &list.Items[0], nil
	}

	cr := &v1alpha1.Sampling{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "sampling-",
		},
		Spec: v1alpha1.SamplingSpec{},
	}

	created, err := kube.DefaultClient.OdigosClient.Samplings(odigosNs).Create(ctx, cr, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create sampling CR: %w", err)
	}
	return created, nil
}

// getSamplingCR returns the single Sampling CR or an error if none exists.
func getSamplingCR(ctx context.Context) (*v1alpha1.Sampling, error) {
	odigosNs := env.GetCurrentNamespace()

	list, err := kube.DefaultClient.OdigosClient.Samplings(odigosNs).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list sampling CRs: %w", err)
	}

	if len(list.Items) == 0 {
		return nil, fmt.Errorf("no sampling CR found")
	}
	// Currently we assume a single Sampling CR for all rules.
	// When grouping is introduced, this will need to accept a group identifier.
	return &list.Items[0], nil
}

func updateSamplingCR(ctx context.Context, cr *v1alpha1.Sampling) (*v1alpha1.Sampling, error) {
	odigosNs := env.GetCurrentNamespace()
	updated, err := kube.DefaultClient.OdigosClient.Samplings(odigosNs).Update(ctx, cr, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update sampling CR: %w", err)
	}
	return updated, nil
}

// ---- Noisy Operations ----

func GetNoisyOperationRules(ctx context.Context) ([]*model.NoisyOperationRule, error) {
	odigosNs := env.GetCurrentNamespace()

	list, err := kube.DefaultClient.OdigosClient.Samplings(odigosNs).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list sampling CRs: %w", err)
	}

	var rules []*model.NoisyOperationRule
	for i := range list.Items {
		cr := &list.Items[i]
		for j := range cr.Spec.NoisyOperations {
			rules = append(rules, convertNoisyOperationToModel(&cr.Spec.NoisyOperations[j]))
		}
	}
	return rules, nil
}

func CreateNoisyOperationRule(ctx context.Context, input model.NoisyOperationRuleInput) (*model.NoisyOperationRule, error) {
	cr, err := getOrCreateSamplingCR(ctx)
	if err != nil {
		return nil, err
	}

	rule := noisyOperationFromInput(input)
	cr.Spec.NoisyOperations = append(cr.Spec.NoisyOperations, rule)

	if _, err := updateSamplingCR(ctx, cr); err != nil {
		return nil, err
	}
	return convertNoisyOperationToModel(&rule), nil
}

func UpdateNoisyOperationRule(ctx context.Context, ruleID string, input model.NoisyOperationRuleInput) (*model.NoisyOperationRule, error) {
	cr, err := getSamplingCR(ctx)
	if err != nil {
		return nil, err
	}

	idx := findNoisyOperationByHash(cr.Spec.NoisyOperations, ruleID)
	if idx < 0 {
		return nil, fmt.Errorf("noisy operation rule %s not found", ruleID)
	}

	rule := noisyOperationFromInput(input)
	cr.Spec.NoisyOperations[idx] = rule

	if _, err := updateSamplingCR(ctx, cr); err != nil {
		return nil, err
	}
	return convertNoisyOperationToModel(&rule), nil
}

func DeleteNoisyOperationRule(ctx context.Context, ruleID string) (bool, error) {
	cr, err := getSamplingCR(ctx)
	if err != nil {
		return false, err
	}

	idx := findNoisyOperationByHash(cr.Spec.NoisyOperations, ruleID)
	if idx < 0 {
		return false, fmt.Errorf("noisy operation rule %s not found", ruleID)
	}

	cr.Spec.NoisyOperations = append(cr.Spec.NoisyOperations[:idx], cr.Spec.NoisyOperations[idx+1:]...)

	if _, err := updateSamplingCR(ctx, cr); err != nil {
		return false, err
	}
	return true, nil
}

func findNoisyOperationByHash(rules []v1alpha1.NoisyOperation, hash string) int {
	for i := range rules {
		if v1alpha1.ComputeNoisyOperationHash(&rules[i]) == hash {
			return i
		}
	}
	return -1
}

// ---- Highly Relevant Operations ----

func GetHighlyRelevantOperationRules(ctx context.Context) ([]*model.HighlyRelevantOperationRule, error) {
	odigosNs := env.GetCurrentNamespace()

	list, err := kube.DefaultClient.OdigosClient.Samplings(odigosNs).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list sampling CRs: %w", err)
	}

	var rules []*model.HighlyRelevantOperationRule
	for i := range list.Items {
		cr := &list.Items[i]
		for j := range cr.Spec.HighlyRelevantOperations {
			rules = append(rules, convertHighlyRelevantOperationToModel(&cr.Spec.HighlyRelevantOperations[j]))
		}
	}
	return rules, nil
}

func CreateHighlyRelevantOperationRule(ctx context.Context, input model.HighlyRelevantOperationRuleInput) (*model.HighlyRelevantOperationRule, error) {
	cr, err := getOrCreateSamplingCR(ctx)
	if err != nil {
		return nil, err
	}

	rule := highlyRelevantOperationFromInput(input)
	cr.Spec.HighlyRelevantOperations = append(cr.Spec.HighlyRelevantOperations, rule)

	if _, err := updateSamplingCR(ctx, cr); err != nil {
		return nil, err
	}
	return convertHighlyRelevantOperationToModel(&rule), nil
}

func UpdateHighlyRelevantOperationRule(ctx context.Context, ruleID string, input model.HighlyRelevantOperationRuleInput) (*model.HighlyRelevantOperationRule, error) {
	cr, err := getSamplingCR(ctx)
	if err != nil {
		return nil, err
	}

	idx := findHighlyRelevantOperationByHash(cr.Spec.HighlyRelevantOperations, ruleID)
	if idx < 0 {
		return nil, fmt.Errorf("highly relevant operation rule %s not found", ruleID)
	}

	rule := highlyRelevantOperationFromInput(input)
	cr.Spec.HighlyRelevantOperations[idx] = rule

	if _, err := updateSamplingCR(ctx, cr); err != nil {
		return nil, err
	}
	return convertHighlyRelevantOperationToModel(&rule), nil
}

func DeleteHighlyRelevantOperationRule(ctx context.Context, ruleID string) (bool, error) {
	cr, err := getSamplingCR(ctx)
	if err != nil {
		return false, err
	}

	idx := findHighlyRelevantOperationByHash(cr.Spec.HighlyRelevantOperations, ruleID)
	if idx < 0 {
		return false, fmt.Errorf("highly relevant operation rule %s not found", ruleID)
	}

	cr.Spec.HighlyRelevantOperations = append(cr.Spec.HighlyRelevantOperations[:idx], cr.Spec.HighlyRelevantOperations[idx+1:]...)

	if _, err := updateSamplingCR(ctx, cr); err != nil {
		return false, err
	}
	return true, nil
}

func findHighlyRelevantOperationByHash(rules []v1alpha1.HighlyRelevantOperation, hash string) int {
	for i := range rules {
		if v1alpha1.ComputeHighlyRelevantOperationHash(&rules[i]) == hash {
			return i
		}
	}
	return -1
}

// ---- Cost Reduction Rules ----

func GetCostReductionRules(ctx context.Context) ([]*model.CostReductionRule, error) {
	odigosNs := env.GetCurrentNamespace()

	list, err := kube.DefaultClient.OdigosClient.Samplings(odigosNs).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list sampling CRs: %w", err)
	}

	var rules []*model.CostReductionRule
	for i := range list.Items {
		cr := &list.Items[i]
		for j := range cr.Spec.CostReductionRules {
			rules = append(rules, convertCostReductionRuleToModel(&cr.Spec.CostReductionRules[j]))
		}
	}
	return rules, nil
}

func CreateCostReductionRule(ctx context.Context, input model.CostReductionRuleInput) (*model.CostReductionRule, error) {
	cr, err := getOrCreateSamplingCR(ctx)
	if err != nil {
		return nil, err
	}

	rule := costReductionRuleFromInput(input)
	cr.Spec.CostReductionRules = append(cr.Spec.CostReductionRules, rule)

	if _, err := updateSamplingCR(ctx, cr); err != nil {
		return nil, err
	}
	return convertCostReductionRuleToModel(&rule), nil
}

func UpdateCostReductionRule(ctx context.Context, ruleID string, input model.CostReductionRuleInput) (*model.CostReductionRule, error) {
	cr, err := getSamplingCR(ctx)
	if err != nil {
		return nil, err
	}

	idx := findCostReductionRuleByHash(cr.Spec.CostReductionRules, ruleID)
	if idx < 0 {
		return nil, fmt.Errorf("cost reduction rule %s not found", ruleID)
	}

	rule := costReductionRuleFromInput(input)
	cr.Spec.CostReductionRules[idx] = rule

	if _, err := updateSamplingCR(ctx, cr); err != nil {
		return nil, err
	}
	return convertCostReductionRuleToModel(&rule), nil
}

func DeleteCostReductionRule(ctx context.Context, ruleID string) (bool, error) {
	cr, err := getSamplingCR(ctx)
	if err != nil {
		return false, err
	}

	idx := findCostReductionRuleByHash(cr.Spec.CostReductionRules, ruleID)
	if idx < 0 {
		return false, fmt.Errorf("cost reduction rule %s not found", ruleID)
	}

	cr.Spec.CostReductionRules = append(cr.Spec.CostReductionRules[:idx], cr.Spec.CostReductionRules[idx+1:]...)

	if _, err := updateSamplingCR(ctx, cr); err != nil {
		return false, err
	}
	return true, nil
}

func findCostReductionRuleByHash(rules []v1alpha1.CostReductionRule, hash string) int {
	for i := range rules {
		if v1alpha1.ComputeCostReductionRuleHash(&rules[i]) == hash {
			return i
		}
	}
	return -1
}
