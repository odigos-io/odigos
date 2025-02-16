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

package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type SourcesDefaulter struct {
	client.Client
}

var _ webhook.CustomDefaulter = &SourcesDefaulter{}

func (s *SourcesDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	source, ok := obj.(*v1alpha1.Source)
	if !ok {
		return fmt.Errorf("expected a Source but got a %T", obj)
	}

	if source.Labels == nil {
		source.Labels = make(map[string]string)
	}

	if _, ok := source.Labels[k8sconsts.WorkloadNameLabel]; !ok {
		source.Labels[k8sconsts.WorkloadNameLabel] = source.Spec.Workload.Name
	}
	if _, ok := source.Labels[k8sconsts.WorkloadNamespaceLabel]; !ok {
		source.Labels[k8sconsts.WorkloadNamespaceLabel] = source.Spec.Workload.Namespace
	}
	if _, ok := source.Labels[k8sconsts.WorkloadKindLabel]; !ok {
		source.Labels[k8sconsts.WorkloadKindLabel] = string(source.Spec.Workload.Kind)
	}

	// Make sure the Source has the right finalizer, so the right controller handles it for deletion.
	// If a normal source has `spec.disableInstrumentation` updated to `true`, it is now an excluded Source.
	// Vice versa for an excluded Source that has `spec.disableInstrumentation` removed.
	// These checks make sure that the right type of Source has the right type of finalizer
	// by toggling what finalizer is set.
	if !v1alpha1.IsDisabledSource(source) {
		if !controllerutil.ContainsFinalizer(source, k8sconsts.DeleteInstrumentationConfigFinalizer) && !k8sutils.IsTerminating(source) {
			controllerutil.AddFinalizer(source, k8sconsts.DeleteInstrumentationConfigFinalizer)
		}
		if controllerutil.ContainsFinalizer(source, k8sconsts.StartLangDetectionFinalizer) {
			controllerutil.RemoveFinalizer(source, k8sconsts.StartLangDetectionFinalizer)
		}
	}
	if v1alpha1.IsDisabledSource(source) {
		if controllerutil.ContainsFinalizer(source, k8sconsts.DeleteInstrumentationConfigFinalizer) {
			controllerutil.RemoveFinalizer(source, k8sconsts.DeleteInstrumentationConfigFinalizer)
		}
		if !controllerutil.ContainsFinalizer(source, k8sconsts.StartLangDetectionFinalizer) && !k8sutils.IsTerminating(source) {
			controllerutil.AddFinalizer(source, k8sconsts.StartLangDetectionFinalizer)
		}
	}

	return nil
}

type SourcesValidator struct {
	client.Client
}

var _ webhook.CustomValidator = &SourcesValidator{}

func (s *SourcesValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList
	source, ok := obj.(*v1alpha1.Source)
	if !ok {
		return nil, fmt.Errorf("expected a Source but got a %T", obj)
	}

	errs := s.validateSourceFields(ctx, source)
	if len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	if len(allErrs) == 0 {
		return nil, nil
	}

	return nil, apierrors.NewInvalid(
		schema.GroupKind{Group: "odigos.io", Kind: "Source"},
		source.Name, allErrs)
}

func (s *SourcesValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList
	old, ok := oldObj.(*v1alpha1.Source)
	if !ok {
		return nil, fmt.Errorf("expected old Source but got a %T", old)
	}
	new, ok := newObj.(*v1alpha1.Source)
	if !ok {
		return nil, fmt.Errorf("expected new Source but got a %T", new)
	}

	if new.GetName() != old.GetName() {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("metadata").Child("name"),
			new.GetName(),
			"Source name is immutable",
		))
	}

	if new.GetNamespace() != old.GetNamespace() {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("metadata").Child("namespace"),
			new.GetNamespace(),
			"Source namespace is immutable",
		))
	}

	if new.Labels[k8sconsts.WorkloadKindLabel] != old.Labels[k8sconsts.WorkloadKindLabel] {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("metadata").Child("labels"),
			new.Labels[k8sconsts.WorkloadKindLabel],
			"Source workload-kind label is immutable",
		))
	}
	if new.Labels[k8sconsts.WorkloadNameLabel] != old.Labels[k8sconsts.WorkloadNameLabel] {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("metadata").Child("labels"),
			new.Labels[k8sconsts.WorkloadNameLabel],
			"Source workload-name label is immutable",
		))
	}
	if new.Labels[k8sconsts.WorkloadNamespaceLabel] != old.Labels[k8sconsts.WorkloadNamespaceLabel] {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("metadata").Child("labels"),
			new.Labels[k8sconsts.WorkloadNamespaceLabel],
			"Source workload-namespace label is immutable",
		))
	}
	if new.Spec.Workload != old.Spec.Workload {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec").Child("workload"),
			new.Spec.Workload,
			"Source workload is immutable",
		))
	}

	errs := s.validateSourceFields(ctx, new)
	if len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	if len(allErrs) == 0 {
		return nil, nil
	}

	return nil, apierrors.NewInvalid(
		schema.GroupKind{Group: "odigos.io", Kind: "Source"},
		new.Name, allErrs)
}

func (s *SourcesValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (s *SourcesValidator) validateSourceFields(ctx context.Context, source *v1alpha1.Source) field.ErrorList {
	allErrs := make([]*field.Error, 0)

	if controllerutil.ContainsFinalizer(source, k8sconsts.DeleteInstrumentationConfigFinalizer) &&
		controllerutil.ContainsFinalizer(source, k8sconsts.StartLangDetectionFinalizer) {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("metadata").Child("finalizers"),
			source.Finalizers,
			"Source may only have one finalizer",
		))
	}

	if source.Labels[k8sconsts.WorkloadNameLabel] != source.Spec.Workload.Name {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("metadata").Child("labels"),
			source.Labels[k8sconsts.WorkloadNameLabel],
			"odigos.io/workload-name must match spec.workload.name",
		))
	}

	if source.Labels[k8sconsts.WorkloadNamespaceLabel] != source.Spec.Workload.Namespace {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("metadata").Child("labels"),
			source.Labels[k8sconsts.WorkloadNamespaceLabel],
			"odigos.io/workload-namespace must match spec.workload.namespace",
		))
	}

	if source.Labels[k8sconsts.WorkloadKindLabel] != string(source.Spec.Workload.Kind) {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("metadata").Child("labels"),
			source.Labels[k8sconsts.WorkloadKindLabel],
			"odigos.io/workload-kind must match spec.workload.kind",
		))
	}

	err := s.validateSourceUniqueness(ctx, source)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec").Child("workload"),
			source.Spec.Workload,
			err.Error(),
		))
	}

	if source.Spec.Workload.Kind == k8sconsts.WorkloadKindNamespace &&
		(source.Spec.Workload.Name != source.Spec.Workload.Namespace) {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec").Child("workload").Child("namespace"),
			source.Spec.Workload.Namespace,
			"namespace Source must have matching workload.name and workload.namespace",
		))
	}

	validKind := workload.IsValidWorkloadKind(source.Spec.Workload.Kind)
	if !validKind {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec").Child("workload").Child("kind"),
			source.Spec.Workload.Kind,
			"workload kind must be one of (Deployment, DaemonSet, StatefulSet, Namespace)",
		))
	}

	if source.Spec.Workload.Namespace != source.GetNamespace() {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec").Child("workload").Child("namespace"),
			source.Spec.Workload.Namespace,
			"Source namespace must match spec.workload.namespace",
		))
	}

	if source.Spec.Workload.Kind == k8sconsts.WorkloadKindNamespace && source.Spec.OtelServiceName != "" {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec").Child("otelServiceName"),
			source.Spec.OtelServiceName,
			"Service name is not valid for Namespace sources, only valid for Workload Sources",
		))
	}

	return allErrs
}

func (s *SourcesValidator) validateSourceUniqueness(ctx context.Context, source *v1alpha1.Source) error {
	sourceList := &v1alpha1.SourceList{}
	selector := labels.SelectorFromSet(labels.Set{
		k8sconsts.WorkloadNameLabel:      source.Labels[k8sconsts.WorkloadNameLabel],
		k8sconsts.WorkloadNamespaceLabel: source.Labels[k8sconsts.WorkloadNamespaceLabel],
		k8sconsts.WorkloadKindLabel:      source.Labels[k8sconsts.WorkloadKindLabel],
	})
	err := s.Client.List(ctx, sourceList, &client.ListOptions{LabelSelector: selector}, client.InNamespace(source.GetNamespace()))
	if err != nil {
		return err
	}
	if len(sourceList.Items) > 0 {
		duplicates := []string{}
		// In theory, there should only ever be at most 1 duplicate. But loop through all to be thorough
		for _, dupe := range sourceList.Items {
			// during an update, this source will show up as existing already
			if dupe.GetName() != source.GetName() {
				duplicates = append(duplicates, dupe.GetName())
			}
		}
		if len(duplicates) > 0 {
			return fmt.Errorf("duplicate source(s) exist for workload: %s", strings.Join(duplicates, ","))
		}
	}

	return nil
}
