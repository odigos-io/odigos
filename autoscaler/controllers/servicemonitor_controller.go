package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ServiceMonitorReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	ImagePullSecrets []string
	OdigosVersion    string
}

var (
	// ServiceMonitor GVK - defined as variables to avoid import dependency
	ServiceMonitorGVK = schema.GroupVersionKind{
		Group:   "monitoring.coreos.com",
		Version: "v1",
		Kind:    "ServiceMonitor",
	}
)

func (r *ServiceMonitorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling ServiceMonitor")

	// Check if ServiceMonitor auto-detection is enabled
	if !r.isServiceMonitorAutoDetectionEnabled(ctx, logger) {
		logger.V(1).Info("ServiceMonitor auto-detection is disabled, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	// Check if ServiceMonitor CRD exists
	if !r.isServiceMonitorCRDAvailable(ctx, logger) {
		logger.V(1).Info("ServiceMonitor CRD not available, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	// Reconcile the cluster collector to include ServiceMonitor scraping
	return ctrl.Result{}, nil
}

func (r *ServiceMonitorReconciler) isServiceMonitorCRDAvailable(ctx context.Context, logger logr.Logger) bool {
	// Check if the ServiceMonitor CRD exists
	crdList := &metav1.PartialObjectMetadataList{}
	crdList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1",
		Kind:    "CustomResourceDefinitionList",
	})

	if err := r.List(ctx, crdList); err != nil {
		logger.V(1).Info("Failed to list CRDs, assuming ServiceMonitor CRD not available", "error", err)
		return false
	}

	// Look for ServiceMonitor CRD
	for _, crd := range crdList.Items {
		if crd.GetName() == "servicemonitors.monitoring.coreos.com" {
			return true
		}
	}

	logger.V(1).Info("ServiceMonitor CRD not found in cluster")
	return false
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
	// Check if ServiceMonitor CRD is available before setting up the controller
	ctx := context.Background()
	logger := mgr.GetLogger().WithName("servicemonitor-setup")
	
	if !r.isServiceMonitorCRDAvailable(ctx, logger) {
		logger.Info("ServiceMonitor CRD not available, skipping controller setup")
		return nil
	}

	// Create unstructured object for ServiceMonitor
	serviceMonitor := &unstructured.Unstructured{}
	serviceMonitor.SetGroupVersionKind(ServiceMonitorGVK)

	return ctrl.NewControllerManagedBy(mgr).
		For(serviceMonitor).
		Complete(r)
}