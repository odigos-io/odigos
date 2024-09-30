package instrumentationconfig

import (
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/instrumentor/controllers/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

func updateInstrumentationConfigForWorkload(ic *odigosv1alpha1.InstrumentationConfig, ia *odigosv1alpha1.InstrumentedApplication, rules *odigosv1alpha1.InstrumentationRuleList) error {

	workloadName, workloadKind, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(ia.Name)
	if err != nil {
		return err
	}
	workload := workload.PodWorkload{
		Name:      workloadName,
		Namespace: ia.Namespace,
		Kind:      workloadKind,
	}

	sdkConfigs := make([]odigosv1alpha1.SdkConfig, 0, len(ia.Spec.RuntimeDetails))

	// create an empty sdk config for each detected programming language
	for _, container := range ia.Spec.RuntimeDetails {
		containerLanguage := container.Language
		if containerLanguage == common.IgnoredProgrammingLanguage || containerLanguage == common.UnknownProgrammingLanguage {
			continue
		}
		sdkConfigs = createDefaultSdkConfig(sdkConfigs, containerLanguage)
	}

	// iterate over all the payload collection rules, and update the instrumentation config accordingly
	for i := range rules.Items {
		rule := &rules.Items[i]
		if rule.Spec.Disabled {
			continue
		}
		// filter out rules where the workload does not match
		participating := utils.IsWorkloadParticipatingInRule(workload, rule)
		if !participating {
			continue
		}

		for i := range sdkConfigs {
			if rule.Spec.InstrumentationLibraries == nil { // nil means a rule in SDK level, that applies unless overridden by library level rule
				if rule.Spec.PayloadCollection != nil {
					sdkConfigs[i].DefaultPayloadCollection.HttpRequest = mergeHttpPayloadCollectionRules(sdkConfigs[i].DefaultPayloadCollection.HttpRequest, rule.Spec.PayloadCollection.HttpRequest)
					sdkConfigs[i].DefaultPayloadCollection.HttpResponse = mergeHttpPayloadCollectionRules(sdkConfigs[i].DefaultPayloadCollection.HttpResponse, rule.Spec.PayloadCollection.HttpResponse)
					sdkConfigs[i].DefaultPayloadCollection.DbQuery = mergeDbPayloadCollectionRules(sdkConfigs[i].DefaultPayloadCollection.DbQuery, rule.Spec.PayloadCollection.DbQuery)
					sdkConfigs[i].DefaultPayloadCollection.Messaging = mergeMessagingPayloadCollectionRules(sdkConfigs[i].DefaultPayloadCollection.Messaging, rule.Spec.PayloadCollection.Messaging)
				}
			} else {
				for _, library := range *rule.Spec.InstrumentationLibraries {
					libraryConfig := findOrCreateSdkLibraryConfig(&sdkConfigs[i], library)
					if libraryConfig == nil {
						// library is not relevant to this SDK
						continue
					}
					if rule.Spec.PayloadCollection != nil {
						libraryConfig.PayloadCollection.HttpRequest = mergeHttpPayloadCollectionRules(libraryConfig.PayloadCollection.HttpRequest, rule.Spec.PayloadCollection.HttpRequest)
						libraryConfig.PayloadCollection.HttpResponse = mergeHttpPayloadCollectionRules(libraryConfig.PayloadCollection.HttpResponse, rule.Spec.PayloadCollection.HttpResponse)
						libraryConfig.PayloadCollection.DbQuery = mergeDbPayloadCollectionRules(libraryConfig.PayloadCollection.DbQuery, rule.Spec.PayloadCollection.DbQuery)
						libraryConfig.PayloadCollection.Messaging = mergeMessagingPayloadCollectionRules(libraryConfig.PayloadCollection.Messaging, rule.Spec.PayloadCollection.Messaging)
					}
				}
			}
		}
	}

	ic.Spec.SdkConfigs = sdkConfigs

	return nil
}

// returns a pointer to the instrumentation library config, creating it if it does not exist
// the pointer can be used to modify the config
func findOrCreateSdkLibraryConfig(sdkConfig *odigosv1alpha1.SdkConfig, library odigosv1alpha1.InstrumentationLibraryGlobalId) *odigosv1alpha1.InstrumentationLibraryConfig {
	if library.Language != sdkConfig.Language {
		return nil
	}

	for i, libConfig := range sdkConfig.InstrumentationLibraryConfigs {
		if libConfig.InstrumentationLibraryId.InstrumentationLibraryName == library.Name &&
			libConfig.InstrumentationLibraryId.SpanKind == library.SpanKind {

			// if already present, return a pointer to it which can be modified by the caller
			return &sdkConfig.InstrumentationLibraryConfigs[i]
		}
	}
	newLibConfig := odigosv1alpha1.InstrumentationLibraryConfig{
		InstrumentationLibraryId: odigosv1alpha1.InstrumentationLibraryId{
			InstrumentationLibraryName: library.Name,
			SpanKind:                   library.SpanKind,
		},
		PayloadCollection: &instrumentationrules.PayloadCollection{},
	}
	sdkConfig.InstrumentationLibraryConfigs = append(sdkConfig.InstrumentationLibraryConfigs, newLibConfig)
	return &sdkConfig.InstrumentationLibraryConfigs[len(sdkConfig.InstrumentationLibraryConfigs)-1]
}

func createDefaultSdkConfig(sdkConfigs []odigosv1alpha1.SdkConfig, containerLanguage common.ProgrammingLanguage) []odigosv1alpha1.SdkConfig {
	// if the language is already present, do nothing
	for _, sdkConfig := range sdkConfigs {
		if sdkConfig.Language == containerLanguage {
			return sdkConfigs
		}
	}
	return append(sdkConfigs, odigosv1alpha1.SdkConfig{
		Language:                 containerLanguage,
		DefaultPayloadCollection: &instrumentationrules.PayloadCollection{},
	})
}

func mergeHttpPayloadCollectionRules(rule1 *instrumentationrules.HttpPayloadCollection, rule2 *instrumentationrules.HttpPayloadCollection) *instrumentationrules.HttpPayloadCollection {

	// nil means a rules has not yet been set, so return the other rule
	if rule1 == nil {
		return rule2
	} else if rule2 == nil {
		return rule1
	}

	// merge of the 2 non nil rules
	mergedRules := instrumentationrules.HttpPayloadCollection{}

	// MimeTypes is extended to include both. nil means "all" so treat it as such
	if rule1.MimeTypes == nil || rule2.MimeTypes == nil {
		mergedRules.MimeTypes = nil
	} else {
		mergeMimeTypeMap := make(map[string]struct{})
		for _, mimeType := range *rule1.MimeTypes {
			mergeMimeTypeMap[mimeType] = struct{}{}
		}
		for _, mimeType := range *rule2.MimeTypes {
			mergeMimeTypeMap[mimeType] = struct{}{}
		}
		mergedMimeTypeSlice := make([]string, 0, len(mergeMimeTypeMap))
		for mimeType := range mergeMimeTypeMap {
			mergedMimeTypeSlice = append(mergedMimeTypeSlice, mimeType)
		}
		mergedRules.MimeTypes = &mergedMimeTypeSlice
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

func mergeDbPayloadCollectionRules(rule1 *instrumentationrules.DbQueryPayloadCollection, rule2 *instrumentationrules.DbQueryPayloadCollection) *instrumentationrules.DbQueryPayloadCollection {
	if rule1 == nil {
		return rule2
	} else if rule2 == nil {
		return rule1
	}

	mergedRules := instrumentationrules.DbQueryPayloadCollection{}

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

func mergeMessagingPayloadCollectionRules(rule1 *instrumentationrules.MessagingPayloadCollection, rule2 *instrumentationrules.MessagingPayloadCollection) *instrumentationrules.MessagingPayloadCollection {
	if rule1 == nil {
		return rule2
	} else if rule2 == nil {
		return rule1
	}

	mergedRules := instrumentationrules.MessagingPayloadCollection{}

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
