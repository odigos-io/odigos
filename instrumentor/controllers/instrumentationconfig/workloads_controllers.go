package instrumentationconfig

import (
	"context"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// These controllers handle update of the InstrumentationConfig's ServiceName
// whenever there are changes in the associated workloads (Deployments, DaemonSets, StatefulSets).

type DeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileWorkload(ctx, r.Client, workload.WorkloadKindDeployment, req)
}

type DaemonSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *DaemonSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileWorkload(ctx, r.Client, workload.WorkloadKindDaemonSet, req)
}

type StatefulSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *StatefulSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileWorkload(ctx, r.Client, workload.WorkloadKindStatefulSet, req)
}

func reconcileWorkload(ctx context.Context, k8sClient client.Client, objKind workload.WorkloadKind, req ctrl.Request) (ctrl.Result, error) {
	obj := workload.ClientObjectFromWorkloadKind(objKind)
	instConfigName := workload.CalculateWorkloadRuntimeObjectName(req.Name, objKind)

	if err := getWorkloadObject(ctx, k8sClient, req, obj); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	annotations := obj.GetAnnotations()
	reportedName := annotations[consts.OdigosReportedNameAnnotation]

	return createOrUpdateInstrumentationConfig(ctx, k8sClient, instConfigName, req.Namespace, reportedName)

}

func getWorkloadObject(ctx context.Context, k8sClient client.Client, req ctrl.Request, obj client.Object) error {
	return k8sClient.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, obj)
}

func createOrUpdateInstrumentationConfig(ctx context.Context, k8sClient client.Client, instConfigName, namespace string, serviceName string) (reconcile.Result, error) {
	logger := log.FromContext(ctx)

	instConfig := &odigosv1alpha1.InstrumentationConfig{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: instConfigName, Namespace: namespace}, instConfig)
	if err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	if instConfig.Spec.ServiceName != serviceName {
		instConfig.Spec.ServiceName = serviceName

		logger.Info("Updating InstrumentationConfig", "name", instConfigName, "namespace", namespace)
		err = k8sClient.Update(ctx, instConfig)
		return utils.K8SUpdateErrorHandler(err)
	}

	return reconcile.Result{}, nil
}
