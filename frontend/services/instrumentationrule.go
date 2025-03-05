package services

import (
	"context"
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

// ListInstrumentationRules fetches all instrumentation rules
func ListInstrumentationRules(ctx context.Context) ([]*model.InstrumentationRule, error) {
	ns := env.GetCurrentNamespace()

	instrumentationRules, err := kube.DefaultClient.OdigosClient.InstrumentationRules(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting instrumentation rules: %w", err)
	}

	var gqlRules []*model.InstrumentationRule
	for _, rule := range instrumentationRules.Items {
		annotations := rule.GetAnnotations()
		profileName := annotations[k8sconsts.OdigosProfileAnnotation]
		mutable := profileName == ""

		gqlRules = append(gqlRules, &model.InstrumentationRule{
			RuleID:                   rule.Name,
			RuleName:                 &rule.Spec.RuleName,
			Notes:                    &rule.Spec.Notes,
			Disabled:                 &rule.Spec.Disabled,
			Mutable:                  mutable,
			ProfileName:              profileName,
			Workloads:                convertWorkloads(rule.Spec.Workloads),
			InstrumentationLibraries: convertInstrumentationLibraries(rule.Spec.InstrumentationLibraries),
			PayloadCollection:        convertPayloadCollection(rule.Spec.PayloadCollection),
			CodeAttributes:           (*model.CodeAttributes)(rule.Spec.CodeAttributes),
		})
	}
	return gqlRules, nil
}

func GetInstrumentationRule(ctx context.Context, id string) (*model.InstrumentationRule, error) {
	ns := env.GetCurrentNamespace()

	rule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(ns).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		return nil, handleNotFoundError(err, id, "instrumentation rule")
	}

	return &model.InstrumentationRule{
		RuleID:                   rule.Name,
		RuleName:                 &rule.Spec.RuleName,
		Notes:                    &rule.Spec.Notes,
		Disabled:                 &rule.Spec.Disabled,
		Workloads:                convertWorkloads(rule.Spec.Workloads),
		InstrumentationLibraries: convertInstrumentationLibraries(rule.Spec.InstrumentationLibraries),
		PayloadCollection:        convertPayloadCollection(rule.Spec.PayloadCollection),
	}, nil
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

		existingRule.Spec.PayloadCollection = payloadCollection
	} else {
		existingRule.Spec.PayloadCollection = nil
	}

	var codeAttributes *instrumentationrules.CodeAttributes
	if input.CodeAttributes != nil {
		codeAttributes = &instrumentationrules.CodeAttributes{}

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

		existingRule.Spec.CodeAttributes = codeAttributes
	} else {
		existingRule.Spec.CodeAttributes = nil
	}

	// Update rule in Kubernetes
	updatedRule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(ns).Update(ctx, existingRule, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error updating instrumentation rule: %w", err)
	}

	annotations := updatedRule.GetAnnotations()
	profileName := annotations[k8sconsts.OdigosProfileAnnotation]

	return &model.InstrumentationRule{
		RuleID:                   updatedRule.Name,
		RuleName:                 &updatedRule.Spec.RuleName,
		Notes:                    &updatedRule.Spec.Notes,
		Disabled:                 &updatedRule.Spec.Disabled,
		Mutable:                  profileName == "",
		ProfileName:              profileName,
		Workloads:                convertWorkloads(updatedRule.Spec.Workloads),
		InstrumentationLibraries: convertInstrumentationLibraries(updatedRule.Spec.InstrumentationLibraries),
		PayloadCollection:        convertPayloadCollection(updatedRule.Spec.PayloadCollection),
		CodeAttributes:           (*model.CodeAttributes)(updatedRule.Spec.CodeAttributes),
	}, nil
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

	var payloadCollection *instrumentationrules.PayloadCollection
	if input.PayloadCollection != nil {
		payloadCollection = &instrumentationrules.PayloadCollection{}

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
	}

	var codeAttributes *instrumentationrules.CodeAttributes
	if input.CodeAttributes != nil {
		codeAttributes = &instrumentationrules.CodeAttributes{}

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
			PayloadCollection:        payloadCollection,
			CodeAttributes:           codeAttributes,
		},
	}

	// Create the rule in Kubernetes
	createdRule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(ns).Create(ctx, newRule, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error creating instrumentation rule: %w", err)
	}

	// Convert to GraphQL model and return
	return &model.InstrumentationRule{
		RuleID:                   createdRule.Name,
		RuleName:                 &createdRule.Spec.RuleName,
		Notes:                    &createdRule.Spec.Notes,
		Disabled:                 &createdRule.Spec.Disabled,
		Mutable:                  true, // New rules are always mutable
		ProfileName:              "",   // New rules are not associated with a profile
		Workloads:                convertWorkloads(createdRule.Spec.Workloads),
		InstrumentationLibraries: convertInstrumentationLibraries(createdRule.Spec.InstrumentationLibraries),
		PayloadCollection:        convertPayloadCollection(createdRule.Spec.PayloadCollection),
		CodeAttributes:           (*model.CodeAttributes)(createdRule.Spec.CodeAttributes),
	}, nil
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
