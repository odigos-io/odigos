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
	"errors"
	"strconv"

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

var supportedProbabilisticSignals = map[common.ObservabilitySignal]struct{}{
	common.TracesObservabilitySignal: {},
}

// DEPRECATED: Use odigosv1.Action instead
type ProbabilisticSamplerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *ProbabilisticSamplerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling ProbabilisticSampler action")
	logger.V(0).Info("WARNING: ProbabilisticSampler action is deprecated and will be removed in a future version. Migrate to odigosv1.Action instead.")

	action := &actionv1.ProbabilisticSampler{}
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

type ProbabilisticSamplerConfig struct {
	Value    float64 `json:"sampling_percentage"`
	HashSeed int     `json:"hash_seed"`
}

func probabilisticSamplerConfig(percentage string) (ProbabilisticSamplerConfig, error) {
	samplingPercentage, err := strconv.ParseFloat(percentage, 32)
	if err != nil {
		return ProbabilisticSamplerConfig{}, err
	}

	if samplingPercentage < 0 || samplingPercentage > 100 {
		return ProbabilisticSamplerConfig{}, errors.New("sampling percentage must be between 0 and 100")
	}

	return ProbabilisticSamplerConfig{Value: samplingPercentage, HashSeed: 123}, nil
}

func (r *ProbabilisticSamplerReconciler) createMigratedAction(action *actionv1.ProbabilisticSampler, migratedActionName string) *v1.Action {
	config := actionv1.ProbabilisticSamplerConfig{
		SamplingPercentage: action.Spec.SamplingPercentage,
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
			Samplers: &actionv1.SamplersConfig{
				ProbabilisticSampler: &config,
			},
		},
	}

	return odigosAction
}
