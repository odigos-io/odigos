package sourceinstrumentation

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/api/k8sconsts"
	sourceutils "github.com/odigos-io/odigos/k8sutils/pkg/source"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

type DeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileWorkload(ctx, r.Client, k8sconsts.WorkloadKindDeployment, req.NamespacedName, r.Scheme)
}

type DaemonSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *DaemonSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileWorkload(ctx, r.Client, k8sconsts.WorkloadKindDaemonSet, req.NamespacedName, r.Scheme)
}

type StatefulSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *StatefulSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileWorkload(ctx, r.Client, k8sconsts.WorkloadKindStatefulSet, req.NamespacedName, r.Scheme)
}

func reconcileWorkload(
	ctx context.Context,
	k8sClient client.Client,
	objKind k8sconsts.WorkloadKind,
	key client.ObjectKey,
	scheme *runtime.Scheme) (ctrl.Result, error) {

	obj := workload.ClientObjectFromWorkloadKind(objKind)
	err := k8sClient.Get(ctx, key, obj)
	if err != nil {
		return ctrl.Result{}, err
	}

	var reconcileFunc reconcileFunction
	enabled, _, err := sourceutils.IsObjectInstrumentedBySource(ctx, k8sClient, obj)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if enabled {
		reconcileFunc = instrumentWorkload
	} else {
		reconcileFunc = uninstrumentWorkload
	}

	podWorkload := k8sconsts.PodWorkload{
		Name:      key.Name,
		Namespace: key.Namespace,
		Kind:      objKind,
	}

	res, err := reconcileFunc(
		ctx,
		k8sClient,
		podWorkload,
		scheme)
	if err != nil {
		return res, err
	}

	return ctrl.Result{}, nil
}
