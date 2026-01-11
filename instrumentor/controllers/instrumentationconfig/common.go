package instrumentationconfig

import (
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/instrumentor/controllers/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func updateInstrumentationConfigForWorkload(ic *odigosv1alpha1.InstrumentationConfig, rules *odigosv1alpha1.InstrumentationRuleList, conf *common.OdigosConfiguration) error {
	workload, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(ic.Name, ic.Namespace)
	if err != nil {
		return err
	}

	sdkConfigs := make([]odigosv1alpha1.SdkConfig, 0, len(ic.Spec.Containers))
	runtimeDetailsByContainer := ic.RuntimeDetailsByContainer()

	for _, runtimeDetails := range runtimeDetailsByContainer {
		if runtimeDetails == nil {
			continue
		}
		containerLanguage := runtimeDetails.Language
		if containerLanguage == "" || containerLanguage == common.UnknownProgrammingLanguage {
			continue
		}
		sdkConfigs = createDefaultSdkConfig(sdkConfigs, containerLanguage)
	}
	// iterate over all the payload collection rules, and update the instrumentation config accordingly
	for i := range rules.Items {
		rule := &rules.Items[i]
		// skip disabled rules
		if rule.Spec.Disabled {
			continue
		}
		// filter out rules where the workload does not match
		participating := utils.IsWorkloadParticipatingInRule(workload, rule)
		if !participating {
			continue
		}
		// merge the rule into all the sdk configs
		for i := range sdkConfigs {
			// If we've recieved nil for the instrumentation libraries, then we apply the given rules as global rules to the SDK.
			if rule.Spec.InstrumentationLibraries == nil {
				if rule.Spec.PayloadCollection != nil {
					sdkConfigs[i].DefaultPayloadCollection.HttpRequest = mergeHttpPayloadCollectionRules(sdkConfigs[i].DefaultPayloadCollection.HttpRequest, rule.Spec.PayloadCollection.HttpRequest)
					sdkConfigs[i].DefaultPayloadCollection.HttpResponse = mergeHttpPayloadCollectionRules(sdkConfigs[i].DefaultPayloadCollection.HttpResponse, rule.Spec.PayloadCollection.HttpResponse)
					sdkConfigs[i].DefaultPayloadCollection.DbQuery = mergeDbPayloadCollectionRules(sdkConfigs[i].DefaultPayloadCollection.DbQuery, rule.Spec.PayloadCollection.DbQuery)
					sdkConfigs[i].DefaultPayloadCollection.Messaging = mergeMessagingPayloadCollectionRules(sdkConfigs[i].DefaultPayloadCollection.Messaging, rule.Spec.PayloadCollection.Messaging)
				}
				if rule.Spec.CodeAttributes != nil {
					sdkConfigs[i].DefaultCodeAttributes = mergeCodeAttributesRules(sdkConfigs[i].DefaultCodeAttributes, rule.Spec.CodeAttributes)
				}
				if rule.Spec.HeadersCollection != nil {
					sdkConfigs[i].DefaultHeadersCollection = mergeHttpHeadersCollectionrules(sdkConfigs[i].DefaultHeadersCollection, rule.Spec.HeadersCollection)
				}
				if rule.Spec.TraceConfig != nil {
					sdkConfigs[i].DefaultTraceConfig = mergeDefaultTracingConfig(sdkConfigs[i].DefaultTraceConfig, rule.Spec.TraceConfig)
				}
				if rule.Spec.CustomInstrumentations != nil {
					sdkConfigs[i].CustomInstrumentations = mergeCustomInstrumentations(sdkConfigs[i].CustomInstrumentations, rule.Spec.CustomInstrumentations)
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
					if rule.Spec.CodeAttributes != nil {
						libraryConfig.CodeAttributes = mergeCodeAttributesRules(libraryConfig.CodeAttributes, rule.Spec.CodeAttributes)
					}
					if rule.Spec.HeadersCollection != nil {
						libraryConfig.HeadersCollection = mergeHttpHeadersCollectionrules(libraryConfig.HeadersCollection, rule.Spec.HeadersCollection)
					}
					if rule.Spec.TraceConfig != nil {
						libraryConfig.TraceConfig = mergeTracingConfig(libraryConfig.TraceConfig, rule.Spec.TraceConfig)
					}
				}
			}
		}
	}

	ic.Spec.SdkConfigs = sdkConfigs

	// populate runtime metrics in sdkConfigs based on effective config and distro support
	populateRuntimeMetricsInSdkConfigs(ic, conf)

	return nil
}

func mergeDefaultTracingConfig(defaultConfig *instrumentationrules.TraceConfig, rule *instrumentationrules.TraceConfig) *instrumentationrules.TraceConfig {
	if defaultConfig == nil {
		return rule
	}
	if rule == nil {
		return defaultConfig
	}

	mergedRules := &instrumentationrules.TraceConfig{}

	// Only set Disabled if we have actual values to work with
	if defaultConfig.Disabled != nil && rule.Disabled != nil {
		// Both values are set, use OR logic: tracing is disabled if either config disables it
		mergedRules.Disabled = boolPtr(*defaultConfig.Disabled || *rule.Disabled)
	} else if defaultConfig.Disabled != nil {
		// Only default config has a value, use it
		mergedRules.Disabled = defaultConfig.Disabled
	} else if rule.Disabled != nil {
		// Only rule has a value, use it
		mergedRules.Disabled = rule.Disabled
	}
	// If both are nil, mergedRules.Disabled remains nil
	return mergedRules
}

func mergeTracingConfig(sdkConfig *odigosv1alpha1.InstrumentationLibraryConfigTraces, rule *instrumentationrules.TraceConfig) *odigosv1alpha1.InstrumentationLibraryConfigTraces {
	// The SDK config uses "Enabled" field to enable/disable tracing.
	// The rule uses "Disabled" field to disable tracing, since the semantics of "Disabled" allows for default nil/false
	// which is clearer for the user facing object.

	if sdkConfig == nil {
		if rule.Disabled != nil {
			return &odigosv1alpha1.InstrumentationLibraryConfigTraces{
				Enabled: boolPtr(!*rule.Disabled),
			}
		}
		// Both sdkConfig and rule.Disabled are nil, return nil config
		return &odigosv1alpha1.InstrumentationLibraryConfigTraces{}
	} else if rule == nil {
		return sdkConfig
	}

	mergedRules := odigosv1alpha1.InstrumentationLibraryConfigTraces{}

	// Only set Enabled if we have actual values to work with
	if sdkConfig.Enabled != nil && rule.Disabled != nil {
		// Both values are set, use AND logic: tracing is enabled only if SDK config enables it AND rule doesn't disable it
		mergedRules.Enabled = boolPtr(*sdkConfig.Enabled && !*rule.Disabled)
	} else if sdkConfig.Enabled != nil {
		// Only SDK config has a value, use it
		mergedRules.Enabled = sdkConfig.Enabled
	} else if rule.Disabled != nil {
		// Only rule has a value, use it
		mergedRules.Enabled = boolPtr(!*rule.Disabled)
	}
	// If both are nil, mergedRules.Enabled remains nil
	return &mergedRules
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

func populateRuntimeMetricsInSdkConfigs(ic *odigosv1alpha1.InstrumentationConfig, effectiveConfig *common.OdigosConfiguration) {
	logger := log.Log.WithName("runtime-metrics")

	// Check if runtime metrics are configured in effective config
	if effectiveConfig == nil ||
		effectiveConfig.MetricsSources == nil ||
		effectiveConfig.MetricsSources.AgentMetrics == nil ||
		effectiveConfig.MetricsSources.AgentMetrics.RuntimeMetrics == nil {
		return
	}

	runtimeMetricsConfig := effectiveConfig.MetricsSources.AgentMetrics.RuntimeMetrics

	// Add runtime metrics to every Java sdkConfig
	for i := range ic.Spec.SdkConfigs {
		sdkConfig := &ic.Spec.SdkConfigs[i]

		// Currently we're adding the runtimeMetricConfig for all java sdkConfigs.
		// This should be changed once we migrate this code to run in the container config where we have distro.
		switch sdkConfig.Language {
		case common.JavaProgrammingLanguage:
			if runtimeMetricsConfig.Java != nil {
				logger.V(0).Info("Adding runtime metrics to Java sdkConfig", "workload", ic.Name)
				sdkConfig.RuntimeMetrics = &common.MetricsSourceAgentRuntimeMetricsConfiguration{
					Java: convertJavaRuntimeMetricsConfig(runtimeMetricsConfig.Java),
				}
			}
		}
	}
}

// convertJavaRuntimeMetricsConfig converts from common config to API config
func convertJavaRuntimeMetricsConfig(commonConfig *common.MetricsSourceAgentJavaRuntimeMetricsConfiguration) *common.MetricsSourceAgentJavaRuntimeMetricsConfiguration {
	if commonConfig == nil {
		return nil
	}

	apiConfig := &common.MetricsSourceAgentJavaRuntimeMetricsConfiguration{
		Disabled: commonConfig.Disabled,
	}

	// Convert metrics array
	if len(commonConfig.Metrics) > 0 {
		apiConfig.Metrics = make([]common.MetricsSourceAgentRuntimeMetricConfiguration, len(commonConfig.Metrics))
		for i, metric := range commonConfig.Metrics {
			apiConfig.Metrics[i] = common.MetricsSourceAgentRuntimeMetricConfiguration{
				Name:     metric.Name,
				Disabled: metric.Disabled,
			}
		}
	}

	return apiConfig
}

func mergeCustomInstrumentations(rule1 *instrumentationrules.CustomInstrumentations, rule2 *instrumentationrules.CustomInstrumentations) *instrumentationrules.CustomInstrumentations {
	if rule1 == nil {
		return rule2
	} else if rule2 == nil {
		return rule1
	}

	mergedRules := &instrumentationrules.CustomInstrumentations{}

	// Merge Golang custom probes
	mergedGolangProbes := make([]instrumentationrules.GolangCustomProbe, 0, len(rule1.Golang)+len(rule2.Golang))
	mergedGolangProbes = append(mergedGolangProbes, rule1.Golang...)
	mergedGolangProbes = append(mergedGolangProbes, rule2.Golang...)
	mergedRules.Golang = mergedGolangProbes

	// Merge Java custom probes
	mergedJavaProbes := make([]instrumentationrules.JavaCustomProbe, 0, len(rule1.Java)+len(rule2.Java))
	mergedJavaProbes = append(mergedJavaProbes, rule1.Java...)
	mergedJavaProbes = append(mergedJavaProbes, rule2.Java...)
	mergedRules.Java = mergedJavaProbes

	return mergedRules
}

func mergeHttpHeadersCollectionrules(rule1 *instrumentationrules.HttpHeadersCollection, rule2 *instrumentationrules.HttpHeadersCollection) *instrumentationrules.HttpHeadersCollection {
	if rule1 == nil {
		return rule2
	} else if rule2 == nil {
		return rule1
	}

	mergedRules := instrumentationrules.HttpHeadersCollection{}

	// Merge the headers collection rules
	var mergedHeaders []string
	if rule1.HeaderKeys != nil {
		mergedHeaders = append(mergedHeaders, rule1.HeaderKeys...)
	}

	if rule2.HeaderKeys != nil {
		mergedHeaders = append(mergedHeaders, rule2.HeaderKeys...)
	}

	mergedRules.HeaderKeys = mergedHeaders
	return &mergedRules
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

// will merge 2 optional boolean fields from 2 instrumentation rules.
// if any of them is true, the result is true.
// if none of them is true, but one is false, the result is false.
// if both are nil, the result is nil
func merge2RuleBooleans(value1 *bool, value2 *bool) *bool {
	if value1 == nil {
		return value2
	} else if value2 == nil {
		return value1
	}
	return boolPtr(*value1 || *value2)
}

func mergeCodeAttributesRules(rule1 *instrumentationrules.CodeAttributes, rule2 *instrumentationrules.CodeAttributes) *instrumentationrules.CodeAttributes {
	if rule1 == nil {
		return rule2
	} else if rule2 == nil {
		return rule1
	}

	mergedRules := instrumentationrules.CodeAttributes{}
	mergedRules.Column = merge2RuleBooleans(rule1.Column, rule2.Column)
	mergedRules.FilePath = merge2RuleBooleans(rule1.FilePath, rule2.FilePath)
	mergedRules.Function = merge2RuleBooleans(rule1.Function, rule2.Function)
	mergedRules.LineNumber = merge2RuleBooleans(rule1.LineNumber, rule2.LineNumber)
	mergedRules.Namespace = merge2RuleBooleans(rule1.Namespace, rule2.Namespace)
	mergedRules.Stacktrace = merge2RuleBooleans(rule1.Stacktrace, rule2.Stacktrace)

	return &mergedRules
}

func boolPtr(b bool) *bool {
	return &b
}
