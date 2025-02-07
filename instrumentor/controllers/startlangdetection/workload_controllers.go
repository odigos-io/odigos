package startlangdetection

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	sourceutils "github.com/odigos-io/odigos/k8sutils/pkg/source"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

type DeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileWorkload(ctx, r.Client, k8sconsts.WorkloadKindDeployment, req, r.Scheme)
}

type DaemonSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *DaemonSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileWorkload(ctx, r.Client, k8sconsts.WorkloadKindDaemonSet, req, r.Scheme)
}

type StatefulSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *StatefulSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileWorkload(ctx, r.Client, k8sconsts.WorkloadKindStatefulSet, req, r.Scheme)
}

func reconcileWorkload(ctx context.Context, k8sClient client.Client, objKind k8sconsts.WorkloadKind, req ctrl.Request, scheme *runtime.Scheme) (ctrl.Result, error) {
	obj := workload.ClientObjectFromWorkloadKind(objKind)
	err := getWorkloadObject(ctx, k8sClient, req, obj)
	if err != nil {
		// Deleted objects should be filtered in the event filter
		return ctrl.Result{}, err
	}

	enabled, reason, message, err := sourceutils.IsObjectInstrumentedBySource(ctx, k8sClient, obj)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !enabled {
		return ctrl.Result{}, nil
	}

	instConfigName := workload.CalculateWorkloadRuntimeObjectName(req.Name, objKind)
	err = requestOdigletsToCalculateRuntimeDetails(ctx, k8sClient, instConfigName, req.Namespace, obj, scheme)

	// update the status with the reason
	ic := odigosv1.InstrumentationConfig{}
	err = k8sClient.Get(ctx, types.NamespacedName{Name: instConfigName, Namespace: req.Namespace}, &ic)
	if err != nil {
		return ctrl.Result{}, err
	}

	cond := metav1.Condition{
		Type:    odigosv1.WorkloadInstrumentationStatusConditionType,
		Status:  metav1.ConditionTrue, // if instrumentation config is created, it is always instrumented.
		Reason:  string(reason),
		Message: message,
	}
	statuschanged := meta.SetStatusCondition(&ic.Status.Conditions, cond)
	if statuschanged {
		err = k8sClient.Status().Update(ctx, &ic)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, err
}

func getWorkloadObject(ctx context.Context, k8sClient client.Client, req ctrl.Request, obj client.Object) error {
	return k8sClient.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, obj)
}

func requestOdigletsToCalculateRuntimeDetails(ctx context.Context, k8sClient client.Client, instConfigName string, namespace string, obj client.Object, scheme *runtime.Scheme) error {
	logger := log.FromContext(ctx)
	instConfig := &odigosv1.InstrumentationConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "InstrumentationConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instConfigName,
			Namespace: namespace,
		},
	}

	serviceName, err := sourceutils.OtelServiceNameBySource(ctx, k8sClient, obj)
	if err != nil {
		return err
	}

	if serviceName != "" {
		instConfig.Spec.ServiceName = serviceName
	}

	if err := ctrl.SetControllerReference(obj, instConfig, scheme); err != nil {
		logger.Error(err, "Failed to set controller reference", "name", instConfigName, "namespace", namespace)
		return err
	}

	err = k8sClient.Create(ctx, instConfig)
	if err != nil {
		return client.IgnoreAlreadyExists(err)
	}

	logger.V(0).Info("Requested calculation of runtime details from odiglets", "name", instConfigName, "namespace", namespace)
	return nil
}
