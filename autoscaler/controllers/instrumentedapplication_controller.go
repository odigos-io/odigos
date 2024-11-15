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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/version"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// InstrumentedApplicationReconciler reconciles a InstrumentedApplication object
type InstrumentedApplicationReconciler struct {
	client.Client
	Scheme               *runtime.Scheme
	ImagePullSecrets     []string
	OdigosVersion        string
	K8sVersion           *version.Version
	DisableNameProcessor bool
}

//+kubebuilder:rbac:groups=odigos.io,resources=instrumentedapplications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=odigos.io,resources=instrumentedapplications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=odigos.io,resources=instrumentedapplications/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the InstrumentedApplication object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *InstrumentedApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling InstrumentedApps")
	err := datacollection.Sync(ctx, r.Client, r.Scheme, r.ImagePullSecrets, r.OdigosVersion, r.K8sVersion, r.DisableNameProcessor)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *InstrumentedApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&odigosv1.InstrumentedApplication{}).
		Complete(r)
}
