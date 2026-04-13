package sampling

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/services"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getSamplingCRByID(ctx context.Context, samplingID string) (*v1alpha1.Sampling, error) {
	odigosNs := env.GetCurrentNamespace()
	var cr v1alpha1.Sampling
	err := kube.CacheClient.Get(ctx, client.ObjectKey{Namespace: odigosNs, Name: samplingID}, &cr)
	if err != nil {
		return nil, fmt.Errorf("sampling CR %q not found: %w", samplingID, err)
	}
	return &cr, nil
}

func getOrCreateSamplingCR(ctx context.Context, samplingID string) (*v1alpha1.Sampling, error) {
	cr, err := getSamplingCRByID(ctx, samplingID)
	if err == nil {
		return cr, nil
	}

	odigosNs := env.GetCurrentNamespace()

	if !apierrors.IsNotFound(err) {
		return nil, err
	}

	newCR := &v1alpha1.Sampling{
		ObjectMeta: metav1.ObjectMeta{
			Name:      samplingID,
			Namespace: odigosNs,
		},
		Spec: v1alpha1.SamplingSpec{
			Name: samplingID,
		},
	}
	created, createErr := kube.DefaultClient.OdigosClient.Samplings(odigosNs).Create(ctx, newCR, metav1.CreateOptions{})
	if createErr != nil {
		return nil, fmt.Errorf("failed to create sampling CR %q: %w", samplingID, createErr)
	}
	return created, nil
}

func updateSamplingCR(ctx context.Context, cr *v1alpha1.Sampling) (*v1alpha1.Sampling, error) {
	odigosNs := env.GetCurrentNamespace()
	updated, err := kube.DefaultClient.OdigosClient.Samplings(odigosNs).Update(ctx, cr, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// GetAllSamplingRuleGroups lists all Sampling CRs and returns each as a SamplingRules group
// with rules eagerly populated.
func GetAllSamplingRuleGroups(ctx context.Context) ([]*model.SamplingRules, error) {
	odigosNs := env.GetCurrentNamespace()

	var list v1alpha1.SamplingList
	if err := kube.CacheClient.List(ctx, &list, client.InNamespace(odigosNs)); err != nil {
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
	rule := noisyOperationFromInput(input)
	var result *model.NoisyOperationRule

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cr, err := getOrCreateSamplingCR(ctx, samplingID)
		if err != nil {
			return err
		}
		cr.Spec.NoisyOperations = append(cr.Spec.NoisyOperations, rule)
		if _, err := updateSamplingCR(ctx, cr); err != nil {
			return err
		}
		result = convertNoisyOperationToModel(&rule)
		return nil
	})

	return result, err
}

func UpdateNoisyOperationRule(ctx context.Context, samplingID string, ruleID string, input model.NoisyOperationRuleInput) (*model.NoisyOperationRule, error) {
	rule := noisyOperationFromInput(input)
	var result *model.NoisyOperationRule

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cr, err := getSamplingCRByID(ctx, samplingID)
		if err != nil {
			return err
		}
		idx := findNoisyOperationByHash(cr.Spec.NoisyOperations, ruleID)
		if idx < 0 {
			return fmt.Errorf("noisy operation rule %s not found in sampling %s", ruleID, samplingID)
		}
		cr.Spec.NoisyOperations[idx] = rule
		if _, err := updateSamplingCR(ctx, cr); err != nil {
			return err
		}
		result = convertNoisyOperationToModel(&rule)
		return nil
	})

	return result, err
}

func DeleteNoisyOperationRule(ctx context.Context, samplingID string, ruleID string) (bool, error) {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cr, err := getSamplingCRByID(ctx, samplingID)
		if err != nil {
			return err
		}
		idx := findNoisyOperationByHash(cr.Spec.NoisyOperations, ruleID)
		if idx < 0 {
			return fmt.Errorf("noisy operation rule %s not found in sampling %s", ruleID, samplingID)
		}
		cr.Spec.NoisyOperations = append(cr.Spec.NoisyOperations[:idx], cr.Spec.NoisyOperations[idx+1:]...)
		_, err = updateSamplingCR(ctx, cr)
		return err
	})

	return err == nil, err
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
	rule := highlyRelevantOperationFromInput(input)
	var result *model.HighlyRelevantOperationRule

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cr, err := getOrCreateSamplingCR(ctx, samplingID)
		if err != nil {
			return err
		}
		cr.Spec.HighlyRelevantOperations = append(cr.Spec.HighlyRelevantOperations, rule)
		if _, err := updateSamplingCR(ctx, cr); err != nil {
			return err
		}
		result = convertHighlyRelevantOperationToModel(&rule)
		return nil
	})

	return result, err
}

func UpdateHighlyRelevantOperationRule(ctx context.Context, samplingID string, ruleID string, input model.HighlyRelevantOperationRuleInput) (*model.HighlyRelevantOperationRule, error) {
	rule := highlyRelevantOperationFromInput(input)
	var result *model.HighlyRelevantOperationRule

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cr, err := getSamplingCRByID(ctx, samplingID)
		if err != nil {
			return err
		}
		idx := findHighlyRelevantOperationByHash(cr.Spec.HighlyRelevantOperations, ruleID)
		if idx < 0 {
			return fmt.Errorf("highly relevant operation rule %s not found in sampling %s", ruleID, samplingID)
		}
		cr.Spec.HighlyRelevantOperations[idx] = rule
		if _, err := updateSamplingCR(ctx, cr); err != nil {
			return err
		}
		result = convertHighlyRelevantOperationToModel(&rule)
		return nil
	})

	return result, err
}

func DeleteHighlyRelevantOperationRule(ctx context.Context, samplingID string, ruleID string) (bool, error) {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cr, err := getSamplingCRByID(ctx, samplingID)
		if err != nil {
			return err
		}
		idx := findHighlyRelevantOperationByHash(cr.Spec.HighlyRelevantOperations, ruleID)
		if idx < 0 {
			return fmt.Errorf("highly relevant operation rule %s not found in sampling %s", ruleID, samplingID)
		}
		cr.Spec.HighlyRelevantOperations = append(cr.Spec.HighlyRelevantOperations[:idx], cr.Spec.HighlyRelevantOperations[idx+1:]...)
		_, err = updateSamplingCR(ctx, cr)
		return err
	})

	return err == nil, err
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
	rule := costReductionRuleFromInput(input)
	var result *model.CostReductionRule

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cr, err := getOrCreateSamplingCR(ctx, samplingID)
		if err != nil {
			return err
		}
		cr.Spec.CostReductionRules = append(cr.Spec.CostReductionRules, rule)
		if _, err := updateSamplingCR(ctx, cr); err != nil {
			return err
		}
		result = convertCostReductionRuleToModel(&rule)
		return nil
	})

	return result, err
}

func UpdateCostReductionRule(ctx context.Context, samplingID string, ruleID string, input model.CostReductionRuleInput) (*model.CostReductionRule, error) {
	rule := costReductionRuleFromInput(input)
	var result *model.CostReductionRule

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cr, err := getSamplingCRByID(ctx, samplingID)
		if err != nil {
			return err
		}
		idx := findCostReductionRuleByHash(cr.Spec.CostReductionRules, ruleID)
		if idx < 0 {
			return fmt.Errorf("cost reduction rule %s not found in sampling %s", ruleID, samplingID)
		}
		cr.Spec.CostReductionRules[idx] = rule
		if _, err := updateSamplingCR(ctx, cr); err != nil {
			return err
		}
		result = convertCostReductionRuleToModel(&rule)
		return nil
	})

	return result, err
}

func DeleteCostReductionRule(ctx context.Context, samplingID string, ruleID string) (bool, error) {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cr, err := getSamplingCRByID(ctx, samplingID)
		if err != nil {
			return err
		}
		idx := findCostReductionRuleByHash(cr.Spec.CostReductionRules, ruleID)
		if idx < 0 {
			return fmt.Errorf("cost reduction rule %s not found in sampling %s", ruleID, samplingID)
		}
		cr.Spec.CostReductionRules = append(cr.Spec.CostReductionRules[:idx], cr.Spec.CostReductionRules[idx+1:]...)
		_, err = updateSamplingCR(ctx, cr)
		return err
	})

	return err == nil, err
}

func findCostReductionRuleByHash(rules []v1alpha1.CostReductionRule, hash string) int {
	for i := range rules {
		if v1alpha1.ComputeCostReductionRuleHash(&rules[i]) == hash {
			return i
		}
	}
	return -1
}
