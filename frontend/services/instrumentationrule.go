package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	instrumentationrules "github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/common"
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

// ListInstrumentationRules fetches all instrumentation rules
func ListInstrumentationRules(ctx context.Context) ([]*model.InstrumentationRule, error) {
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
			Workloads:                convertWorkloads(r.Spec.Workloads),
			InstrumentationLibraries: convertInstrumentationLibraries(r.Spec.InstrumentationLibraries),
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
		Workloads:                convertWorkloads(r.Spec.Workloads),
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

	if input.PayloadCollection.HTTPRequest != nil {
		payloadCollection.HttpRequest = &instrumentationrules.HttpPayloadCollection{}
	}
	if input.PayloadCollection.HTTPResponse != nil {
		payloadCollection.HttpResponse = &instrumentationrules.HttpPayloadCollection{}
	}
	if input.PayloadCollection.DbQuery != nil {
		payloadCollection.DbQuery = &instrumentationrules.DbQueryPayloadCollection{}
	}
	if input.PayloadCollection.Messaging != nil {
		payloadCollection.Messaging = &instrumentationrules.MessagingPayloadCollection{}
	}

	return payloadCollection
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
	seen := make(map[string]struct{})
	for _, probe := range customInstrumentations.Golang {
		var key string
		key = "pkg:" + probe.PackageName + "|"
		if probe.FunctionName != "" {
			key += "function:" + probe.FunctionName
		} else {
			key += "receiver:" + probe.ReceiverName + "|method:" + probe.ReceiverMethodName
		}
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			uniqueGolangProbes = append(uniqueGolangProbes, probe)
		}
	}
	customInstrumentations.Golang = uniqueGolangProbes

	// Remove duplicate Java probes
	uniqueJavaProbes := make([]instrumentationrules.JavaCustomProbe, 0, len(customInstrumentations.Java))
	seen = make(map[string]struct{})
	for _, probe := range customInstrumentations.Java {
		var key string
		if probe.ClassName != "" && probe.MethodName != "" {
			key = "class:" + probe.ClassName + "|method:" + probe.MethodName
		}
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			uniqueJavaProbes = append(uniqueJavaProbes, probe)
		}
	}
	customInstrumentations.Java = uniqueJavaProbes

	fmt.Printf("FRONTEND: Custom Instrumentations: %+v\n", customInstrumentations)
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
	if input.Workloads != nil {
		convertedWorkloads := make([]k8sconsts.PodWorkload, len(input.Workloads))
		for i, w := range input.Workloads {
			convertedWorkloads[i] = k8sconsts.PodWorkload{
				Name:      w.Name,
				Namespace: w.Namespace,
				Kind:      k8sconsts.WorkloadKind(w.Kind),
			}
		}
		existingRule.Spec.Workloads = &convertedWorkloads
	} else {
		existingRule.Spec.Workloads = nil
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
	// Print if the rule is enabled or disabled for debugging
	fmt.Printf("Updating Instrumentation Rule %s: Disabled=%v, Custom Instrumentations: %+v\n", id, existingRule.Spec.Disabled, existingRule.Spec.CustomInstrumentations)
	// Print the custom instrumentations for debugging
	fmt.Printf("Updating Instrumentation Rule %s with Custom Instrumentations: %+v\n", id, existingRule.Spec.CustomInstrumentations)
	// Update rule in Kubernetes
	updatedRule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(ns).Update(ctx, existingRule, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error updating instrumentation rule: %w", err)
	}

	annotations := updatedRule.GetAnnotations()
	profileName := annotations[k8sconsts.OdigosProfileAnnotation]

	// print the custom instrumentation probes for debugging
	if updatedRule.Spec.CustomInstrumentations != nil {
		probesJson, _ := json.MarshalIndent(updatedRule.Spec.CustomInstrumentations, "", "  ")
		fmt.Printf("XXXXX Updated Instrumentation Rule %s Custom Instrumentation Probes: %s\n", id, string(probesJson))
	}

	rule := model.InstrumentationRule{
		RuleID:                   updatedRule.Name,
		RuleName:                 &updatedRule.Spec.RuleName,
		Notes:                    &updatedRule.Spec.Notes,
		Disabled:                 &updatedRule.Spec.Disabled,
		Mutable:                  profileName == "",
		ProfileName:              profileName,
		Workloads:                convertWorkloads(updatedRule.Spec.Workloads),
		InstrumentationLibraries: convertInstrumentationLibraries(updatedRule.Spec.InstrumentationLibraries),
		CodeAttributes:           (*model.CodeAttributes)(updatedRule.Spec.CodeAttributes),
		HeadersCollection:        convertHeadersCollection(updatedRule.Spec.HeadersCollection),
		PayloadCollection:        convertPayloadCollection(updatedRule.Spec.PayloadCollection),
		CustomInstrumentations:   convertCustomInstrumentations(updatedRule.Spec.CustomInstrumentations),
	}
	rule.Type = deriveTypeFromRule(&rule)
	// Print all the probes from the custom instrumentations for debugging
	if rule.CustomInstrumentations != nil {
		probesJson, _ := json.MarshalIndent(rule.CustomInstrumentations, "", "  ")
		fmt.Printf("Updated Instrumentation Rule %s Custom Instrumentation Probes: %s\n", id, string(probesJson))
	}

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

	var workloads *[]k8sconsts.PodWorkload
	if input.Workloads != nil {
		convertedWorkloads := make([]k8sconsts.PodWorkload, len(input.Workloads))
		for i, w := range input.Workloads {
			convertedWorkloads[i] = k8sconsts.PodWorkload{
				Name:      w.Name,
				Namespace: w.Namespace,
				Kind:      k8sconsts.WorkloadKind(w.Kind),
			}
		}
		workloads = &convertedWorkloads
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
			Workloads:                workloads,
			InstrumentationLibraries: instrumentationLibraries,
			CodeAttributes:           getCodeAttributesInput(input),
			HeadersCollection:        getHeadersCollectionInput(input),
			PayloadCollection:        getPayloadCollectionInput(input),
			CustomInstrumentations:   getCustomInstrumentationsInput(input),
		},
	}
	// Print the custom instrumentations for debugging
	fmt.Printf("Creating Instrumentation Rule with Custom Instrumentations: %+v\n", newRule.Spec.CustomInstrumentations)
	// Print the disabled status for debugging
	fmt.Printf("Creating Instrumentation Rule: Disabled=%v\n", newRule.Spec.Disabled)
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
		Workloads:                convertWorkloads(createdRule.Spec.Workloads),
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

// Converts Workloads to GraphQL-compatible format
func convertWorkloads(workloads *[]k8sconsts.PodWorkload) []*model.PodWorkload {
	var gqlWorkloads []*model.PodWorkload
	if workloads != nil {
		for _, w := range *workloads {
			gqlWorkloads = append(gqlWorkloads, &model.PodWorkload{
				Namespace: w.Namespace,
				Kind:      model.K8sResourceKind(w.Kind),
				Name:      w.Name,
			})
		}
	}
	return gqlWorkloads
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

// Helpers to create empty payloads if they exist
func toHTTPPayload(payload *instrumentationrules.HttpPayloadCollection) *model.HTTPPayloadCollection {
	if payload == nil {
		return nil
	}
	return &model.HTTPPayloadCollection{}
}

func toDbQueryPayload(payload *instrumentationrules.DbQueryPayloadCollection) *model.DbQueryPayloadCollection {
	if payload == nil {
		return nil
	}
	return &model.DbQueryPayloadCollection{}
}

func toMessagingPayload(payload *instrumentationrules.MessagingPayloadCollection) *model.MessagingPayloadCollection {
	if payload == nil {
		return nil
	}
	return &model.MessagingPayloadCollection{}
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

	// Json stringify the probes for debugging
	probesJson, _ := json.MarshalIndent(customInstruAsGqlModel, "", "  ")
	fmt.Printf("Converted Custom Instrumentation Probes: %s\n", string(probesJson))

	return customInstruAsGqlModel
}
