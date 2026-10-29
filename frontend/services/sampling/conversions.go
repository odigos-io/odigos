package sampling

import (
	"fmt"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/services"
)

// ---- Input → CRD converters ----

func noisyOperationFromInput(input model.NoisyOperationRuleInput) v1alpha1.NoisyOperation {
	return v1alpha1.NoisyOperation{
		Name:             services.DerefString(input.Name.Value()),
		Disabled:         services.DerefBool(input.Disabled.Value()),
		SourceScopes:     services.SourcesScopesInputToCRD(input.SourceScopes.Value()),
		Operation:        headSamplingOperationMatcherInputToCRD(input.Operation.Value()),
		PercentageAtMost: input.PercentageAtMost.Value(),
		Notes:            services.DerefString(input.Notes.Value()),
	}
}

func highlyRelevantOperationFromInput(input model.HighlyRelevantOperationRuleInput) v1alpha1.HighlyRelevantOperation {
	return v1alpha1.HighlyRelevantOperation{
		Name:              services.DerefString(input.Name.Value()),
		Disabled:          services.DerefBool(input.Disabled.Value()),
		SourceScopes:      services.SourcesScopesInputToCRD(input.SourceScopes.Value()),
		Error:             services.DerefBool(input.Error.Value()),
		DurationAtLeastMs: input.DurationAtLeastMs.Value(),
		Operation:         tailSamplingOperationMatcherInputToCRD(input.Operation.Value()),
		PercentageAtLeast: input.PercentageAtLeast.Value(),
		Notes:             services.DerefString(input.Notes.Value()),
	}
}

func costReductionRuleFromInput(input model.CostReductionRuleInput) v1alpha1.CostReductionRule {
	return v1alpha1.CostReductionRule{
		Name:             services.DerefString(input.Name.Value()),
		Disabled:         services.DerefBool(input.Disabled.Value()),
		SourceScopes:     services.SourcesScopesInputToCRD(input.SourceScopes.Value()),
		Operation:        tailSamplingOperationMatcherInputToCRD(input.Operation.Value()),
		PercentageAtMost: input.PercentageAtMost,
		Notes:            services.DerefString(input.Notes.Value()),
	}
}

// Updates are partial: a field omitted from the GraphQL input keeps its current value, while an
// explicit null clears it. Nil SourceScopes/Operation match all sources/operations, so null is how
// a client widens a scoped rule, and omission must not silently widen one.

func mergeNoisyOperationUpdate(existing v1alpha1.NoisyOperation, input model.NoisyOperationRuleInput) v1alpha1.NoisyOperation {
	rule := noisyOperationFromInput(input)
	if !input.Name.IsSet() {
		rule.Name = existing.Name
	}
	if !input.Disabled.IsSet() {
		rule.Disabled = existing.Disabled
	}
	if !input.SourceScopes.IsSet() {
		rule.SourceScopes = existing.SourceScopes
	}
	if !input.Operation.IsSet() {
		rule.Operation = existing.Operation
	}
	if !input.PercentageAtMost.IsSet() {
		rule.PercentageAtMost = existing.PercentageAtMost
	}
	if !input.Notes.IsSet() {
		rule.Notes = existing.Notes
	}
	// the GraphQL input has no surge settings: an update keeps the rule's surge.
	rule.Surge = existing.Surge
	return rule
}

func mergeHighlyRelevantOperationUpdate(existing v1alpha1.HighlyRelevantOperation, input model.HighlyRelevantOperationRuleInput) v1alpha1.HighlyRelevantOperation {
	rule := highlyRelevantOperationFromInput(input)
	if !input.Name.IsSet() {
		rule.Name = existing.Name
	}
	if !input.Disabled.IsSet() {
		rule.Disabled = existing.Disabled
	}
	if !input.SourceScopes.IsSet() {
		rule.SourceScopes = existing.SourceScopes
	}
	if !input.Error.IsSet() {
		rule.Error = existing.Error
	}
	if !input.DurationAtLeastMs.IsSet() {
		rule.DurationAtLeastMs = existing.DurationAtLeastMs
	}
	if !input.Operation.IsSet() {
		rule.Operation = existing.Operation
	}
	if !input.PercentageAtLeast.IsSet() {
		rule.PercentageAtLeast = existing.PercentageAtLeast
	}
	if !input.Notes.IsSet() {
		rule.Notes = existing.Notes
	}
	return rule
}

func mergeCostReductionRuleUpdate(existing v1alpha1.CostReductionRule, input model.CostReductionRuleInput) v1alpha1.CostReductionRule {
	rule := costReductionRuleFromInput(input)
	if !input.Name.IsSet() {
		rule.Name = existing.Name
	}
	if !input.Disabled.IsSet() {
		rule.Disabled = existing.Disabled
	}
	if !input.SourceScopes.IsSet() {
		rule.SourceScopes = existing.SourceScopes
	}
	if !input.Operation.IsSet() {
		rule.Operation = existing.Operation
	}
	if !input.Notes.IsSet() {
		rule.Notes = existing.Notes
	}
	return rule
}

// ---- CRD → Model converters ----

func convertNoisyOperationToModel(rule *v1alpha1.NoisyOperation) *model.NoisyOperationRule {
	return &model.NoisyOperationRule{
		RuleID:           v1alpha1.ComputeNoisyOperationHash(rule),
		Name:             services.StringPtrIfNotEmpty(rule.Name),
		Disabled:         rule.Disabled,
		SourceScopes:     services.SourcesScopesCRDToModel(rule.SourceScopes),
		Operation:        headSamplingOperationMatcherCRDToModel(rule.Operation),
		PercentageAtMost: rule.PercentageAtMost,
		Notes:            services.StringPtrIfNotEmpty(rule.Notes),
		Surge:            convertSurgeToModel(rule.Surge),
	}
}

func convertHighlyRelevantOperationToModel(rule *v1alpha1.HighlyRelevantOperation) *model.HighlyRelevantOperationRule {
	return &model.HighlyRelevantOperationRule{
		RuleID:            v1alpha1.ComputeHighlyRelevantOperationHash(rule),
		Name:              services.StringPtrIfNotEmpty(rule.Name),
		Disabled:          rule.Disabled,
		SourceScopes:      services.SourcesScopesCRDToModel(rule.SourceScopes),
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
		SourceScopes:     services.SourcesScopesCRDToModel(rule.SourceScopes),
		Operation:        tailSamplingOperationMatcherCRDToModel(rule.Operation),
		PercentageAtMost: rule.PercentageAtMost,
		Notes:            services.StringPtrIfNotEmpty(rule.Notes),
	}
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
			QueryParams: headSamplingQueryParamsInputToCRD(input.HTTPServer.QueryParams),
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
			QueryParams: headSamplingQueryParamsCRDToModel(matcher.HttpServer.QueryParams),
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

func headSamplingQueryParamsInputToCRD(in []*model.HeadSamplingQueryParamMatcherInput) []commonapisampling.QueryParamMatcher {
	if len(in) == 0 {
		return nil
	}
	out := make([]commonapisampling.QueryParamMatcher, 0, len(in))
	for _, param := range in {
		if param == nil {
			continue
		}
		out = append(out, commonapisampling.QueryParamMatcher{
			Name:       param.Name,
			ValueExact: param.ValueExact,
		})
	}
	return out
}

func headSamplingQueryParamsCRDToModel(in []commonapisampling.QueryParamMatcher) []*model.HeadSamplingQueryParamMatcher {
	if len(in) == 0 {
		return nil
	}
	out := make([]*model.HeadSamplingQueryParamMatcher, 0, len(in))
	for i := range in {
		out = append(out, &model.HeadSamplingQueryParamMatcher{
			Name:       in[i].Name,
			ValueExact: in[i].ValueExact,
		})
	}
	return out
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

// trace surges are evaluated per service, including the services the rule's traces reach.
const (
	surgeEvaluationPerService = "per_service"
)

func convertSurgeToModel(surge *v1alpha1.TraceSurgeSettings) *model.HeadSamplingSurge {
	if surge == nil {
		return nil
	}
	return &model.HeadSamplingSurge{
		Version:             surge.Version,
		IncludeDownstream:   true,
		Evaluation:          surgeEvaluationPerService,
		Metric:              string(surge.Metric),
		Threshold:           surge.Threshold,
		SustainedSeconds:    surge.SustainedSeconds,
		MinimumRequests:     surge.MinimumRequests,
		BoostPercent:        surge.BoostPercent,
		RecoveryThreshold:   surge.RecoveryThreshold,
		RecoverySeconds:     surge.RecoverySeconds,
		MinimumBoostSeconds: surge.MinimumBoostSeconds,
	}
}

func surgeFromInput(input *model.HeadSamplingSurgeInput, version string) *v1alpha1.TraceSurgeSettings {
	if input == nil {
		return nil
	}
	return &v1alpha1.TraceSurgeSettings{
		Version:             version,
		Metric:              v1alpha1.TraceSurgeMetric(input.Metric),
		Threshold:           input.Threshold,
		SustainedSeconds:    input.SustainedSeconds,
		MinimumRequests:     input.MinimumRequests,
		BoostPercent:        input.BoostPercent,
		RecoveryThreshold:   input.RecoveryThreshold,
		RecoverySeconds:     input.RecoverySeconds,
		MinimumBoostSeconds: input.MinimumBoostSeconds,
	}
}

// validateSurge mirrors the UI's validation of a surge for a rule whose normal percentage is normalPercent.
func validateSurge(input *model.HeadSamplingSurgeInput, normalPercent float64) error {
	if !input.IncludeDownstream || input.Evaluation != surgeEvaluationPerService {
		return fmt.Errorf("a trace surge evaluates every service separately, including downstream services")
	}
	switch v1alpha1.TraceSurgeMetric(input.Metric) {
	case v1alpha1.TraceSurgeMetricErrorRate:
		if input.Threshold > 100 {
			return fmt.Errorf("an error rate threshold is at most 100%%")
		}
	case v1alpha1.TraceSurgeMetricLatencyP95, v1alpha1.TraceSurgeMetricRequestRate:
	default:
		return fmt.Errorf("unknown trace surge metric %q", input.Metric)
	}
	switch {
	case normalPercent <= 0 || normalPercent > 100:
		return fmt.Errorf("a rule with a trace surge needs a normal percentage above 0%% and at most 100%%")
	case input.Threshold <= 0:
		return fmt.Errorf("the trace surge threshold must be above 0")
	case input.RecoveryThreshold < 0 || input.RecoveryThreshold > input.Threshold:
		return fmt.Errorf("the recovery threshold must be at least 0 and at most the threshold")
	case input.BoostPercent <= normalPercent || input.BoostPercent > 100:
		return fmt.Errorf("the surge percentage must be above the normal percentage and at most 100%%")
	case input.SustainedSeconds <= 0 || input.RecoverySeconds <= 0 || input.MinimumBoostSeconds <= 0 || input.MinimumRequests <= 0:
		return fmt.Errorf("the trace surge durations and minimum requests must be positive")
	}
	return nil
}
