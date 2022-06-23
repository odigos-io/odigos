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
	"errors"
	v1 "github.com/keyval-dev/odigos/api/v1alpha1"
	"github.com/keyval-dev/odigos/common/consts"
	"github.com/keyval-dev/odigos/instrumentor/patch"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	instAppOwnerKey   = ".metadata.controller"
	IgnoredNamespaces = []string{"kube-system", "local-path-storage", consts.DefaultNamespace}
)

// DeploymentReconciler reconciles a Deployment object
type DeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments/finalizers,verbs=update

// Reconcile is responsible for creating InstrumentedApplication objects for every Deployment.
// In addition, Reconcile patch the deployment according to the discovered language and keeps the `instrumented` field
// of InstrumentedApplication up to date with the deployment spec.
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

	instApps, err := r.getInstrumentedApps(ctx, &req)
	if err != nil {
		logger.Error(err, "error finding InstrumentedApp objects")
		return ctrl.Result{}, err
	}

	if len(instApps.Items) == 0 {
		if dep.Status.ReadyReplicas == 0 {
			logger.V(0).Info("not enough ready replicas, waiting for pods to be ready")
			return ctrl.Result{}, nil
		}

		instrumentedApp := v1.InstrumentedApplication{
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.Name,
				Namespace: req.Namespace,
			},
		}

		err = ctrl.SetControllerReference(&dep, &instrumentedApp, r.Scheme)
		if err != nil {
			logger.Error(err, "error creating InstrumentedApp object")
			return ctrl.Result{}, err
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

		return ctrl.Result{}, nil
	}

	if len(instApps.Items) > 1 {
		return ctrl.Result{}, errors.New("found more than one InstrumentedApp per deployment")
	}

	// If lang not detected yet - nothing to do
	instApp := instApps.Items[0]
	if len(instApp.Spec.Languages) == 0 || instApp.Status.LangDetection.Phase != v1.CompletedLangDetectionPhase {
		return ctrl.Result{}, nil
	}

	// if scheduled
	if instApp.Spec.CollectorAddr != "" {
		// Compute .status.instrumented field
		instrumneted, err := patch.IsInstrumented(&dep.Spec.Template, &instApp)
		if err != nil {
			logger.Error(err, "error computing instrumented status")
			return ctrl.Result{}, err
		}
		if instrumneted != instApp.Status.Instrumented {
			logger.V(0).Info("updating .status.instrumented", "instrumented", instrumneted)
			instApp.Status.Instrumented = instrumneted
			err = r.Status().Update(ctx, &instApp)
			if err != nil {
				logger.Error(err, "error computing instrumented status")
				return ctrl.Result{}, err
			}
		}

		// If not instrumented - patch deployment
		if !instrumneted {
			err = patch.ModifyObject(&dep.Spec.Template, &instApp)
			if err != nil {
				logger.Error(err, "error patching deployment")
				return ctrl.Result{}, err
			}

			err = r.Update(ctx, &dep)
			if err != nil {
				logger.Error(err, "error instrumenting application")
				return ctrl.Result{}, err
			}
		}
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

func (r *DeploymentReconciler) getInstrumentedApps(ctx context.Context, req *ctrl.Request) (*v1.InstrumentedApplicationList, error) {
	var instrumentedApps v1.InstrumentedApplicationList
	err := r.List(ctx, &instrumentedApps, client.InNamespace(req.Namespace), client.MatchingFields{instAppOwnerKey: req.Name})
	if err != nil {
		return nil, err
	}

	return &instrumentedApps, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Index InstrumentedApps by owner for fast lookup
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1.InstrumentedApplication{}, instAppOwnerKey, func(rawObj client.Object) []string {
		instApp := rawObj.(*v1.InstrumentedApplication)
		owner := metav1.GetControllerOf(instApp)
		if owner == nil {
			return nil
		}

		if owner.APIVersion != appsv1.SchemeGroupVersion.String() || owner.Kind != "Deployment" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Owns(&v1.InstrumentedApplication{}).
		Complete(r)
}
