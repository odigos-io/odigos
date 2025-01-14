package runtime_details

import (
	"context"
	"errors"
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	criwrapper "github.com/odigos-io/odigos/k8sutils/pkg/cri"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	kubeutils "github.com/odigos-io/odigos/odiglet/pkg/kube/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type instrumentationConfigPredicate struct{}

func (p *instrumentationConfigPredicate) Create(e event.CreateEvent) bool {
	obj, ok := e.Object.(*odigosv1.InstrumentationConfig)
	if !ok {
		return false
	}
	// we only care about new InstrumentationConfig objects
	// the event will be triggered also when odiglet starts, and the instrumentation config is created
	// in controller-runtime cache
	// checking the RuntimeDetailsByContainer map should filter cases where we do not want to trigger the reconcile
	// for workload which has already been reconciled previously
	return len(obj.Status.RuntimeDetailsByContainer) == 0
}

func (p *instrumentationConfigPredicate) Update(e event.UpdateEvent) bool {
	return false
}

func (p *instrumentationConfigPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (p *instrumentationConfigPredicate) Generic(e event.GenericEvent) bool {
	return false
}

type InstrumentationConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// the clientset is used to interact with the k8s API directly,
	// without pulling in specific objects into the controller runtime cache
	// which can be expensive (memory and CPU)
	Clientset *kubernetes.Clientset
	CriClient *criwrapper.CriClient
}

func (r *InstrumentationConfigReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {

	logger := log.FromContext(ctx)

	var instrumentationConfig odigosv1.InstrumentationConfig
	err := r.Get(ctx, request.NamespacedName, &instrumentationConfig)
	if err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	if len(instrumentationConfig.OwnerReferences) != 1 {
		return reconcile.Result{}, fmt.Errorf("InstrumentationConfig %s/%s has %d owner references, expected 1", instrumentationConfig.Namespace, instrumentationConfig.Name, len(instrumentationConfig.OwnerReferences))
	}

	workload, labels, err := getWorkloadAndLabelsfromOwner(ctx, r.Client, instrumentationConfig.Namespace, instrumentationConfig.OwnerReferences[0])
	if err != nil {
		logger.Error(err, "Failed to get workload and labels from owner")
		return reconcile.Result{}, err
	}

	pods, err := kubeutils.GetRunningPods(ctx, labels, workload.GetNamespace(), r.Client)
	if err != nil {
		return reconcile.Result{}, err
	}

	odigosConfig, err := k8sutils.GetCurrentOdigosConfig(ctx, r.Client)
	if err != nil {
		return k8sutils.K8SNoEffectiveConfigErrorHandler(err)
	}

	var selectedPods []corev1.Pod
	if len(pods) > 0 {
		selectedPods = append(selectedPods, pods[0])
	}

	runtimeResults, err := runtimeInspection(ctx, selectedPods, odigosConfig.IgnoredContainers, r.CriClient)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = persistRuntimeDetailsToInstrumentationConfig(ctx, r.Client, &instrumentationConfig, odigosv1.InstrumentationConfigStatus{
		RuntimeDetailsByContainer: runtimeResults,
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	logger.Info("Completed runtime detection for new instrumentation config", "namespace", request.Namespace, "name", request.Name)
	return reconcile.Result{}, nil
}

func getWorkloadAndLabelsfromOwner(ctx context.Context, k8sClient client.Client, ns string, ownerReference metav1.OwnerReference) (client.Object, map[string]string, error) {
	workloadName, workloadKind, err := workload.GetWorkloadFromOwnerReference(ownerReference)
	if err != nil {
		return nil, nil, err
	}

	switch workloadKind {
	case "Deployment":
		var dep appsv1.Deployment
		err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: workloadName}, &dep)
		if err != nil {
			return nil, nil, err
		}
		return &dep, dep.Spec.Selector.MatchLabels, nil
	case "DaemonSet":
		var ds appsv1.DaemonSet
		err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: workloadName}, &ds)
		if err != nil {
			return nil, nil, err
		}

		return &ds, ds.Spec.Selector.MatchLabels, nil
	case "StatefulSet":
		var sts appsv1.StatefulSet
		err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: workloadName}, &sts)
		if err != nil {
			return nil, nil, err
		}

		return &sts, sts.Spec.Selector.MatchLabels, nil
	}

	return nil, nil, errors.New("workload kind not supported")
}
