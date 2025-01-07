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

package main

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type SourcesDefaulter struct {
	client.Client
}

var _ webhook.CustomDefaulter = &SourcesDefaulter{}

func (p *SourcesDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	source, ok := obj.(*v1alpha1.Source)
	if !ok {
		return fmt.Errorf("expected a Source but got a %T", obj)
	}

	if source.Labels == nil {
		source.Labels = make(map[string]string)
	}

	if source.Labels[consts.WorkloadNameLabel] != source.Spec.Workload.Name ||
		source.Labels[consts.WorkloadNamespaceLabel] != source.Spec.Workload.Namespace ||
		source.Labels[consts.WorkloadKindLabel] != string(source.Spec.Workload.Kind) {

		source.Labels[consts.WorkloadNameLabel] = source.Spec.Workload.Name
		source.Labels[consts.WorkloadNamespaceLabel] = source.Spec.Workload.Namespace
		source.Labels[consts.WorkloadKindLabel] = string(source.Spec.Workload.Kind)
	}

	if !v1alpha1.IsWorkloadExcludedSource(source) &&
		source.DeletionTimestamp.IsZero() &&
		!controllerutil.ContainsFinalizer(source, consts.DeleteInstrumentationConfigFinalizer) {
		controllerutil.AddFinalizer(source, consts.DeleteInstrumentationConfigFinalizer)
	}
	if v1alpha1.IsWorkloadExcludedSource(source) &&
		source.DeletionTimestamp.IsZero() &&
		!controllerutil.ContainsFinalizer(source, consts.StartLangDetectionFinalizer) {
		controllerutil.AddFinalizer(source, consts.StartLangDetectionFinalizer)
	}

	return nil
}
