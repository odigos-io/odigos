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

type DeleteAttributeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *DeleteAttributeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling DeleteAttribute action")

	action := &actionv1.DeleteAttribute{}
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

func (r *DeleteAttributeReconciler) ReportReconciledToProcessorFailed(ctx context.Context, action *actionv1.DeleteAttribute, reason string, msg string) error {
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

func (r *DeleteAttributeReconciler) ReportReconciledToProcessor(ctx context.Context, action *actionv1.DeleteAttribute) error {
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

func (r *DeleteAttributeReconciler) convertToProcessor(action *actionv1.DeleteAttribute) (*v1.Processor, error) {

	config := TransformProcessorConfig{
		ErrorMode: "propagate", // deleting attributes is a security sensitive operation, so we should propagate errors
	}

	if action.Spec.Signals == nil {
		return nil, fmt.Errorf("Signals must be set")
	}

	ottlDeleteKeyStatements := make([]string, len(action.Spec.AttributeNamesToDelete))
	for i, attr := range action.Spec.AttributeNamesToDelete {
		ottlDeleteKeyStatements[i] = fmt.Sprintf("delete_key(attributes, \"%s\")", attr)
	}

	for _, signal := range action.Spec.Signals {
		switch signal {

		case common.LogsObservabilitySignal:
			config.LogStatements = []OttlStatementConfig{
				{
					Context:    "resource",
					Statements: ottlDeleteKeyStatements,
				},
				{
					Context:    "scope",
					Statements: ottlDeleteKeyStatements,
				},
				{
					Context:    "log",
					Statements: ottlDeleteKeyStatements,
				},
			}

		case common.MetricsObservabilitySignal:
			config.MetricStatements = []OttlStatementConfig{
				{
					Context:    "resource",
					Statements: ottlDeleteKeyStatements,
				},
				{
					Context:    "scope",
					Statements: ottlDeleteKeyStatements,
				},
				{
					Context:    "datapoint",
					Statements: ottlDeleteKeyStatements,
				},
			}

		case common.TracesObservabilitySignal:
			config.TraceStatements = []OttlStatementConfig{
				{
					Context:    "resource",
					Statements: ottlDeleteKeyStatements,
				},
				{
					Context:    "scope",
					Statements: ottlDeleteKeyStatements,
				},
				{
					Context:    "span",
					Statements: ottlDeleteKeyStatements,
				},
				{
					Context:    "spanevent",
					Statements: ottlDeleteKeyStatements,
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
			OrderHint:       -100, // since this is a security action, it should be applied as soon as possible
			ProcessorConfig: runtime.RawExtension{Raw: configJson},
		},
	}

	return &processor, nil
}
