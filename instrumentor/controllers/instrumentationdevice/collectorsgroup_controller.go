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

package instrumentationdevice

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CollectorsGroupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *CollectorsGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	isDataCollectionReady := isDataCollectionReady(ctx, r.Client)

	var instApps odigosv1.InstrumentedApplicationList
	if err := r.List(ctx, &instApps); err != nil {
		return ctrl.Result{}, err
	}

	for _, runtimeDetails := range instApps.Items {
		err := reconcileSingleWorkload(ctx, r.Client, &runtimeDetails, isDataCollectionReady)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}
