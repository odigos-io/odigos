package instrumentation_ebpf

import (
	"context"

	"github.com/odigos-io/odigos/common/consts"
	"gopkg.in/yaml.v3"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	runtime_details "github.com/odigos-io/odigos/odiglet/pkg/kube/runtime_details"
	kubeutils "github.com/odigos-io/odigos/odiglet/pkg/kube/utils"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type PodsReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Directors ebpf.DirectorsMap
}

func (p *PodsReconciler) isNamespaceIgnored(ctx context.Context, ns string) bool {
	var configMap v1.ConfigMap
	var odigosConfig common.OdigosConfiguration

	err := p.Client.Get(ctx, client.ObjectKey{Name: consts.OdigosConfigurationName, Namespace: env.GetCurrentNamespace()}, &configMap)
	if err != nil {
		return false
	}
	if err := yaml.Unmarshal([]byte(configMap.Data[consts.OdigosConfigurationFileName]), odigosConfig); err != nil {
		return false
	}

	ignoredNamespaces := odigosConfig.IgnoredNamespaces
	for _, ignoredNamespace := range ignoredNamespaces {
		if ignoredNamespace == ns {
			return true
		}
	}

	return false
}

func (p *PodsReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if request.Namespace == env.GetCurrentNamespace() || p.isNamespaceIgnored(ctx, request.Namespace) {
		return ctrl.Result{}, nil
	}

	var pod corev1.Pod
	err := p.Client.Get(ctx, request.NamespacedName, &pod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			cleanupEbpf(p.Directors, request.NamespacedName)
			return ctrl.Result{}, nil
		}

		logger.Error(err, "error fetching pod object")
		return ctrl.Result{}, err
	}

	if !kubeutils.IsPodInCurrentNode(&pod) {
		return ctrl.Result{}, nil
	}

	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		logger.Info("pod is not running, removing instrumentation")
		cleanupEbpf(p.Directors, request.NamespacedName)
		return ctrl.Result{}, nil
	}

	podWorkload, err := p.getPodWorkloadObject(ctx, &pod)
	if err != nil {
		logger.Error(err, "error getting pod workload object")
		return ctrl.Result{}, err
	}
	if podWorkload == nil {
		// pod is not managed by a controller
		return ctrl.Result{}, nil
	}

	if pod.Status.Phase == corev1.PodRunning {
		err, instrumentedEbpf := p.instrumentWithEbpf(ctx, &pod, podWorkload)
		if err != nil {
			logger.Error(err, "error instrumenting pod")
			cleanupEbpf(p.Directors, request.NamespacedName)
			return ctrl.Result{}, err
		} else if !instrumentedEbpf {
			cleanupEbpf(p.Directors, request.NamespacedName)
			return ctrl.Result{}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (p *PodsReconciler) instrumentWithEbpf(ctx context.Context, pod *corev1.Pod, podWorkload *common.PodWorkload) (error, bool) {
	runtimeDetails, err := runtime_details.GetRuntimeDetails(ctx, p.Client, podWorkload)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Probably shutdown in progress, cleanup will be done as soon as the pod object is deleted
			return nil, false
		}
		return err, false
	}

	return instrumentPodWithEbpf(ctx, pod, p.Directors, runtimeDetails, podWorkload)
}

func (p *PodsReconciler) getPodWorkloadObject(ctx context.Context, pod *corev1.Pod) (*common.PodWorkload, error) {
	for _, owner := range pod.OwnerReferences {
		name, kind, err := kubeutils.GetWorkloadNameFromOwnerReference(owner)
		if err != nil {
			if kubeutils.IsErrorKindNotSupported(err) {
				return nil, nil
			}

			return nil, err
		}

		return &common.PodWorkload{
			Name:      name,
			Kind:      kind,
			Namespace: pod.Namespace,
		}, nil
	}

	// Pod does not necessarily have to be managed by a controller
	return nil, nil
}
