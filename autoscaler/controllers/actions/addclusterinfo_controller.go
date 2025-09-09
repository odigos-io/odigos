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

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	v1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type AddClusterInfoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *AddClusterInfoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling AddClusterInfo action")

	action := &actionv1.AddClusterInfo{}
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
		updatedAction := r.createMigratedAction(action, migratedActionName)
		updatedAction.ResourceVersion = odigosAction.ResourceVersion
		err = r.Update(ctx, updatedAction)
		if err != nil {
			return utils.K8SUpdateErrorHandler(err)
		}
	}

	return ctrl.Result{}, nil
}

type addClusterInfoAttributeConfig struct {
	Key    string  `json:"key"`
	Value  *string `json:"value"`
	Action string  `json:"action"`
}

type addclusterinfoConfig struct {
	Attributes []addClusterInfoAttributeConfig `json:"attributes"`
}

func (r *AddClusterInfoReconciler) createMigratedAction(action *actionv1.AddClusterInfo, migratedActionName string) *v1.Action {
	config := actionv1.AddClusterInfoConfig{
		ClusterAttributes:       action.Spec.ClusterAttributes,
		OverwriteExistingValues: action.Spec.OverwriteExistingValues,
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
			ActionName:     action.Spec.ActionName,
			Notes:          action.Spec.Notes,
			Disabled:       action.Spec.Disabled,
			Signals:        action.Spec.Signals,
			AddClusterInfo: &config,
		},
	}

	return odigosAction
}

func addClusterInfoConfig(cfg []actionv1.OtelAttributeWithValue, overwriteExistingValues bool) addclusterinfoConfig {
	action := "insert"
	if overwriteExistingValues {
		action = "upsert"
	}
	config := addclusterinfoConfig{
		Attributes: []addClusterInfoAttributeConfig{},
	}
	for _, attr := range cfg {
		config.Attributes = append(config.Attributes, addClusterInfoAttributeConfig{
			Key:    attr.AttributeName,
			Value:  attr.AttributeStringValue,
			Action: action,
		})
	}
	return config
}
