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

package instrumentationconfig

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type InstrumentedApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *InstrumentedApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var ia odigosv1.InstrumentedApplication
	err := r.Client.Get(ctx, req.NamespacedName, &ia)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var ic odigosv1.InstrumentationConfig
	err = r.Client.Get(ctx, req.NamespacedName, &ic)
	if err != nil {
		// each InstrumentedApplication should have a corresponding InstrumentationConfig
		// but it might rarely happen that the InstrumentationConfig is deleted before the InstrumentedApplication
		if apierrors.IsNotFound(err) {
			logger.V(0).Info("Ignoring InstrumentedApplication without InstrumentationConfig", "runtime object name", ia.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	instrumentationRules := &odigosv1.InstrumentationRuleList{}
	err = r.Client.List(ctx, instrumentationRules)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	err = updateInstrumentationConfigForWorkload(&ic, &ia, instrumentationRules)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Client.Update(ctx, &ic)
	if err == nil {
		logger.V(0).Info("Updated instrumentation config", "workload", ia.Name)
	}
	return utils.K8SUpdateErrorHandler(err)
}
