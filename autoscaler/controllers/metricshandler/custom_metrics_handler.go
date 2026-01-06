package metricshandler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/prometheus/common/expfmt"
)

const (
	gatewayRejectionMetricName = "odigos_gateway_memory_limiter_rejections_total"

	// Custom Metrics API constants
	customMetricsAPIGroup     = "custom.metrics.k8s.io"
	customMetricsAPIVersion   = "v1beta1"
	customMetricsGroupVersion = customMetricsAPIGroup + "/" + customMetricsAPIVersion
	NewAPIServiceName         = customMetricsAPIVersion + "." + customMetricsAPIGroup
	// TODO remove after migration is completed
	legacyAPIServiceName = "v1beta1.custom.metrics.k8s.io"
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

var lastSample sync.Map

// MetricHandler aggregates gateway rejection metrics across all pods
func MetricHandler(ctx context.Context, k8sClient client.Client, namespace string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := ctrl.Log.WithName("gateway-metric-handler")

		var podList corev1.PodList
		err := k8sClient.List(ctx, &podList,
			client.InNamespace(namespace),
			client.MatchingLabels{
				k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleClusterGateway),
			})
		if err != nil {
			log.Error(err, "Failed to list gateway pods")
			http.Error(w, fmt.Sprintf("failed to list gateway pods: %v", err), http.StatusInternalServerError)
			return
		}

		totalPods := len(podList.Items)
		if totalPods == 0 {
			http.Error(w, "no gateway pods found", http.StatusNotFound)
			return
		}

		now := time.Now()
		rejectingPods := 0

		for _, pod := range podList.Items {
			// Check if pod is OOMKilled (CrashLoopBackOff due to OOM)
			// if so, we consider it as rejecting requests due to memory pressure
			if isPodOOMKilled(&pod) {
				rejectingPods++
				continue
			}

			value, err := scrapeGatewayMetric(pod.Status.PodIP)
			if err != nil {
				log.V(1).Info("Failed to scrape gateway pod metrics",
					"pod", pod.Name,
					"podIP", pod.Status.PodIP,
					"error", err,
				)
				continue // skip unreachable pods
			}

			key := pod.Name
			prevVal, loaded := lastSample.LoadOrStore(key, value)

			var delta float64
			if loaded {
				delta = value - prevVal.(float64)
				if delta < 0 {
					delta = 0 // counter reset
				}
			} else {
				delta = 0 // First sample, no delta
			}

			if delta > 0 {
				rejectingPods++
			}

			lastSample.Store(key, value)
		}

		// Calculate rejection ratio
		rejectRatio := float64(rejectingPods) / float64(totalPods)

		// Binary metric: 1 if ≥50% pods reject, else 0
		var metricVal float64
		if rejectRatio >= 0.5 {
			metricVal = 1
		} else {
			metricVal = 0
		}

		// Only log if there are actually rejections
		if rejectingPods > 0 {
			log.Info("Observed gateway pods that rejecting data",
				"totalGatewayPodsRejectingData", rejectingPods,
				"totalGatewayPods", totalPods,
			)
		}

		resp := MetricValueList{
			APIVersion: customMetricsGroupVersion,
			Kind:       "MetricValueList",
			Items: []MetricValue{
				{
					MetricName: "odigos_gateway_rejections",
					Timestamp:  now,
					Value:      fmt.Sprintf("%.2f", metricVal),
					DescribedObject: map[string]string{
						"kind":      "Deployment",
						"namespace": namespace,
						"name":      "odigos-gateway",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error(err, "Failed to encode JSON response")
		}
	}
}

// DiscoveryHandler responds with the APIResourceList so the API server knows what metrics you offer
func DiscoveryHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"kind":         "APIResourceList",
		"apiVersion":   "v1",
		"groupVersion": customMetricsGroupVersion,
		"resources": []map[string]interface{}{
			{
				"name":         "deployments.apps/odigos_gateway_rejections",
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

	namespace := env.GetCurrentNamespace()
	// Load CA from Secret created by the rotator
	secret := &corev1.Secret{}
	if err := mgr.GetClient().Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      k8sconsts.AutoscalerWebhookSecretName,
	}, secret); err != nil {
		return fmt.Errorf("failed to get cert secret: %w", err)
	}

	caData, ok := secret.Data["ca.crt"]
	if !ok {
		return fmt.Errorf("ca.crt not found in secret %s", secret.Name)
	}

	helmManagedAPIService := &apiregv1.APIService{}
	err := mgr.GetClient().Get(ctx, client.ObjectKey{Name: NewAPIServiceName}, helmManagedAPIService)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Register handlers anyway, they'll work once Helm creates the APIService
		} else {
			return fmt.Errorf("failed to get Helm-managed APIService: %w", err)
		}
	} else {
		// Update CA bundle if needed
		if base64.StdEncoding.EncodeToString(helmManagedAPIService.Spec.CABundle) !=
			base64.StdEncoding.EncodeToString(caData) {
			helmManagedAPIService.Spec.CABundle = caData
			if err := mgr.GetClient().Update(ctx, helmManagedAPIService); err != nil {
				return fmt.Errorf("failed to update Helm-managed APIService CA bundle: %w", err)
			}
			ctrl.Log.Info("Updated CA bundle for Helm-managed APIService")
		}
	}

	// Register HTTP handlers on the webhook server
	webhookServer := mgr.GetWebhookServer()

	discoveryPath := fmt.Sprintf("/apis/%s", customMetricsGroupVersion)
	webhookServer.Register(discoveryPath, http.HandlerFunc(DiscoveryHandler))

	deploymentMetricPath := fmt.Sprintf(
		"/apis/%s/namespaces/%s/deployments.apps/odigos-gateway/odigos_gateway_rejections",
		customMetricsGroupVersion,
		namespace,
	)
	webhookServer.Register(deploymentMetricPath, MetricHandler(ctx, mgr.GetClient(), namespace))

	ctrl.Log.Info("Custom Metrics API registered successfully")
	return nil
}

func scrapeGatewayMetric(podIP string) (float64, error) {
	url := fmt.Sprintf("http://%s:%d/metrics", podIP, k8sconsts.OdigosClusterCollectorOwnTelemetryPortDefault)

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to reach pod: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}

	parser := expfmt.TextParser{}
	metricFamilies, err := parser.TextToMetricFamilies(bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("failed to parse metrics: %w", err)
	}

	mf, ok := metricFamilies[gatewayRejectionMetricName]
	if !ok {
		// metric not found → treat as 0 rejections
		// This can happen if the gateway is not running or the metric is not available yet.
		return 0, nil
	}

	var total float64
	for _, m := range mf.Metric {
		if m.Counter != nil && m.Counter.Value != nil {
			total += *m.Counter.Value
		}
	}

	return total, nil
}

// isPodOOMKilled checks if the pod is in CrashLoopBackOff due to OOM or currently OOMKilled
func isPodOOMKilled(pod *corev1.Pod) bool {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		// Scenario 1: Container is waiting in CrashLoopBackOff after being OOMKilled
		if containerStatus.State.Waiting != nil &&
			containerStatus.State.Waiting.Reason == "CrashLoopBackOff" {
			// Check if last termination was due to OOM
			if containerStatus.LastTerminationState.Terminated != nil &&
				containerStatus.LastTerminationState.Terminated.Reason == "OOMKilled" {
				return true
			}
		}

		// Scenario 2: Container is currently terminated due to OOM
		if containerStatus.State.Terminated != nil &&
			containerStatus.State.Terminated.Reason == "OOMKilled" {
			return true
		}
	}

	return false
}
