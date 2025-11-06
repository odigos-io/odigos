package metricshandler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

type MetricValueList struct {
	APIVersion string        `json:"apiVersion"`
	Kind       string        `json:"kind"`
	Items      []MetricValue `json:"items"`
}

type MetricValue struct {
	DescribedObject map[string]string `json:"describedObject"`
	MetricName      string            `json:"metricName"`
	Timestamp       time.Time         `json:"timestamp"`
	Value           string            `json:"value"`
}

// Simple handler returning a constant metric value
func ConstantMetricHandler(w http.ResponseWriter, r *http.Request) {
	resp := MetricValueList{
		APIVersion: "custom.metrics.k8s.io/v1beta1",
		Kind:       "MetricValueList",
		Items: []MetricValue{
			{
				MetricName: "odigos_gateway_rejections",
				Timestamp:  time.Now(),
				Value:      "5", // constant value for now
				DescribedObject: map[string]string{
					"kind":      "Pod",
					"namespace": "odigos-system",
					"name":      "gateway-1",
				},
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// DiscoveryHandler responds with the APIResourceList so the API server knows what metrics you offer
func DiscoveryHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"kind":         "APIResourceList",
		"apiVersion":   "v1",
		"groupVersion": "custom.metrics.k8s.io/v1beta1",
		"resources": []map[string]interface{}{
			{
				"name":         "pods/odigos_gateway_rejections",
				"singularName": "",
				"namespaced":   true,
				"kind":         "MetricValueList",
				"verbs":        []string{"get"},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// RegisterCustomMetricsAPI sets up both the APIService and HTTP routes
func RegisterCustomMetricsAPI(mgr ctrl.Manager) error {
	ctx := context.Background()

	// 1️⃣ Load CA from Secret created by the rotator
	secret := &corev1.Secret{}
	if err := mgr.GetClient().Get(ctx, client.ObjectKey{
		Namespace: env.GetCurrentNamespace(),
		Name:      k8sconsts.AutoscalerWebhookSecretName,
	}, secret); err != nil {
		return fmt.Errorf("failed to get cert secret: %w", err)
	}

	caData, ok := secret.Data["ca.crt"]
	if !ok {
		return fmt.Errorf("ca.crt not found in secret %s", secret.Name)
	}

	// 2️⃣ Create or update APIService for custom.metrics.k8s.io
	port := int32(9443)
	apiSvc := &apiregv1.APIService{
		ObjectMeta: metav1.ObjectMeta{
			Name: "v1beta1.custom.metrics.k8s.io",
		},
		Spec: apiregv1.APIServiceSpec{
			Service: &apiregv1.ServiceReference{
				Name:      "odigos-autoscaler",
				Namespace: env.GetCurrentNamespace(),
				Port:      &port,
			},
			Group:                 "custom.metrics.k8s.io",
			Version:               "v1beta1",
			InsecureSkipTLSVerify: false,
			CABundle:              caData,
			GroupPriorityMinimum:  100,
			VersionPriority:       100,
		},
	}

	existing := &apiregv1.APIService{}
	err := mgr.GetClient().Get(ctx, client.ObjectKey{Name: apiSvc.Name}, existing)
	if client.IgnoreNotFound(err) != nil {
		return err
	}

	if err != nil {
		// Not found -> create
		if err := mgr.GetClient().Create(ctx, apiSvc); err != nil {
			return fmt.Errorf("failed to create APIService: %w", err)
		}
	} else {
		// Found -> update if CA changed
		if base64.StdEncoding.EncodeToString(existing.Spec.CABundle) !=
			base64.StdEncoding.EncodeToString(caData) {
			existing.Spec.CABundle = caData
			if err := mgr.GetClient().Update(ctx, existing); err != nil {
				return fmt.Errorf("failed to update APIService: %w", err)
			}
		}
	}

	// 3️⃣ Register HTTP handlers on the webhook server
	webhookServer := mgr.GetWebhookServer()

	discoveryPath := "/apis/custom.metrics.k8s.io/v1beta1"
	webhookServer.Register(discoveryPath, http.HandlerFunc(DiscoveryHandler))

	metricPath := fmt.Sprintf("/apis/custom.metrics.k8s.io/v1beta1/namespaces/%s/pods/*/odigos_gateway_rejections", env.GetCurrentNamespace())
	webhookServer.Register(metricPath, http.HandlerFunc(ConstantMetricHandler))

	ctrl.Log.Info("Custom Metrics API registered successfully")
	return nil
}
