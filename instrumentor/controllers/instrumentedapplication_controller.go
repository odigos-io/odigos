/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"errors"
	v1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	istioAnnotationKey     = "sidecar.istio.io/inject"
	istioAnnotationValue   = "false"
	linkerdAnnotationKey   = "linkerd.io/inject"
	linkerdAnnotationValue = "disabled"
)

var (
	podOwnerKey = ".metadata.controller"
	apiGVStr    = v1.SchemeGroupVersion.String()
)

// InstrumentedApplicationReconciler reconciles a InstrumentedApplication object
type InstrumentedApplicationReconciler struct {
	client.Client
	Scheme                 *runtime.Scheme
	LangDetectorTag        string
	LangDetectorImage      string
	DeleteLangDetectorPods bool
}

//+kubebuilder:rbac:groups=odigos.io,resources=instrumentedapplications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=odigos.io,resources=instrumentedapplications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=odigos.io,resources=instrumentedapplications/finalizers,verbs=update
//+kubebuilder:rbac:groups=odigos.io,resources=odigosconfigurations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods/status,verbs=get;update;patch

// Reconcile is responsible for language detection. The function starts the lang detection process if the InstrumentedApplication
// object does not have a languages field. In addition, Reconcile will clean up lang detection pods upon completion / error
func (r *InstrumentedApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	var instrumentedApp v1.InstrumentedApplication
	err := r.Get(ctx, req.NamespacedName, &instrumentedApp)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		logger.Error(err, "error fetching object")
		return ctrl.Result{}, err
	}

	// If language already detected - there is nothing to do
	if r.isLangDetected(&instrumentedApp) {
		return ctrl.Result{}, nil
	}

	// Language detection is in progress, check if lang detection pods finished
	if instrumentedApp.Status.LangDetection.Phase == v1.RunningLangDetectionPhase {
		var childPods corev1.PodList
		err = r.List(ctx, &childPods, client.InNamespace(req.Namespace), client.MatchingFields{podOwnerKey: req.Name})
		if err != nil {
			logger.Error(err, "could not find child pods")
			return ctrl.Result{}, err
		}
		for _, pod := range childPods.Items {
			// If pod finished -  read detection result
			if pod.Status.Phase == corev1.PodSucceeded && len(pod.Status.ContainerStatuses) > 0 {
				containerStatus := pod.Status.ContainerStatuses[0]
				if containerStatus.State.Terminated == nil {
					continue
				}

				// Write detection result
				result := containerStatus.State.Terminated.Message
				var detectionResult []common.LanguageByContainer
				err = json.Unmarshal([]byte(result), &detectionResult)
				if err != nil {
					logger.Error(err, "error parsing detection result")
					return ctrl.Result{}, err
				} else {
					instrumentedApp.Spec.Languages = detectionResult
					err = r.Update(ctx, &instrumentedApp)
					if err != nil {
						logger.Error(err, "error updating InstrumentedApp object with detection result")
						return ctrl.Result{}, err
					}

					instrumentedApp.Status.LangDetection.Phase = v1.CompletedLangDetectionPhase
					err = r.Status().Update(ctx, &instrumentedApp)
					if err != nil {
						logger.Error(err, "error updating InstrumentedApp status with detection result")
						return ctrl.Result{}, err
					}
				}
			} else if pod.Status.Phase == corev1.PodFailed {
				logger.V(0).Info("lang detection pod failed. marking as error")
				instrumentedApp.Status.LangDetection.Phase = v1.ErrorLangDetectionPhase
				err = r.Status().Update(ctx, &instrumentedApp)
				if err != nil {
					logger.Error(err, "error updating InstrumentedApp status")
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, nil
			}
		}
	}

	// Clean up finished pods
	if instrumentedApp.Status.LangDetection.Phase == v1.CompletedLangDetectionPhase ||
		instrumentedApp.Status.LangDetection.Phase == v1.ErrorLangDetectionPhase {
		var childPods corev1.PodList
		err = r.List(ctx, &childPods, client.InNamespace(req.Namespace), client.MatchingFields{podOwnerKey: req.Name})
		if err != nil {
			logger.Error(err, "could not find child pods")
			return ctrl.Result{}, err
		}

		for _, pod := range childPods.Items {
			if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
				if !r.DeleteLangDetectorPods {
					return ctrl.Result{}, nil
				}

				err = r.Client.Delete(ctx, &pod)
				if client.IgnoreNotFound(err) != nil {
					logger.Error(err, "failed to delete lang detection pod")
					return ctrl.Result{}, err
				}
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *InstrumentedApplicationReconciler) isLangDetected(app *v1.InstrumentedApplication) bool {
	return len(app.Spec.Languages) > 0
}

func (r *InstrumentedApplicationReconciler) getContainerNames(pod *corev1.Pod) []string {
	var result []string
	for _, c := range pod.Spec.Containers {
		if !r.skipContainer(c.Name) {
			result = append(result, c.Name)
		}
	}

	return result
}

func (r *InstrumentedApplicationReconciler) skipContainer(name string) bool {
	return name == "istio-proxy" || name == "linkerd-proxy"
}

func (r *InstrumentedApplicationReconciler) getOwnerTemplateLabels(ctx context.Context, instrumentedApp *v1.InstrumentedApplication) (map[string]string, error) {
	owner := metav1.GetControllerOf(instrumentedApp)
	if owner == nil {
		return nil, errors.New("could not find owner for InstrumentedApp")
	}

	if owner.Kind == "Deployment" && owner.APIVersion == appsv1.SchemeGroupVersion.String() {
		var dep appsv1.Deployment
		err := r.Get(ctx, client.ObjectKey{
			Namespace: instrumentedApp.Namespace,
			Name:      owner.Name,
		}, &dep)
		if err != nil {
			return nil, err
		}

		return dep.Spec.Template.Labels, nil
	} else if owner.Kind == "StatefulSet" && owner.APIVersion == appsv1.SchemeGroupVersion.String() {
		var ss appsv1.StatefulSet
		err := r.Get(ctx, client.ObjectKey{
			Namespace: instrumentedApp.Namespace,
			Name:      owner.Name,
		}, &ss)
		if err != nil {
			return nil, err
		}

		return ss.Spec.Template.Labels, nil
	}

	return nil, errors.New("unrecognized owner kind")
}

// SetupWithManager sets up the controller with the Manager.
func (r *InstrumentedApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Index pods by owner for fast lookup
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, podOwnerKey, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		owner := metav1.GetControllerOf(pod)
		if owner == nil {
			return nil
		}

		if owner.APIVersion != apiGVStr || owner.Kind != "InstrumentedApplication" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.InstrumentedApplication{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
