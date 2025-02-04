package instrumentationdevice

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
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
	instrumentedAppName := workload.CalculateWorkloadRuntimeObjectName(req.Name, k8sconsts.WorkloadKindDeployment)
	err := reconcileSingleInstrumentedApplicationByName(ctx, r.Client, instrumentedAppName, req.Namespace)
	return utils.K8SUpdateErrorHandler(err)
}

type DaemonSetReconciler struct {
	client.Client
}

func (r *DaemonSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	instrumentedAppName := workload.CalculateWorkloadRuntimeObjectName(req.Name, k8sconsts.WorkloadKindDaemonSet)
	err := reconcileSingleInstrumentedApplicationByName(ctx, r.Client, instrumentedAppName, req.Namespace)
	return utils.K8SUpdateErrorHandler(err)
}

type StatefulSetReconciler struct {
	client.Client
}

func (r *StatefulSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	instrumentedAppName := workload.CalculateWorkloadRuntimeObjectName(req.Name, k8sconsts.WorkloadKindStatefulSet)
	err := reconcileSingleInstrumentedApplicationByName(ctx, r.Client, instrumentedAppName, req.Namespace)
	return utils.K8SUpdateErrorHandler(err)
}

func reconcileSingleInstrumentedApplicationByName(ctx context.Context, k8sClient client.Client, instrumentedAppName string, namespace string) error {
	var instrumentationConfig odigosv1.InstrumentationConfig
	err := k8sClient.Get(ctx, types.NamespacedName{Name: instrumentedAppName, Namespace: namespace}, &instrumentationConfig)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// if there is no instrumentation config, make sure the device is removed from the workload pod template manifest
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

	return reconcileSingleWorkload(ctx, k8sClient, &instrumentationConfig, isNodeCollectorReady)
}
