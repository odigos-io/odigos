package metricshandler

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const helmManagedByLabel = "app.kubernetes.io/managed-by"

// APIServiceDeleteMigration deletes a leftover Odigos-owned custom-metrics APIService
// that Helm does not yet own, so a later helm upgrade can create it.
type APIServiceDeleteMigration struct {
	Client client.Client
	Logger logr.Logger
}

func (m *APIServiceDeleteMigration) NeedLeaderElection() bool {
	return true
}

func (m *APIServiceDeleteMigration) Start(ctx context.Context) error {
	return wait.ExponentialBackoff(wait.Backoff{
		Duration: 100 * time.Millisecond,
		Factor:   2.0,
		Jitter:   0.1,
		Steps:    5,
	}, func() (bool, error) {
		apiSvc := &apiregv1.APIService{}
		err := m.Client.Get(ctx, client.ObjectKey{Name: k8sconsts.CustomMetricsAPIServiceName}, apiSvc)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, nil
		}

		if !IsOwnedByOdigos(apiSvc) {
			return true, nil
		}
		if apiSvc.Labels[helmManagedByLabel] == "Helm" {
			return true, nil
		}

		err = m.Client.Delete(ctx, apiSvc)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, nil
		}
		m.Logger.Info("Deleted leftover custom-metrics APIService so Helm can own it on the next upgrade",
			"apiService", k8sconsts.CustomMetricsAPIServiceName)
		return true, nil
	})
}
