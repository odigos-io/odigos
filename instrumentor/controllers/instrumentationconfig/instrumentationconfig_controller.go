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

	var ic odigosv1.InstrumentationConfig
	err := r.Client.Get(ctx, req.NamespacedName, &ic)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	instrumentationRules := &odigosv1.InstrumentationRuleList{}
	err = r.Client.List(ctx, instrumentationRules)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	conf, err := utils.GetCurrentOdigosConfiguration(ctx, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = updateInstrumentationConfigForWorkload(&ic, instrumentationRules, &conf)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Client.Update(ctx, &ic)
	if err == nil {
		logger.V(0).Info("Updated instrumentation config", "workload", ic.Name)
	}
	return utils.K8SUpdateErrorHandler(err)
}
