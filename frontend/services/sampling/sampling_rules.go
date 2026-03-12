package sampling

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/services"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getSamplingCRByID(ctx context.Context, samplingID string) (*v1alpha1.Sampling, error) {
	odigosNs := env.GetCurrentNamespace()
	cr, err := kube.DefaultClient.OdigosClient.Samplings(odigosNs).Get(ctx, samplingID, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("sampling CR %q not found: %w", samplingID, err)
	}
	return cr, nil
}

func updateSamplingCR(ctx context.Context, cr *v1alpha1.Sampling) (*v1alpha1.Sampling, error) {
	odigosNs := env.GetCurrentNamespace()
	updated, err := kube.DefaultClient.OdigosClient.Samplings(odigosNs).Update(ctx, cr, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update sampling CR: %w", err)
	}
	return updated, nil
}

// GetAllSamplingRuleGroups lists all Sampling CRs and returns each as a SamplingRules group
// with rules eagerly populated.
func GetAllSamplingRuleGroups(ctx context.Context) ([]*model.SamplingRules, error) {
	odigosNs := env.GetCurrentNamespace()

	list, err := kube.DefaultClient.OdigosClient.Samplings(odigosNs).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list sampling CRs: %w", err)
	}

	groups := make([]*model.SamplingRules, 0, len(list.Items))
	for i := range list.Items {
		cr := &list.Items[i]
		groups = append(groups, samplingCRToModel(cr))
	}
	return groups, nil
}

func samplingCRToModel(cr *v1alpha1.Sampling) *model.SamplingRules {
	noisy := make([]*model.NoisyOperationRule, 0, len(cr.Spec.NoisyOperations))
	for j := range cr.Spec.NoisyOperations {
		noisy = append(noisy, convertNoisyOperationToModel(&cr.Spec.NoisyOperations[j]))
	}

	relevant := make([]*model.HighlyRelevantOperationRule, 0, len(cr.Spec.HighlyRelevantOperations))
	for j := range cr.Spec.HighlyRelevantOperations {
		relevant = append(relevant, convertHighlyRelevantOperationToModel(&cr.Spec.HighlyRelevantOperations[j]))
	}

	cost := make([]*model.CostReductionRule, 0, len(cr.Spec.CostReductionRules))
	for j := range cr.Spec.CostReductionRules {
		cost = append(cost, convertCostReductionRuleToModel(&cr.Spec.CostReductionRules[j]))
	}

	return &model.SamplingRules{
		ID:                       cr.Name,
		Name:                     services.StringPtrIfNotEmpty(cr.Spec.Name),
		NoisyOperations:          noisy,
		HighlyRelevantOperations: relevant,
		CostReductionRules:       cost,
	}
}

// ---- Noisy Operations ----

func CreateNoisyOperationRule(ctx context.Context, samplingID string, input model.NoisyOperationRuleInput) (*model.NoisyOperationRule, error) {
	cr, err := getSamplingCRByID(ctx, samplingID)
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

func UpdateNoisyOperationRule(ctx context.Context, samplingID string, ruleID string, input model.NoisyOperationRuleInput) (*model.NoisyOperationRule, error) {
	cr, err := getSamplingCRByID(ctx, samplingID)
	if err != nil {
		return nil, err
	}

	idx := findNoisyOperationByHash(cr.Spec.NoisyOperations, ruleID)
	if idx < 0 {
		return nil, fmt.Errorf("noisy operation rule %s not found in sampling %s", ruleID, samplingID)
	}

	rule := noisyOperationFromInput(input)
	cr.Spec.NoisyOperations[idx] = rule

	if _, err := updateSamplingCR(ctx, cr); err != nil {
		return nil, err
	}
	return convertNoisyOperationToModel(&rule), nil
}

func DeleteNoisyOperationRule(ctx context.Context, samplingID string, ruleID string) (bool, error) {
	cr, err := getSamplingCRByID(ctx, samplingID)
	if err != nil {
		return false, err
	}

	idx := findNoisyOperationByHash(cr.Spec.NoisyOperations, ruleID)
	if idx < 0 {
		return false, fmt.Errorf("noisy operation rule %s not found in sampling %s", ruleID, samplingID)
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

func CreateHighlyRelevantOperationRule(ctx context.Context, samplingID string, input model.HighlyRelevantOperationRuleInput) (*model.HighlyRelevantOperationRule, error) {
	cr, err := getSamplingCRByID(ctx, samplingID)
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

func UpdateHighlyRelevantOperationRule(ctx context.Context, samplingID string, ruleID string, input model.HighlyRelevantOperationRuleInput) (*model.HighlyRelevantOperationRule, error) {
	cr, err := getSamplingCRByID(ctx, samplingID)
	if err != nil {
		return nil, err
	}

	idx := findHighlyRelevantOperationByHash(cr.Spec.HighlyRelevantOperations, ruleID)
	if idx < 0 {
		return nil, fmt.Errorf("highly relevant operation rule %s not found in sampling %s", ruleID, samplingID)
	}

	rule := highlyRelevantOperationFromInput(input)
	cr.Spec.HighlyRelevantOperations[idx] = rule

	if _, err := updateSamplingCR(ctx, cr); err != nil {
		return nil, err
	}
	return convertHighlyRelevantOperationToModel(&rule), nil
}

func DeleteHighlyRelevantOperationRule(ctx context.Context, samplingID string, ruleID string) (bool, error) {
	cr, err := getSamplingCRByID(ctx, samplingID)
	if err != nil {
		return false, err
	}

	idx := findHighlyRelevantOperationByHash(cr.Spec.HighlyRelevantOperations, ruleID)
	if idx < 0 {
		return false, fmt.Errorf("highly relevant operation rule %s not found in sampling %s", ruleID, samplingID)
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

func CreateCostReductionRule(ctx context.Context, samplingID string, input model.CostReductionRuleInput) (*model.CostReductionRule, error) {
	cr, err := getSamplingCRByID(ctx, samplingID)
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

func UpdateCostReductionRule(ctx context.Context, samplingID string, ruleID string, input model.CostReductionRuleInput) (*model.CostReductionRule, error) {
	cr, err := getSamplingCRByID(ctx, samplingID)
	if err != nil {
		return nil, err
	}

	idx := findCostReductionRuleByHash(cr.Spec.CostReductionRules, ruleID)
	if idx < 0 {
		return nil, fmt.Errorf("cost reduction rule %s not found in sampling %s", ruleID, samplingID)
	}

	rule := costReductionRuleFromInput(input)
	cr.Spec.CostReductionRules[idx] = rule

	if _, err := updateSamplingCR(ctx, cr); err != nil {
		return nil, err
	}
	return convertCostReductionRuleToModel(&rule), nil
}

func DeleteCostReductionRule(ctx context.Context, samplingID string, ruleID string) (bool, error) {
	cr, err := getSamplingCRByID(ctx, samplingID)
	if err != nil {
		return false, err
	}

	idx := findCostReductionRuleByHash(cr.Spec.CostReductionRules, ruleID)
	if idx < 0 {
		return false, fmt.Errorf("cost reduction rule %s not found in sampling %s", ruleID, samplingID)
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
