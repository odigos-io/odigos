package instrumentationdevice

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
		if apierrors.IsNotFound(err) {
			// if there is no instrumented application, make sure the device is removed from the workload pod template manifest
			workloadName, workloadKind, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(instrumentedAppName)
			if err != nil {
				return err
			}
			err = removeInstrumentationDeviceFromWorkload(ctx, k8sClient, namespace, workloadKind, workloadName, ApplyInstrumentationDeviceReasonNoRuntimeDetails)
			return err
		} else {
			return err
		}
	}
	isNodeCollectorReady := isDataCollectionReady(ctx, k8sClient)

	return reconcileSingleWorkload(ctx, k8sClient, &instrumentedApplication, isNodeCollectorReady)
}
