package nodecollector

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Runnable to delete deprecated datacollection daemonset we used to have in the past.
// We used to manage the content of the daemonset from autoscaler controller - but now the Helm/CLI is managing it.
type DataCollectionDSMigration struct {
	Client                  client.Client
	Logger                  logr.Logger
	DataCollectionDaemonSet types.NamespacedName
}

func (m *DataCollectionDSMigration) NeedLeaderElection() bool {
	// make sure we run it only from one instance of the autoscaler
	return true
}

func (m *DataCollectionDSMigration) Start(ctx context.Context) error {
	err := wait.ExponentialBackoff(wait.Backoff{
		Duration: 100 * time.Millisecond,
		Factor:   2.0,
		Jitter:   0.1,
		Steps:    5,
	}, func() (bool, error) {
		s := appsv1.DaemonSet{}
		err := m.Client.Get(ctx, m.DataCollectionDaemonSet, &s)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, nil
		}

		err = m.Client.Delete(ctx, &s)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, nil
		}
		m.Logger.Info("Successfully deleted deprecated Daemonset", "daemonset", m.DataCollectionDaemonSet)
		return true, nil
	})

	return err
}
