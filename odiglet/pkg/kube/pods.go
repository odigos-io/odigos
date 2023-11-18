package kube

import (
	"context"
	"errors"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/common/utils"
	"github.com/keyval-dev/odigos/odiglet/pkg/ebpf"
	"github.com/keyval-dev/odigos/odiglet/pkg/env"
	"github.com/keyval-dev/odigos/odiglet/pkg/process"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type PodsReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Director ebpf.Director
}

func (p *PodsReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	var pod corev1.Pod
	err := p.Client.Get(ctx, request.NamespacedName, &pod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			p.Director.Cleanup(request.NamespacedName)
			return ctrl.Result{}, nil
		}

		logger.Error(err, "error fetching pod object")
		return ctrl.Result{}, err
	}

	if !p.shouldInstrument(&pod) {
		return ctrl.Result{}, nil
	}

	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		logger.Info("pod is not running, removing instrumentation")
		p.Director.Cleanup(request.NamespacedName)
		return ctrl.Result{}, nil
	}

	if pod.Status.Phase == corev1.PodRunning {
		err = p.instrument(ctx, &pod)
		if err != nil {
			logger.Error(err, "error instrumenting pod")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// shouldInstrument returns true if the pod should be instrumented.
// A pod should be instrumented if:
// - it is running
// - it is scheduled on the same node as the odiglet
// - it has instrumentation.odigos.io/go device attached
func (p *PodsReconciler) shouldInstrument(pod *corev1.Pod) bool {
	return pod.Spec.NodeName == env.Current.NodeName && hasInstrumentationDevice(pod)
}

func (p *PodsReconciler) instrument(ctx context.Context, pod *corev1.Pod) error {
	logger := log.FromContext(ctx)
	containers, ownerName, err := p.findAllGoContainers(ctx, pod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Probably shutdown in progress, cleanup will be done as soon as the pod object is deleted
			return nil
		}

		logger.Error(err, "error finding go containers")
		return err
	}

	pids := make(map[int]string)
	for _, c := range containers {
		details, err := process.FindAllInContainer(string(pod.UID), c)
		if err != nil {
			logger.Error(err, "error finding processes")
			return err
		}

		for _, d := range details {
			appName := c
			if ownerName != "" {
				appName = ownerName
			}
			pids[d.ProcessID] = appName
		}
	}

	if len(pids) == 0 {
		// Probably shutdown in progress, cleanup will be done as soon as the pod object is deleted
		return nil
	}

	for pid, appName := range pids {
		err = p.Director.Instrument(ctx, pid, types.NamespacedName{
			Namespace: pod.Namespace,
			Name:      pod.Name,
		}, appName)

		if err != nil {
			logger.Error(err, "error instrumenting process", "pid", pid)
			return err
		}
	}
	return nil
}

func (p *PodsReconciler) findAllGoContainers(ctx context.Context, pod *corev1.Pod) ([]string, string, error) {
	runtimeDetails, err := p.getRuntimeDetails(ctx, pod)
	if err != nil {
		return nil, "", err
	}

	var containers []string
	for _, container := range runtimeDetails.Spec.Languages {
		if container.Language == common.GoProgrammingLanguage {
			containers = append(containers, container.ContainerName)
		}
	}

	ownerName := ""
	if len(runtimeDetails.Spec.Languages) == 1 && len(runtimeDetails.OwnerReferences) > 0 {
		ownerName = runtimeDetails.OwnerReferences[0].Name
	}
	return containers, ownerName, nil
}

func (p *PodsReconciler) getRuntimeDetails(ctx context.Context, pod *corev1.Pod) (*odigosv1.InstrumentedApplication, error) {
	name, err := p.getRuntimeDetailsName(ctx, pod)
	if err != nil {
		return nil, err
	}

	var runtimeDetails odigosv1.InstrumentedApplication
	err = p.Client.Get(ctx, client.ObjectKey{
		Namespace: pod.Namespace,
		Name:      name,
	}, &runtimeDetails)
	if err != nil {
		return nil, err
	}

	return &runtimeDetails, nil
}

func (p *PodsReconciler) getRuntimeDetailsName(ctx context.Context, pod *corev1.Pod) (string, error) {
	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "ReplicaSet" {
			var rs appsv1.ReplicaSet
			err := p.Client.Get(ctx, client.ObjectKey{
				Namespace: pod.Namespace,
				Name:      owner.Name,
			}, &rs)
			if err != nil {
				return "", err
			}

			if rs.OwnerReferences == nil {
				return "", errors.New("replicaset has no owner reference")
			}

			for _, rsOwner := range rs.OwnerReferences {
				if rsOwner.Kind == "Deployment" || rsOwner.Kind == "DaemonSet" || rsOwner.Kind == "StatefulSet" {
					return utils.GetRuntimeObjectName(rsOwner.Name, rsOwner.Kind), nil
				}
			}
		} else if owner.Kind == "DaemonSet" || owner.Kind == "Deployment" || owner.Kind == "StatefulSet" {
			return utils.GetRuntimeObjectName(owner.Name, owner.Kind), nil
		}
	}

	return "", errors.New("pod has no owner reference")
}

// / hasInstrumentationDevice returns true if the pod has go instrumentation device attached.
func hasInstrumentationDevice(pod *corev1.Pod) bool {
	for _, c := range pod.Spec.Containers {
		if c.Resources.Limits != nil {
			_, exists := c.Resources.Limits[corev1.ResourceName("instrumentation.odigos.io/go")]
			return exists
		}
	}

	return false
}
