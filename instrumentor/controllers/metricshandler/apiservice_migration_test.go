package metricshandler

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// The backoff in APIServiceDeleteMigration.Start runs the condition at most this many times.
const cmMigrationAttempts = 5

func TestAPIServiceDeleteMigration_NeedsLeaderElection(t *testing.T) {
	t.Parallel()

	assert.True(t, (&APIServiceDeleteMigration{}).NeedLeaderElection(),
		"a one-shot cluster-scoped delete must run on a single replica")
}

// v1beta1.custom.metrics.k8s.io is a cluster-scoped singleton. Deleting one that belongs to the
// Prometheus Adapter, KEDA or any other adapter takes down that product's autoscaling, so the
// ownership guard is the most important thing in this file.
func TestAPIServiceDeleteMigration_LeavesForeignAPIServicesAlone(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
	}{
		{name: "prometheus adapter", serviceName: "prometheus-adapter"},
		{name: "keda", serviceName: "keda-operator-metrics-apiserver"},
		{name: "an external name with no backing service", serviceName: ""},
		{name: "a lookalike of the instrumentor service", serviceName: k8sconsts.InstrumentorServiceName + "-metrics"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writes := &cmWrites{}
			foreign := cmAPIService(tt.serviceName, []byte("somebody elses ca"))
			c := fake.NewClientBuilder().WithScheme(cmScheme(t)).
				WithObjects(foreign).
				WithInterceptorFuncs(writes.funcs(interceptor.Funcs{})).Build()

			require.NoError(t, cmRunMigration(t, c))

			assert.Zero(t, writes.total(), "a foreign APIService must not be touched")
			assert.NotNil(t, cmAPIServiceOrNil(t, c))
		})
	}
}

// Once Helm owns the object the migration's job is done; deleting it again would make every
// upgrade drop the custom metric for a reconcile.
func TestAPIServiceDeleteMigration_LeavesTheHelmOwnedAPIServiceAlone(t *testing.T) {
	writes := &cmWrites{}
	adopted := cmAPIService(k8sconsts.InstrumentorServiceName, []byte("ca"))
	adopted.Labels = map[string]string{cmHelmManagedByLabel: "Helm"}

	c := fake.NewClientBuilder().WithScheme(cmScheme(t)).
		WithObjects(adopted).
		WithInterceptorFuncs(writes.funcs(interceptor.Funcs{})).Build()

	require.NoError(t, cmRunMigration(t, c))

	assert.Zero(t, writes.total())
	assert.NotNil(t, cmAPIServiceOrNil(t, c))
}

func TestAPIServiceDeleteMigration_DeletesTheLeftoverOdigosAPIService(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		labels      map[string]string
	}{
		{
			name:        "left behind by the autoscaler",
			serviceName: k8sconsts.AutoScalerWebhookServiceName,
		},
		{
			name:        "left behind by an instrumentor that created it itself",
			serviceName: k8sconsts.InstrumentorServiceName,
		},
		{
			// Helm writes the value "Helm"; anything else is not a Helm-owned object, and
			// skipping on it would strand the migration forever.
			name:        "carrying a managed-by value that is not Helm",
			serviceName: k8sconsts.InstrumentorServiceName,
			labels:      map[string]string{cmHelmManagedByLabel: "helm"},
		},
		{
			name:        "carrying an unrelated managed-by value",
			serviceName: k8sconsts.AutoScalerWebhookServiceName,
			labels:      map[string]string{cmHelmManagedByLabel: "kustomize"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writes := &cmWrites{}
			leftover := cmAPIService(tt.serviceName, []byte("ca"))
			leftover.Labels = tt.labels

			c := fake.NewClientBuilder().WithScheme(cmScheme(t)).
				WithObjects(leftover).
				WithInterceptorFuncs(writes.funcs(interceptor.Funcs{})).Build()

			require.NoError(t, cmRunMigration(t, c))

			assert.Equal(t, 1, writes.deletes)
			assert.Equal(t, 1, writes.total(), "the migration only deletes")
			assert.Nil(t, cmAPIServiceOrNil(t, c))
		})
	}
}

func TestAPIServiceDeleteMigration_NothingToDo(t *testing.T) {
	writes := &cmWrites{}
	c := fake.NewClientBuilder().WithScheme(cmScheme(t)).
		WithInterceptorFuncs(writes.funcs(interceptor.Funcs{})).Build()

	require.NoError(t, cmRunMigration(t, c))

	assert.Zero(t, writes.total())
}

// Another replica, or the garbage collector, can remove the object between the read and the
// delete. That race is a success, not a startup failure.
func TestAPIServiceDeleteMigration_ConcurrentDeleteIsSuccess(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(cmScheme(t)).
		WithObjects(cmAPIService(k8sconsts.AutoScalerWebhookServiceName, []byte("ca"))).
		WithInterceptorFuncs(interceptor.Funcs{
			Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
				return apierrors.NewNotFound(
					schema.GroupResource{Group: "apiregistration.k8s.io", Resource: "apiservices"},
					k8sconsts.CustomMetricsAPIServiceName)
			},
		}).Build()

	assert.NoError(t, cmRunMigration(t, c))
}

func TestAPIServiceDeleteMigration_RetriesTransientFailures(t *testing.T) {
	tests := []struct {
		name         string
		failingCalls int
		wantDeleted  bool
	}{
		{name: "recovers on the last attempt", failingCalls: cmMigrationAttempts - 1, wantDeleted: true},
		{name: "recovers immediately", failingCalls: 0, wantDeleted: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gets := 0
			c := fake.NewClientBuilder().WithScheme(cmScheme(t)).
				WithObjects(cmAPIService(k8sconsts.AutoScalerWebhookServiceName, []byte("ca"))).
				WithInterceptorFuncs(interceptor.Funcs{
					Get: func(ctx context.Context, cl client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
						gets++
						if gets <= tt.failingCalls {
							return errors.New("the api server is still starting")
						}
						return cl.Get(ctx, key, obj, opts...)
					},
				}).Build()

			require.NoError(t, cmRunMigration(t, c))

			assert.Equal(t, tt.failingCalls+1, gets)
			assert.Equal(t, tt.wantDeleted, cmAPIServiceOrNil(t, c) == nil)
		})
	}
}

func TestAPIServiceDeleteMigration_GivesUpAfterTheBackoffIsExhausted(t *testing.T) {
	tests := []struct {
		name       string
		funcs      func(calls *int) interceptor.Funcs
		wantDelete bool
	}{
		{
			name: "the read never succeeds",
			funcs: func(calls *int) interceptor.Funcs {
				return interceptor.Funcs{
					Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
						*calls++
						return errors.New("apiservices get is forbidden")
					},
				}
			},
			// An RBAC denial on the read must never be mistaken for "the object is gone".
			wantDelete: false,
		},
		{
			name: "the delete never succeeds",
			funcs: func(calls *int) interceptor.Funcs {
				return interceptor.Funcs{
					Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
						*calls++
						return errors.New("apiservices delete is forbidden")
					},
				}
			},
			wantDelete: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			writes := &cmWrites{}
			c := fake.NewClientBuilder().WithScheme(cmScheme(t)).
				WithObjects(cmAPIService(k8sconsts.AutoScalerWebhookServiceName, []byte("ca"))).
				WithInterceptorFuncs(writes.funcs(tt.funcs(&calls))).Build()

			err := cmRunMigration(t, c)

			require.Error(t, err, "an exhausted migration must surface, not report success")
			assert.Equal(t, cmMigrationAttempts, calls)
			if tt.wantDelete {
				assert.Equal(t, cmMigrationAttempts, writes.deletes)
			} else {
				assert.Zero(t, writes.total(), "a read the controller could not complete must not lead to a delete")
			}
		})
	}
}

func cmRunMigration(t *testing.T, c client.Client) error {
	t.Helper()
	return (&APIServiceDeleteMigration{Client: c, Logger: logr.Discard()}).Start(context.Background())
}
