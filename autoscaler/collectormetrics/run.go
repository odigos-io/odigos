package collectormetrics

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	metricsUrlPattern = "http://%s:8888/metrics"
)

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
			if err := a.getCollectorsMetrics(ctx); err != nil {
				logger.Error(err, "Failed to get collectors metrics")
			}
		}
	}
}

func (a *Autoscaler) updateIPsMap(notification Notification) {
	if notification.Reason == NewIPDiscovered {
		a.podIPs[notification.PodName] = notification.IP
	} else if notification.Reason == IPRemoved {
		delete(a.podIPs, notification.PodName)
	}
}

func (a *Autoscaler) getCollectorsMetrics(ctx context.Context) error {
	logger := log.FromContext(ctx)
	results := make(chan string, len(a.podIPs))
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	for podName, podIP := range a.podIPs {
		go func(podName, podIP string, results chan string) {
			// Get metrics from the collector pod
			urlStr := fmt.Sprintf(metricsUrlPattern, podIP)
			req, err := http.NewRequestWithContext(timeoutCtx, http.MethodGet, urlStr, nil)
			if err != nil {
				logger.Error(err, "Failed to create request", "url", urlStr)
				return
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				logger.Error(err, "Failed to get metrics", "url", urlStr)
				return
			}

			// Log resp body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				logger.Error(err, "Failed to read response body", "url", urlStr)
				return
			}

			results <- string(body)
		}(podName, podIP, results)
	}

	// TODO: write error to results as well
	// Fetch all results from channel
	for i := 0; i < len(a.podIPs); i++ {
		result := <-results
		logger.V(0).Info("Got metrics", "metrics", result)
	}

	return nil
}
