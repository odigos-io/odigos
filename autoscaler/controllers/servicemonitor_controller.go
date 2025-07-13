package controllers

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/autoscaler/controllers/clustercollector"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ServiceMonitorReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	ImagePullSecrets []string
	OdigosVersion    string
}

func (r *ServiceMonitorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling ServiceMonitor")

	// Check if ServiceMonitor auto-detection is enabled
	if !r.isServiceMonitorAutoDetectionEnabled(ctx, logger) {
		logger.V(1).Info("ServiceMonitor auto-detection is disabled, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	// Reconcile the cluster collector to include ServiceMonitor scraping
	return clustercollector.ReconcileClusterCollector(ctx, r.Client, r.Scheme, r.ImagePullSecrets, r.OdigosVersion)
}

func (r *ServiceMonitorReconciler) isServiceMonitorAutoDetectionEnabled(ctx context.Context, logger logr.Logger) bool {
	// Get the Odigos configuration
	odigosNs := env.GetCurrentNamespace()
	configMap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      consts.OdigosConfigurationName,
		Namespace: odigosNs,
	}, configMap)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(1).Info("Odigos configuration not found, ServiceMonitor auto-detection disabled")
			return false
		}
		logger.Error(err, "Failed to get Odigos configuration")
		return false
	}

	// Parse the configuration
	configData, ok := configMap.Data[consts.OdigosConfigurationFileName]
	if !ok {
		logger.V(1).Info("Odigos configuration data not found, ServiceMonitor auto-detection disabled")
		return false
	}

	var odigosConfig common.OdigosConfiguration
	if err := common.UnmarshalYAML([]byte(configData), &odigosConfig); err != nil {
		logger.Error(err, "Failed to unmarshal Odigos configuration")
		return false
	}

	// Check if ServiceMonitor auto-detection is enabled
	if odigosConfig.ServiceMonitorAutoDetectionEnabled == nil {
		return false
	}

	return *odigosConfig.ServiceMonitorAutoDetectionEnabled
}

func (r *ServiceMonitorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&promv1.ServiceMonitor{}).
		Complete(r)
}

// GetServiceMonitorTargets returns Prometheus scrape targets from ServiceMonitor CRDs
func GetServiceMonitorTargets(ctx context.Context, client client.Client) ([]PrometheusTarget, error) {
	var targets []PrometheusTarget

	// List all ServiceMonitor CRDs
	var serviceMonitors promv1.ServiceMonitorList
	if err := client.List(ctx, &serviceMonitors); err != nil {
		return nil, fmt.Errorf("failed to list ServiceMonitors: %w", err)
	}

	for _, sm := range serviceMonitors.Items {
		// Get the services that match the selector
		services, err := getServicesForServiceMonitor(ctx, client, &sm)
		if err != nil {
			// Log error but continue processing other ServiceMonitors
			continue
		}

		// Convert ServiceMonitor endpoints to Prometheus targets
		for _, service := range services {
			for _, endpoint := range sm.Spec.Endpoints {
				targets = append(targets, createPrometheusTarget(service, endpoint, &sm))
			}
		}
	}

	return targets, nil
}

func getServicesForServiceMonitor(ctx context.Context, client client.Client, sm *promv1.ServiceMonitor) ([]corev1.Service, error) {
	var services []corev1.Service

	// Get all services in the target namespaces
	namespaces := []string{sm.Namespace}
	if sm.Spec.NamespaceSelector != nil {
		if sm.Spec.NamespaceSelector.Any {
			// TODO: Get all namespaces
			namespaces = []string{""}
		} else if len(sm.Spec.NamespaceSelector.MatchNames) > 0 {
			namespaces = sm.Spec.NamespaceSelector.MatchNames
		}
	}

	for _, ns := range namespaces {
		var serviceList corev1.ServiceList
		listOpts := []client.ListOption{}
		if ns != "" {
			listOpts = append(listOpts, client.InNamespace(ns))
		}

		if err := client.List(ctx, &serviceList, listOpts...); err != nil {
			return nil, fmt.Errorf("failed to list services in namespace %s: %w", ns, err)
		}

		// Filter services based on selector
		for _, service := range serviceList.Items {
			if serviceMatchesSelector(service, sm.Spec.Selector) {
				services = append(services, service)
			}
		}
	}

	return services, nil
}

func serviceMatchesSelector(service corev1.Service, selector metav1.LabelSelector) bool {
	// Simple label matching - could be enhanced with more complex matching
	for key, value := range selector.MatchLabels {
		if service.Labels[key] != value {
			return false
		}
	}
	return true
}

func createPrometheusTarget(service corev1.Service, endpoint promv1.Endpoint, sm *promv1.ServiceMonitor) PrometheusTarget {
	// Default values
	path := "/metrics"
	if endpoint.Path != "" {
		path = endpoint.Path
	}

	interval := "15s"
	if endpoint.Interval != "" {
		interval = string(endpoint.Interval)
	}

	// Build target URL
	var port int32
	if endpoint.Port != "" {
		// Find port by name
		for _, servicePort := range service.Spec.Ports {
			if servicePort.Name == endpoint.Port {
				port = servicePort.Port
				break
			}
		}
	} else if endpoint.TargetPort != nil {
		port = endpoint.TargetPort.IntVal
	}

	target := PrometheusTarget{
		Targets: []string{fmt.Sprintf("%s.%s.svc.cluster.local:%d", service.Name, service.Namespace, port)},
		Labels: map[string]string{
			"__metrics_path__":     path,
			"__scrape_interval__":  interval,
			"job":                  fmt.Sprintf("%s/%s", sm.Namespace, sm.Name),
			"service":              service.Name,
			"namespace":            service.Namespace,
			"servicemonitor":       sm.Name,
			"servicemonitor_namespace": sm.Namespace,
		},
	}

	// Add custom labels from ServiceMonitor
	for key, value := range sm.Labels {
		target.Labels[key] = value
	}

	return target
}

// PrometheusTarget represents a Prometheus scrape target
type PrometheusTarget struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}