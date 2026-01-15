/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package actions

import (
	"context"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	actionsv1alpha1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

var validActionConfigNames = []string{
	actionsv1alpha1.ActionNameAddClusterInfo,
	actionsv1alpha1.ActionNameDeleteAttribute,
	actionsv1alpha1.ActionNameRenameAttribute,
	actionsv1alpha1.ActionNamePiiMasking,
	actionsv1alpha1.ActionNameK8sAttributes,
	actionsv1alpha1.ActionNameSamplers,
	actions.ActionNameURLTemplatization,
	actions.ActionSpanRenamer,
}

type ActionsValidator struct {
	client.Client
}

var _ webhook.CustomValidator = &ActionsValidator{}

func (s *ActionsValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	action, ok := obj.(*v1alpha1.Action)
	if !ok {
		return nil, fmt.Errorf("expected an Action but got a %T", obj)
	}

	err := s.validateAction(ctx, action)
	if err != nil {
		return nil, apierrors.NewInvalid(
			schema.GroupKind{Group: "odigos.io", Kind: "Action"},
			action.Name, err,
		)
	}

	return nil, nil
}

func (s *ActionsValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	old, ok := oldObj.(*v1alpha1.Action)
	if !ok {
		return nil, fmt.Errorf("expected old Action but got a %T", old)
	}
	new, ok := newObj.(*v1alpha1.Action)
	if !ok {
		return nil, fmt.Errorf("expected new Action but got a %T", new)
	}

	err := s.validateAction(ctx, new)
	if err != nil {
		return nil, apierrors.NewInvalid(
			schema.GroupKind{Group: "odigos.io", Kind: "Action"},
			new.Name, err,
		)
	}

	return nil, nil
}

func (s *ActionsValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// TODO: Refactor Action config to cast a generic config field in v1alpha2 to one of the matching action configs to allow type-switch validation.
func (a *ActionsValidator) validateAction(ctx context.Context, action *v1alpha1.Action) field.ErrorList {

	odigosNamespace := env.GetCurrentNamespace()
	if action.Namespace != odigosNamespace {
		return field.ErrorList{field.Invalid(field.NewPath("namespace"), action.Namespace, "actions are only allowed in the odigos namespace '"+odigosNamespace+"'")}
	}

	var allErrs field.ErrorList
	fields := make(map[*field.Path]ActionConfig)
	if action.Spec.AddClusterInfo != nil {
		path := field.NewPath("spec").Child("addClusterInfo")
		fields[path] = action.Spec.AddClusterInfo
	}
	if action.Spec.DeleteAttribute != nil {
		path := field.NewPath("spec").Child("deleteAttribute")
		fields[path] = action.Spec.DeleteAttribute
	}
	if action.Spec.RenameAttribute != nil {
		path := field.NewPath("spec").Child("renameAttribute")
		fields[path] = action.Spec.RenameAttribute
	}
	if action.Spec.PiiMasking != nil {
		path := field.NewPath("spec").Child("piiMasking")
		fields[path] = action.Spec.PiiMasking
	}
	if action.Spec.K8sAttributes != nil {
		path := field.NewPath("spec").Child("k8sAttributes")
		fields[path] = action.Spec.K8sAttributes
	}
	if action.Spec.Samplers != nil {
		path := field.NewPath("spec").Child("samplers")
		fields[path] = action.Spec.Samplers

		var validSamplerFields = []string{
			actionsv1alpha1.ActionNameSpanAttributeSampler,
			actionsv1alpha1.ActionNameLatencySampler,
			actionsv1alpha1.ActionNameErrorSampler,
			actionsv1alpha1.ActionNameServiceNameSampler,
			actionsv1alpha1.ActionNameProbabilisticSampler,
			actionsv1alpha1.ActionNameIgnoreHealthChecks,
		}

		samplerFields := make(map[*field.Path]interface{})
		if action.Spec.Samplers.SpanAttributeSampler != nil {
			path := field.NewPath("spec").Child("samplers").Child("spanAttributeSampler")
			samplerFields[path] = action.Spec.Samplers.SpanAttributeSampler
		}
		if action.Spec.Samplers.LatencySampler != nil {
			path := field.NewPath("spec").Child("samplers").Child("latencySampler")
			samplerFields[path] = action.Spec.Samplers.LatencySampler
		}
		if action.Spec.Samplers.ErrorSampler != nil {
			path := field.NewPath("spec").Child("samplers").Child("errorSampler")
			samplerFields[path] = action.Spec.Samplers.ErrorSampler
		}
		if action.Spec.Samplers.ServiceNameSampler != nil {
			path := field.NewPath("spec").Child("samplers").Child("serviceNameSampler")
			samplerFields[path] = action.Spec.Samplers.ServiceNameSampler
		}
		if action.Spec.Samplers.ProbabilisticSampler != nil {
			path := field.NewPath("spec").Child("samplers").Child("probabilisticSampler")
			samplerFields[path] = action.Spec.Samplers.ProbabilisticSampler
		}
		if action.Spec.Samplers.IgnoreHealthChecks != nil {
			// check if the ignore health checks fraction is in range 0-1
			if action.Spec.Samplers.IgnoreHealthChecks.FractionToRecord < 0 || action.Spec.Samplers.IgnoreHealthChecks.FractionToRecord > 1 {
				allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("samplers").Child("ignoreHealthChecks").Child("fractionToRecord"), action.Spec.Samplers.IgnoreHealthChecks.FractionToRecord, "fractionToRecord must be in range 0-1"))
			} else {
				path := field.NewPath("spec").Child("samplers").Child("ignoreHealthChecks")
				samplerFields[path] = action.Spec.Samplers.IgnoreHealthChecks
			}
		}
		if len(samplerFields) == 0 {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("samplers"), samplerFields, fmt.Sprintf("At least one of (%s) must be set", strings.Join(validSamplerFields, ", "))))
		}
		if len(samplerFields) > 1 {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("samplers"), samplerFields, fmt.Sprintf("Only one of (%s) may be set", strings.Join(validSamplerFields, ", "))))
		}
	}
	if action.Spec.URLTemplatization != nil {
		path := field.NewPath("spec").Child("urlTemplatization")
		fields[path] = action.Spec.URLTemplatization
	}
	if action.Spec.SpanRenamer != nil {
		path := field.NewPath("spec").Child("spanRenamer")
		fields[path] = action.Spec.SpanRenamer
	}

	if len(fields) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("spec"), fmt.Sprintf("At least one of (%s) must be set", strings.Join(validActionConfigNames, ", "))))
	}
	if len(fields) > 1 {
		for path, cfg := range fields {
			allErrs = append(allErrs, field.Invalid(
				path,
				cfg,
				fmt.Sprintf("Only one of (%s) may be set", strings.Join(validActionConfigNames, ", ")),
			))
		}
	}

	return allErrs
}
