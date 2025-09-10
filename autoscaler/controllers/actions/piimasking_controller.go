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

package actions

import (
	"context"
	"fmt"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	v1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var piiMaskingSupportedSignals = map[common.ObservabilitySignal]struct{}{
	common.TracesObservabilitySignal: {},
}

// DEPRECATED: Use odigosv1.Action instead
type PiiMaskingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *PiiMaskingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling PiiMasking action")
	logger.V(0).Info("WARNING: PiiMasking action is deprecated and will be removed in a future version. Migrate to odigosv1.Action instead.")

	action := &actionv1.PiiMasking{}
	err := r.Get(ctx, req.NamespacedName, action)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Migrate to odigosv1.Action
	migratedActionName := v1.ActionMigratedLegacyPrefix + action.Name
	odigosAction := &v1.Action{}
	err = r.Get(ctx, client.ObjectKey{Name: migratedActionName, Namespace: action.Namespace}, odigosAction)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		logger.V(0).Info("Migrating legacy Action to odigosv1.Action. This is a one-way change, and modifications to the legacy Action will not be reflected in the migrated Action.")
		// Action doesn't exist, create new one
		odigosAction = r.createMigratedAction(action, migratedActionName)
		err = r.Create(ctx, odigosAction)
		return ctrl.Result{}, err
	}
	logger.V(0).Info("Migrated Action already exists, skipping update")
	return ctrl.Result{}, nil
}

type PiiMaskingConfig struct {
	AllowAllKeys  bool     `json:"allow_all_keys"`
	BlockedValues []string `json:"blocked_values"`
}

func piiMaskingConfig(cfg []actionv1.PiiCategory) (PiiMaskingConfig, error) {
	PiiCategories := cfg
	if len(PiiCategories) == 0 {
		return PiiMaskingConfig{}, fmt.Errorf("no PII categories are configured, so this processor is not needed")
	}

	// Allow all attributes to be traced. If set to false it removes all attributes not in allowed_keys which is all attributes
	config := PiiMaskingConfig{
		AllowAllKeys: true,
	}

	for _, piiCategory := range PiiCategories {
		switch piiCategory {
		case actionv1.CreditCardMasking:
			config.BlockedValues = append(config.BlockedValues, []string{
				"4[0-9]{12}(?:[0-9]{3})?", // Visa credit card number
				"(5[1-5][0-9]{14})",       // MasterCard number
			}...)
		}
	}

	return config, nil
}

func (r *PiiMaskingReconciler) createMigratedAction(action *actionv1.PiiMasking, migratedActionName string) *v1.Action {
	config := actionv1.PiiMaskingConfig{
		PiiCategories: action.Spec.PiiCategories,
	}

	odigosAction := &v1.Action{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "Action",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      migratedActionName,
			Namespace: action.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: action.APIVersion,
					Kind:       action.Kind,
					Name:       action.Name,
					UID:        action.UID,
				},
			},
		},
		Spec: v1.ActionSpec{
			ActionName: action.Spec.ActionName,
			Notes:      action.Spec.Notes,
			Disabled:   action.Spec.Disabled,
			Signals:    action.Spec.Signals,
			PiiMasking: &config,
		},
	}

	return odigosAction
}
