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
	"fmt"
	v1 "github.com/keyval-dev/odigos/instrumentor/api/v1"
	"github.com/keyval-dev/odigos/instrumentor/consts"
	"github.com/keyval-dev/odigos/instrumentor/utils"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var IgnoredNamespaces = []string{"kube-system", "local-path-storage", consts.DefaultNamespace}

// DeploymentReconciler reconciles a Deployment object
type DeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the Deployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	if r.shouldSkipDeployment(&req) {
		logger.V(5).Info("skipped deployment")
		return ctrl.Result{}, nil
	}

	var dep appsv1.Deployment
	err := r.Get(ctx, req.NamespacedName, &dep)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		logger.Error(err, "error fetching deployment object")
		return ctrl.Result{}, err
	}

	isObjectExists, err := r.isInstrumentedAppObjectExists(ctx, &req)
	if err != nil {
		logger.Error(err, "error finding if InstrumentedApp object exists")
		return ctrl.Result{}, err
	}

	if !isObjectExists {
		if dep.Status.ReadyReplicas == 0 {
			logger.V(0).Info("not enough ready replicas, waiting for pods to be ready")
			return ctrl.Result{}, nil
		}

		instrumentedApp := v1.InstrumentedApplication{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: fmt.Sprintf("%s-", req.Name),
				Namespace:    utils.GetCurrentNamespace(),
			},
			Spec: v1.InstrumentedApplicationSpec{
				Ref: v1.ApplicationReference{
					Type:      "deployment",
					Namespace: req.Namespace,
					Name:      req.Name,
				},
				Instrumented: false,
			},
		}
		err = r.Create(ctx, &instrumentedApp)
		if err != nil {
			logger.Error(err, "error creating InstrumentedApp object")
			return ctrl.Result{}, err
		}

		instrumentedApp.Status = v1.InstrumentedApplicationStatus{
			LangDetection: v1.LangDetectionStatus{
				Phase: v1.PendingLangDetectionPhase,
			},
		}
		err = r.Status().Update(ctx, &instrumentedApp)
		if err != nil {
			logger.Error(err, "error creating InstrumentedApp object")
		}

		logger.V(0).Info("requested language detection")
	}
	return ctrl.Result{}, nil
}

func (r *DeploymentReconciler) shouldSkipDeployment(req *ctrl.Request) bool {
	for _, ns := range IgnoredNamespaces {
		if req.Namespace == ns {
			return true
		}
	}

	return false
}

func (r *DeploymentReconciler) isInstrumentedAppObjectExists(ctx context.Context, req *ctrl.Request) (bool, error) {
	var instrumentedApps v1.InstrumentedApplicationList
	err := r.List(ctx, &instrumentedApps)

	if err != nil {
		return false, err
	}

	for _, app := range instrumentedApps.Items {
		ref := app.Spec.Ref
		if ref.Type == v1.DeploymentApplicationType &&
			ref.Name == req.Name &&
			ref.Namespace == req.Namespace {
			return true, nil
		}
	}

	return false, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Complete(r)
}
