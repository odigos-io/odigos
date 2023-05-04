package kube

import (
	"context"
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/odiglet/pkg/inspectors"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	"github.com/keyval-dev/odigos/odiglet/pkg/process"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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

	// Find processes
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

	// Log container results
	log.Logger.V(0).Info("Container results", "results", containerResults)

	return reconcile.Result{}, nil
}

func (p *PodsReconciler) InjectClient(c client.Client) error {
	p.kubeClient = c
	return nil
}
