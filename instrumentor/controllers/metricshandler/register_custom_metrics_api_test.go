package metricshandler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// cmStubManager exposes only what RegisterCustomMetricsAPI uses. The embedded nil interface makes
// any other manager call panic, so the test cannot silently start depending on more of it.
type cmStubManager struct {
	manager.Manager
	c             client.Client
	webhookServer *cmRecordingWebhookServer
}

func (m *cmStubManager) GetClient() client.Client         { return m.c }
func (m *cmStubManager) GetWebhookServer() webhook.Server { return m.webhookServer }

type cmRecordingWebhookServer struct {
	webhook.Server
	registered map[string]http.Handler
}

func (s *cmRecordingWebhookServer) Register(path string, hook http.Handler) {
	s.registered[path] = hook
}

func cmRegister(t *testing.T, c client.Client) (*cmRecordingWebhookServer, error) {
	t.Helper()
	t.Setenv(consts.CurrentNamespaceEnvVar, cmNamespace)

	srv := &cmRecordingWebhookServer{registered: map[string]http.Handler{}}
	return srv, RegisterCustomMetricsAPI(&cmStubManager{c: c, webhookServer: srv})
}

// The aggregated API server routes a request for the advertised resource to a path it builds from
// the discovery document. The path is assembled from string literals here, so the only thing
// keeping the two in step is this contract.
func TestRegisterCustomMetricsAPI_RegistersThePathsDiscoveryAdvertises(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(cmScheme(t)).WithObjects(
		cmCertSecret(map[string][]byte{"ca.crt": []byte("ca")}),
		cmAPIService(k8sconsts.InstrumentorServiceName, []byte("ca")),
	).Build()

	srv, err := cmRegister(t, c)
	require.NoError(t, err)

	group, version := cmServedGroupVersion(t)
	discoveryPath := fmt.Sprintf("/apis/%s/%s", group, version)
	metricPath := fmt.Sprintf("%s/namespaces/%s/%s/%s/%s",
		discoveryPath, cmNamespace,
		cmAdvertisedResourceType(t),
		k8sconsts.OdigosClusterCollectorDeploymentName,
		cmAdvertisedMetricName(t))

	assert.ElementsMatch(t, []string{discoveryPath, metricPath}, cmPaths(srv))

	rec := cmCall(t, srv.registered[discoveryPath].ServeHTTP)
	assert.Contains(t, rec.Body.String(), cmAdvertisedMetricName(t))
}

// Helm creates the APIService; until it does, or when another adapter owns it, registration still
// has to bring the HTTP routes up so the metric is served the moment the object appears.
func TestRegisterCustomMetricsAPI_ServesEvenWithoutAnOdigosAPIService(t *testing.T) {
	tests := []struct {
		name    string
		apiSvc  client.Object
		wantCA  []byte
		writing bool
	}{
		{name: "helm has not created it yet"},
		{
			name:   "another adapter owns it",
			apiSvc: cmAPIService("prometheus-adapter", []byte("somebody elses ca")),
			wantCA: []byte("somebody elses ca"),
		},
		{
			name:   "it is already in sync",
			apiSvc: cmAPIService(k8sconsts.InstrumentorServiceName, []byte("ca")),
			wantCA: []byte("ca"),
		},
		{
			name:    "its CA is stale",
			apiSvc:  cmAPIService(k8sconsts.InstrumentorServiceName, []byte("the autoscaler era ca")),
			wantCA:  []byte("ca"),
			writing: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := []client.Object{cmCertSecret(map[string][]byte{"ca.crt": []byte("ca")})}
			if tt.apiSvc != nil {
				objects = append(objects, tt.apiSvc)
			}

			writes := &cmWrites{}
			c := fake.NewClientBuilder().WithScheme(cmScheme(t)).WithObjects(objects...).
				WithInterceptorFuncs(writes.funcs(interceptor.Funcs{})).Build()

			srv, err := cmRegister(t, c)
			require.NoError(t, err)
			assert.Len(t, cmPaths(srv), 2, "the metric must be served regardless of the APIService")

			if tt.writing {
				assert.Equal(t, 1, writes.updates)
			} else {
				assert.Zero(t, writes.total())
			}
			if tt.apiSvc != nil {
				assert.Equal(t, tt.wantCA, cmAPIServiceOrNil(t, c).Spec.CABundle)
			}
		})
	}
}

func TestRegisterCustomMetricsAPI_Failures(t *testing.T) {
	tests := []struct {
		name    string
		objects []client.Object
		funcs   interceptor.Funcs
		wantErr string
	}{
		{
			name:    "the cert secret is missing",
			wantErr: "failed to get cert secret",
		},
		{
			name:    "the cert secret has no ca.crt",
			objects: []client.Object{cmCertSecret(map[string][]byte{"tls.crt": []byte("leaf")})},
			wantErr: "ca.crt not found in secret",
		},
		{
			name:    "the APIService cannot be read",
			objects: []client.Object{cmCertSecret(map[string][]byte{"ca.crt": []byte("ca")})},
			funcs: interceptor.Funcs{
				Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if key.Name == k8sconsts.CustomMetricsAPIServiceName {
						return errors.New("apiservices get is forbidden")
					}
					return c.Get(ctx, key, obj, opts...)
				},
			},
			wantErr: "apiservices get is forbidden",
		},
		{
			name: "the APIService cannot be updated",
			objects: []client.Object{
				cmCertSecret(map[string][]byte{"ca.crt": []byte("ca")}),
				cmAPIService(k8sconsts.InstrumentorServiceName, []byte("stale")),
			},
			funcs: interceptor.Funcs{
				Update: func(context.Context, client.WithWatch, client.Object, ...client.UpdateOption) error {
					return errors.New("apiservices update is forbidden")
				},
			},
			wantErr: "failed to update APIService",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(cmScheme(t)).
				WithObjects(tt.objects...).WithInterceptorFuncs(tt.funcs).Build()

			srv, err := cmRegister(t, c)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.Empty(t, cmPaths(srv), "a failed registration must not leave half the API served")
		})
	}
}

func cmPaths(srv *cmRecordingWebhookServer) []string {
	paths := make([]string, 0, len(srv.registered))
	for p := range srv.registered {
		paths = append(paths, p)
	}
	return paths
}

func cmAdvertisedResourceType(t *testing.T) string {
	t.Helper()
	resourceType, _ := cmSplitAdvertisedResource(t)
	return resourceType
}

func cmAdvertisedMetricName(t *testing.T) string {
	t.Helper()
	_, metricName := cmSplitAdvertisedResource(t)
	return metricName
}
