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
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type InstrumentationConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *InstrumentationConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var instConfig odigosv1.InstrumentationConfig
	err := r.Client.Get(ctx, req.NamespacedName, &instConfig)
	if err != nil {

		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		// runtime details deleted: remove instrumentation from resource requests
		workloadName, workloadKind, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(req.Name)
		if err != nil {
			logger.Error(err, "error parsing workload info from runtime object name")
			return ctrl.Result{}, err
		}
		err = removeInstrumentationDeviceFromWorkload(ctx, r.Client, req.Namespace, workloadKind, workloadName, ApplyInstrumentationDeviceReasonNoRuntimeDetails)
		return utils.K8SUpdateErrorHandler(err)
	}

	isNodeCollectorReady := isDataCollectionReady(ctx, r.Client)
	err = reconcileSingleWorkload(ctx, r.Client, &instConfig, isNodeCollectorReady)
	return utils.K8SUpdateErrorHandler(err)
}
