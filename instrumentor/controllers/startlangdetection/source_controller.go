package startlangdetection

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

type SourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *SourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling Source object", "name", req.Name, "namespace", req.Namespace)
	source := &v1alpha1.Source{}
	err := r.Get(ctx, req.NamespacedName, source)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	obj := workload.ClientObjectFromWorkloadKind(source.Spec.Workload.Kind)
	err = r.Client.Get(ctx, types.NamespacedName{Name: source.Spec.Workload.Name, Namespace: source.Spec.Workload.Namespace}, obj)
	if err != nil {
		// TODO: Deleted objects should be filtered in the event filter
		return ctrl.Result{}, err
	}
	instConfigName := workload.CalculateWorkloadRuntimeObjectName(source.Spec.Workload.Name, source.Spec.Workload.Kind)

	if source.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(source, consts.SourceFinalizer) {
			controllerutil.AddFinalizer(source, consts.SourceFinalizer)
			// Removed by deleteinstrumentedapplication controller
			controllerutil.AddFinalizer(source, consts.InstrumentedApplicationFinalizer)

			if source.Labels == nil {
				source.Labels = make(map[string]string)
			}
			source.Labels[consts.WorkloadNameLabel] = source.Spec.Workload.Name
			source.Labels[consts.WorkloadNamespaceLabel] = source.Spec.Workload.Namespace
			source.Labels[consts.WorkloadKindLabel] = string(source.Spec.Workload.Kind)

			if err := r.Update(ctx, source); err != nil {
				return k8sutils.K8SUpdateErrorHandler(err)
			}

			err = requestOdigletsToCalculateRuntimeDetails(ctx, r.Client, instConfigName, req.Namespace, obj, r.Scheme)
			return ctrl.Result{}, err
		}
	} else {
		// Source is being deleted
		if controllerutil.ContainsFinalizer(source, consts.SourceFinalizer) {

			instConfig := &v1alpha1.InstrumentationConfig{}
			err = r.Client.Get(ctx, types.NamespacedName{Name: instConfigName, Namespace: req.Namespace}, instConfig)
			if err != nil {
				if !apierrors.IsNotFound(err) {
					// report any real error so the controller can retry
					return ctrl.Result{}, err
				} // else, no need to delete the InstrumentationConfig, just continue to remove the finalizer
			} else {
				err = r.Client.Delete(ctx, instConfig)
				if err != nil {
					// report any real error so the controller can retry
					return ctrl.Result{}, err
				}
			}

			controllerutil.RemoveFinalizer(source, consts.SourceFinalizer)
			if err := r.Update(ctx, source); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}
	}

	return ctrl.Result{}, err
}
