package kube

import (
	"context"
	"errors"
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/odiglet/pkg/inspectors"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	"github.com/keyval-dev/odigos/odiglet/pkg/process"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	errPodOwnerNotFound = errors.New("pod owner not found")
)

type PodsReconciler struct {
	kubeClient client.Client
}

func (p *PodsReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Logger.V(0).Info("Reconciling pod", "request", request)

	// Get pod
	pod := &corev1.Pod{}
	if err := p.kubeClient.Get(ctx, request.NamespacedName, pod); err != nil && client.IgnoreNotFound(err) != nil {
		log.Logger.Error(err, "Failed to get pod")
		return reconcile.Result{}, err
	}

	// Detect languages
	var containerResults []common.LanguageByContainer
	for _, c := range pod.Spec.Containers {
		processes, err := process.FindAllInContainer(string(pod.UID), c.Name)
		if err != nil {
			log.Logger.Error(err, "Failed to find processes")
			return reconcile.Result{}, err
		}

		processResults, processName := inspectors.DetectLanguage(processes)
		if len(processResults) > 0 {
			containerResults = append(containerResults, common.LanguageByContainer{
				ContainerName: c.Name,
				Language:      processResults[0],
				ProcessName:   processName,
			})
		}
	}

	log.Logger.V(0).Info("language detected", "name", request.Name, "namespace", request.Namespace, "results", containerResults)
	return reconcile.Result{}, nil
}

func (p *PodsReconciler) persistLanguageDetectionResults(pod *corev1.Pod, results []common.LanguageByContainer) error {
	// Get owner
	_, err := p.getPodRootOwner(pod)
	if err != nil {
		log.Logger.Error(err, "Failed to get pod root owner")
		return err
	}

	return nil
}

func (p *PodsReconciler) getInstrumentedApplication(ownerName string, ownerKind string, ns string) (*odigosv1.InstrumentedApplication, error) {
	// Get instrumented application
	ia := &odigosv1.InstrumentedApplication{}
	return ia, nil
}

func (p *PodsReconciler) getPodRootOwner(pod *corev1.Pod) (string, error) {
	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "DaemonSet" {
			return owner.Name, nil
		}

		if owner.Kind == "ReplicaSet" {
			// Get replica set
			replicaSet := &appsv1.ReplicaSet{}
			if err := p.kubeClient.Get(context.Background(), client.ObjectKey{
				Namespace: pod.Namespace,
				Name:      owner.Name,
			}, replicaSet); err != nil {
				return "", err
			}

			// Get owner of replica set
			for _, owner := range replicaSet.OwnerReferences {
				if owner.Kind == "Deployment" || owner.Kind == "StatefulSet" {
					return owner.Name, nil
				}
			}
		}
	}
	return "", errPodOwnerNotFound
}

func (p *PodsReconciler) InjectClient(c client.Client) error {
	p.kubeClient = c
	return nil
}
