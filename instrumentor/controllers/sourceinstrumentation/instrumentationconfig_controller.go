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

	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstrumentationConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// syncs the instrumentation config (create / delete) in the followig scenarios:
// 1. when a source is deleted and the workload is no longer instrumented -> delete IC
// 2. when IC exists and workload is deleted -> delete IC
// 3. when IC is deleted accidentally when it should exist -> create IC
func (r *InstrumentationConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pw, pwErr := workload.ExtractWorkloadInfoFromRuntimeObjectName(req.Name, req.Namespace)
	if pwErr != nil {
		return ctrl.Result{}, nil
	}
	return syncWorkload(ctx, r.Client, r.Scheme, pw)
}
