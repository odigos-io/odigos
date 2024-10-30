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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListInstrumentationRules fetches all instrumentation rules
func ListInstrumentationRules(ctx context.Context) ([]*model.InstrumentationRule, error) {
	odigosns := consts.DefaultOdigosNamespace
	instrumentationRules, err := kube.DefaultClient.OdigosClient.InstrumentationRules(odigosns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting instrumentation rules: %w", err)
	}

	var gqlRules []*model.InstrumentationRule
	for _, rule := range instrumentationRules.Items {

		var gqlWorkloads []*model.PodWorkload
		if rule.Spec.Workloads != nil {
			for _, workload := range *rule.Spec.Workloads {
				gqlWorkloads = append(gqlWorkloads, &model.PodWorkload{
					Namespace: workload.Namespace,
					Kind:      model.K8sResourceKind(workload.Kind),
					Name:      workload.Name,
				})
			}
		}

		var gqlLibraries []*model.InstrumentationLibraryGlobalID
		if rule.Spec.InstrumentationLibraries != nil {
			for _, library := range *rule.Spec.InstrumentationLibraries {

				spanKind := model.SpanKind(library.SpanKind)
				language := model.ProgrammingLanguage(library.Language)
				gqlLibraries = append(gqlLibraries, &model.InstrumentationLibraryGlobalID{
					Name:     library.Name,
					SpanKind: &spanKind,
					Language: &language,
				})
			}
		}

		var gqlPayloadCollection *model.PayloadCollection
		if rule.Spec.PayloadCollection != nil {
			var gqlHttpRequest *model.HTTPPayloadCollection
			if rule.Spec.PayloadCollection.HttpRequest != nil {
				gqlHttpRequest = &model.HTTPPayloadCollection{}
			}

			var gqlHttpResponse *model.HTTPPayloadCollection
			if rule.Spec.PayloadCollection.HttpResponse != nil {
				gqlHttpResponse = &model.HTTPPayloadCollection{}
			}

			var gqlDbQuery *model.DbQueryPayloadCollection
			if rule.Spec.PayloadCollection.DbQuery != nil {
				gqlDbQuery = &model.DbQueryPayloadCollection{}
			}

			var gqlMessaging *model.MessagingPayloadCollection
			if rule.Spec.PayloadCollection.Messaging != nil {
				gqlMessaging = &model.MessagingPayloadCollection{}
			}

			gqlPayloadCollection = &model.PayloadCollection{
				HTTPRequest:  gqlHttpRequest,
				HTTPResponse: gqlHttpResponse,
				DbQuery:      gqlDbQuery,
				Messaging:    gqlMessaging,
			}
		}

		gqlRules = append(gqlRules, &model.InstrumentationRule{
			RuleID:                   rule.Name,
			RuleName:                 &rule.Spec.RuleName,
			Notes:                    &rule.Spec.Notes,
			Disabled:                 &rule.Spec.Disabled,
			Workloads:                gqlWorkloads,
			InstrumentationLibraries: gqlLibraries,
			PayloadCollection:        gqlPayloadCollection,
		})
	}
	return gqlRules, nil
}

func GetInstrumentationRule(ctx context.Context, id string) (*model.InstrumentationRule, error) {
	odigosns := consts.DefaultOdigosNamespace

	rule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(odigosns).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("instrumentation rule with ID %s not found", id)
		}
		return nil, fmt.Errorf("error getting instrumentation rule: %w", err)
	}
	var gqlWorkloads []*model.PodWorkload
	if rule.Spec.Workloads != nil {
		for _, workload := range *rule.Spec.Workloads {
			gqlWorkloads = append(gqlWorkloads, &model.PodWorkload{
				Namespace: workload.Namespace,
				Kind:      model.K8sResourceKind(workload.Kind),
				Name:      workload.Name,
			})
		}
	}

	var gqlLibraries []*model.InstrumentationLibraryGlobalID
	if rule.Spec.InstrumentationLibraries != nil {
		for _, library := range *rule.Spec.InstrumentationLibraries {
			spanKind := model.SpanKind(library.SpanKind)
			language := model.ProgrammingLanguage(library.Language)
			gqlLibraries = append(gqlLibraries, &model.InstrumentationLibraryGlobalID{
				Name:     library.Name,
				SpanKind: &spanKind,
				Language: &language,
			})
		}
	}

	var gqlPayloadCollection *model.PayloadCollection
	if rule.Spec.PayloadCollection != nil {
		var gqlHttpRequest *model.HTTPPayloadCollection
		if rule.Spec.PayloadCollection.HttpRequest != nil {
			gqlHttpRequest = &model.HTTPPayloadCollection{}
		}

		var gqlHttpResponse *model.HTTPPayloadCollection
		if rule.Spec.PayloadCollection.HttpResponse != nil {
			gqlHttpResponse = &model.HTTPPayloadCollection{}
		}

		var gqlDbQuery *model.DbQueryPayloadCollection
		if rule.Spec.PayloadCollection.DbQuery != nil {
			gqlDbQuery = &model.DbQueryPayloadCollection{}
		}

		var gqlMessaging *model.MessagingPayloadCollection
		if rule.Spec.PayloadCollection.Messaging != nil {
			gqlMessaging = &model.MessagingPayloadCollection{}
		}

		gqlPayloadCollection = &model.PayloadCollection{
			HTTPRequest:  gqlHttpRequest,
			HTTPResponse: gqlHttpResponse,
			DbQuery:      gqlDbQuery,
			Messaging:    gqlMessaging,
		}
	}

	return &model.InstrumentationRule{
		RuleID:                   rule.Name,
		RuleName:                 &rule.Spec.RuleName,
		Notes:                    &rule.Spec.Notes,
		Disabled:                 &rule.Spec.Disabled,
		Workloads:                gqlWorkloads,
		InstrumentationLibraries: gqlLibraries,
		PayloadCollection:        gqlPayloadCollection,
	}, nil
}

func UpdateInstrumentationRule(ctx context.Context, id string, input model.InstrumentationRuleInput) (*model.InstrumentationRule, error) {
	odigosns := consts.DefaultOdigosNamespace

	// Retrieve existing rule
	existingRule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(odigosns).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("instrumentation rule with ID %s not found", id)
		}
		return nil, fmt.Errorf("error getting instrumentation rule: %w", err)
	}

	// Update the existing rule's specification
	existingRule.Spec.RuleName = *input.RuleName
	existingRule.Spec.Notes = *input.Notes
	existingRule.Spec.Disabled = *input.Disabled
	if input.Workloads != nil {
		convertedWorkloads := make([]workload.PodWorkload, len(input.Workloads))
		for i, w := range input.Workloads {
			convertedWorkloads[i] = workload.PodWorkload{
				Name:      w.Name,
				Namespace: w.Namespace,
				Kind:      workload.WorkloadKind(w.Kind),
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

	// Update rule in Kubernetes
	updatedRule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(odigosns).Update(ctx, existingRule, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error updating instrumentation rule: %w", err)
	}

	var gqlWorkloads []*model.PodWorkload
	if input.Workloads != nil {
		for _, w := range input.Workloads {
			gqlWorkloads = append(gqlWorkloads, &model.PodWorkload{
				Name:      w.Name,
				Namespace: w.Namespace,
				Kind:      model.K8sResourceKind(w.Kind),
			})
		}
	}

	var gqlLibraries []*model.InstrumentationLibraryGlobalID
	if input.InstrumentationLibraries != nil {
		for _, lib := range input.InstrumentationLibraries {
			spanKind := model.SpanKind(*lib.SpanKind)
			language := model.ProgrammingLanguage(*lib.Language)
			gqlLibraries = append(gqlLibraries, &model.InstrumentationLibraryGlobalID{
				Name:     lib.Name,
				SpanKind: &spanKind,
				Language: &language,
			})
		}
	}

	var gqlPayloadCollection *model.PayloadCollection
	if input.PayloadCollection != nil {
		var gqlHTTPRequest, gqlHTTPResponse *model.HTTPPayloadCollection
		var gqlDbQuery *model.DbQueryPayloadCollection
		var gqlMessaging *model.MessagingPayloadCollection

		if input.PayloadCollection.HTTPRequest != nil {
			gqlHTTPRequest = &model.HTTPPayloadCollection{}
		}
		if input.PayloadCollection.HTTPResponse != nil {
			gqlHTTPResponse = &model.HTTPPayloadCollection{}
		}
		if input.PayloadCollection.DbQuery != nil {
			gqlDbQuery = &model.DbQueryPayloadCollection{}
		}
		if input.PayloadCollection.Messaging != nil {
			gqlMessaging = &model.MessagingPayloadCollection{}
		}

		gqlPayloadCollection = &model.PayloadCollection{
			HTTPRequest:  gqlHTTPRequest,
			HTTPResponse: gqlHTTPResponse,
			DbQuery:      gqlDbQuery,
			Messaging:    gqlMessaging,
		}
	}

	return &model.InstrumentationRule{
		RuleID:                   updatedRule.Name,
		RuleName:                 &updatedRule.Spec.RuleName,
		Notes:                    &updatedRule.Spec.Notes,
		Disabled:                 &updatedRule.Spec.Disabled,
		Workloads:                gqlWorkloads,
		InstrumentationLibraries: gqlLibraries,
		PayloadCollection:        gqlPayloadCollection,
	}, nil
}

func DeleteInstrumentationRule(ctx context.Context, id string) (bool, error) {
	odigosns := consts.DefaultOdigosNamespace

	err := kube.DefaultClient.OdigosClient.InstrumentationRules(odigosns).Delete(ctx, id, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, fmt.Errorf("instrumentation rule with ID %s not found", id)
		}
		return false, fmt.Errorf("error deleting instrumentation rule: %w", err)
	}

	return true, nil
}

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
	// Convert to GraphQL model and return
	return &model.InstrumentationRule{
		RuleID: createdRule.Name,
	}, nil
}
