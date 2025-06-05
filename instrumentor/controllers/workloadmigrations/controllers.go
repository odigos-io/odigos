package workloadmigrations

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

type NamespacesReconciler struct {
	client.Client
}

func (n *NamespacesReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	var ns corev1.Namespace
	err := n.Get(ctx, request.NamespacedName, &ns)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return migrateFromWorkload(ctx, n.Client, &ns, k8sconsts.WorkloadKindNamespace)
}

type DeploymentReconciler struct {
	client.Client
}

func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileWorkload(ctx, r.Client, k8sconsts.WorkloadKindDeployment, req)
}

type DaemonSetReconciler struct {
	client.Client
}

func (r *DaemonSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileWorkload(ctx, r.Client, k8sconsts.WorkloadKindDaemonSet, req)
}

type StatefulSetReconciler struct {
	client.Client
}

func (r *StatefulSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileWorkload(ctx, r.Client, k8sconsts.WorkloadKindStatefulSet, req)
}

func migrateFromWorkload(ctx context.Context, k8sClient client.Client, obj client.Object, objKind k8sconsts.WorkloadKind) (ctrl.Result, error) {
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

	annotations := obj.GetAnnotations()
	var serviceName string
	if annotations != nil {
		serviceName = annotations[consts.OdigosReportedNameAnnotation]
	}

	if disabled || labeled || serviceName != "" {
		err := createOrUpdateSourceForObject(ctx, logger, k8sClient, obj, objKind, disable, serviceName)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func reconcileWorkload(ctx context.Context, k8sClient client.Client, objKind k8sconsts.WorkloadKind, req ctrl.Request) (ctrl.Result, error) {
	obj := workload.ClientObjectFromWorkloadKind(objKind)
	err := k8sClient.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, obj)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return migrateFromWorkload(ctx, k8sClient, obj, objKind)
}

// createOrUpdateSourceForObject creates a Source for an object if one does not exist
// The created Source will have a randomly generated name and be in the object's Namespace.
// If the source is annotated with the "odigos.io/reported-name" annotation, the Source will have the same value in the OtelServiceName field.
func createOrUpdateSourceForObject(ctx context.Context,
	logger logr.Logger,
	k8sClient client.Client,
	obj client.Object,
	kind k8sconsts.WorkloadKind,
	disableInstrumentation bool,
	serviceNameFromWorkload string) error {
	if !workload.IsValidWorkloadKind(kind) {
		return fmt.Errorf("invalid workload kind %s", kind)
	}

	namespace := obj.GetNamespace()
	if namespace == "" && kind == k8sconsts.WorkloadKindNamespace {
		namespace = obj.GetName()
	}

	sources, err := v1alpha1.GetSources(ctx, k8sClient, obj)
	if err != nil {
		return err
	}
	var source *v1alpha1.Source

	if kind == k8sconsts.WorkloadKindNamespace {
		if sources.Namespace != nil {
			source = sources.Namespace
		}
	} else {
		if sources.Workload != nil {
			source = sources.Workload
		}
	}

	if source != nil {
		logger.Info("Source already exists for workload. Please remove deprecated odigos-instrumentation label and migrate to Source objects. Support for this will be entirely removed in a future version. See https://docs.odigos.io/pipeline/sources.", "workload", obj.GetName(), "kind", kind, "namespace", namespace, "source", source.Spec)
		return nil
	}

	source = &v1alpha1.Source{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: workload.CalculateWorkloadRuntimeObjectName(obj.GetName(), kind),
			Namespace:    namespace,
		},
		Spec: v1alpha1.SourceSpec{
			Workload: k8sconsts.PodWorkload{
				Name:      obj.GetName(),
				Namespace: namespace,
				Kind:      kind,
			},
		},
	}

	source.Spec.DisableInstrumentation = disableInstrumentation
	// migrate the reported name from the workload to the Source only if the Source is not a namespace Source
	// and the workload Source does not already have a service name.
	//
	// If a user have set the annotation on the workload we will use it in the source only if no reported name is set.
	// If that happens once, and the reported name was taken from the annotation,
	// any more changes to the annotation will not be reflected in the source.
	// This is valid, since the annotation is deprecated and we want to encourage users to use Source CR.
	if kind != k8sconsts.WorkloadKindNamespace && source.Spec.OtelServiceName == "" && serviceNameFromWorkload != "" {
		source.Spec.OtelServiceName = serviceNameFromWorkload
		logger.Info("legacy reported name annotation is deprecated; migrating to source OtelServiceName field",
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", kind,
			"serviceName", serviceNameFromWorkload)
	}

	logger.Info("creating source", "source", source.Spec)
	return k8sClient.Create(ctx, source)
}
