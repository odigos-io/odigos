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

package instrumentationdevice

import (
	"context"

	"github.com/go-logr/logr"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/utils"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CollectorsGroupReconciler reconciles a CollectorsGroup object
type CollectorsGroupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=odigos.io,resources=collectorsgroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=odigos.io,resources=collectorsgroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=odigos.io,resources=collectorsgroups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the CollectorsGroup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *CollectorsGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	if isDataCollectionReady(ctx, r.Client) {
		logger.V(0).Info("data collection is ready, instrumenting selected applications")
		var instApps odigosv1.InstrumentedApplicationList
		if err := r.List(ctx, &instApps); err != nil {
			logger.Error(err, "failed to list InstrumentedApps")
			return ctrl.Result{}, err
		}

		for _, instApp := range instApps.Items {
			err := instrument(logger, ctx, r.Client, &instApp)
			if err != nil {
				logger.Error(err, "failed to instrument application", "application", instApp.Name, "namespace", instApp.Namespace)
				return ctrl.Result{}, err
			}
		}
	} else {
		err := r.removeAllInstrumentations(ctx, logger)
		if err != nil {
			logger.Error(err, "failed to remove instrumentations")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *CollectorsGroupReconciler) removeAllInstrumentations(ctx context.Context, logger logr.Logger) error {
	var instApps odigosv1.InstrumentedApplicationList
	if err := r.List(ctx, &instApps); err != nil {
		return err
	}

	for _, instApp := range instApps.Items {
		name, kind, err := utils.GetTargetFromRuntimeName(instApp.Name)
		if err != nil {
			return err
		}

		err = uninstrument(logger, ctx, r.Client, instApp.Namespace, name, kind, UnInstrumentReasonRemoveAll)
		if err != nil {
			logger.Error(err, "failed to remove instrumentation", "application", name, "namespace", instApp.Namespace, "kind", kind)
			return err
		}
	}

	return nil
}
