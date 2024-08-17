package sdkconfig

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"k8s.io/apimachinery/pkg/runtime"

	appsv1 "k8s.io/api/apps/v1"

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

func reconcileWorkload(ctx context.Context, k8sClient client.Client, obj client.Object, objKind string, req ctrl.Request, scheme *runtime.Scheme) (ctrl.Result, error) {
	instConfigName := workload.CalculateWorkloadRuntimeObjectName(req.Name, objKind)
	fmt.Println(instConfigName)
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
		// deleting odigos workload objects when instrumentation is not effective
		// is handled by the deleteinstrumentedapplication controllers
		// TODO: consider consolidating the logic here
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
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
		Spec: odigosv1.InstrumentationConfigSpec{
			RuntimeDetailsInvalidated: true,
		},
	}

	if err := ctrl.SetControllerReference(obj, instConfig, scheme); err != nil {
		logger.Error(err, "Failed to set controller reference", "name", instConfigName, "namespace", namespace)
		return err
	}

	instConfigBytes, _ := yaml.Marshal(instConfig)

	force := true
	patchOptions := client.PatchOptions{
		FieldManager: "instrumentor",
		Force:        &force,
	}

	logger.V(0).Info("Requested calculation of runtime details from odiglets", "name", instConfigName, "namespace", namespace)
	return k8sClient.Patch(ctx, instConfig, client.RawPatch(types.ApplyPatchType, instConfigBytes), &patchOptions)
}
