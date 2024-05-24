package collectormetrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	autoscalingv1 "k8s.io/api/autoscaling/v1"

	"github.com/odigos-io/odigos/autoscaler/controllers/gateway"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	metricsUrlPattern = "http://%s:8888/metrics"
)

var (
	errObjectNotFoundForCollectorsGroup = fmt.Errorf("object not found for collectors group")
	errDecisionOutOfRange               = fmt.Errorf("decision out of range")
)

type MetricFetchResult struct {
	PodName string
	Error   error
	Metrics map[string]*dto.MetricFamily
}

func (a *Autoscaler) Run(ctx context.Context) {
	logger := log.FromContext(ctx)
	logger = logger.WithName("autoscaler")

	for {
		select {
		case notification := <-a.notifications:
			logger.V(5).Info("Got ip change notification", "notification", notification)
			a.updateIPsMap(notification)
		case <-ctx.Done():
			logger.V(0).Info("Shutting down autoscaler", "collectorsGroup", a.options.collectorsGroup)
			a.ticker.Stop()
			close(a.notifications)
			return
		case <-a.ticker.C:
			logger.V(0).Info("Checking collectors metrics")
			results := a.getCollectorsMetrics(ctx)
			decision := a.options.algorithm.Decide(ctx, results)
			a.executeDecision(ctx, decision)
		}
	}
}

func (a *Autoscaler) executeDecision(ctx context.Context, decision AutoscalerDecision) bool {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Executing decision", "decision", decision)
	obj := a.getTargetKubernetesObject()
	if obj == nil {
		logger.Error(errObjectNotFoundForCollectorsGroup, "No target object found", "group", a.options.collectorsGroup)
		return false
	}

	scale := &autoscalingv1.Scale{
		Spec: autoscalingv1.ScaleSpec{
			Replicas: int32(decision),
		},
	}

	err := a.kubeClient.SubResource("scale").Update(ctx, obj, client.WithSubResourceBody(scale))
	if err != nil {
		logger.Error(err, "Failed to scale object", "object", obj)
		return false
	}

	return true
}

func (a *Autoscaler) getTargetKubernetesObject() client.Object {
	if a.options.collectorsGroup == odigosv1.CollectorsGroupRoleClusterGateway {
		return &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      gateway.KubeObjectName,
				Namespace: env.GetCurrentNamespace(),
			},
		}
	}

	return nil
}

func (a *Autoscaler) updateIPsMap(notification Notification) {
	if notification.Reason == NewIPDiscovered {
		a.podIPs[notification.PodName] = notification.IP
	} else if notification.Reason == IPRemoved {
		delete(a.podIPs, notification.PodName)
	}
}

func (a *Autoscaler) getCollectorsMetrics(ctx context.Context) []MetricFetchResult {
	logger := log.FromContext(ctx)
	results := make(chan MetricFetchResult, len(a.podIPs))
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	for podName, podIP := range a.podIPs {
		go func(podName, podIP string, results chan MetricFetchResult) {
			result := MetricFetchResult{
				PodName: podName,
			}

			// Get metrics from the collector pod
			urlStr := fmt.Sprintf(metricsUrlPattern, podIP)
			req, err := http.NewRequestWithContext(timeoutCtx, http.MethodGet, urlStr, nil)
			if err != nil {
				result.Error = err
				results <- result
				return
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				result.Error = err
				results <- result
				return
			}

			defer resp.Body.Close()
			var parser expfmt.TextParser
			metricFamilies, err := parser.TextToMetricFamilies(resp.Body)
			if err != nil {
				result.Error = err
				results <- result
				return
			}

			result.Metrics = metricFamilies
			results <- result
		}(podName, podIP, results)
	}

	// Fetch all results from channel
	var successfulResults []MetricFetchResult
	for i := 0; i < len(a.podIPs); i++ {
		result := <-results
		if result.Error != nil {
			logger.Error(result.Error, "Failed to get metrics from pod", "pod", result.PodName)
		} else {
			successfulResults = append(successfulResults, result)
		}
	}

	return successfulResults
}
