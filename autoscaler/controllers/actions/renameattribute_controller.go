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
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type RenameAttributeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *RenameAttributeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling RenameAttribute action")

	action := &actionv1.RenameAttribute{}
	err := r.Get(ctx, req.NamespacedName, action)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Migrate to odigosv1.Action
	migratedActionName := v1.ActionMigratedLegacyPrefix + action.Name
	odigosAction := &v1.Action{}
	err = r.Get(ctx, client.ObjectKey{Name: migratedActionName, Namespace: action.Namespace}, odigosAction)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		// Action doesn't exist, create new one
		odigosAction = r.createMigratedAction(action, migratedActionName)
		err = r.Create(ctx, odigosAction)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		// Action exists, update it
		updatedAction := r.updateMigratedAction(action, odigosAction)
		updatedAction.ResourceVersion = odigosAction.ResourceVersion
		err = r.Update(ctx, updatedAction)
		if err != nil {
			return utils.K8SUpdateErrorHandler(err)
		}
	}

	return ctrl.Result{}, nil
}

func renameAttributeConfig(cfg map[string]string, signals []common.ObservabilitySignal) (TransformProcessorConfig, error) {
	config := TransformProcessorConfig{
		ErrorMode: "ignore",
	}

	if signals == nil {
		return TransformProcessorConfig{}, fmt.Errorf("Signals must be set")
	}

	// Every rename produces 2 OTTL statement
	ottlStatements := make([]string, 2*len(cfg))
	i := 0
	for from, to := range cfg {
		ottlStatements[i] = fmt.Sprintf("set(attributes[\"%s\"], attributes[\"%s\"])", to, from)
		ottlStatements[i+1] = fmt.Sprintf("delete_key(attributes, \"%s\")", from)
		i += 2
	}

	for _, signal := range signals {
		switch signal {

		case common.LogsObservabilitySignal:
			config.LogStatements = []OttlStatementConfig{
				{
					Context:    "resource",
					Statements: ottlStatements,
				},
				{
					Context:    "scope",
					Statements: ottlStatements,
				},
				{
					Context:    "log",
					Statements: ottlStatements,
				},
			}

		case common.MetricsObservabilitySignal:
			config.MetricStatements = []OttlStatementConfig{
				{
					Context:    "resource",
					Statements: ottlStatements,
				},
				{
					Context:    "scope",
					Statements: ottlStatements,
				},
				{
					Context:    "datapoint",
					Statements: ottlStatements,
				},
			}

		case common.TracesObservabilitySignal:
			config.TraceStatements = []OttlStatementConfig{
				{
					Context:    "resource",
					Statements: ottlStatements,
				},
				{
					Context:    "scope",
					Statements: ottlStatements,
				},
				{
					Context:    "span",
					Statements: ottlStatements,
				},
				{
					Context:    "spanevent",
					Statements: ottlStatements,
				},
			}
		}
	}
	return config, nil
}

func (r *RenameAttributeReconciler) createMigratedAction(action *actionv1.RenameAttribute, migratedActionName string) *v1.Action {
	config := actionv1.RenameAttributeConfig{
		Renames: action.Spec.Renames,
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
			ActionName:      action.Spec.ActionName,
			Notes:           action.Spec.Notes,
			Disabled:        action.Spec.Disabled,
			Signals:         action.Spec.Signals,
			RenameAttribute: &config,
		},
	}

	return odigosAction
}

func (r *RenameAttributeReconciler) updateMigratedAction(action *actionv1.RenameAttribute, odigosAction *v1.Action) *v1.Action {
	odigosAction.Spec.Notes = action.Spec.Notes
	odigosAction.Spec.Disabled = action.Spec.Disabled
	odigosAction.Spec.Signals = action.Spec.Signals
	odigosAction.Spec.RenameAttribute = &actionv1.RenameAttributeConfig{
		Renames: action.Spec.Renames,
	}
	return odigosAction
}
