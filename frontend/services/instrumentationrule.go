package services

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	instrumentationrules "github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// // ListInstrumentationRules fetches all instrumentation rules
// func ListInstrumentationRules(ctx context.Context) ([]*model.InstrumentationRule, error) {
// 	odigosns := consts.DefaultOdigosNamespace
// 	instrumentationRules, err := kube.DefaultClient.OdigosClient.InstrumentationRules(odigosns).List(ctx, metav1.ListOptions{})
// 	if err != nil {
// 		return nil, fmt.Errorf("error getting instrumentation rules: %w", err)
// 	}

// 	var gqlRules []*model.InstrumentationRule
// 	for _, rule := range instrumentationRules.Items {
// 		gqlRules = append(gqlRules, &model.InstrumentationRule{
// 			RuleId:                   rule.Name,
// 			RuleName:                 rule.Spec.RuleName,
// 			Notes:                    rule.Spec.Notes,
// 			Disabled:                 rule.Spec.Disabled,
// 			Workloads:                rule.Spec.Workloads,
// 			InstrumentationLibraries: rule.Spec.InstrumentationLibraries,
// 			PayloadCollection:        rule.Spec.PayloadCollection,
// 			OtelSdks:                 rule.Spec.OtelSdks,
// 		})
// 	}
// 	return gqlRules, nil
// }

// func GetInstrumentationRule(ctx context.Context, id string) (*model.InstrumentationRule, error) {
// 	odigosns := consts.DefaultOdigosNamespace

// 	rule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(odigosns).Get(ctx, id, metav1.GetOptions{})
// 	if err != nil {
// 		if apierrors.IsNotFound(err) {
// 			return nil, fmt.Errorf("instrumentation rule with ID %s not found", id)
// 		}
// 		return nil, fmt.Errorf("error getting instrumentation rule: %w", err)
// 	}

// 	return &model.InstrumentationRule{
// 		RuleId:                   rule.Name,
// 		RuleName:                 rule.Spec.RuleName,
// 		Notes:                    rule.Spec.Notes,
// 		Disabled:                 rule.Spec.Disabled,
// 		Workloads:                rule.Spec.Workloads,
// 		InstrumentationLibraries: rule.Spec.InstrumentationLibraries,
// 		PayloadCollection:        rule.Spec.PayloadCollection,
// 		OtelSdks:                 rule.Spec.OtelSdks,
// 	}, nil
// }

// func UpdateInstrumentationRule(ctx context.Context, id string, input model.InstrumentationRuleInput) (*model.InstrumentationRule, error) {
// 	odigosns := consts.DefaultOdigosNamespace

// 	// Retrieve existing rule
// 	existingRule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(odigosns).Get(ctx, id, metav1.GetOptions{})
// 	if err != nil {
// 		if apierrors.IsNotFound(err) {
// 			return nil, fmt.Errorf("instrumentation rule with ID %s not found", id)
// 		}
// 		return nil, fmt.Errorf("error getting instrumentation rule: %w", err)
// 	}

// 	// Update the existing rule's specification
// 	existingRule.Spec.RuleName = input.RuleName
// 	existingRule.Spec.Notes = input.Notes
// 	existingRule.Spec.Disabled = input.Disabled
// 	existingRule.Spec.Workloads = input.Workloads
// 	existingRule.Spec.InstrumentationLibraries = input.InstrumentationLibraries
// 	existingRule.Spec.PayloadCollection = input.PayloadCollection
// 	existingRule.Spec.OtelSdks = input.OtelSdks

// 	// Update rule in Kubernetes
// 	updatedRule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(odigosns).Update(ctx, existingRule, metav1.UpdateOptions{})
// 	if err != nil {
// 		return nil, fmt.Errorf("error updating instrumentation rule: %w", err)
// 	}

// 	return &model.InstrumentationRule{
// 		RuleId:                   updatedRule.Name,
// 		RuleName:                 updatedRule.Spec.RuleName,
// 		Notes:                    updatedRule.Spec.Notes,
// 		Disabled:                 updatedRule.Spec.Disabled,
// 		Workloads:                updatedRule.Spec.Workloads,
// 		InstrumentationLibraries: updatedRule.Spec.InstrumentationLibraries,
// 		PayloadCollection:        updatedRule.Spec.PayloadCollection,
// 		OtelSdks:                 updatedRule.Spec.OtelSdks,
// 	}, nil
// }

// func DeleteInstrumentationRule(ctx context.Context, id string) (bool, error) {
// 	odigosns := consts.DefaultOdigosNamespace

// 	err := kube.DefaultClient.OdigosClient.InstrumentationRules(odigosns).Delete(ctx, id, metav1.DeleteOptions{})
// 	if err != nil {
// 		if apierrors.IsNotFound(err) {
// 			return false, fmt.Errorf("instrumentation rule with ID %s not found", id)
// 		}
// 		return false, fmt.Errorf("error deleting instrumentation rule: %w", err)
// 	}

// 	return true, nil
// }

func CreateInstrumentationRule(ctx context.Context, input model.InstrumentationRuleInput) (*model.InstrumentationRule, error) {
	odigosns := consts.DefaultOdigosNamespace

	ruleName := *input.RuleName
	notes := *input.Notes
	disabled := *input.Disabled

	var workloads *[]workload.PodWorkload
	if input.Workloads != nil {
		convertedWorkloads := make([]workload.PodWorkload, len(input.Workloads))
		for i, w := range input.Workloads {
			convertedWorkloads[i] = workload.PodWorkload{
				Name:      w.Name,
				Namespace: w.Namespace,
				Kind:      workload.WorkloadKind(w.Kind),
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

			mineTypes := make([]string, len(input.PayloadCollection.HTTPRequest.MimeTypes))
			for i, mt := range input.PayloadCollection.HTTPRequest.MimeTypes {
				mineTypes[i] = *mt
			}

			maxPayloadLength := int64(*input.PayloadCollection.HTTPRequest.MaxPayloadLength)

			payloadCollection.HttpRequest = &instrumentationrules.HttpPayloadCollection{
				MimeTypes:           &mineTypes,
				MaxPayloadLength:    &maxPayloadLength,
				DropPartialPayloads: input.PayloadCollection.HTTPRequest.DropPartialPayloads,
			}
		}

		if input.PayloadCollection.HTTPResponse != nil {
			mimeTypes := make([]string, len(input.PayloadCollection.HTTPResponse.MimeTypes))
			for i, mt := range input.PayloadCollection.HTTPResponse.MimeTypes {
				mimeTypes[i] = *mt
			}
			maxPayloadLength := int64(*input.PayloadCollection.HTTPResponse.MaxPayloadLength)

			payloadCollection.HttpResponse = &instrumentationrules.HttpPayloadCollection{
				MimeTypes:           &mimeTypes,
				MaxPayloadLength:    &maxPayloadLength,
				DropPartialPayloads: input.PayloadCollection.HTTPResponse.DropPartialPayloads,
			}
		}

		if input.PayloadCollection.DbQuery != nil {
			maxPayloadLength := int64(*input.PayloadCollection.DbQuery.MaxPayloadLength)
			payloadCollection.DbQuery = &instrumentationrules.DbQueryPayloadCollection{
				MaxPayloadLength:    &maxPayloadLength,
				DropPartialPayloads: input.PayloadCollection.DbQuery.DropPartialPayloads,
			}
		}

		if input.PayloadCollection.Messaging != nil {
			maxPayloadLength := int64(*input.PayloadCollection.Messaging.MaxPayloadLength)
			payloadCollection.Messaging = &instrumentationrules.MessagingPayloadCollection{
				MaxPayloadLength:    &maxPayloadLength,
				DropPartialPayloads: input.PayloadCollection.Messaging.DropPartialPayloads,
			}
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
		},
	}

	// Create the rule in Kubernetes
	createdRule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(odigosns).Create(ctx, newRule, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error creating instrumentation rule: %w", err)
	}
	println(createdRule.Name)
	// Convert to GraphQL model and return
	return &model.InstrumentationRule{}, nil
}
