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

	actionv1 "github.com/keyval-dev/odigos/api/odigos/action/v1alpha1"
	v1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type InsertClusterAttributesReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *InsertClusterAttributesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling InsertClusterAttributes action")

	action := &actionv1.InsertClusterAttributes{}
	err := r.Get(ctx, req.NamespacedName, action)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	processor, err := r.convertToProcessor(action)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Patch(ctx, processor, client.Apply, client.FieldOwner(action.Name), client.ForceOwnership)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

type insertclusterattributesAttributeConfig struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Action string `json:"action"`
}

type insertclusterattributesConfig struct {
	Attributes []insertclusterattributesAttributeConfig `json:"attributes"`
}

func (r *InsertClusterAttributesReconciler) convertToProcessor(action *actionv1.InsertClusterAttributes) (*v1.Processor, error) {

	config := insertclusterattributesConfig{
		Attributes: []insertclusterattributesAttributeConfig{},
	}
	for _, attr := range action.Spec.ClusterAttributes {
		if attr.AttributeStringValue == nil {
			continue
		}
		config.Attributes = append(config.Attributes, insertclusterattributesAttributeConfig{
			Key:    attr.AttributeName,
			Value:  *attr.AttributeStringValue,
			Action: "insert",
		})
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
			Type:            "resource",
			ProcessorName:   action.Spec.ActionName,
			Disabled:        action.Spec.Disabled,
			Notes:           action.Spec.Notes,
			Signals:         action.Spec.Signals,
			CollectorRoles:  []v1.CollectorsGroupRole{v1.CollectorsGroupRoleClusterGateway},
			OrderHint:       1, // it doesn't really matters the order, but better to have it in the beginning if downstream processors depend on it
			ProcessorConfig: runtime.RawExtension{Raw: configJson},
		},
	}

	return &processor, nil
}
