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
	"errors"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type EffectiveConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *EffectiveConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// When effective config changes, we need to reconcile ALL InstrumentationConfig objects
	allInstrumentationConfigs := odigosv1.InstrumentationConfigList{}
	err := r.Client.List(ctx, &allInstrumentationConfigs)
	if err != nil {
		return ctrl.Result{}, err
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

	// Process each InstrumentationConfig
	var allErrs error
	updatedCount := 0
	for _, ic := range allInstrumentationConfigs.Items {
		icName := types.NamespacedName{
			Name:      ic.Name,
			Namespace: ic.Namespace,
		}

		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			// Get the latest version
			latestIC := &odigosv1.InstrumentationConfig{}
			if err := r.Client.Get(ctx, icName, latestIC); err != nil {
				return err
			}

			// Apply the update logic
			if err := updateInstrumentationConfigForWorkload(latestIC, instrumentationRules, &conf); err != nil {
				return err
			}

			// Attempt to update
			return r.Client.Update(ctx, latestIC)
		})

		if err != nil {
			allErrs = errors.Join(allErrs, err)
		} else {
			updatedCount++
		}
	}

	if updatedCount > 0 {
		logger.V(0).Info("Updated instrumentation configs from effective config change", "count", updatedCount)
	}

	return ctrl.Result{}, allErrs
}
