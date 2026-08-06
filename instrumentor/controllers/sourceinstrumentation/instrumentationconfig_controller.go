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

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	sourceutils "github.com/odigos-io/odigos/k8sutils/pkg/source"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstrumentationConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *InstrumentationConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := commonlogger.FromContext(ctx)

	var instrumentationConfig odigosv1.InstrumentationConfig
	err := r.Client.Get(ctx, req.NamespacedName, &instrumentationConfig)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		// It's possible for an IC to be deleted by accident
		// This catches that case and syncs the workload to re-create the IC if necessary
		// The IC is essentially an operand of the instrumentor, so it should reconcile accidental deletions to recreate it.
		pw, pwErr := workload.ExtractWorkloadInfoFromRuntimeObjectName(req.Name, req.Namespace)
		if pwErr != nil {
			return ctrl.Result{}, nil
		}

		logger.Debug("instrumentation config deleted, syncing workload",
			"workload", pw.Name, "namespace", pw.Namespace, "kind", pw.Kind)
		return syncWorkload(ctx, r.Client, r.Scheme, pw)
	}

	pw, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(req.Name, req.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	sources, err := odigosv1.GetSources(ctx, r.Client, pw)
	enabled, _, err := sourceutils.IsObjectInstrumentedBySource(ctx, sources, err)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !enabled {
		logger.Info("Deleting instrumentationconfig for non-enabled workload")
		err := r.Client.Delete(ctx, &instrumentationConfig)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return ctrl.Result{}, nil
}
