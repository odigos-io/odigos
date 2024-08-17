package instrumentationdevice

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeploymentReconciler struct {
	client.Client
}

func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	instrumentedAppName := workload.CalculateWorkloadRuntimeObjectName(req.Name, workload.WorkloadKindDeployment)
	err := reconcileSingleInstrumentedApplicationByName(ctx, r.Client, instrumentedAppName, req.Namespace)
	return ctrl.Result{}, err
}

type DaemonSetReconciler struct {
	client.Client
}

func (r *DaemonSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	instrumentedAppName := workload.CalculateWorkloadRuntimeObjectName(req.Name, workload.WorkloadKindDaemonSet)
	err := reconcileSingleInstrumentedApplicationByName(ctx, r.Client, instrumentedAppName, req.Namespace)
	return ctrl.Result{}, err
}

type StatefulSetReconciler struct {
	client.Client
}

func (r *StatefulSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	instrumentedAppName := workload.CalculateWorkloadRuntimeObjectName(req.Name, workload.WorkloadKindStatefulSet)
	err := reconcileSingleInstrumentedApplicationByName(ctx, r.Client, instrumentedAppName, req.Namespace)
	return ctrl.Result{}, err
}

func reconcileSingleInstrumentedApplicationByName(ctx context.Context, k8sClient client.Client, instrumentedAppName string, namespace string) error {
	var instrumentedApplication odigosv1.InstrumentedApplication
	err := k8sClient.Get(ctx, types.NamespacedName{Name: instrumentedAppName, Namespace: namespace}, &instrumentedApplication)
	if err != nil {
		// changes in workload when there is no instrumented application is not interesting
		return client.IgnoreNotFound(err)
	}
	isNodeCollectorReady := isDataCollectionReady(ctx, k8sClient)
	return reconcileSingleWorkload(ctx, k8sClient, &instrumentedApplication, isNodeCollectorReady)
}
