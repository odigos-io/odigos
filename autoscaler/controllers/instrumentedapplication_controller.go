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

package controllers

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/autoscaler/controllers/datacollection"
	predicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstrumentationConfigReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	ImagePullSecrets []string
	OdigosVersion    string
}

func (r *InstrumentationConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	err := datacollection.Sync(ctx, r.Client, r.Scheme, r.ImagePullSecrets, r.OdigosVersion)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *InstrumentationConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&odigosv1.InstrumentationConfig{}).
		// this controller only cares about the instrumented application existence.
		// when it is created or removed, the node collector config map needs to be updated to scrape logs for it's pods.
		WithEventFilter(&predicate.ExistencePredicate{}).
		Complete(r)
}
