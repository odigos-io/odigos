package services

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func deriveTypeFromRule(rule *model.InstrumentationRule) model.InstrumentationRuleType {
	if rule.CodeAttributes != nil {
		if rule.CodeAttributes.Column != nil || rule.CodeAttributes.FilePath != nil || rule.CodeAttributes.Function != nil || rule.CodeAttributes.LineNumber != nil || rule.CodeAttributes.Namespace != nil || rule.CodeAttributes.Stacktrace != nil {
			return model.InstrumentationRuleTypeCodeAttributes
		}
	}

	if rule.HeadersCollection != nil {
		if rule.HeadersCollection.HeaderKeys != nil {
			return model.InstrumentationRuleTypeHeadersCollection
		}
	}

	if rule.PayloadCollection != nil {
		if rule.PayloadCollection.HTTPRequest != nil || rule.PayloadCollection.HTTPResponse != nil || rule.PayloadCollection.DbQuery != nil || rule.PayloadCollection.Messaging != nil {
			return model.InstrumentationRuleTypePayloadCollection
		}
	}

	if rule.CustomInstrumentations != nil {
		return model.InstrumentationRuleTypeCustomInstrumentation
	}

	return model.InstrumentationRuleTypeUnknownType
}

// GetInstrumentationRules fetches all instrumentation rules
func GetInstrumentationRules(ctx context.Context) ([]*model.InstrumentationRule, error) {
	ns := env.GetCurrentNamespace()

	instrumentationRules, err := kube.DefaultClient.OdigosClient.InstrumentationRules(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting instrumentation rules: %w", err)
	}

	var gqlRules []*model.InstrumentationRule
	for _, r := range instrumentationRules.Items {
		annotations := r.GetAnnotations()
		profileName := annotations[k8sconsts.OdigosProfileAnnotation]
		mutable := profileName == ""

		rule := &model.InstrumentationRule{
			RuleID:                   r.Name,
			RuleName:                 &r.Spec.RuleName,
			Notes:                    &r.Spec.Notes,
			Disabled:                 &r.Spec.Disabled,
			Mutable:                  mutable,
			ProfileName:              profileName,
			SourcesScopes:            convertSourcesScope(r.Spec.Scopes),
			InstrumentationLibraries: convertInstrumentationLibraries(r.Spec.InstrumentationLibraries),
			Conditions:               ConvertConditions(r.Status.Conditions),
			CodeAttributes:           (*model.CodeAttributes)(r.Spec.CodeAttributes),
			HeadersCollection:        convertHeadersCollection(r.Spec.HeadersCollection),
			PayloadCollection:        convertPayloadCollection(r.Spec.PayloadCollection),
			CustomInstrumentations:   convertCustomInstrumentations(r.Spec.CustomInstrumentations),
		}
		rule.Type = deriveTypeFromRule(rule)

		gqlRules = append(gqlRules, rule)
	}
	return gqlRules, nil
}

func GetInstrumentationRule(ctx context.Context, id string) (*model.InstrumentationRule, error) {
	ns := env.GetCurrentNamespace()

	r, err := kube.DefaultClient.OdigosClient.InstrumentationRules(ns).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		return nil, handleNotFoundError(err, id, "instrumentation rule")
	}

	annotations := r.GetAnnotations()
	profileName := annotations[k8sconsts.OdigosProfileAnnotation]
	mutable := profileName == ""

	rule := &model.InstrumentationRule{
		RuleID:                   r.Name,
		RuleName:                 &r.Spec.RuleName,
		Notes:                    &r.Spec.Notes,
		Disabled:                 &r.Spec.Disabled,
		Mutable:                  mutable,
		ProfileName:              profileName,
		SourcesScopes:            convertSourcesScope(r.Spec.Scopes),
		InstrumentationLibraries: convertInstrumentationLibraries(r.Spec.InstrumentationLibraries),
		CodeAttributes:           (*model.CodeAttributes)(r.Spec.CodeAttributes),
		HeadersCollection:        convertHeadersCollection(r.Spec.HeadersCollection),
		PayloadCollection:        convertPayloadCollection(r.Spec.PayloadCollection),
		CustomInstrumentations:   convertCustomInstrumentations(r.Spec.CustomInstrumentations),
	}
	rule.Type = deriveTypeFromRule(rule)

	return rule, nil
}

func getPayloadCollectionInput(input model.InstrumentationRuleInput) *instrumentationrules.PayloadCollection {
	if input.PayloadCollection == nil {
		return nil
	}

	payloadCollection := &instrumentationrules.PayloadCollection{}

	if in := input.PayloadCollection.HTTPRequest; in != nil {
		payloadCollection.HttpRequest = fromHTTPPayloadInput(in)
	}
	if in := input.PayloadCollection.HTTPResponse; in != nil {
		payloadCollection.HttpResponse = fromHTTPPayloadInput(in)
	}
	if in := input.PayloadCollection.DbQuery; in != nil {
		payloadCollection.DbQuery = &instrumentationrules.DbQueryPayloadCollection{
			MaxPayloadLength:    intToInt64Ptr(in.MaxPayloadLength),
			DropPartialPayloads: in.DropPartialPayloads,
		}
	}
	if in := input.PayloadCollection.Messaging; in != nil {
		payloadCollection.Messaging = &instrumentationrules.MessagingPayloadCollection{
			MaxPayloadLength:    intToInt64Ptr(in.MaxPayloadLength),
			DropPartialPayloads: in.DropPartialPayloads,
		}
	}

	return payloadCollection
}

// fromHTTPPayloadInput copies the advanced HTTP payload options (mime types,
// max length, drop-partial) from the GraphQL input into the CRD config. The
// GraphQL layer uses `[]*string` / `*int`, the CRD uses `*[]string` / `*int64`.
func fromHTTPPayloadInput(in *model.HTTPPayloadCollectionInput) *instrumentationrules.HttpPayloadCollection {
	cfg := &instrumentationrules.HttpPayloadCollection{
		MaxPayloadLength:    intToInt64Ptr(in.MaxPayloadLength),
		DropPartialPayloads: in.DropPartialPayloads,
	}
	if in.MimeTypes != nil {
		mimeTypes := make([]string, 0, len(in.MimeTypes))
		for _, m := range in.MimeTypes {
			if m != nil {
				mimeTypes = append(mimeTypes, *m)
			}
		}
		cfg.MimeTypes = &mimeTypes
	}
	return cfg
}

func intToInt64Ptr(v *int) *int64 {
	if v == nil {
		return nil
	}
	i := int64(*v)
	return &i
}

func int64ToIntPtr(v *int64) *int {
	if v == nil {
		return nil
	}
	i := int(*v)
	return &i
}

func getHeadersCollectionInput(input model.InstrumentationRuleInput) *instrumentationrules.HttpHeadersCollection {
	if input.HeadersCollection == nil {
		return nil
	}

	headersCollection := &instrumentationrules.HttpHeadersCollection{}

	if input.HeadersCollection.HeaderKeys != nil {
		headersCollection.HeaderKeys = make([]string, 0, len(input.HeadersCollection.HeaderKeys))
		for _, key := range input.HeadersCollection.HeaderKeys {
			headersCollection.HeaderKeys = append(headersCollection.HeaderKeys, *key)
		}
	}

	return headersCollection
}

func getCodeAttributesInput(input model.InstrumentationRuleInput) *instrumentationrules.CodeAttributes {
	if input.CodeAttributes == nil {
		return nil
	}

	codeAttributes := &instrumentationrules.CodeAttributes{}

	if input.CodeAttributes.Column != nil {
		codeAttributes.Column = input.CodeAttributes.Column
	}
	if input.CodeAttributes.FilePath != nil {
		codeAttributes.FilePath = input.CodeAttributes.FilePath
	}
	if input.CodeAttributes.Function != nil {
		codeAttributes.Function = input.CodeAttributes.Function
	}
	if input.CodeAttributes.LineNumber != nil {
		codeAttributes.LineNumber = input.CodeAttributes.LineNumber
	}
	if input.CodeAttributes.Namespace != nil {
		codeAttributes.Namespace = input.CodeAttributes.Namespace
	}
	if input.CodeAttributes.Stacktrace != nil {
		codeAttributes.Stacktrace = input.CodeAttributes.Stacktrace
	}

	return codeAttributes
}

func getCustomInstrumentationsInput(input model.InstrumentationRuleInput) *instrumentationrules.CustomInstrumentations {
	if input.CustomInstrumentations == nil {
		return nil
	}
	customInstrumentations := &instrumentationrules.CustomInstrumentations{}
	// Iterate Java custom probes and verify input
	if input.CustomInstrumentations.Java != nil {
		customInstrumentations.Java = make([]instrumentationrules.JavaCustomProbe, 0, len(input.CustomInstrumentations.Java))
		for _, probe := range input.CustomInstrumentations.Java {
			apiProbe := instrumentationrules.JavaCustomProbe{}
			if probe.ClassName != nil {
				apiProbe.ClassName = *probe.ClassName
			} else {
				apiProbe.ClassName = ""
			}
			if probe.MethodName != nil {
				apiProbe.MethodName = *probe.MethodName
			} else {
				apiProbe.MethodName = ""
			}
			customInstrumentations.Java = append(customInstrumentations.Java, apiProbe)
		}
	}

	if input.CustomInstrumentations.Golang != nil {
		customInstrumentations.Golang = make([]instrumentationrules.GolangCustomProbe, 0, len(input.CustomInstrumentations.Golang))
		for _, probe := range input.CustomInstrumentations.Golang {
			apiProbe := instrumentationrules.GolangCustomProbe{}
			if probe.PackageName != nil {
				apiProbe.PackageName = *probe.PackageName
			} else {
				apiProbe.PackageName = ""
			}
			if probe.FunctionName != nil {
				apiProbe.FunctionName = *probe.FunctionName
			} else {
				apiProbe.FunctionName = ""
			}
			if probe.ReceiverName != nil {
				apiProbe.ReceiverName = *probe.ReceiverName
			} else {
				apiProbe.ReceiverName = ""
			}
			if probe.ReceiverMethodName != nil {
				apiProbe.ReceiverMethodName = *probe.ReceiverMethodName
			} else {
				apiProbe.ReceiverMethodName = ""
			}
			customInstrumentations.Golang = append(customInstrumentations.Golang, apiProbe)
		}
	}

	// Remove duplicate Golang probes
	uniqueGolangProbes := make([]instrumentationrules.GolangCustomProbe, 0, len(customInstrumentations.Golang))
	uniqGoProbes := make(map[instrumentationrules.GolangCustomProbe]struct{})
	for _, probe := range customInstrumentations.Golang {
		uniqGoProbes[probe] = struct{}{}
	}
	for probe := range uniqGoProbes {
		uniqueGolangProbes = append(uniqueGolangProbes, probe)
	}
	customInstrumentations.Golang = uniqueGolangProbes

	// Remove duplicate Java probes
	uniqueJavaProbes := make([]instrumentationrules.JavaCustomProbe, 0, len(customInstrumentations.Java))
	javaSeen := make(map[instrumentationrules.JavaCustomProbe]struct{})
	for _, probe := range customInstrumentations.Java {
		javaSeen[probe] = struct{}{}
	}
	for probe := range javaSeen {
		uniqueJavaProbes = append(uniqueJavaProbes, probe)
	}
	customInstrumentations.Java = uniqueJavaProbes
	return customInstrumentations
}

func UpdateInstrumentationRule(ctx context.Context, id string, input model.InstrumentationRuleInput) (*model.InstrumentationRule, error) {
	ns := env.GetCurrentNamespace()

	// Retrieve existing rule
	existingRule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(ns).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		return nil, handleNotFoundError(err, id, "instrumentation rule")
	}
	// Update the existing rule's specification
	existingRule.Spec.RuleName = *input.RuleName
	existingRule.Spec.Notes = *input.Notes
	existingRule.Spec.Disabled = *input.Disabled

	if input.SourcesScopes != nil {
		existingRule.Spec.Scopes = convertSourcesScopeInput(input.SourcesScopes)
	} else {
		existingRule.Spec.Scopes = nil
	}

	if input.InstrumentationLibraries != nil {
		convertedLibraries := make([]v1alpha1.InstrumentationLibraryGlobalId, len(input.InstrumentationLibraries))
		for i, lib := range input.InstrumentationLibraries {
			convertedLibraries[i] = v1alpha1.InstrumentationLibraryGlobalId{
				Name:     lib.Name,
				SpanKind: common.SpanKind(*lib.SpanKind),
				Language: common.ProgrammingLanguage(*lib.Language),
			}
		}
		existingRule.Spec.InstrumentationLibraries = &convertedLibraries
	} else {
		existingRule.Spec.InstrumentationLibraries = nil
	}

	if input.PayloadCollection != nil {
		existingRule.Spec.PayloadCollection = getPayloadCollectionInput(input)
	} else {
		existingRule.Spec.PayloadCollection = nil
	}

	if input.CodeAttributes != nil {
		existingRule.Spec.CodeAttributes = getCodeAttributesInput(input)
	} else {
		existingRule.Spec.CodeAttributes = nil
	}

	if input.HeadersCollection != nil {
		existingRule.Spec.HeadersCollection = getHeadersCollectionInput(input)
	} else {
		existingRule.Spec.HeadersCollection = nil
	}

	if input.CustomInstrumentations != nil {
		existingRule.Spec.CustomInstrumentations = getCustomInstrumentationsInput(input)
	} else {
		existingRule.Spec.CustomInstrumentations = nil
	}
	// Update rule in Kubernetes
	updatedRule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(ns).Update(ctx, existingRule, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error updating instrumentation rule: %w", err)
	}

	annotations := updatedRule.GetAnnotations()
	profileName := annotations[k8sconsts.OdigosProfileAnnotation]

	rule := model.InstrumentationRule{
		RuleID:                   updatedRule.Name,
		RuleName:                 &updatedRule.Spec.RuleName,
		Notes:                    &updatedRule.Spec.Notes,
		Disabled:                 &updatedRule.Spec.Disabled,
		Mutable:                  profileName == "",
		ProfileName:              profileName,
		SourcesScopes:            convertSourcesScope(updatedRule.Spec.Scopes),
		InstrumentationLibraries: convertInstrumentationLibraries(updatedRule.Spec.InstrumentationLibraries),
		CodeAttributes:           (*model.CodeAttributes)(updatedRule.Spec.CodeAttributes),
		HeadersCollection:        convertHeadersCollection(updatedRule.Spec.HeadersCollection),
		PayloadCollection:        convertPayloadCollection(updatedRule.Spec.PayloadCollection),
		CustomInstrumentations:   convertCustomInstrumentations(updatedRule.Spec.CustomInstrumentations),
	}
	rule.Type = deriveTypeFromRule(&rule)
	return &rule, nil
}

func DeleteInstrumentationRule(ctx context.Context, id string) (bool, error) {
	ns := env.GetCurrentNamespace()

	err := kube.DefaultClient.OdigosClient.InstrumentationRules(ns).Delete(ctx, id, metav1.DeleteOptions{})
	if err != nil {
		return false, handleNotFoundError(err, id, "instrumentation rule")
	}

	return true, nil
}

func CreateInstrumentationRule(ctx context.Context, input model.InstrumentationRuleInput) (*model.InstrumentationRule, error) {
	ns := env.GetCurrentNamespace()

	ruleName := *input.RuleName
	notes := *input.Notes
	disabled := *input.Disabled

	var sourcesScopes *k8sconsts.SourcesScopes
	if input.SourcesScopes != nil {
		sourcesScopes = convertSourcesScopeInput(input.SourcesScopes)
	}

	var instrumentationLibraries *[]v1alpha1.InstrumentationLibraryGlobalId
	if input.InstrumentationLibraries != nil {
		convertedLibraries := make([]v1alpha1.InstrumentationLibraryGlobalId, len(input.InstrumentationLibraries))
		for i, lib := range input.InstrumentationLibraries {
			convertedLibraries[i] = v1alpha1.InstrumentationLibraryGlobalId{
				Name:     lib.Name,
				SpanKind: common.SpanKind(*lib.SpanKind),
				Language: common.ProgrammingLanguage(*lib.Language),
			}
		}
		instrumentationLibraries = &convertedLibraries
	}

	// Define the new rule spec based on the input
	newRule := &v1alpha1.InstrumentationRule{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "ui-instrumentation-rule-",
		},
		Spec: v1alpha1.InstrumentationRuleSpec{
			RuleName:                 ruleName,
			Notes:                    notes,
			Disabled:                 disabled,
			Scopes:                   sourcesScopes,
			InstrumentationLibraries: instrumentationLibraries,
			CodeAttributes:           getCodeAttributesInput(input),
			HeadersCollection:        getHeadersCollectionInput(input),
			PayloadCollection:        getPayloadCollectionInput(input),
			CustomInstrumentations:   getCustomInstrumentationsInput(input),
		},
	}
	// Create the rule in Kubernetes
	createdRule, err := CreateResourceWithGenerateName(ctx, func() (*v1alpha1.InstrumentationRule, error) {
		return kube.DefaultClient.OdigosClient.InstrumentationRules(ns).Create(ctx, newRule, metav1.CreateOptions{})
	})
	if err != nil {
		return nil, fmt.Errorf("error creating instrumentation rule: %w", err)
	}

	// Convert to GraphQL model and return
	rule := model.InstrumentationRule{
		RuleID:                   createdRule.Name,
		RuleName:                 &createdRule.Spec.RuleName,
		Notes:                    &createdRule.Spec.Notes,
		Disabled:                 &createdRule.Spec.Disabled,
		Mutable:                  true, // New rules are always mutable
		ProfileName:              "",   // New rules are not associated with a profile
		SourcesScopes:            convertSourcesScope(createdRule.Spec.Scopes),
		InstrumentationLibraries: convertInstrumentationLibraries(createdRule.Spec.InstrumentationLibraries),
		CodeAttributes:           (*model.CodeAttributes)(createdRule.Spec.CodeAttributes),
		HeadersCollection:        convertHeadersCollection(createdRule.Spec.HeadersCollection),
		PayloadCollection:        convertPayloadCollection(createdRule.Spec.PayloadCollection),
		CustomInstrumentations:   convertCustomInstrumentations(createdRule.Spec.CustomInstrumentations),
	}
	rule.Type = deriveTypeFromRule(&rule)

	return &rule, nil
}

func handleNotFoundError(err error, id string, entity string) error {
	if apierrors.IsNotFound(err) {
		return fmt.Errorf("%s with ID %s not found", entity, id)
	}
	return fmt.Errorf("error getting %s: %w", entity, err)
}

// convertSourcesScopeInput folds the row-per-criterion GraphQL shape into the
// single-object tri-list CRD shape. Each input row contributes to at most one
// of Sources/Namespaces (workload identity vs. namespace-only) plus optionally
// Languages. The CRD model has no equivalent for the legacy `containerName`
// field, so it is dropped on conversion.
func convertSourcesScopeInput(scopes []*model.InstrumentationRuleSourcesScopeInput) *k8sconsts.SourcesScopes {
	if len(scopes) == 0 {
		return nil
	}
	out := &k8sconsts.SourcesScopes{}
	for _, scope := range scopes {
		if scope == nil {
			continue
		}
		name := DerefString(scope.WorkloadName)
		kind := DerefK8sResourceKind(scope.WorkloadKind)
		namespace := DerefString(scope.WorkloadNamespace)
		language := DerefSamplingWorkloadLanguage(scope.WorkloadLanguage)

		switch {
		case name != "" || kind != "":
			out.Sources = append(out.Sources, k8sconsts.PodWorkload{
				Name:      name,
				Namespace: namespace,
				Kind:      k8sconsts.WorkloadKind(kind),
			})
		case namespace != "":
			out.Namespaces = append(out.Namespaces, namespace)
		}
		if language != "" {
			out.Languages = append(out.Languages, language)
		}
	}
	if len(out.Sources) == 0 && len(out.Namespaces) == 0 && len(out.Languages) == 0 {
		return nil
	}
	return out
}

// convertSourcesScope unfolds the single-object tri-list CRD shape back into the
// row-per-criterion GraphQL shape: one row per Source, one per Namespace, one
// per Language. This is the inverse of convertSourcesScopeInput for the common
// case of single-dimension rows.
func convertSourcesScope(scopes *k8sconsts.SourcesScopes) []*model.InstrumentationRuleSourcesScope {
	if scopes == nil {
		return nil
	}
	var gqlSourcesScope []*model.InstrumentationRuleSourcesScope
	for _, src := range scopes.Sources {
		row := &model.InstrumentationRuleSourcesScope{
			WorkloadName:      StringPtrIfNotEmpty(src.Name),
			WorkloadKind:      K8sResourceKindPtrIfNotEmpty(string(src.Kind)),
			WorkloadNamespace: StringPtrIfNotEmpty(src.Namespace),
		}
		gqlSourcesScope = append(gqlSourcesScope, row)
	}
	for _, ns := range scopes.Namespaces {
		ns := ns
		gqlSourcesScope = append(gqlSourcesScope, &model.InstrumentationRuleSourcesScope{
			WorkloadNamespace: &ns,
		})
	}
	for _, lang := range scopes.Languages {
		l := model.SamplingWorkloadLanguage(lang)
		gqlSourcesScope = append(gqlSourcesScope, &model.InstrumentationRuleSourcesScope{
			WorkloadLanguage: &l,
		})
	}
	return gqlSourcesScope
}

// Converts InstrumentationLibraries to GraphQL-compatible format
func convertInstrumentationLibraries(libraries *[]v1alpha1.InstrumentationLibraryGlobalId) []*model.InstrumentationLibraryGlobalID {
	var gqlLibraries []*model.InstrumentationLibraryGlobalID
	if libraries != nil {
		for _, lib := range *libraries {
			spanKind := model.SpanKind(lib.SpanKind)
			language := model.ProgrammingLanguage(lib.Language)
			gqlLibraries = append(gqlLibraries, &model.InstrumentationLibraryGlobalID{
				Name:     lib.Name,
				SpanKind: &spanKind,
				Language: &language,
			})
		}
	}
	return gqlLibraries
}

func convertHeadersCollection(headers *instrumentationrules.HttpHeadersCollection) *model.HeadersCollection {
	if headers == nil {
		return nil
	}

	headerKeys := make([]*string, len(headers.HeaderKeys))
	for i := range headers.HeaderKeys {
		headerKeys[i] = &headers.HeaderKeys[i]
	}

	return &model.HeadersCollection{
		HeaderKeys: headerKeys,
	}
}

// Converts PayloadCollection to GraphQL-compatible format
func convertPayloadCollection(payload *instrumentationrules.PayloadCollection) *model.PayloadCollection {
	if payload == nil {
		return nil
	}

	return &model.PayloadCollection{
		HTTPRequest:  toHTTPPayload(payload.HttpRequest),
		HTTPResponse: toHTTPPayload(payload.HttpResponse),
		DbQuery:      toDbQueryPayload(payload.DbQuery),
		Messaging:    toMessagingPayload(payload.Messaging),
	}
}

// Helpers to map the stored payload configs back to the GraphQL model, carrying
// the advanced options (mime types / max length / drop-partial) so they survive
// a round-trip and render in the UI.
func toHTTPPayload(payload *instrumentationrules.HttpPayloadCollection) *model.HTTPPayloadCollection {
	if payload == nil {
		return nil
	}
	out := &model.HTTPPayloadCollection{
		MaxPayloadLength:    int64ToIntPtr(payload.MaxPayloadLength),
		DropPartialPayloads: payload.DropPartialPayloads,
	}
	if payload.MimeTypes != nil {
		mimeTypes := make([]*string, 0, len(*payload.MimeTypes))
		for i := range *payload.MimeTypes {
			v := (*payload.MimeTypes)[i]
			mimeTypes = append(mimeTypes, &v)
		}
		out.MimeTypes = mimeTypes
	}
	return out
}

func toDbQueryPayload(payload *instrumentationrules.DbQueryPayloadCollection) *model.DbQueryPayloadCollection {
	if payload == nil {
		return nil
	}
	return &model.DbQueryPayloadCollection{
		MaxPayloadLength:    int64ToIntPtr(payload.MaxPayloadLength),
		DropPartialPayloads: payload.DropPartialPayloads,
	}
}

func toMessagingPayload(payload *instrumentationrules.MessagingPayloadCollection) *model.MessagingPayloadCollection {
	if payload == nil {
		return nil
	}
	return &model.MessagingPayloadCollection{
		MaxPayloadLength:    int64ToIntPtr(payload.MaxPayloadLength),
		DropPartialPayloads: payload.DropPartialPayloads,
	}
}

// Converts CustomInstrumentations to GraphQL-compatible format
func convertCustomInstrumentations(customInstruAsInstruRule *instrumentationrules.CustomInstrumentations) *model.CustomInstrumentations {
	if customInstruAsInstruRule == nil {
		return nil
	}
	customInstruAsGqlModel := &model.CustomInstrumentations{}
	if customInstruAsInstruRule.Golang != nil {
		for _, golangProbe := range customInstruAsInstruRule.Golang {
			customInstruAsGqlModel.Golang = append(customInstruAsGqlModel.Golang, &model.GolangCustomProbe{
				PackageName:        &golangProbe.PackageName,
				FunctionName:       &golangProbe.FunctionName,
				ReceiverName:       &golangProbe.ReceiverName,
				ReceiverMethodName: &golangProbe.ReceiverMethodName,
			})
		}
	}
	if customInstruAsInstruRule.Java != nil {
		for _, javaProbe := range customInstruAsInstruRule.Java {
			customInstruAsGqlModel.Java = append(customInstruAsGqlModel.Java, &model.JavaCustomProbe{
				ClassName:  &javaProbe.ClassName,
				MethodName: &javaProbe.MethodName,
			})
		}
	}
	return customInstruAsGqlModel
}
