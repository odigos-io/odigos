package metricshandler

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

type caSyncResult struct {
	err     error
	updates int
	stored  *apiregv1.APIService
}

// syncCertSecret reconciles the given secret name and reports what the reconciler did to the
// APIService.
func syncCertSecret(t *testing.T, secretName string, objects []client.Object, funcs interceptor.Funcs) caSyncResult {
	t.Helper()

	updates := 0
	userUpdate := funcs.Update
	funcs.Update = func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
		updates++
		if userUpdate != nil {
			return userUpdate(ctx, c, obj, opts...)
		}
		return c.Update(ctx, obj, opts...)
	}

	scheme := metricsHandlerScheme(t)
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).
		WithInterceptorFuncs(funcs).Build()
	reconciler := &CAUpdaterReconciler{Client: k8sClient, Scheme: scheme}

	ctx := logr.NewContext(context.Background(), logr.Discard())
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: secretName, Namespace: gatewayTestNamespace},
	})

	result := caSyncResult{err: err, updates: updates}
	stored := &apiregv1.APIService{}
	if getErr := k8sClient.Get(ctx, client.ObjectKey{Name: k8sconsts.CustomMetricsAPIServiceName}, stored); getErr == nil {
		result.stored = stored
	}
	return result
}

func TestCAUpdaterReconciler_SyncsARotatedCA(t *testing.T) {
	result := syncCertSecret(t, k8sconsts.AutoscalerWebhookSecretName, []client.Object{
		webhookCertSecret(map[string][]byte{"ca.crt": []byte("rotated-ca")}),
		odigosOwnedAPIService([]byte("expired-ca")),
	}, interceptor.Funcs{})

	require.NoError(t, result.err)
	require.NotNil(t, result.stored)
	assert.Equal(t, []byte("rotated-ca"), result.stored.Spec.CABundle)
}

func TestCAUpdaterReconciler_IgnoresOtherSecrets(t *testing.T) {
	// odigos still ships the deprecated cert secret next to the current one, and it carries a CA
	// of its own; taking that one would make the aggregation layer reject the autoscaler's cert.
	deprecated := webhookCertSecret(map[string][]byte{"ca.crt": []byte("unrelated-ca")})
	deprecated.Name = k8sconsts.DeprecatedAutoscalerWebhookSecretName

	result := syncCertSecret(t, k8sconsts.DeprecatedAutoscalerWebhookSecretName, []client.Object{
		deprecated,
		odigosOwnedAPIService([]byte("expired-ca")),
	}, interceptor.Funcs{})

	require.NoError(t, result.err)
	assert.Zero(t, result.updates)
	require.NotNil(t, result.stored)
	assert.Equal(t, []byte("expired-ca"), result.stored.Spec.CABundle)
}

func TestCAUpdaterReconciler_DeletedSecretIsNotAnError(t *testing.T) {
	result := syncCertSecret(t, k8sconsts.AutoscalerWebhookSecretName, []client.Object{
		odigosOwnedAPIService([]byte("expired-ca")),
	}, interceptor.Funcs{})

	require.NoError(t, result.err)
	assert.Zero(t, result.updates)
}

func TestCAUpdaterReconciler_SecretLookupFailureIsRetried(t *testing.T) {
	result := syncCertSecret(t, k8sconsts.AutoscalerWebhookSecretName, []client.Object{
		odigosOwnedAPIService([]byte("expired-ca")),
	}, interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if _, ok := obj.(*corev1.Secret); ok {
				return errors.New("the cache is not synced")
			}
			return c.Get(ctx, key, obj, opts...)
		},
	})

	require.Error(t, result.err)
	assert.Contains(t, result.err.Error(), "the cache is not synced")
}

func TestCAUpdaterReconciler_SecretWithoutUsableCA(t *testing.T) {
	for _, tc := range []struct {
		name string
		data map[string][]byte
	}{
		{name: "no ca.crt key", data: map[string][]byte{"tls.crt": []byte("leaf")}},
		// the rotator creates the secret before it has issued a CA
		{name: "empty ca.crt", data: map[string][]byte{"ca.crt": {}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := syncCertSecret(t, k8sconsts.AutoscalerWebhookSecretName, []client.Object{
				webhookCertSecret(tc.data),
				odigosOwnedAPIService([]byte("expired-ca")),
			}, interceptor.Funcs{})

			require.NoError(t, result.err)
			assert.Zero(t, result.updates)
			require.NotNil(t, result.stored)
			assert.Equal(t, []byte("expired-ca"), result.stored.Spec.CABundle)
		})
	}
}

func TestCAUpdaterReconciler_MissingAPIServiceIsNotAnError(t *testing.T) {
	// odigos never registered the custom metrics API (or it was removed by hand)
	result := syncCertSecret(t, k8sconsts.AutoscalerWebhookSecretName, []client.Object{
		webhookCertSecret(map[string][]byte{"ca.crt": []byte("rotated-ca")}),
	}, interceptor.Funcs{})

	require.NoError(t, result.err)
	assert.Zero(t, result.updates)
}

func TestCAUpdaterReconciler_APIServiceLookupFailureIsRetried(t *testing.T) {
	result := syncCertSecret(t, k8sconsts.AutoscalerWebhookSecretName, []client.Object{
		webhookCertSecret(map[string][]byte{"ca.crt": []byte("rotated-ca")}),
	}, interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if _, ok := obj.(*apiregv1.APIService); ok {
				return errors.New("apiservices is forbidden")
			}
			return c.Get(ctx, key, obj, opts...)
		},
	})

	require.Error(t, result.err)
	assert.Contains(t, result.err.Error(), "apiservices is forbidden")
}

func TestCAUpdaterReconciler_LeavesAForeignAPIServiceAlone(t *testing.T) {
	foreign := odigosOwnedAPIService([]byte("prometheus-adapter-ca"))
	foreign.Spec.Service.Name = "prometheus-adapter"

	result := syncCertSecret(t, k8sconsts.AutoscalerWebhookSecretName, []client.Object{
		webhookCertSecret(map[string][]byte{"ca.crt": []byte("odigos-ca")}),
		foreign,
	}, interceptor.Funcs{})

	require.NoError(t, result.err)
	assert.Zero(t, result.updates)
	require.NotNil(t, result.stored)
	assert.Equal(t, []byte("prometheus-adapter-ca"), result.stored.Spec.CABundle)
}

func TestCAUpdaterReconciler_UnchangedCAIsNotRewritten(t *testing.T) {
	// the secret is resynced periodically; rewriting an identical CA would churn the APIService
	result := syncCertSecret(t, k8sconsts.AutoscalerWebhookSecretName, []client.Object{
		webhookCertSecret(map[string][]byte{"ca.crt": []byte("the-ca")}),
		odigosOwnedAPIService([]byte("the-ca")),
	}, interceptor.Funcs{})

	require.NoError(t, result.err)
	assert.Zero(t, result.updates)
}

func TestCAUpdaterReconciler_UpdateFailureIsRetried(t *testing.T) {
	result := syncCertSecret(t, k8sconsts.AutoscalerWebhookSecretName, []client.Object{
		webhookCertSecret(map[string][]byte{"ca.crt": []byte("rotated-ca")}),
		odigosOwnedAPIService([]byte("expired-ca")),
	}, interceptor.Funcs{
		Update: func(context.Context, client.WithWatch, client.Object, ...client.UpdateOption) error {
			return errors.New("conflict")
		},
	})

	require.Error(t, result.err)
	assert.Contains(t, result.err.Error(), "conflict")
}
