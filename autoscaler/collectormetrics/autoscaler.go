package collectormetrics

import (
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	"github.com/odigos-io/odigos/autoscaler/controllers/datacollection"
	"github.com/odigos-io/odigos/autoscaler/controllers/gateway"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"sigs.k8s.io/controller-runtime/pkg/predicate"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

const (
	defaultInterval      = 15 * time.Second
	defaultMinReplicas   = 1
	defaultMaxReplicas   = 5
	notificationChanSize = 10
)

type AutoscalerOptions struct {
	interval        time.Duration
	collectorsGroup odigosv1.CollectorsGroupRole
	minReplicas     int
	maxReplicas     int
	algorithm       AutoscalerAlgorithm
}

type AutoscalerOption func(*AutoscalerOptions)

func WithInterval(interval time.Duration) AutoscalerOption {
	return func(o *AutoscalerOptions) {
		o.interval = interval
	}
}

func WithScaleRange(minReplicas, maxReplicas int) AutoscalerOption {
	return func(o *AutoscalerOptions) {
		o.minReplicas = minReplicas
		o.maxReplicas = maxReplicas
	}
}

func WithCollectorsGroup(collectorsGroup odigosv1.CollectorsGroupRole) AutoscalerOption {
	return func(o *AutoscalerOptions) {
		o.collectorsGroup = collectorsGroup
	}
}

func WithAlgorithm(algorithm AutoscalerAlgorithm) AutoscalerOption {
	return func(o *AutoscalerOptions) {
		o.algorithm = algorithm
	}
}

type Autoscaler struct {
	kubeClient    client.Client
	options       AutoscalerOptions
	ticker        *time.Ticker
	notifications chan Notification
	podIPs        map[string]string
	odigosConfig  *odigosv1.OdigosConfiguration
}

func NewAutoscaler(kubeClient client.Client, opts ...AutoscalerOption) *Autoscaler {
	// Set default options
	options := AutoscalerOptions{
		interval:    defaultInterval,
		minReplicas: defaultMinReplicas,
		maxReplicas: defaultMaxReplicas,
	}

	for _, opt := range opts {
		opt(&options)
	}

	return &Autoscaler{
		kubeClient:    kubeClient,
		options:       options,
		ticker:        time.NewTicker(options.interval),
		notifications: make(chan Notification, notificationChanSize),
		podIPs:        make(map[string]string),
	}
}

func (a *Autoscaler) Predicate() predicate.Predicate {
	ns := env.GetCurrentNamespace()
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldPod := e.ObjectOld.(*corev1.Pod)
			newPod := e.ObjectNew.(*corev1.Pod)
			if oldPod.Namespace != ns || newPod.Namespace != ns {
				return false
			}

			// If pods are not related to this collectors group, ignore them
			if val, ok := newPod.Labels[a.collectorsGroupLabelKey()]; !ok || val != "true" {
				return false
			}
			if val, ok := oldPod.Labels[a.collectorsGroupLabelKey()]; !ok || val != "true" {
				return false
			}

			// Filter updates if IP changes or phase becomes running
			return oldPod.Status.PodIP != newPod.Status.PodIP ||
				(newPod.Status.Phase == corev1.PodRunning && oldPod.Status.Phase != corev1.PodRunning)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			if e.Object.GetNamespace() != ns {
				return false
			}

			// If pods are not related to this collectors group, ignore them
			if val, ok := e.Object.GetLabels()[a.collectorsGroupLabelKey()]; !ok || val != "true" {
				return false
			}

			return true
		},
		CreateFunc: func(e event.CreateEvent) bool {
			if e.Object.GetNamespace() != ns {
				return false
			}

			// If pods are not related to this collectors group, ignore them
			if val, ok := e.Object.GetLabels()[a.collectorsGroupLabelKey()]; !ok || val != "true" {
				return false
			}

			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

func (a *Autoscaler) collectorsGroupLabelKey() string {
	if a.options.collectorsGroup == odigosv1.CollectorsGroupRoleClusterGateway {
		return gateway.CollectorLabel
	}

	return datacollection.CollectorLabel
}

func (a *Autoscaler) Notify() chan<- Notification {
	return a.notifications
}
