package kube

import (
	"context"
	"errors"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/common/consts"
	"github.com/keyval-dev/odigos/common/utils"
	"github.com/keyval-dev/odigos/odiglet/pkg/ebpf"
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
	Scheme    *runtime.Scheme
	Directors map[common.ProgrammingLanguage]ebpf.Director
}

func (p *PodsReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	var pod corev1.Pod
	err := p.Client.Get(ctx, request.NamespacedName, &pod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			p.cleanup(request.NamespacedName)
			return ctrl.Result{}, nil
		}

		logger.Error(err, "error fetching pod object")
		return ctrl.Result{}, err
	}

	if !isPodInThisNode(&pod) {
		return ctrl.Result{}, nil
	}

	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		logger.Info("pod is not running, removing instrumentation")
		p.cleanup(request.NamespacedName)
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

	ebpfInstrumented, err := p.isEbpfInstrumented(ctx, podWorkload)
	if err != nil {
		logger.Error(err, "error checking if pod is ebpf instrumented")
		return ctrl.Result{}, err
	}
	if !ebpfInstrumented {
		p.cleanup(request.NamespacedName)
		return ctrl.Result{}, nil
	}

	if pod.Status.Phase == corev1.PodRunning {
		err := p.instrumentWithEbpf(ctx, &pod, podWorkload)
		if err != nil {
			logger.Error(err, "error instrumenting pod")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (p *PodsReconciler) cleanup(name types.NamespacedName) {
	// cleanup using all available directors
	// the Cleanup method is idempotent, so no harm in calling it multiple times
	for _, director := range p.Directors {
		director.Cleanup(name)
	}
}

func (p *PodsReconciler) instrumentWithEbpf(ctx context.Context, pod *corev1.Pod, podWorkload *PodWorkload) error {
	logger := log.FromContext(ctx)
	runtimeDetails, err := p.getRuntimeDetails(ctx, podWorkload)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Probably shutdown in progress, cleanup will be done as soon as the pod object is deleted
			return nil
		}
		return err
	}

	podUid := string(pod.UID)
	for _, container := range runtimeDetails.Spec.Languages {

		director := p.Directors[container.Language]
		if director == nil {
			return errors.New("no director found for language " + string(container.Language))
		}

		appName := container.ContainerName
		if len(runtimeDetails.Spec.Languages) == 1 && len(runtimeDetails.OwnerReferences) > 0 {
			appName = runtimeDetails.OwnerReferences[0].Name
		}

		details, err := process.FindAllInContainer(podUid, container.ContainerName)
		if err != nil {
			logger.Error(err, "error finding processes")
			return err
		}

		for _, d := range details {
			err = director.Instrument(d.ProcessID, types.NamespacedName{
				Namespace: pod.Namespace,
				Name:      pod.Name,
			}, appName)

			if err != nil {
				logger.Error(err, "error instrumenting process", "pid", d.ProcessID)
				return err
			}
		}
	}

	return nil
}

func (p *PodsReconciler) getRuntimeDetails(ctx context.Context, podWorkload *PodWorkload) (*odigosv1.InstrumentedApplication, error) {
	instrumentedApplicationName := utils.GetRuntimeObjectName(podWorkload.Name, podWorkload.Kind)

	var runtimeDetails odigosv1.InstrumentedApplication
	err := p.Client.Get(ctx, client.ObjectKey{
		Namespace: podWorkload.Namespace,
		Name:      instrumentedApplicationName,
	}, &runtimeDetails)
	if err != nil {
		return nil, err
	}

	return &runtimeDetails, nil
}

// PodWorkload represents the higher-level controller managing a specific Pod within a Kubernetes cluster.
// It contains essential details about the controller such as its Name, Namespace, and Kind.
// 'Kind' refers to the type of controller, which can be a Deployment, StatefulSet, or DaemonSet.
// This struct is useful for identifying and interacting with the overarching entity
// that governs the lifecycle and behavior of a Pod, especially in contexts where
// understanding the relationship between a Pod and its controlling workload is crucial.
type PodWorkload struct {
	Name      string
	Namespace string
	Kind      string
}

func (p *PodsReconciler) getPodWorkloadObject(ctx context.Context, pod *corev1.Pod) (*PodWorkload, error) {
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
					return &PodWorkload{
						Name:      rsOwner.Name,
						Namespace: pod.Namespace,
						Kind:      rsOwner.Kind,
					}, nil
				}
			}
		} else if owner.Kind == "DaemonSet" || owner.Kind == "Deployment" || owner.Kind == "StatefulSet" {
			return &PodWorkload{
				Name:      owner.Name,
				Namespace: pod.Namespace,
				Kind:      owner.Kind,
			}, nil
		}
	}

	// Pod does not necessarily have to be managed by a controller
	return nil, nil
}

func (p *PodsReconciler) isEbpfInstrumented(ctx context.Context, podWorkload *PodWorkload) (bool, error) {
	// TODO: this is better done with a dynamic client
	switch podWorkload.Kind {
	case "Deployment":
		var dep appsv1.Deployment
		err := p.Client.Get(ctx, client.ObjectKey{
			Namespace: podWorkload.Namespace,
			Name:      podWorkload.Name,
		}, &dep)
		return hasEbpfInstrumentationAnnotation(&dep), err
	case "DaemonSet":
		var ds appsv1.DaemonSet
		err := p.Client.Get(ctx, client.ObjectKey{
			Namespace: podWorkload.Namespace,
			Name:      podWorkload.Name,
		}, &ds)
		return hasEbpfInstrumentationAnnotation(&ds), err
	case "StatefulSet":
		var sts appsv1.StatefulSet
		err := p.Client.Get(ctx, client.ObjectKey{
			Namespace: podWorkload.Namespace,
			Name:      podWorkload.Name,
		}, &sts)
		return hasEbpfInstrumentationAnnotation(&sts), err
	default:
		return false, errors.New("unknown pod workload kind")
	}
}

func hasEbpfInstrumentationAnnotation(obj client.Object) bool {
	if obj == nil {
		return false
	}

	annotations := obj.GetAnnotations()
	if annotations == nil {
		return false
	}

	_, exists := annotations[consts.EbpfInstrumentationAnnotation]
	return exists
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
