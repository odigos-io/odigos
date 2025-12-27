package services

import (
	"context"
	"fmt"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	RecommendationsConfigMapName = "odigos-dissmissed-recommendations"
)

// GetDismissedRecommendations retrieves the list of dismissed recommendation types from the ConfigMap.
// Returns a map where the key is the recommendation type and the value indicates if it's dismissed.
func GetDismissedRecommendations(ctx context.Context) (map[model.RecommendationType]bool, error) {
	ns := env.GetCurrentNamespace()

	cm, err := kube.DefaultClient.CoreV1().ConfigMaps(ns).Get(ctx, RecommendationsConfigMapName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return make(map[model.RecommendationType]bool), nil
		}
		return nil, fmt.Errorf("failed to get recommendations ConfigMap: %w", err)
	}

	if cm.Data == nil {
		return make(map[model.RecommendationType]bool), nil
	}

	dismissedMap := make(map[model.RecommendationType]bool)
	// Each key in the ConfigMap data is a recommendation type, and its value is the dismissal timestamp
	for key := range cm.Data {
		// Validate that the key is a valid recommendation type
		recType := model.RecommendationType(key)
		if recType.IsValid() {
			dismissedMap[recType] = true
		}
	}

	return dismissedMap, nil
}

// DismissRecommendation adds a recommendation type to the dismissed list in the ConfigMap.
// The key is the recommendation type string, and the value is the RFC3339 timestamp when it was dismissed.
func DismissRecommendation(ctx context.Context, recType model.RecommendationType) error {
	ns := env.GetCurrentNamespace()

	// Get current timestamp in RFC3339 format
	timestamp := time.Now().UTC().Format(time.RFC3339)

	cm, err := kube.DefaultClient.CoreV1().ConfigMaps(ns).Get(ctx, RecommendationsConfigMapName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create new ConfigMap with the dismissed recommendation
			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      RecommendationsConfigMapName,
					Namespace: ns,
					Labels: map[string]string{
						k8sconsts.OdigosSystemConfigLabelKey: "recommendations",
					},
				},
				Data: map[string]string{
					string(recType): timestamp,
				},
			}
			_, err = kube.DefaultClient.CoreV1().ConfigMaps(ns).Create(ctx, cm, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create recommendations ConfigMap: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get recommendations ConfigMap: %w", err)
	}

	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}

	// Set the key (recommendation type) with the current timestamp
	cm.Data[string(recType)] = timestamp

	_, err = kube.DefaultClient.CoreV1().ConfigMaps(ns).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update recommendations ConfigMap: %w", err)
	}

	return nil
}
