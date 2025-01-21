package labelmigration

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

type NamespacesReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (n *NamespacesReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	var ns corev1.Namespace
	err := n.Get(ctx, request.NamespacedName, &ns)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return migrateObject(ctx, n.Client, &ns, workload.WorkloadKindNamespace)
}

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

func migrateObject(ctx context.Context, k8sClient client.Client, obj client.Object, objKind workload.WorkloadKind) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	disable := false
	disabled := workload.IsInstrumentationDisabledExplicitly(obj)
	if disabled {
		logger.Info("legacy instrumentation label is deprecated; excluding source for workload",
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objKind)
		disable = true
	}

	labeled := workload.IsObjectLabeledForInstrumentation(obj)
	if labeled {
		logger.Info("legacy instrumentation label is deprecated; creating source for workload",
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objKind)
		disable = false
	}

	if disabled || labeled {
		err := CreateOrUpdateSourceForObject(ctx, k8sClient, obj, objKind, disable)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func reconcileWorkload(ctx context.Context, k8sClient client.Client, objKind workload.WorkloadKind, req ctrl.Request, scheme *runtime.Scheme) (ctrl.Result, error) {
	obj := workload.ClientObjectFromWorkloadKind(objKind)
	err := k8sClient.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, obj)
	if err != nil {
		// Deleted objects should be filtered in the event filter
		return ctrl.Result{}, err
	}

	return migrateObject(ctx, k8sClient, obj, objKind)
}

// CreateOrUpdateSourceForObject creates a Source for an object if one does not exist
// The created Source will have a randomly generated name and be in the object's Namespace.
func CreateOrUpdateSourceForObject(ctx context.Context,
	k8sClient client.Client,
	obj client.Object,
	kind workload.WorkloadKind,
	disableInstrumentation bool) error {
	if !workload.IsValidWorkloadKind(kind) {
		return fmt.Errorf("invalid workload kind %s", kind)
	}

	namespace := obj.GetNamespace()
	if namespace == "" && kind == workload.WorkloadKindNamespace {
		namespace = obj.GetName()
	}

	sources, err := v1alpha1.GetSources(ctx, k8sClient, obj)
	if err != nil {
		return err
	}
	var source *v1alpha1.Source

	if kind == workload.WorkloadKindNamespace {
		if sources.Namespace != nil {
			source = sources.Namespace
		}
	} else {
		if sources.Workload != nil {
			source = sources.Workload
		}
	}

	create := false
	if source == nil {
		create = true
		source = &v1alpha1.Source{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: workload.CalculateWorkloadRuntimeObjectName(obj.GetName(), kind),
				Namespace:    namespace,
			},
			Spec: v1alpha1.SourceSpec{
				Workload: workload.PodWorkload{
					Name:      obj.GetName(),
					Namespace: namespace,
					Kind:      kind,
				},
			},
		}
	}
	source.Spec.DisableInstrumentation = disableInstrumentation

	if create {
		log.FromContext(ctx).Info("creating source", "source", source.Spec)
		return k8sClient.Create(ctx, source)
	}
	log.FromContext(ctx).Info("updating source", "source", source.Spec)
	return k8sClient.Update(ctx, source)
}
