package sampling

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
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

func sourcesScopeInputToCRD(in *model.SourcesScopesInput) *k8sconsts.SourcesScopes {
	if in == nil {
		return nil
	}
	out := &k8sconsts.SourcesScopes{}
	if len(in.Sources) > 0 {
		out.Sources = make([]k8sconsts.PodWorkload, 0, len(in.Sources))
		for _, s := range in.Sources {
			if s == nil {
				continue
			}
			out.Sources = append(out.Sources, k8sconsts.PodWorkload{
				Name:      s.Name,
				Namespace: s.Namespace,
				Kind:      k8sconsts.WorkloadKind(s.Kind),
			})
		}
	}
	if len(in.Namespaces) > 0 {
		out.Namespaces = append([]string(nil), in.Namespaces...)
	}
	if len(in.Languages) > 0 {
		out.Languages = make([]common.ProgrammingLanguage, 0, len(in.Languages))
		for _, l := range in.Languages {
			out.Languages = append(out.Languages, common.ProgrammingLanguage(l))
		}
	}
	return out
}

func sourcesScopeCRDToModel(in *k8sconsts.SourcesScopes) *model.SourcesScopes {
	if in == nil {
		return nil
	}
	out := &model.SourcesScopes{}
	if len(in.Sources) > 0 {
		out.Sources = make([]*model.K8sWorkloadID, 0, len(in.Sources))
		for _, s := range in.Sources {
			out.Sources = append(out.Sources, &model.K8sWorkloadID{
				Name:      s.Name,
				Namespace: s.Namespace,
				Kind:      model.K8sResourceKind(s.Kind),
			})
		}
	}
	if len(in.Namespaces) > 0 {
		out.Namespaces = append([]string(nil), in.Namespaces...)
	}
	if len(in.Languages) > 0 {
		out.Languages = make([]model.SamplingWorkloadLanguage, 0, len(in.Languages))
		for _, l := range in.Languages {
			out.Languages = append(out.Languages, model.SamplingWorkloadLanguage(l))
		}
	}
	return out
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
	if input.GrpcServer != nil {
		matcher.GrpcServer = &commonapisampling.HeadSamplingGrpcServerOperationMatcher{
			Method:  services.DerefString(input.GrpcServer.Method),
			Service: services.DerefString(input.GrpcServer.Service),
		}
	}
	if input.GrpcClient != nil {
		matcher.GrpcClient = &commonapisampling.HeadSamplingGrpcClientOperationMatcher{
			Method:        services.DerefString(input.GrpcClient.Method),
			Service:       services.DerefString(input.GrpcClient.Service),
			ServerAddress: services.DerefString(input.GrpcClient.ServerAddress),
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
	if matcher.GrpcServer != nil {
		result.GrpcServer = &model.HeadSamplingGrpcServerMatcher{
			Method:  services.StringPtrIfNotEmpty(matcher.GrpcServer.Method),
			Service: services.StringPtrIfNotEmpty(matcher.GrpcServer.Service),
		}
	}
	if matcher.GrpcClient != nil {
		result.GrpcClient = &model.HeadSamplingGrpcClientMatcher{
			Method:        services.StringPtrIfNotEmpty(matcher.GrpcClient.Method),
			Service:       services.StringPtrIfNotEmpty(matcher.GrpcClient.Service),
			ServerAddress: services.StringPtrIfNotEmpty(matcher.GrpcClient.ServerAddress),
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
