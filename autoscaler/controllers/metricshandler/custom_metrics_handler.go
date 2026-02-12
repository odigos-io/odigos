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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

const (
	gatewayRejectionMetricName = "odigos_gateway_memory_limiter_rejections_total"
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
			APIVersion: "custom.metrics.k8s.io/v1beta1",
			Kind:       "MetricValueList",
			Items: []MetricValue{
				{
					MetricName: "odigos_gateway_rejections",
					Timestamp:  now,
					Value:      fmt.Sprintf("%.2f", metricVal),
					DescribedObject: map[string]string{
						"kind":      "Deployment",
						"namespace": namespace,
						"name":      k8sconsts.OdigosClusterCollectorDeploymentName,
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
		"groupVersion": "custom.metrics.k8s.io/v1beta1",
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

	// Create or update APIService for custom.metrics.k8s.io
	apiSvcName := "v1beta1.custom.metrics.k8s.io"
	existing := &apiregv1.APIService{}
	err := mgr.GetClient().Get(ctx, client.ObjectKey{Name: apiSvcName}, existing)
	if client.IgnoreNotFound(err) != nil {
		return err
	}

	if err != nil {
		// NotFound error -> create the APIService
		port := int32(9443)
		apiSvc := &apiregv1.APIService{
			ObjectMeta: metav1.ObjectMeta{
				Name: apiSvcName,
			},
			Spec: apiregv1.APIServiceSpec{
				Service: &apiregv1.ServiceReference{
					Name:      k8sconsts.AutoScalerWebhookServiceName,
					Namespace: namespace,
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

	// Register HTTP handlers on the webhook server
	webhookServer := mgr.GetWebhookServer()

	discoveryPath := "/apis/custom.metrics.k8s.io/v1beta1"
	webhookServer.Register(discoveryPath, http.HandlerFunc(DiscoveryHandler))

	deploymentMetricPath := fmt.Sprintf(
		"/apis/custom.metrics.k8s.io/v1beta1/namespaces/%s/deployments.apps/odigos-gateway/odigos_gateway_rejections",
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

	parser := expfmt.NewTextParser(model.UTF8Validation)
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
