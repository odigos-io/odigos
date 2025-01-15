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

	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type StatefulSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *StatefulSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var ss appsv1.StatefulSet
	err := r.Get(ctx, req.NamespacedName, &ss)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		logger.Error(err, "error fetching statefulset object")
		return ctrl.Result{}, err
	}

	err = reconcileWorkloadObject(ctx, r.Client, &ss)
	return k8sutils.K8SUpdateErrorHandler(err)
}
