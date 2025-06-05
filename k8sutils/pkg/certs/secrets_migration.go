package certs

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Runnable to delete deprecated secrets we used to have in the past.
// We used to manage the cootents of the secrets from Helm/CLI - but now the controller is managing it.
type SecretDeleteMigration struct {
	Client client.Client
	Logger logr.Logger
	Secret types.NamespacedName
}

func (m *SecretDeleteMigration) NeedLeaderElection() bool {
	// make sure we run it only from one instance of an instrumentor
	return true
}

func (m *SecretDeleteMigration) Start(ctx context.Context) error {
	err := wait.ExponentialBackoff(wait.Backoff{
		Duration: 100 * time.Millisecond,
		Factor:   2.0,
		Jitter:   0.1,
		Steps:    5,
	}, func() (bool, error) {
		s := corev1.Secret{}
		err := m.Client.Get(ctx, m.Secret, &s)
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
		m.Logger.Info("Successfully deleted deprecated secret", "secret", m.Secret)
		return true, nil
	})

	return err
}
