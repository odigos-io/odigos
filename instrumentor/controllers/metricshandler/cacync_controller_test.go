package metricshandler

import (
	"context"
	"errors"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// cmWrites counts every mutating call a reconcile makes, so a test can prove a guard refused to
// touch the cluster at all rather than merely writing back the same value.
type cmWrites struct {
	creates int
	updates int
	patches int
	deletes int
}

func (w *cmWrites) total() int { return w.creates + w.updates + w.patches + w.deletes }

func (w *cmWrites) funcs(base interceptor.Funcs) interceptor.Funcs {
	wrapped := base

	wrapped.Create = func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
		w.creates++
		if base.Create != nil {
			return base.Create(ctx, c, obj, opts...)
		}
		return c.Create(ctx, obj, opts...)
	}
	wrapped.Update = func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
		w.updates++
		if base.Update != nil {
			return base.Update(ctx, c, obj, opts...)
		}
		return c.Update(ctx, obj, opts...)
	}
	wrapped.Patch = func(ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
		w.patches++
		if base.Patch != nil {
			return base.Patch(ctx, c, obj, patch, opts...)
		}
		return c.Patch(ctx, obj, patch, opts...)
	}
	wrapped.Delete = func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
		w.deletes++
		if base.Delete != nil {
			return base.Delete(ctx, c, obj, opts...)
		}
		return c.Delete(ctx, obj, opts...)
	}

	return wrapped
}

func cmCertSecret(data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.InstrumentorWebhookSecretName,
			Namespace: cmNamespace,
		},
		Data: data,
	}
}

func cmAPIService(serviceName string, caBundle []byte) *apiregv1.APIService {
	apiSvc := &apiregv1.APIService{
		ObjectMeta: metav1.ObjectMeta{Name: k8sconsts.CustomMetricsAPIServiceName},
		Spec: apiregv1.APIServiceSpec{
			Group:    "custom.metrics.k8s.io",
			Version:  "v1beta1",
			CABundle: caBundle,
		},
	}
	if serviceName != "" {
		apiSvc.Spec.Service = &apiregv1.ServiceReference{Name: serviceName, Namespace: cmNamespace}
	}
	return apiSvc
}

func cmCertSecretRequest() ctrl.Request {
	return ctrl.Request{NamespacedName: client.ObjectKey{
		Name:      k8sconsts.InstrumentorWebhookSecretName,
		Namespace: cmNamespace,
	}}
}

// The CA sync is the only thing that lets the API server trust the instrumentor's aggregated
// endpoint. Every guard in front of the write is load-bearing: the APIService is a cluster-scoped
// singleton that another adapter may own, and overwriting its CA would break that adapter.
func TestCAUpdaterReconcile_DoesNotWrite(t *testing.T) {
	tests := []struct {
		name    string
		request ctrl.Request
		objects []client.Object
	}{
		{
			// The decoy carries a ca.crt of its own, so dropping the name guard would sync the
			// wrong CA rather than quietly doing nothing.
			name:    "a secret that is not the webhook cert",
			request: ctrl.Request{NamespacedName: client.ObjectKey{Name: "some-other-secret", Namespace: cmNamespace}},
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "some-other-secret", Namespace: cmNamespace},
					Data:       map[string][]byte{"ca.crt": []byte("an unrelated ca")},
				},
				cmAPIService(k8sconsts.InstrumentorServiceName, []byte("stale")),
			},
		},
		{
			name:    "the cert secret does not exist yet",
			request: cmCertSecretRequest(),
			objects: []client.Object{cmAPIService(k8sconsts.InstrumentorServiceName, []byte("stale"))},
		},
		{
			name:    "the cert secret carries no ca.crt",
			request: cmCertSecretRequest(),
			objects: []client.Object{
				cmCertSecret(map[string][]byte{"tls.crt": []byte("leaf")}),
				cmAPIService(k8sconsts.InstrumentorServiceName, []byte("stale")),
			},
		},
		{
			name:    "the cert secret carries an empty ca.crt",
			request: cmCertSecretRequest(),
			objects: []client.Object{
				cmCertSecret(map[string][]byte{"ca.crt": {}}),
				cmAPIService(k8sconsts.InstrumentorServiceName, []byte("stale")),
			},
		},
		{
			name:    "helm has not created the APIService yet",
			request: cmCertSecretRequest(),
			objects: []client.Object{cmCertSecret(map[string][]byte{"ca.crt": []byte("ca")})},
		},
		{
			name:    "the APIService belongs to another custom metrics adapter",
			request: cmCertSecretRequest(),
			objects: []client.Object{
				cmCertSecret(map[string][]byte{"ca.crt": []byte("ca")}),
				cmAPIService("prometheus-adapter", []byte("somebody elses ca")),
			},
		},
		{
			name:    "the APIService has no backing service at all",
			request: cmCertSecretRequest(),
			objects: []client.Object{
				cmCertSecret(map[string][]byte{"ca.crt": []byte("ca")}),
				cmAPIService("", []byte("somebody elses ca")),
			},
		},
		{
			name:    "the CA bundle is already in sync",
			request: cmCertSecretRequest(),
			objects: []client.Object{
				cmCertSecret(map[string][]byte{"ca.crt": []byte("ca")}),
				cmAPIService(k8sconsts.InstrumentorServiceName, []byte("ca")),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writes := &cmWrites{}
			c := fake.NewClientBuilder().WithScheme(cmScheme(t)).
				WithObjects(tt.objects...).
				WithInterceptorFuncs(writes.funcs(interceptor.Funcs{})).Build()

			before := cmAPIServiceOrNil(t, c)
			res, err := (&CAUpdaterReconciler{Client: c}).Reconcile(context.Background(), tt.request)

			require.NoError(t, err)
			assert.Equal(t, ctrl.Result{}, res)
			assert.Zero(t, writes.total(), "the reconcile must not touch the cluster")
			assert.Equal(t, before, cmAPIServiceOrNil(t, c))
		})
	}
}

func TestCAUpdaterReconcile_SyncsTheCABundle(t *testing.T) {
	writes := &cmWrites{}
	existing := cmAPIService(k8sconsts.InstrumentorServiceName, []byte("the autoscaler era ca"))
	existing.Labels = map[string]string{cmHelmManagedByLabel: "Helm"}

	c := fake.NewClientBuilder().WithScheme(cmScheme(t)).
		WithObjects(cmCertSecret(map[string][]byte{"ca.crt": []byte("the instrumentor ca")}), existing).
		WithInterceptorFuncs(writes.funcs(interceptor.Funcs{})).Build()

	_, err := (&CAUpdaterReconciler{Client: c}).Reconcile(context.Background(), cmCertSecretRequest())
	require.NoError(t, err)

	got := cmAPIServiceOrNil(t, c)
	require.NotNil(t, got)
	assert.Equal(t, []byte("the instrumentor ca"), got.Spec.CABundle)
	assert.Equal(t, 1, writes.updates)
	assert.Equal(t, 1, writes.total(), "only the APIService may be written")

	want := existing.DeepCopy()
	want.Spec.CABundle = got.Spec.CABundle
	want.ResourceVersion = got.ResourceVersion
	assert.Equal(t, want.Spec, got.Spec, "nothing but the CA bundle may change")
	assert.Equal(t, want.Labels, got.Labels)
}

// An autoscaler-era APIService is still Odigos-owned: the CA has to be re-synced onto it so the
// metric keeps working across the upgrade that moves the service reference.
func TestCAUpdaterReconcile_SyncsOntoTheAutoscalerEraAPIService(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(cmScheme(t)).WithObjects(
		cmCertSecret(map[string][]byte{"ca.crt": []byte("the instrumentor ca")}),
		cmAPIService(k8sconsts.AutoScalerWebhookServiceName, nil),
	).Build()

	_, err := (&CAUpdaterReconciler{Client: c}).Reconcile(context.Background(), cmCertSecretRequest())
	require.NoError(t, err)

	got := cmAPIServiceOrNil(t, c)
	require.NotNil(t, got)
	assert.Equal(t, []byte("the instrumentor ca"), got.Spec.CABundle)
}

func TestCAUpdaterReconcile_PropagatesFailures(t *testing.T) {
	tests := []struct {
		name    string
		funcs   interceptor.Funcs
		wantErr string
	}{
		{
			name: "reading the cert secret fails",
			funcs: interceptor.Funcs{
				Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if _, ok := obj.(*corev1.Secret); ok {
						return errors.New("secret get is forbidden")
					}
					return c.Get(ctx, key, obj, opts...)
				},
			},
			wantErr: "secret get is forbidden",
		},
		{
			name: "reading the APIService fails",
			funcs: interceptor.Funcs{
				Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if _, ok := obj.(*apiregv1.APIService); ok {
						return errors.New("apiservice get is forbidden")
					}
					return c.Get(ctx, key, obj, opts...)
				},
			},
			wantErr: "apiservice get is forbidden",
		},
		{
			name: "writing the APIService fails",
			funcs: interceptor.Funcs{
				Update: func(context.Context, client.WithWatch, client.Object, ...client.UpdateOption) error {
					return errors.New("apiservice update is forbidden")
				},
			},
			wantErr: "apiservice update is forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(cmScheme(t)).WithObjects(
				cmCertSecret(map[string][]byte{"ca.crt": []byte("ca")}),
				cmAPIService(k8sconsts.InstrumentorServiceName, []byte("stale")),
			).WithInterceptorFuncs(tt.funcs).Build()

			_, err := (&CAUpdaterReconciler{Client: c}).Reconcile(context.Background(), cmCertSecretRequest())

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func cmAPIServiceOrNil(t *testing.T, c client.Client) *apiregv1.APIService {
	t.Helper()

	apiSvc := &apiregv1.APIService{}
	err := c.Get(context.Background(), client.ObjectKey{Name: k8sconsts.CustomMetricsAPIServiceName}, apiSvc)
	if err != nil {
		return nil
	}
	return apiSvc
}
