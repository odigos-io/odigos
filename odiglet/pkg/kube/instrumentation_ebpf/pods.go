package instrumentation_ebpf

import (
	"context"
	"errors"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	kubeutils "github.com/odigos-io/odigos/odiglet/pkg/kube/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

func (p *PodsReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
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
	runtimeDetails, err := getRuntimeDetails(ctx, p.Client, podWorkload)
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
		if owner.Kind == "ReplicaSet" {
			var rs appsv1.ReplicaSet
			err := p.Client.Get(ctx, client.ObjectKey{
				Namespace: pod.Namespace,
				Name:      owner.Name,
			}, &rs)
			if err != nil {
				return nil, err
			}

			if rs.OwnerReferences == nil {
				return nil, errors.New("replicaset has no owner reference")
			}

			for _, rsOwner := range rs.OwnerReferences {
				if rsOwner.Kind == "Deployment" || rsOwner.Kind == "DaemonSet" || rsOwner.Kind == "StatefulSet" {
					return &common.PodWorkload{
						Name:      rsOwner.Name,
						Namespace: pod.Namespace,
						Kind:      rsOwner.Kind,
					}, nil
				}
			}
		} else if owner.Kind == "DaemonSet" || owner.Kind == "Deployment" || owner.Kind == "StatefulSet" {
			return &common.PodWorkload{
				Name:      owner.Name,
				Namespace: pod.Namespace,
				Kind:      owner.Kind,
			}, nil
		}
	}

	// Pod does not necessarily have to be managed by a controller
	return nil, nil
}
