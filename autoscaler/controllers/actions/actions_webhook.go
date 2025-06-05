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
)

var validActionConfigNames = []string{
	actionsv1alpha1.ActionNameAddClusterInfo,
	actionsv1alpha1.ActionNameDeleteAttribute,
	actionsv1alpha1.ActionNameRenameAttribute,
	actionsv1alpha1.ActionNamePiiMasking,
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
