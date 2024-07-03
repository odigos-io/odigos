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
	"encoding/json"
	"fmt"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	v1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"k8s.io/apimachinery/pkg/api/meta"
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

	processor, err := r.convertToProcessor(action)
	if err != nil {
		r.ReportReconciledToProcessorFailed(ctx, action, FailedToTransformToProcessorReason, err.Error())
		return ctrl.Result{}, err
	}

	err = r.Patch(ctx, processor, client.Apply, client.FieldOwner(action.Name), client.ForceOwnership)
	if err != nil {
		r.ReportReconciledToProcessorFailed(ctx, action, FailedToCreateProcessorReason, err.Error())
		return ctrl.Result{}, err
	}

	err = r.ReportReconciledToProcessor(ctx, action)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RenameAttributeReconciler) ReportReconciledToProcessorFailed(ctx context.Context, action *actionv1.RenameAttribute, reason string, msg string) error {
	changed := meta.SetStatusCondition(&action.Status.Conditions, metav1.Condition{
		Type:               ActionTransformedToProcessorType,
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            msg,
		ObservedGeneration: action.Generation,
	})

	if changed {
		err := r.Status().Update(ctx, action)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *RenameAttributeReconciler) ReportReconciledToProcessor(ctx context.Context, action *actionv1.RenameAttribute) error {
	changed := meta.SetStatusCondition(&action.Status.Conditions, metav1.Condition{
		Type:               ActionTransformedToProcessorType,
		Status:             metav1.ConditionTrue,
		Reason:             ProcessorCreatedReason,
		Message:            "The action has been reconciled to a processor resource.",
		ObservedGeneration: action.Generation,
	})

	if changed {
		err := r.Status().Update(ctx, action)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *RenameAttributeReconciler) convertToProcessor(action *actionv1.RenameAttribute) (*v1.Processor, error) {

	config := TransformProcessorConfig{
		ErrorMode: "ignore",
	}

	if action.Spec.Signals == nil {
		return nil, fmt.Errorf("Signals must be set")
	}

	// Every rename produces 2 OTTL statement
	ottlStatements := make([]string, 2*len(action.Spec.Renames))
	i := 0
	for from, to := range action.Spec.Renames {
		ottlStatements[i] = fmt.Sprintf("set(attributes[\"%s\"], attributes[\"%s\"])", to, from)
		ottlStatements[i+1] = fmt.Sprintf("delete_key(attributes, \"%s\")", from)
		i += 2
	}

	for _, signal := range action.Spec.Signals {
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

	configJson, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	processor := v1.Processor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "Processor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      action.Name,
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
		Spec: v1.ProcessorSpec{
			Type:            "transform",
			ProcessorName:   action.Spec.ActionName,
			Disabled:        action.Spec.Disabled,
			Notes:           action.Spec.Notes,
			Signals:         action.Spec.Signals,
			CollectorRoles:  []v1.CollectorsGroupRole{v1.CollectorsGroupRoleClusterGateway},
			OrderHint:       -50,
			ProcessorConfig: runtime.RawExtension{Raw: configJson},
		},
	}

	return &processor, nil
}
