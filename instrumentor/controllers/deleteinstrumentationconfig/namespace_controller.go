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

	sourceutils "github.com/odigos-io/odigos/k8sutils/pkg/source"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type NamespaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("namespace reconcile - will delete instrumentation for workloads that are not enabled in this namespace", "namespace", req.Name)

	var ns corev1.Namespace
	err := r.Get(ctx, client.ObjectKey{Name: req.Name}, &ns)
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "error fetching namespace object")
		return ctrl.Result{}, err
	}

	if err := sourceutils.MigrateInstrumentationLabelToDisabledSource(ctx, r.Client, &ns, workload.WorkloadKindNamespace); err != nil {
		return k8sutils.K8SUpdateErrorHandler(err)
	}

	enabled, err := sourceutils.IsObjectInstrumentedBySource(ctx, r.Client, &ns)
	if err != nil {
		return ctrl.Result{}, err
	}
	if enabled {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, syncNamespaceWorkloads(ctx, r.Client, req)
}
