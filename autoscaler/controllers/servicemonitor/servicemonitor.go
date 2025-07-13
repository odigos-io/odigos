package servicemonitor

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PrometheusTarget represents a Prometheus scrape target
type PrometheusTarget struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

// GetServiceMonitorTargets returns Prometheus scrape targets from ServiceMonitor CRDs
func GetServiceMonitorTargets(ctx context.Context, kubeClient client.Client) ([]PrometheusTarget, error) {
	var targets []PrometheusTarget

	// Check if ServiceMonitor CRD exists
	crdList := &metav1.PartialObjectMetadataList{}
	crdList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1",
		Kind:    "CustomResourceDefinitionList",
	})

	if err := kubeClient.List(ctx, crdList); err != nil {
		return targets, nil // Return empty targets if we can't check CRDs
	}

	// Look for ServiceMonitor CRD
	serviceMonitorCRDExists := false
	for _, crd := range crdList.Items {
		if crd.GetName() == "servicemonitors.monitoring.coreos.com" {
			serviceMonitorCRDExists = true
			break
		}
	}

	if !serviceMonitorCRDExists {
		return targets, nil // Return empty targets if CRD doesn't exist
	}

	// List all ServiceMonitor CRDs using unstructured client
	serviceMonitors := &unstructured.UnstructuredList{}
	serviceMonitors.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "monitoring.coreos.com",
		Version: "v1",
		Kind:    "ServiceMonitorList",
	})

	if err := kubeClient.List(ctx, serviceMonitors); err != nil {
		return nil, fmt.Errorf("failed to list ServiceMonitors: %w", err)
	}

	for _, sm := range serviceMonitors.Items {
		// Get the services that match the selector
		services, err := getServicesForServiceMonitor(ctx, kubeClient, &sm)
		if err != nil {
			// Log error but continue processing other ServiceMonitors
			continue
		}

		// Convert ServiceMonitor endpoints to Prometheus targets
		for _, service := range services {
			if endpoints, found, err := unstructured.NestedSlice(sm.Object, "spec", "endpoints"); err == nil && found {
				for _, endpointRaw := range endpoints {
					if endpoint, ok := endpointRaw.(map[string]interface{}); ok {
						targets = append(targets, createPrometheusTargetFromUnstructured(service, endpoint, &sm))
					}
				}
			}
		}
	}

	return targets, nil
}

func getServicesForServiceMonitor(ctx context.Context, kubeClient client.Client, sm *unstructured.Unstructured) ([]corev1.Service, error) {
	var services []corev1.Service

	// Get all services in the target namespaces
	namespaces := []string{sm.GetNamespace()}
	
	// Check for namespace selector
	if nsSelector, found, err := unstructured.NestedMap(sm.Object, "spec", "namespaceSelector"); err == nil && found {
		if any, found := nsSelector["any"]; found && any == true {
			// TODO: Get all namespaces - for now use current namespace
			namespaces = []string{""}
		} else if matchNames, found, err := unstructured.NestedStringSlice(nsSelector, "matchNames"); err == nil && found {
			namespaces = matchNames
		}
	}

	for _, ns := range namespaces {
		var serviceList corev1.ServiceList
		if ns != "" {
			if err := kubeClient.List(ctx, &serviceList, client.InNamespace(ns)); err != nil {
				return nil, fmt.Errorf("failed to list services in namespace %s: %w", ns, err)
			}
		} else {
			if err := kubeClient.List(ctx, &serviceList); err != nil {
				return nil, fmt.Errorf("failed to list services: %w", err)
			}
		}

		// Filter services based on selector
		for _, service := range serviceList.Items {
			if serviceMatchesSelectorFromUnstructured(service, sm) {
				services = append(services, service)
			}
		}
	}

	return services, nil
}

func serviceMatchesSelectorFromUnstructured(service corev1.Service, sm *unstructured.Unstructured) bool {
	// Get selector from ServiceMonitor
	selector, found, err := unstructured.NestedMap(sm.Object, "spec", "selector")
	if err != nil || !found {
		return false
	}

	matchLabels, found, err := unstructured.NestedStringMap(selector, "matchLabels")
	if err != nil || !found {
		return false
	}

	// Simple label matching
	for key, value := range matchLabels {
		if service.Labels[key] != value {
			return false
		}
	}
	return true
}

func createPrometheusTargetFromUnstructured(service corev1.Service, endpoint map[string]interface{}, sm *unstructured.Unstructured) PrometheusTarget {
	// Default values
	path := "/metrics"
	if pathVal, found := endpoint["path"]; found {
		if pathStr, ok := pathVal.(string); ok {
			path = pathStr
		}
	}

	interval := "15s"
	if intervalVal, found := endpoint["interval"]; found {
		if intervalStr, ok := intervalVal.(string); ok {
			interval = intervalStr
		}
	}

	// Build target URL
	var port int32
	if portVal, found := endpoint["port"]; found {
		if portStr, ok := portVal.(string); ok {
			// Find port by name
			for _, servicePort := range service.Spec.Ports {
				if servicePort.Name == portStr {
					port = servicePort.Port
					break
				}
			}
		}
	} else if targetPortVal, found := endpoint["targetPort"]; found {
		if targetPortInt, ok := targetPortVal.(int64); ok {
			port = int32(targetPortInt)
		}
	}

	target := PrometheusTarget{
		Targets: []string{fmt.Sprintf("%s.%s.svc.cluster.local:%d", service.Name, service.Namespace, port)},
		Labels: map[string]string{
			"__metrics_path__":             path,
			"__scrape_interval__":          interval,
			"job":                          fmt.Sprintf("%s/%s", sm.GetNamespace(), sm.GetName()),
			"service":                      service.Name,
			"namespace":                    service.Namespace,
			"servicemonitor":               sm.GetName(),
			"servicemonitor_namespace":     sm.GetNamespace(),
		},
	}

	// Add custom labels from ServiceMonitor
	for key, value := range sm.GetLabels() {
		target.Labels[key] = fmt.Sprintf("%v", value)
	}

	return target
}