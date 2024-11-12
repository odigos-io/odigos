package startlangdetection

import (
	"context"

	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// This reconiler is responsible for recalculating the instrumented application for potential changes of ignored container list,
// and trigger runtime detection.
type OdigosConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *OdigosConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// TODO: this logic can be improved by iterating over the instrumentation configs and marking the invalidate flag
	// we currently no changing this logic because all the runtime detection logic is still under development
	logger := log.FromContext(ctx)
	logger.V(0).Info("Odigos Configuration changed, recalculating instrumentated application for potential changes of ignored container list")

	var deps appsv1.DeploymentList
	err := r.Client.List(ctx, &deps)
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "error fetching deployments")
		return ctrl.Result{}, err
	}

	for _, dep := range deps.Items {
		err := r.reconcileUnDisabledFreshWorkload(ctx, workload.WorkloadKindDeployment, client.ObjectKey{Name: dep.Name, Namespace: dep.Namespace})
		if err != nil {
			logger.Error(err, "error requesting runtime details from odiglets", "name", dep.Name, "namespace", dep.Namespace)
		}
	}

	var sts appsv1.StatefulSetList
	err = r.Client.List(ctx, &sts)
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "error fetching statefulsets")
		return ctrl.Result{}, err
	}

	for _, st := range sts.Items {
		err := r.reconcileUnDisabledFreshWorkload(ctx, workload.WorkloadKindStatefulSet, client.ObjectKey{Name: st.Name, Namespace: st.Namespace})
		if err != nil {
			logger.Error(err, "error requesting runtime details from odiglets", "name", st.Name, "namespace", st.Namespace)
		}
	}

	var dss appsv1.DaemonSetList
	err = r.Client.List(ctx, &dss)
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "error fetching daemonsets")
		return ctrl.Result{}, err
	}

	for _, ds := range dss.Items {
		err := r.reconcileUnDisabledFreshWorkload(ctx, workload.WorkloadKindDaemonSet, client.ObjectKey{Name: ds.Name, Namespace: ds.Namespace})
		if err != nil {
			logger.Error(err, "error requesting runtime details from odiglets", "name", ds.Name, "namespace", ds.Namespace)
		}
	}

	return ctrl.Result{}, nil
}

func (r *OdigosConfigReconciler) reconcileUnDisabledFreshWorkload(ctx context.Context, kind workload.WorkloadKind, key client.ObjectKey) error {
	// it is very important that we make the changes based on a fresh copy of the workload object
	// if a list operation pulled in state and is now slowly iterating over it, we might be working with stale data
	freshWorkloadCopy := workload.ClientObjectFromWorkloadKind(kind)
	workloadGetErr := r.Client.Get(ctx, key, freshWorkloadCopy)
	if workloadGetErr != nil {
		if apierrors.IsNotFound(workloadGetErr) {
			// if the workload been deleted, we don't need to do anything
			return nil
		} else {
			return workloadGetErr
		}
	}

	var err error
	// a more accurate approach here might be to che
	// if workload.IsWorkloadInstrumentationEffectiveEnabled
	if !workload.IsInstrumentationDisabledExplicitly(freshWorkloadCopy) {
		req := ctrl.Request{NamespacedName: key}
<<<<<<< HEAD
		_, err = reconcileWorkload(ctx, r.Client, workload.ClientObjectFromWorkloadKind(kind), kind, req, r.Scheme)
=======
		_, err = reconcileWorkload(ctx, r.Client, kind, req, r.Scheme)
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
	}
	return err
}
