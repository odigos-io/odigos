package startlangdetection

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
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
	return reconcileWorkload(ctx, r.Client, workload.WorkloadKindDeployment, req, r.Scheme)
}

type DaemonSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *DaemonSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileWorkload(ctx, r.Client, workload.WorkloadKindDaemonSet, req, r.Scheme)
}

type StatefulSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *StatefulSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileWorkload(ctx, r.Client, workload.WorkloadKindStatefulSet, req, r.Scheme)
}

func reconcileWorkload(ctx context.Context, k8sClient client.Client, objKind workload.WorkloadKind, req ctrl.Request, scheme *runtime.Scheme) (ctrl.Result, error) {
	obj := workload.ClientObjectFromWorkloadKind(objKind)
	instConfigName := workload.CalculateWorkloadRuntimeObjectName(req.Name, objKind)
	err := getWorkloadObject(ctx, k8sClient, req, obj)
	if err != nil {
		// Deleted objects should be filtered in the event filter
		return ctrl.Result{}, err
	}

	instrumented, err := workload.IsWorkloadInstrumentationEffectiveEnabled(ctx, k8sClient, obj)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !instrumented {
		// Check if a Source object exists for this workload
		sourceList, err := v1alpha1.GetSourceListForWorkload(ctx, k8sClient, obj)
		if err != nil {
			return ctrl.Result{}, err
		}
		if len(sourceList.Items) == 0 {
			return ctrl.Result{}, nil
		}
		// if this is explicitly excluded (ie, namespace instrumentation), skip
		for _, s := range sourceList.Items {
			if _, exists := s.Labels[consts.OdigosWorkloadExcludedLabel]; exists {
				return ctrl.Result{}, nil
			}
		}
	}

	err = requestOdigletsToCalculateRuntimeDetails(ctx, k8sClient, instConfigName, req.Namespace, obj, scheme)
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

	if err := ctrl.SetControllerReference(obj, instConfig, scheme); err != nil {
		logger.Error(err, "Failed to set controller reference", "name", instConfigName, "namespace", namespace)
		return err
	}

	err := k8sClient.Create(ctx, instConfig)
	if err != nil {
		return client.IgnoreAlreadyExists(err)
	}

	logger.V(0).Info("Requested calculation of runtime details from odiglets", "name", instConfigName, "namespace", namespace)
	return nil
}
