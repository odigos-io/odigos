package kube

import (
	"context"
	"errors"
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/odiglet/pkg/inspectors"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	"github.com/keyval-dev/odigos/odiglet/pkg/process"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

//func (p *PodsReconciler) persistLanguageDetectionResults(pod *corev1.Pod, results []common.LanguageByContainer) error {
//	// Get owner
//	owner, err := p.getPodRootOwner(pod)
//	if err != nil {
//		log.Logger.Error(err, "Failed to get pod root owner")
//		return err
//	}

//updatedIa := &odigosv1.InstrumentedApplication{
//	ObjectMeta: metav1.ObjectMeta{
//		Name:      p.getInstrumentedAppName(owner.Name, owner.Kind),
//		Namespace: pod.Namespace,
//		OwnerReferences: []metav1.OwnerReference{
//			*owner,
//		},
//	},
//}
//
//operationResult, err := controllerutil.CreateOrPatch(context.Background(), p.kubeClient, updatedIa, func() error {
//	updatedIa.Spec.Languages = results
//	return nil
//})
//
//if err != nil {
//	log.Logger.Error(err, "Failed to patch instrumented application")
//	return err
//}
//
//log.Logger.V(0).Info("instrumented application updated", "name", updatedIa.Name, "namespace", updatedIa.Namespace, "operation", operationResult)
//return nil
//ia, err := p.getInstrumentedApplication(ownerName, ownerKind, pod.Namespace)
//if err != nil {
//	if apierrors.IsNotFound(err) {
//		ia := &odigosv1.InstrumentedApplication{
//			ObjectMeta: metav1.ObjectMeta{
//				Name: p.getInstrumentedAppName(ownerName, ownerKind),
//			},
//			Spec: odigosv1.InstrumentedApplicationSpec{
//				Languages: results,
//			},
//		}
//
//		if err := p.kubeClient.Create(context.Background(), ia); err != nil {
//			log.Logger.Error(err, "Failed to create instrumented application")
//			return err
//		}
//	}
//
//	log.Logger.Error(err, "Failed to get instrumented application")
//	return err
//}
//
//// Patch instrumented application
//updatedIa := ia.DeepCopy()
//updatedIa.Spec.Languages = results
//if err := p.kubeClient.Patch(context.Background(), updatedIa, client.MergeFrom(ia)); err != nil {
//	log.Logger.Error(err, "Failed to patch instrumented application")
//	return err
//}
//}

//func (p *PodsReconciler) getInstrumentedApplication(ownerName string, ownerKind string, ns string) (*odigosv1.InstrumentedApplication, error) {
//	// Get instrumented application
//	ia := &odigosv1.InstrumentedApplication{}
//	name := p.getInstrumentedAppName(ownerName, ownerKind)
//	if err := p.kubeClient.Get(context.Background(), client.ObjectKey{
//		Namespace: ns,
//		Name:      name,
//	}, ia); err != nil {
//		return nil, err
//	}
//
//	return ia, nil
//}

func (p *PodsReconciler) getPodRootOwner(pod *corev1.Pod) (*metav1.OwnerReference, error) {
	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "DaemonSet" {
			return &owner, nil
		}

		if owner.Kind == "ReplicaSet" {
			// Get replica set
			replicaSet := &appsv1.ReplicaSet{}
			if err := p.kubeClient.Get(context.Background(), client.ObjectKey{
				Namespace: pod.Namespace,
				Name:      owner.Name,
			}, replicaSet); err != nil {
				return nil, err
			}

			// Get owner of replica set
			for _, owner := range replicaSet.OwnerReferences {
				if owner.Kind == "Deployment" || owner.Kind == "StatefulSet" {
					return &owner, nil
				}
			}
		}
	}
	return nil, errPodOwnerNotFound
}

func (p *PodsReconciler) InjectClient(c client.Client) error {
	p.kubeClient = c
	return nil
}
