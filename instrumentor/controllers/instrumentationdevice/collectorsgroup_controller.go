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
	"errors"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"

	"github.com/odigos-io/odigos/k8sutils/pkg/consts"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// CollectorsGroupReconciler is responsible for reconciling the instrumented workloads
// once the collectors group becomes ready - by adding the instrumentation device to the workloads.
// This is necessary to ensure that we won't instrument any workload before the
// node collectors are ready to receive the data.
type CollectorsGroupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type collectorsGroupReadyPredicate struct {
	predicate.Funcs
}

func (w collectorsGroupReadyPredicate) Create(e event.CreateEvent) bool {
	if e.Object == nil {
		return false
	}

	if e.Object.GetName() != consts.OdigosNodeCollectorCollectorGroupName {
		return false
	}

	cg, ok := e.Object.(*odigosv1.CollectorsGroup)
	if !ok {
		return false
	}

	return cg.Status.Ready
}

func (w collectorsGroupReadyPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	if e.ObjectNew.GetName() != consts.OdigosNodeCollectorCollectorGroupName {
		return false
	}

	oldCG, ok := e.ObjectOld.(*odigosv1.CollectorsGroup)
	if !ok {
		return false
	}

	newCG, ok := e.ObjectNew.(*odigosv1.CollectorsGroup)
	if !ok {
		return false
	}

	wasReady := oldCG.Status.Ready
	isReady := newCG.Status.Ready

	return !wasReady && isReady
}

func (w collectorsGroupReadyPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (w collectorsGroupReadyPredicate) Generic(e event.GenericEvent) bool {
	return false
}

func (r *CollectorsGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	isDataCollectionReady := isDataCollectionReady(ctx, r.Client)

	var instApps odigosv1.InstrumentedApplicationList
	if err := r.List(ctx, &instApps); err != nil {
		return ctrl.Result{}, err
	}

	logger.V(0).Info("Reconciling instrumented applications after node collectors group became ready", "count", len(instApps.Items))

	var reconcileErr error

	for _, runtimeDetails := range instApps.Items {
		var currentInstApp odigosv1.InstrumentedApplication
		err := r.Get(ctx, client.ObjectKey{Namespace: runtimeDetails.Namespace, Name: runtimeDetails.Name}, &currentInstApp)
		if apierrors.IsNotFound(err) {
			// the loop can take time, so the instrumented application might get deleted
			// in the meantime, so we ignore the error
			continue
		}

		if err != nil {
			reconcileErr = errors.Join(reconcileErr, err)
			continue
		}

		err = reconcileSingleWorkload(ctx, r.Client, &currentInstApp, isDataCollectionReady)
		if err != nil {
			reconcileErr = errors.Join(reconcileErr, err)
		}
	}

	return ctrl.Result{}, reconcileErr
}
