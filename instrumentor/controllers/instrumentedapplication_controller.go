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
	"fmt"
	v1 "github.com/keyval-dev/odigos/instrumentor/api/v1"
	"github.com/keyval-dev/odigos/instrumentor/consts"
	"github.com/keyval-dev/odigos/instrumentor/patch"
	"github.com/keyval-dev/odigos/instrumentor/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// InstrumentedApplicationReconciler reconciles a InstrumentedApplication object
type InstrumentedApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=observability.control.plane.keyval.dev,resources=instrumentedapplications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=observability.control.plane.keyval.dev,resources=instrumentedapplications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=observability.control.plane.keyval.dev,resources=instrumentedapplications/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the InstrumentedApplication object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
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

	ref, err := r.getReference(ctx, &instrumentedApp)
	if err != nil {
		logger.Error(err, "error fetching ref object")
		return ctrl.Result{}, err
	}

	if r.isLangDetected(&instrumentedApp) {
		if r.shouldInstrument(&instrumentedApp) {
			err = patch.AccordingToInstrumentationApp(ref.PodTemplateSpec(), &instrumentedApp)
			if err != nil {
				logger.Error(err, "error instrumenting application", "app",
					instrumentedApp.Spec.Ref.Name)
				return ctrl.Result{}, err
			}

			err = ref.Update(r.Client, ctx)
			if err != nil {
				logger.Error(err, "error instrumenting application", "app",
					instrumentedApp.Spec.Ref.Name)
				return ctrl.Result{}, err
			}

			instrumentedApp.Spec.Instrumented = true
			err = r.Update(ctx, &instrumentedApp)
			if err != nil {
				logger.Error(err, "error instrumenting application", "app",
					instrumentedApp.Spec.Ref.Name)
				return ctrl.Result{}, err
			}
		}
	}

	if r.shouldStartLangDetection(&instrumentedApp) {
		logger.V(0).Info("starting lang detection process")

		instrumentedApp.Status.LangDetection.Phase = v1.RunningLangDetectionPhase
		err = r.Status().Update(ctx, &instrumentedApp)
		if err != nil {
			logger.Error(err, "error updating instrument app status")
			return ctrl.Result{}, err
		}

		err = r.detectLanguage(ctx, &instrumentedApp, ref.PodTemplateSpec().Labels)
		if err != nil {
			logger.Error(err, "error detecting language")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *InstrumentedApplicationReconciler) shouldStartLangDetection(app *v1.InstrumentedApplication) bool {
	return app.Status.LangDetection.Phase == v1.PendingLangDetectionPhase
}

func (r *InstrumentedApplicationReconciler) shouldInstrument(app *v1.InstrumentedApplication) bool {
	// TODO: check requirments like destinations exists
	return !app.Spec.Instrumented && app.Spec.CollectorAddr != ""
}

func (r *InstrumentedApplicationReconciler) isLangDetected(app *v1.InstrumentedApplication) bool {
	return len(app.Spec.Languages) > 0
}

func (r *InstrumentedApplicationReconciler) detectLanguage(ctx context.Context, app *v1.InstrumentedApplication, labels map[string]string) error {
	pod, err := r.choosePods(ctx, labels, app.Spec.Ref.Namespace)
	if err != nil {
		return err
	}

	langDetectionPod := r.createLangDetectionPod(pod, app)
	err = r.Create(ctx, langDetectionPod)
	return err
}

func (r *InstrumentedApplicationReconciler) choosePods(ctx context.Context, labels map[string]string, namespace string) (*corev1.Pod, error) {
	var podList corev1.PodList
	err := r.List(ctx, &podList, client.MatchingLabels(labels), client.InNamespace(namespace))
	if err != nil {
		return nil, err
	}

	if len(podList.Items) == 0 {
		return nil, consts.PodsNotFoundErr
	}

	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodRunning {
			return &pod, nil
		}
	}

	return nil, consts.PodsNotFoundErr
}

func (r *InstrumentedApplicationReconciler) createLangDetectionPod(targetPod *corev1.Pod, instrumentedApp *v1.InstrumentedApplication) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-lang-detection-", targetPod.Name),
			Namespace:    utils.GetCurrentNamespace(),
			Annotations: map[string]string{
				consts.LangDetectionContainerAnnotationKey: "true",
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: consts.LangDetectorServiceAccount,
			Containers: []corev1.Container{
				{
					Name:  "lang-detector",
					Image: fmt.Sprintf("%s:%s", consts.LangDetectorContainer, utils.GetDetectorVersion()),
					Args: []string{
						fmt.Sprintf("--instrumented-app=%s", instrumentedApp.Name),
						fmt.Sprintf("--namespace=%s", utils.GetCurrentNamespace()),
						fmt.Sprintf("--pod-uid=%s", targetPod.UID),
						fmt.Sprintf("--container-names=%s", strings.Join(r.getContainerNames(targetPod), ",")),
					},
					SecurityContext: &corev1.SecurityContext{
						Capabilities: &corev1.Capabilities{
							Add: []corev1.Capability{"SYS_PTRACE"},
						},
					},
				},
			},
			RestartPolicy: "Never",
			NodeName:      targetPod.Spec.NodeName,
			HostPID:       true,
		},
	}
}

func (r *InstrumentedApplicationReconciler) getContainerNames(pod *corev1.Pod) []string {
	var result []string
	for _, c := range pod.Spec.Containers {
		result = append(result, c.Name)
	}

	return result
}

func (r *InstrumentedApplicationReconciler) getReference(ctx context.Context, app *v1.InstrumentedApplication) (*ReferencedApp, error) {
	key := client.ObjectKey{
		Namespace: app.Spec.Ref.Namespace,
		Name:      app.Spec.Ref.Name,
	}

	if app.Spec.Ref.Type == v1.DeploymentApplicationType {
		var dep appsv1.Deployment
		err := r.Get(ctx, key, &dep)
		if err != nil {
			return nil, err
		}

		return ReferenceFromDeployment(&dep), nil
	}

	var ss appsv1.StatefulSet
	err := r.Get(ctx, key, &ss)
	if err != nil {
		return nil, err
	}

	return ReferenceFromStatefulSet(&ss), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *InstrumentedApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.InstrumentedApplication{}).
		Complete(r)
}
