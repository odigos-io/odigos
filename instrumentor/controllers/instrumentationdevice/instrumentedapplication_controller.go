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
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// InstrumentedApplicationReconciler reconciles a InstrumentedApplication object
type InstrumentedApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=odigos.io,resources=instrumentedapplications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=odigos.io,resources=instrumentedapplications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=odigos.io,resources=instrumentedapplications/finalizers,verbs=update
//+kubebuilder:rbac:groups=odigos.io,resources=odigosconfigurations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods/status,verbs=get;update;patch

// Reconcile is responsible for instrumenting deployment/statefulset/daemonset. In order for instrumentation to happen two things must be true:
// 1. InstrumentedApplication must have at least one language specified
// 2. Data collection pods must be running (DataCollection CollectorsGroup .status.ready == true)
func (r *InstrumentedApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var runtimeDetails odigosv1.InstrumentedApplication
	err := r.Client.Get(ctx, req.NamespacedName, &runtimeDetails)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "error fetching instrumented application")
			return ctrl.Result{}, err
		}

		// runtime details deleted: remove instrumentation from resource requests
		err = removeInstrumentation(logger, ctx, r.Client, req.NamespacedName, UnInstrumentReasonNoRuntimeDetails)
		if err != nil {
			logger.Error(err, "error removing instrumentation")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	err = reconcileSingleInstrumentedApplication(ctx, r.Client, &runtimeDetails)
	return ctrl.Result{}, err
}

// this function is extracted so we can call it from other reconcilers like when odigos config changes
func reconcileSingleInstrumentedApplication(ctx context.Context, kubeClient client.Client, runtimeDetails *odigosv1.InstrumentedApplication) error {
	logger := log.FromContext(ctx)

	runtimeDetailsNamespacedName := client.ObjectKeyFromObject(runtimeDetails)

	if len(runtimeDetails.Spec.RuntimeDetails) == 0 {
		err := removeInstrumentation(logger, ctx, kubeClient, runtimeDetailsNamespacedName, UnInstrumentReasonNoRuntimeDetails)
		if err != nil {
			logger.Error(err, "error removing instrumentation")
			return err
		}

		return nil
	}

	if !isDataCollectionReady(ctx, kubeClient) {
		err := removeInstrumentation(logger, ctx, kubeClient, runtimeDetailsNamespacedName, UnInstrumentReasonDataCollectionNotReady)
		if err != nil {
			logger.Error(err, "error removing instrumentation")
			return err
		}
	} else {
		err := instrument(logger, ctx, kubeClient, runtimeDetails)
		if err != nil {
			logger.Error(err, "error instrumenting")
			return err
		}
	}

	return nil
}

func removeInstrumentation(logger logr.Logger, ctx context.Context, kubeClient client.Client, instrumentedApplicationName types.NamespacedName, reason UnInstrumentReason) error {
	name, kind, err := utils.GetTargetFromRuntimeName(instrumentedApplicationName.Name)
	if err != nil {
		return err
	}

	err = uninstrument(logger, ctx, kubeClient, instrumentedApplicationName.Namespace, name, kind, reason)
	if err != nil {
		logger.Error(err, "error removing instrumentation")
		return err
	}

	return nil
}
