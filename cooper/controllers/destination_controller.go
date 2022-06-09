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
	"github.com/go-logr/logr"
	"github.com/keyval-dev/odigos/cooper/controllers/collectorconfig"
	"github.com/keyval-dev/odigos/cooper/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/keyval-dev/odigos/cooper/api/v1"
)

var commonLabels = map[string]string{
	utils.CollectorLabel: "true",
}

// DestinationReconciler reconciles a Destination object
type DestinationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=observability.control.plane.keyval.dev,resources=destinations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=observability.control.plane.keyval.dev,resources=destinations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=observability.control.plane.keyval.dev,resources=destinations/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the Destination object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *DestinationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var dest v1.Destination
	err := r.Get(ctx, req.NamespacedName, &dest)
	if err != nil {
		err = client.IgnoreNotFound(err)
		if err != nil {
			logger.Error(err, "error fetching destination")
		}
		return ctrl.Result{}, err
	}

	collectors, err := r.listCollectors(ctx)
	if err != nil {
		logger.Error(err, "error getting existing collectors")
		return ctrl.Result{}, err
	}

	if len(collectors) == 0 {
		logger.V(0).Info("no running collectors, creating new one")
		createdCol, err := r.createCollectors(ctx, &dest, &logger)
		if err != nil {
			logger.Error(err, "error creating new collector")
			return ctrl.Result{}, err
		}

		collectors = createdCol
		logger.V(0).Info("finished setting up a new collector")
	} else {
		err = r.updateExistingCollectors(&dest)
		if err != nil {
			logger.Error(err, "failed updating existing collectors with destination change")
			return ctrl.Result{}, err
		}
	}

	// TODO: move to pod controller
	//err = r.scheduleAppsToCollectors(collectors)
	//if err != nil {
	//	logger.Error(err, "failed scheduling apps to collectors")
	//	return ctrl.Result{}, err
	//}

	return ctrl.Result{}, nil
}

func (r *DestinationReconciler) listCollectors(ctx context.Context) ([]corev1.Pod, error) {
	var podList corev1.PodList
	err := r.List(ctx, &podList, client.MatchingLabels(commonLabels), client.InNamespace(utils.GetCurrentNamespace()))
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}

func (r *DestinationReconciler) createConfigMap(ctx context.Context, dest *v1.Destination) (*corev1.ConfigMap, error) {
	cm, err := collectorconfig.GetConfigForCollector(dest)
	if err != nil {
		return nil, err
	}

	// Create new configmap
	configmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.CommonConfigMapName,
			Namespace: utils.GetCurrentNamespace(),
			Labels:    commonLabels,
		},
		Data: map[string]string{
			"collector-conf": cm,
		},
	}

	err = r.Create(ctx, configmap)
	if err != nil {
		return nil, err
	}

	return configmap, nil
}

func (r *DestinationReconciler) createCollectors(ctx context.Context, dest *v1.Destination, logger *logr.Logger) ([]corev1.Pod, error) {
	// create configmap if not exists
	configmap := &corev1.ConfigMap{}
	err := r.Get(ctx, client.ObjectKey{
		Namespace: utils.GetCurrentNamespace(),
		Name:      utils.CommonConfigMapName,
	}, configmap)
	if err != nil {
		err = client.IgnoreNotFound(err)
		if err == nil {
			configmap, err = r.createConfigMap(ctx, dest)
			if err != nil {
				logger.Error(err, "failed to create configmap")
				return nil, err
			}
			logger.V(0).Info("created configmap")
		} else {
			logger.Error(err, "failed to get configmap")
			return nil, err
		}
	}

	// Create pod
	collectorPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "odigos-collector-",
			Namespace:    utils.GetCurrentNamespace(),
			Labels:       commonLabels,
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "collector-conf",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "collector-conf",
							},
							Items: []corev1.KeyToPath{
								{
									Key:  "collector-conf",
									Path: "collector-conf.yaml",
								},
							},
						},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:    "collector",
					Image:   r.getCollectorContainerImage(),
					Command: []string{"/otelcol", "--config=/conf/collector-conf.yaml"},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "collector-conf",
							MountPath: "/conf",
						},
					},
				},
			},
		},
	}

	err = r.Create(ctx, &collectorPod)
	if err != nil {
		logger.Error(err, "error creating collector pod")
		return nil, err
	}

	// Create service
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      collectorPod.Name,
			Namespace: utils.GetCurrentNamespace(),
			Labels:    commonLabels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "otlp",
					Protocol:   "TCP",
					Port:       4317,
					TargetPort: intstr.FromInt(4317),
				},
				{
					Name:       "zipkin",
					Protocol:   "TCP",
					Port:       9411,
					TargetPort: intstr.FromInt(9411),
				},
				{
					Name: "metrics",
					Port: 8888,
				},
			},
			Selector: commonLabels,
		},
	}

	err = r.Create(ctx, svc)
	if err != nil {
		return nil, err
	}

	return []corev1.Pod{collectorPod}, nil
}

func (r *DestinationReconciler) getCollectorContainerImage() string {
	return "otel/opentelemetry-collector:0.53.0"
}

func (r *DestinationReconciler) updateExistingCollectors(dest *v1.Destination) error {
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DestinationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Destination{}).
		Complete(r)
}
