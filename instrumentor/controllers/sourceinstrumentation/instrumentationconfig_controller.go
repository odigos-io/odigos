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

package sourceinstrumentation

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	sourceutils "github.com/odigos-io/odigos/k8sutils/pkg/source"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func getObjectByOwnerReference(ctx context.Context, k8sClient client.Client, ownerRef metav1.OwnerReference, namespace string) (client.Object, error) {
	key := client.ObjectKey{
		Name:      ownerRef.Name,
		Namespace: namespace,
	}

	if ownerRef.Kind == "Deployment" {
		dep := &appsv1.Deployment{}
		err := k8sClient.Get(ctx, key, dep)
		return dep, err
	}
	if ownerRef.Kind == "DaemonSet" {
		ds := &appsv1.DaemonSet{}
		err := k8sClient.Get(ctx, key, ds)
		return ds, err
	}
	if ownerRef.Kind == "StatefulSet" {
		ss := &appsv1.StatefulSet{}
		err := k8sClient.Get(ctx, key, ss)
		return ss, err
	}

	if ownerRef.Kind == "CronJob" {
		cj := &batchv1.CronJob{}
		err := k8sClient.Get(ctx, key, cj)
		return cj, err
	}

	return nil, fmt.Errorf("unsupported owner kind %s", ownerRef.Kind)
}

type InstrumentationConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *InstrumentationConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var instrumentationConfig odigosv1.InstrumentationConfig
	err := r.Client.Get(ctx, req.NamespacedName, &instrumentationConfig)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// find the workload object which is the owner of the InstrumentationConfig
	ownerReferences := instrumentationConfig.GetOwnerReferences()
	if len(ownerReferences) != 1 {
		logger.Info("InstrumentationConfig should have exactly one owner reference")
		return ctrl.Result{}, nil
	}
	workloadObject, err := getObjectByOwnerReference(ctx, r.Client, ownerReferences[0], req.Namespace)
	if err != nil {
		logger.Error(err, "error fetching owner object")
		return ctrl.Result{}, err
	}

	pw := k8sconsts.PodWorkload{
		Name:      workloadObject.GetName(),
		Namespace: workloadObject.GetNamespace(),
		Kind:      k8sconsts.WorkloadKind(workloadObject.GetObjectKind().GroupVersionKind().Kind),
	}
	sources, err := odigosv1.GetSources(ctx, r.Client, pw)

	enabled, _, err := sourceutils.IsObjectInstrumentedBySource(ctx, sources, err)
	if err != nil {
		return ctrl.Result{}, err
	}

	// check if the workload is enabled by deprecated labels,
	// once we fully remove the support for the instrumentation labels, we can remove this check
	enabledByDeprecatedLabels, err := workload.IsWorkloadInstrumentationEffectiveEnabled(ctx, r.Client, workloadObject)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !enabled && !enabledByDeprecatedLabels {
		logger.Info("Deleting instrumentationconfig for non-enabled workload")
		err := r.Client.Delete(ctx, &instrumentationConfig)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return ctrl.Result{}, nil
}
