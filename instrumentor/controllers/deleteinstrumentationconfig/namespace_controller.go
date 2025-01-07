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

package deleteinstrumentationconfig

import (
	"context"
	"errors"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type NsLabelBecameDisabledPredicate struct{}

func (i *NsLabelBecameDisabledPredicate) Create(e event.CreateEvent) bool {
	// new namespace should start empty.
	// existing namespace inserted into cache on startup should not trigger this
	return false
}

func (i *NsLabelBecameDisabledPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	oldNs, ok := e.ObjectOld.(*corev1.Namespace)
	if !ok {
		return false
	}
	newNs, ok := e.ObjectNew.(*corev1.Namespace)
	if !ok {
		return false
	}

	// if the namespace was not labeled for instrumentation before, and now it is, we should not trigger
	// if the namespace was labeled for instrumentation before, and now it is not, we should trigger
	return workload.IsObjectLabeledForInstrumentation(oldNs) && !workload.IsObjectLabeledForInstrumentation(newNs)
}

func (i *NsLabelBecameDisabledPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (i *NsLabelBecameDisabledPredicate) Generic(e event.GenericEvent) bool {
	return false
}

type NamespaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("namespace reconcile - will delete instrumentation for workloads that are not labeled in this namespace", "namespace", req.Name)

	var ns corev1.Namespace
	err := r.Get(ctx, client.ObjectKey{Name: req.Name}, &ns)
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "error fetching namespace object")
		return ctrl.Result{}, err
	}
	// this reconciler handles cases where ns becomes uninstrumented,
	// and workloads that are not labeled and instrumented inherently from the ns should be deleted
	if err == nil && workload.IsObjectLabeledForInstrumentation(&ns) {
		return ctrl.Result{}, nil
	}

	// Because of cache settings in the controller, when namespace is unlabelled, it is appearing as not found
	var deps appsv1.DeploymentList
	err = r.Client.List(ctx, &deps, client.InNamespace(req.Name))
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "error fetching deployments")
		return ctrl.Result{}, err
	}

	for _, dep := range deps.Items {
		err := syncGenericWorkloadListToNs(ctx, r.Client, workload.WorkloadKindDeployment, client.ObjectKey{Namespace: dep.Namespace, Name: dep.Name})
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	var ss appsv1.StatefulSetList
	err = r.Client.List(ctx, &ss, client.InNamespace(req.Name))
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "error fetching statefulsets")
		return ctrl.Result{}, err
	}

	for _, s := range ss.Items {
		err := syncGenericWorkloadListToNs(ctx, r.Client, workload.WorkloadKindStatefulSet, client.ObjectKey{Namespace: s.Namespace, Name: s.Name})
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	var ds appsv1.DaemonSetList
	err = r.Client.List(ctx, &ds, client.InNamespace(req.Name))
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "error fetching daemonsets")
		return ctrl.Result{}, err
	}

	for _, d := range ds.Items {
		err := syncGenericWorkloadListToNs(ctx, r.Client, workload.WorkloadKindDaemonSet, client.ObjectKey{Namespace: d.Namespace, Name: d.Name})
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func syncGenericWorkloadListToNs(ctx context.Context, c client.Client, kind workload.WorkloadKind, key client.ObjectKey) error {
	// it is very important that we make the changes based on a fresh copy of the workload object
	// if a list operation pulled in state and is now slowly iterating over it, we might be working with stale data
	freshWorkloadCopy := workload.ClientObjectFromWorkloadKind(kind)
	workloadGetErr := c.Get(ctx, key, freshWorkloadCopy)
	if workloadGetErr != nil {
		if apierrors.IsNotFound(workloadGetErr) {
			// if the workload been deleted, we don't need to do anything
			return nil
		} else {
			return workloadGetErr
		}
	}

	var err error
	inheriting, err := isInheritingInstrumentationFromNs(ctx, c, freshWorkloadCopy)
	if err != nil {
		return err
	}
	if !inheriting {
		return nil
	}

	err = errors.Join(err, deleteWorkloadInstrumentationConfig(ctx, c, freshWorkloadCopy))
	err = errors.Join(err, removeReportedNameAnnotation(ctx, c, freshWorkloadCopy))
	return err
}

// this function indicates that the odigos instrumentation label is missing from the workload object manifest.
// when reconciling the namespace, the usecase is to delete instrumentation for workloads that were only
// instrumented due to the label on the namespace. These are workloads with the label missing.
// (they inherit the instrumentation from the namespace this way)
func isInheritingInstrumentationFromNs(ctx context.Context, c client.Client, obj client.Object) (bool, error) {
	sourceList, err := v1alpha1.GetSourceListForWorkload(ctx, c, obj)
	if err != nil {
		return false, err
	}

	if sourceList.Namespace != nil && sourceList.Namespace.DeletionTimestamp.IsZero() {
		return true, nil
	}

	if sourceList.Workload != nil && sourceList.Workload.DeletionTimestamp.IsZero() {
		return false, nil
	}

	labels := obj.GetLabels()
	if labels == nil {
		return true, nil
	}
	_, exists := labels[consts.OdigosInstrumentationLabel]
	return !exists, nil
}
