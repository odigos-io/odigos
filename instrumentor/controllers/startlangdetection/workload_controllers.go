package startlangdetection

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/odigos-io/odigos/instrumentor/controllers/utils"

	appsv1 "k8s.io/api/apps/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileWorkload(ctx, r.Client, &appsv1.Deployment{}, "Deployment", req, r.Scheme)
}

type DaemonSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *DaemonSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileWorkload(ctx, r.Client, &appsv1.DaemonSet{}, "DaemonSet", req, r.Scheme)
}

type StatefulSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *StatefulSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileWorkload(ctx, r.Client, &appsv1.StatefulSet{}, "StatefulSet", req, r.Scheme)
}

func reconcileWorkload(ctx context.Context, k8sClient client.Client, obj client.Object, objString string, req ctrl.Request, scheme *runtime.Scheme) (ctrl.Result, error) {
	instConfigName := workload.GetRuntimeObjectName(req.Name, objString)
	err := getWorkloadObject(ctx, k8sClient, req, obj)
	if err != nil {
		// Deleted objects should be filtered in the event filter
		return ctrl.Result{}, err
	}

	instrumented, err := utils.IsWorkloadInstrumentationEffectiveEnabled(ctx, k8sClient, obj)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !instrumented {
		return ctrl.Result{}, nil
	}

	err = requestOdigletsToCalculateRuntimeDetails(ctx, k8sClient, instConfigName, req.Namespace, obj, scheme)
	return ctrl.Result{}, err
}

func getWorkloadObject(ctx context.Context, k8sClient client.Client, req ctrl.Request, obj client.Object) error {
	return k8sClient.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, obj)
}

func requestOdigletsToCalculateRuntimeDetails(ctx context.Context, k8sClient client.Client, instConfigName string, namespace string, obj client.Object, scheme *runtime.Scheme) error {
	logger := log.FromContext(ctx)
	var instConfig odigosv1.InstrumentationConfig
	err := k8sClient.Get(ctx, types.NamespacedName{Name: instConfigName, Namespace: namespace}, &instConfig)
	if err != nil {
		if apierrors.IsNotFound(err) {
			instConfig = odigosv1.InstrumentationConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instConfigName,
					Namespace: namespace,
				},
				Spec: odigosv1.InstrumentationConfigSpec{
					Config: []odigosv1.WorkloadInstrumentationConfig{},
				},
			}

			if err = ctrl.SetControllerReference(obj, &instConfig, scheme); err != nil {
				logger.Error(err, "Failed to set controller reference", "name", instConfigName, "namespace", namespace)
				return err
			}

			err = k8sClient.Create(ctx, &instConfig)
			if err != nil {
				logger.Error(err, "Failed to create instrumentation config", "name", instConfigName, "namespace", namespace)
				return err
			} else {
				logger.V(0).Info("Requested language detection from odiglets", "name", instConfigName, "namespace", namespace)
				return nil
			}
		}

		logger.Error(err, "Failed to get instrumentation config", "name", instConfigName, "namespace", namespace)
		return err
	}

	// TODO(edenfed): Already exists - request recalculating language detection
	// Recalculation happens in three cases:
	// 1. Workload restarted / spec changed
	// 2. Instrumentation config changed
	// 3. Namespace labeled for instrumentation
	return nil
}
