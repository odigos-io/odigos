package instrumentationconfig

import (
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	rulesv1alpha1 "github.com/odigos-io/odigos/api/rules/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

func updateInstrumentationConfigForWorkload(ic *odigosv1alpha1.InstrumentationConfig, ia *odigosv1alpha1.InstrumentedApplication, payloadcollectionrules []rulesv1alpha1.PayloadCollection) error {

	workloadName, workloadKind, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(ia.Name)
	if err != nil {
		return err
	}
	workload := workload.PodWorkload{
		Name:      workloadName,
		Namespace: ia.Namespace,
		Kind:      workloadKind,
	}

	// delete all existing sdk configs to re-calculate them
	ic.Spec.SdkConfigs = []odigosv1alpha1.SdkConfig{}

	// create an empty sdk config for each detected programming language
ContainersIteration:
	for _, container := range ia.Spec.RuntimeDetails {
		containerLanguage := container.Language
		if containerLanguage == common.IgnoredProgrammingLanguage || containerLanguage == common.UnknownProgrammingLanguage {
			continue
		}
		for _, sdkConfig := range ic.Spec.SdkConfigs {
			if sdkConfig.Language == containerLanguage {
				continue ContainersIteration
			}
		}
		ic.Spec.SdkConfigs = append(ic.Spec.SdkConfigs, odigosv1alpha1.SdkConfig{
			Language: containerLanguage,
		})
	}

	// iterate over all the payload collection rules, and update the instrumentation config accordingly
	for _, rule := range payloadcollectionrules {
		if rule.Spec.Disabled {
			continue
		}
		// filter out rules where the workload does not match
		participating := isWorkloadParticipatingInRule(workload, &rule)
		if !participating {
			continue
		}

		for i := range ic.Spec.SdkConfigs {
			ic.Spec.SdkConfigs[i].DefaultHttpPayloadCollection = mergeHttpPayloadCollectionRules(ic.Spec.SdkConfigs[i].DefaultHttpPayloadCollection, rule.Spec.HttpPayloadCollectionRule)
			ic.Spec.SdkConfigs[i].DefaultDbPayloadCollection = mergeDbPayloadCollectionRules(ic.Spec.SdkConfigs[i].DefaultDbPayloadCollection, rule.Spec.DbPayloadCollectionRule)
		}
	}

	return nil
}

// naive implementation, can be optimized.
// assumption is that the list of workloads is small
func isWorkloadParticipatingInRule(workload workload.PodWorkload, rule *rulesv1alpha1.PayloadCollection) bool {
	// nil means all workloads are participating
	if rule.Spec.Workloads == nil {
		return true
	}
	for _, allowedWorkload := range *rule.Spec.Workloads {
		if allowedWorkload == workload {
			return true
		}
	}
	return false
}

func mergeHttpPayloadCollectionRules(rule1 *rulesv1alpha1.HttpPayloadCollectionRule, rule2 *rulesv1alpha1.HttpPayloadCollectionRule) *rulesv1alpha1.HttpPayloadCollectionRule {

	// nil means a rules has not yet been set, so return the other rule
	if rule1 == nil {
		return rule2
	} else if rule2 == nil {
		return rule1
	}

	// merge of the 2 non nil rules
	mergedRules := rulesv1alpha1.HttpPayloadCollectionRule{}

	// AllowedMimeType is extended to include both. nil means "all" so treat it as such
	if rule1.AllowedMimeType == nil || rule2.AllowedMimeType == nil {
		mergedRules.AllowedMimeType = nil
	} else {
		mergedMimeTypes := append(*rule1.AllowedMimeType, *rule2.AllowedMimeType...)
		mergedRules.AllowedMimeType = &mergedMimeTypes
	}

	// MaxPayloadLength - choose the smallest value, as this is the maximum allowed
	if rule1.MaxPayloadLength == nil {
		mergedRules.MaxPayloadLength = rule2.MaxPayloadLength
	} else if rule2.MaxPayloadLength == nil {
		mergedRules.MaxPayloadLength = rule1.MaxPayloadLength
	} else {
		if *rule1.MaxPayloadLength < *rule2.MaxPayloadLength {
			mergedRules.MaxPayloadLength = rule1.MaxPayloadLength
		} else {
			mergedRules.MaxPayloadLength = rule2.MaxPayloadLength
		}
	}

	// DropPartialPayloads - if any of the rules is set to drop, the merged rule will drop
	if rule1.DropPartialPayloads == nil {
		mergedRules.DropPartialPayloads = rule2.DropPartialPayloads
	} else if rule2.DropPartialPayloads == nil {
		mergedRules.DropPartialPayloads = rule1.DropPartialPayloads
	} else {
		mergedRules.DropPartialPayloads = boolPtr(*rule1.DropPartialPayloads || *rule2.DropPartialPayloads)
	}

	return &mergedRules
}

func mergeDbPayloadCollectionRules(rule1 *rulesv1alpha1.DbPayloadCollectionRule, rule2 *rulesv1alpha1.DbPayloadCollectionRule) *rulesv1alpha1.DbPayloadCollectionRule {
	if rule1 == nil {
		return rule2
	} else if rule2 == nil {
		return rule1
	}

	mergedRules := rulesv1alpha1.DbPayloadCollectionRule{}

	// MaxPayloadLength - choose the smallest value, as this is the maximum allowed
	if rule1.MaxPayloadLength == nil {
		mergedRules.MaxPayloadLength = rule2.MaxPayloadLength
	} else if rule2.MaxPayloadLength == nil {
		mergedRules.MaxPayloadLength = rule1.MaxPayloadLength
	} else {
		if *rule1.MaxPayloadLength < *rule2.MaxPayloadLength {
			mergedRules.MaxPayloadLength = rule1.MaxPayloadLength
		} else {
			mergedRules.MaxPayloadLength = rule2.MaxPayloadLength
		}
	}

	// DropPartialPayloads - if any of the rules is set to drop, the merged rule will drop
	if rule1.DropPartialPayloads == nil {
		mergedRules.DropPartialPayloads = rule2.DropPartialPayloads
	} else if rule2.DropPartialPayloads == nil {
		mergedRules.DropPartialPayloads = rule1.DropPartialPayloads
	} else {
		mergedRules.DropPartialPayloads = boolPtr(*rule1.DropPartialPayloads || *rule2.DropPartialPayloads)
	}

	return &mergedRules
}

func boolPtr(b bool) *bool {
	return &b
}
