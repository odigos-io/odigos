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
	"regexp"
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
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
var defaultDataStreamLabel = k8sconsts.SourceDataStreamLabelPrefix + consts.DefaultDataStream

func (s *SourcesDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	source, ok := obj.(*v1alpha1.Source)
	if !ok {
		return fmt.Errorf("expected a Source but got a %T", obj)
	}

	if source.Labels == nil {
		source.Labels = make(map[string]string)
	}

	// Set the workload name label - use hash for regex patterns since Kubernetes labels
	// cannot contain regex special characters like *
	if !source.Spec.MatchWorkloadNameAsRegex {
		// For non-regex sources, use the exact workload name
		if _, ok := source.Labels[k8sconsts.WorkloadNameLabel]; !ok {
			source.Labels[k8sconsts.WorkloadNameLabel] = source.Spec.Workload.Name
		}
	}
	if _, ok := source.Labels[k8sconsts.WorkloadNamespaceLabel]; !ok {
		source.Labels[k8sconsts.WorkloadNamespaceLabel] = source.Spec.Workload.Namespace
	}
	if _, ok := source.Labels[k8sconsts.WorkloadKindLabel]; !ok {
		source.Labels[k8sconsts.WorkloadKindLabel] = string(source.Spec.Workload.Kind)
	}
	if !doesSourceHaveDataStreamLabel(source) {
		source.Labels[defaultDataStreamLabel] = "true"
	}

	// Remove old split finalizers
	if controllerutil.ContainsFinalizer(source, k8sconsts.StartLangDetectionFinalizer) {
		controllerutil.RemoveFinalizer(source, k8sconsts.StartLangDetectionFinalizer)
	}
	if controllerutil.ContainsFinalizer(source, k8sconsts.DeleteInstrumentationConfigFinalizer) {
		controllerutil.RemoveFinalizer(source, k8sconsts.DeleteInstrumentationConfigFinalizer)
	}

	if source.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(source, k8sconsts.SourceInstrumentationFinalizer) {
			controllerutil.AddFinalizer(source, k8sconsts.SourceInstrumentationFinalizer)
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
	if new.Spec.MatchWorkloadNameAsRegex != old.Spec.MatchWorkloadNameAsRegex {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec").Child("MatchWorkloadNameAsRegex"),
			new.Spec.MatchWorkloadNameAsRegex,
			"Source MatchWorkloadNameAsRegex is immutable",
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

	// When MatchWorkloadNameAsRegex is true, the label should be a hash of the regex pattern
	// (since Kubernetes labels cannot contain regex special characters like *)
	// When MatchWorkloadNameAsRegex is false, the label should match the exact workload name
	if !source.Spec.MatchWorkloadNameAsRegex {
		if source.Labels[k8sconsts.WorkloadNameLabel] != source.Spec.Workload.Name {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("metadata").Child("labels"),
				source.Labels[k8sconsts.WorkloadNameLabel],
				fmt.Sprintf("%s must match spec.workload.name", k8sconsts.WorkloadNameLabel),
			))
		}
	} else {
		// Validate that the regex pattern is valid
		_, err := regexp.Compile(source.Spec.Workload.Name)
		if err != nil {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec").Child("workload").Child("name"),
				source.Spec.Workload.Name,
				fmt.Sprintf("invalid regex pattern: %s", err.Error()),
			))
		}
	}

	if source.Labels[k8sconsts.WorkloadNamespaceLabel] != source.Spec.Workload.Namespace {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("metadata").Child("labels"),
			source.Labels[k8sconsts.WorkloadNamespaceLabel],
			fmt.Sprintf("%s must match spec.workload.namespace", k8sconsts.WorkloadNamespaceLabel),
		))
	}

	if source.Labels[k8sconsts.WorkloadKindLabel] != string(source.Spec.Workload.Kind) {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("metadata").Child("labels"),
			source.Labels[k8sconsts.WorkloadKindLabel],
			fmt.Sprintf("%s must match spec.workload.kind", k8sconsts.WorkloadKindLabel),
		))
	}

	if !doesSourceHaveDataStreamLabel(source) {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("metadata").Child("labels"),
			source.Labels[defaultDataStreamLabel],
			fmt.Sprintf("Source must have at least one %s* label to indicate a data stream group", k8sconsts.SourceDataStreamLabelPrefix),
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
			"workload kind must be one of (Deployment, DaemonSet, StatefulSet, Namespace, Rollout)",
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

	// MatchWorkloadNameAsRegex is not valid for namespace sources
	if source.Spec.Workload.Kind == k8sconsts.WorkloadKindNamespace && source.Spec.MatchWorkloadNameAsRegex {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec").Child("MatchWorkloadNameAsRegex"),
			source.Spec.MatchWorkloadNameAsRegex,
			"MatchWorkloadNameAsRegex is not valid for Namespace sources, only valid for Workload Sources",
		))
	}

	return allErrs
}

func (s *SourcesValidator) validateSourceUniqueness(ctx context.Context, source *v1alpha1.Source) error {
	sourceList := &v1alpha1.SourceList{}
	// For regex sources, we can't use exact label matching for uniqueness validation
	// Instead, we list all sources with matching namespace and kind, then check for conflicts
	selector := labels.SelectorFromSet(labels.Set{
		k8sconsts.WorkloadNamespaceLabel: source.Labels[k8sconsts.WorkloadNamespaceLabel],
		k8sconsts.WorkloadKindLabel:      source.Labels[k8sconsts.WorkloadKindLabel],
	})
	err := s.Client.List(ctx, sourceList, &client.ListOptions{LabelSelector: selector}, client.InNamespace(source.GetNamespace()))
	if err != nil {
		return err
	}
	if len(sourceList.Items) > 0 {
		duplicates := []string{}
		// Check for duplicates: exact match on workload name (when not using regex)
		// or overlapping regex patterns
		for _, dupe := range sourceList.Items {
			// during an update, this source will show up as existing already
			if dupe.GetName() == source.GetName() {
				continue
			}

			// For non-regex sources, check exact match
			if !source.Spec.MatchWorkloadNameAsRegex && !dupe.Spec.MatchWorkloadNameAsRegex {
				if source.Spec.Workload.Name == dupe.Spec.Workload.Name {
					duplicates = append(duplicates, dupe.GetName())
				}
			} else if source.Spec.MatchWorkloadNameAsRegex && !dupe.Spec.MatchWorkloadNameAsRegex {
				// Current source uses regex, check if it matches the other source's workload name
				matched, err := regexp.MatchString(source.Spec.Workload.Name, dupe.Spec.Workload.Name)
				if err == nil && matched {
					duplicates = append(duplicates, dupe.GetName())
				}
			} else if !source.Spec.MatchWorkloadNameAsRegex && dupe.Spec.MatchWorkloadNameAsRegex {
				// Other source uses regex, check if it matches current source's workload name
				matched, err := regexp.MatchString(dupe.Spec.Workload.Name, source.Spec.Workload.Name)
				if err == nil && matched {
					duplicates = append(duplicates, dupe.GetName())
				}
			}
			// If both use regex, we allow them (they might match different sets of workloads)
		}
		if len(duplicates) > 0 {
			return fmt.Errorf("duplicate source(s) exist for workload: %s", strings.Join(duplicates, ","))
		}
	}

	return nil
}

func doesSourceHaveDataStreamLabel(source *v1alpha1.Source) bool {
	for labelKey, labelValue := range source.Labels {
		if strings.HasPrefix(labelKey, k8sconsts.SourceDataStreamLabelPrefix) && labelValue == "true" {
			return true
		}
	}
	return false
}
