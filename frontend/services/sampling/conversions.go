package sampling

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/services"
)

// ---- Input → CRD converters ----

func noisyOperationFromInput(input model.NoisyOperationRuleInput) v1alpha1.NoisyOperation {
	return v1alpha1.NoisyOperation{
		Name:             services.DerefString(input.Name),
		Disabled:         services.DerefBool(input.Disabled),
		SourceScopes:     sourcesScopeInputToCRD(input.SourceScopes),
		Operation:        headSamplingOperationMatcherInputToCRD(input.Operation),
		PercentageAtMost: input.PercentageAtMost,
		Notes:            services.DerefString(input.Notes),
	}
}

func highlyRelevantOperationFromInput(input model.HighlyRelevantOperationRuleInput) v1alpha1.HighlyRelevantOperation {
	return v1alpha1.HighlyRelevantOperation{
		Name:              services.DerefString(input.Name),
		Disabled:          services.DerefBool(input.Disabled),
		SourceScopes:      sourcesScopeInputToCRD(input.SourceScopes),
		Error:             services.DerefBool(input.Error),
		DurationAtLeastMs: input.DurationAtLeastMs,
		Operation:         tailSamplingOperationMatcherInputToCRD(input.Operation),
		PercentageAtLeast: input.PercentageAtLeast,
		Notes:             services.DerefString(input.Notes),
	}
}

func costReductionRuleFromInput(input model.CostReductionRuleInput) v1alpha1.CostReductionRule {
	return v1alpha1.CostReductionRule{
		Name:             services.DerefString(input.Name),
		Disabled:         services.DerefBool(input.Disabled),
		SourceScopes:     sourcesScopeInputToCRD(input.SourceScopes),
		Operation:        tailSamplingOperationMatcherInputToCRD(input.Operation),
		PercentageAtMost: input.PercentageAtMost,
		Notes:            services.DerefString(input.Notes),
	}
}

// ---- CRD → Model converters ----

func convertNoisyOperationToModel(rule *v1alpha1.NoisyOperation) *model.NoisyOperationRule {
	return &model.NoisyOperationRule{
		RuleID:           v1alpha1.ComputeNoisyOperationHash(rule),
		Name:             services.StringPtrIfNotEmpty(rule.Name),
		Disabled:         rule.Disabled,
		SourceScopes:     sourcesScopeCRDToModel(rule.SourceScopes),
		Operation:        headSamplingOperationMatcherCRDToModel(rule.Operation),
		PercentageAtMost: rule.PercentageAtMost,
		Notes:            services.StringPtrIfNotEmpty(rule.Notes),
	}
}

func convertHighlyRelevantOperationToModel(rule *v1alpha1.HighlyRelevantOperation) *model.HighlyRelevantOperationRule {
	return &model.HighlyRelevantOperationRule{
		RuleID:            v1alpha1.ComputeHighlyRelevantOperationHash(rule),
		Name:              services.StringPtrIfNotEmpty(rule.Name),
		Disabled:          rule.Disabled,
		SourceScopes:      sourcesScopeCRDToModel(rule.SourceScopes),
		Error:             rule.Error,
		DurationAtLeastMs: rule.DurationAtLeastMs,
		Operation:         tailSamplingOperationMatcherCRDToModel(rule.Operation),
		PercentageAtLeast: rule.PercentageAtLeast,
		Notes:             services.StringPtrIfNotEmpty(rule.Notes),
	}
}

func convertCostReductionRuleToModel(rule *v1alpha1.CostReductionRule) *model.CostReductionRule {
	return &model.CostReductionRule{
		RuleID:           v1alpha1.ComputeCostReductionRuleHash(rule),
		Name:             services.StringPtrIfNotEmpty(rule.Name),
		Disabled:         rule.Disabled,
		SourceScopes:     sourcesScopeCRDToModel(rule.SourceScopes),
		Operation:        tailSamplingOperationMatcherCRDToModel(rule.Operation),
		PercentageAtMost: rule.PercentageAtMost,
		Notes:            services.StringPtrIfNotEmpty(rule.Notes),
	}
}

// ---- Shared conversion helpers ----

func sourcesScopeInputToCRD(scopes []*model.SourcesScopeInput) []k8sconsts.SourcesScope {
	if scopes == nil {
		return nil
	}
	result := make([]k8sconsts.SourcesScope, len(scopes))
	for i, s := range scopes {
		result[i] = k8sconsts.SourcesScope{
			WorkloadName:      services.DerefString(s.WorkloadName),
			WorkloadKind:      services.DerefK8sResourceKind(s.WorkloadKind),
			WorkloadNamespace: services.DerefString(s.WorkloadNamespace),
			WorkloadLanguage:  services.DerefProgrammingLanguage(s.WorkloadLanguage),
		}
	}
	return result
}

func sourcesScopeCRDToModel(scopes []k8sconsts.SourcesScope) []*model.SourcesScope {
	if len(scopes) == 0 {
		return nil
	}
	result := make([]*model.SourcesScope, len(scopes))
	for i, s := range scopes {
		result[i] = &model.SourcesScope{
			WorkloadName:      services.StringPtrIfNotEmpty(s.WorkloadName),
			WorkloadKind:      services.K8sResourceKindPtrIfNotEmpty(s.WorkloadKind),
			WorkloadNamespace: services.StringPtrIfNotEmpty(s.WorkloadNamespace),
			WorkloadLanguage:  services.ProgrammingLanguagePtrIfNotEmpty(s.WorkloadLanguage),
		}
	}
	return result
}

func headSamplingOperationMatcherInputToCRD(input *model.HeadSamplingOperationMatcherInput) *commonapisampling.HeadSamplingOperationMatcher {
	if input == nil {
		return nil
	}
	matcher := &commonapisampling.HeadSamplingOperationMatcher{}
	if input.HTTPServer != nil {
		matcher.HttpServer = &commonapisampling.HeadSamplingHttpServerOperationMatcher{
			Route:       services.DerefString(input.HTTPServer.Route),
			RoutePrefix: services.DerefString(input.HTTPServer.RoutePrefix),
			Method:      services.DerefString(input.HTTPServer.Method),
		}
	}
	if input.HTTPClient != nil {
		matcher.HttpClient = &commonapisampling.HeadSamplingHttpClientOperationMatcher{
			ServerAddress:       services.DerefString(input.HTTPClient.ServerAddress),
			TemplatedPath:       services.DerefString(input.HTTPClient.TemplatedPath),
			TemplatedPathPrefix: services.DerefString(input.HTTPClient.TemplatedPathPrefix),
			Method:              services.DerefString(input.HTTPClient.Method),
		}
	}
	return matcher
}

func headSamplingOperationMatcherCRDToModel(matcher *commonapisampling.HeadSamplingOperationMatcher) *model.HeadSamplingOperationMatcher {
	if matcher == nil {
		return nil
	}
	result := &model.HeadSamplingOperationMatcher{}
	if matcher.HttpServer != nil {
		result.HTTPServer = &model.HeadSamplingHTTPServerMatcher{
			Route:       services.StringPtrIfNotEmpty(matcher.HttpServer.Route),
			RoutePrefix: services.StringPtrIfNotEmpty(matcher.HttpServer.RoutePrefix),
			Method:      services.StringPtrIfNotEmpty(matcher.HttpServer.Method),
		}
	}
	if matcher.HttpClient != nil {
		result.HTTPClient = &model.HeadSamplingHTTPClientMatcher{
			ServerAddress:       services.StringPtrIfNotEmpty(matcher.HttpClient.ServerAddress),
			TemplatedPath:       services.StringPtrIfNotEmpty(matcher.HttpClient.TemplatedPath),
			TemplatedPathPrefix: services.StringPtrIfNotEmpty(matcher.HttpClient.TemplatedPathPrefix),
			Method:              services.StringPtrIfNotEmpty(matcher.HttpClient.Method),
		}
	}
	return result
}

func tailSamplingOperationMatcherInputToCRD(input *model.TailSamplingOperationMatcherInput) *commonapisampling.TailSamplingOperationMatcher {
	if input == nil {
		return nil
	}
	matcher := &commonapisampling.TailSamplingOperationMatcher{}
	if input.HTTPServer != nil {
		matcher.HttpServer = &commonapisampling.TailSamplingHttpServerOperationMatcher{
			Route:       services.DerefString(input.HTTPServer.Route),
			RoutePrefix: services.DerefString(input.HTTPServer.RoutePrefix),
			Method:      services.DerefString(input.HTTPServer.Method),
		}
	}
	if input.KafkaConsumer != nil {
		matcher.KafkaConsumer = &commonapisampling.TailSamplingKafkaOperationMatcher{
			KafkaTopic: services.DerefString(input.KafkaConsumer.KafkaTopic),
		}
	}
	if input.KafkaProducer != nil {
		matcher.KafkaProducer = &commonapisampling.TailSamplingKafkaOperationMatcher{
			KafkaTopic: services.DerefString(input.KafkaProducer.KafkaTopic),
		}
	}
	return matcher
}

func tailSamplingOperationMatcherCRDToModel(matcher *commonapisampling.TailSamplingOperationMatcher) *model.TailSamplingOperationMatcher {
	if matcher == nil {
		return nil
	}
	result := &model.TailSamplingOperationMatcher{}
	if matcher.HttpServer != nil {
		result.HTTPServer = &model.TailSamplingHTTPServerMatcher{
			Route:       services.StringPtrIfNotEmpty(matcher.HttpServer.Route),
			RoutePrefix: services.StringPtrIfNotEmpty(matcher.HttpServer.RoutePrefix),
			Method:      services.StringPtrIfNotEmpty(matcher.HttpServer.Method),
		}
	}
	if matcher.KafkaConsumer != nil {
		result.KafkaConsumer = &model.TailSamplingKafkaMatcher{
			KafkaTopic: services.StringPtrIfNotEmpty(matcher.KafkaConsumer.KafkaTopic),
		}
	}
	if matcher.KafkaProducer != nil {
		result.KafkaProducer = &model.TailSamplingKafkaMatcher{
			KafkaTopic: services.StringPtrIfNotEmpty(matcher.KafkaProducer.KafkaTopic),
		}
	}
	return result
}
