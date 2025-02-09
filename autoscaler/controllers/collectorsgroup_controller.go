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

package controllers

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/autoscaler/controllers/datacollection"
	"github.com/odigos-io/odigos/autoscaler/controllers/gateway"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/version"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CollectorsGroupReconciler struct {
	client.Client
	Scheme               *runtime.Scheme
	ImagePullSecrets     []string
	OdigosVersion        string
	K8sVersion           *version.Version
	DisableNameProcessor bool
}

func (r *CollectorsGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling CollectorsGroup")

	err := gateway.Sync(ctx, r.Client, r.Scheme, r.ImagePullSecrets, r.OdigosVersion)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = datacollection.Sync(ctx, r.Client, r.Scheme, r.ImagePullSecrets, r.OdigosVersion, r.K8sVersion, r.DisableNameProcessor)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *CollectorsGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&odigosv1.CollectorsGroup{}).
		Owns(&appsv1.DaemonSet{}).  // in case the ds is deleted or modified for any reason, this will reconcile and recreate it
		Owns(&appsv1.Deployment{}). // in case the deployment is deleted or modified for any reason, this will reconcile and recreate it
		Owns(&corev1.ConfigMap{}).  // in case the configmap is deleted or modified for any reason, this will reconcile and recreate it
		// we assume everything in the collectorsgroup spec is the configuration for the collectors to generate.
		// thus, we need to monitor any change to the spec which is what the generation field is for.
		WithEventFilter(&predicate.GenerationChangedPredicate{}).
		Complete(r)
}
