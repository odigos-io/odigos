package utils

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
)

const tierNamespace = "odigos-tier-test-ns"

// tierAPIServer is a minimal stand-in for the apiserver. GetCurrentOdigosTier takes a concrete
// *kubernetes.Clientset, so the fake clientset cannot be used and a real client has to be pointed
// at an HTTP stub instead.
type tierAPIServer struct {
	objects map[string][]byte
	paths   []string
}

func newTierAPIServer() *tierAPIServer {
	return &tierAPIServer{objects: map[string][]byte{}}
}

func (s *tierAPIServer) put(t *testing.T, path string, object any) {
	t.Helper()

	encoded, err := json.Marshal(object)
	require.NoError(t, err)
	s.objects[path] = encoded
}

func (s *tierAPIServer) secretPath(namespace, name string) string {
	return "/api/v1/namespaces/" + namespace + "/secrets/" + name
}

func (s *tierAPIServer) daemonSetPath(namespace, name string) string {
	return "/apis/apps/v1/namespaces/" + namespace + "/daemonsets/" + name
}

func (s *tierAPIServer) stored(t *testing.T, path string, into any) {
	t.Helper()

	body, ok := s.objects[path]
	require.True(t, ok, "nothing was stored at %s", path)
	require.NoError(t, json.Unmarshal(body, into))
}

func (s *tierAPIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.paths = append(s.paths, r.Method+" "+r.URL.Path)
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		s.objects[r.URL.Path] = body
		_, _ = w.Write(body)
		return
	case http.MethodPost:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var created metav1.PartialObjectMetadata
		if err := json.Unmarshal(body, &created); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		s.objects[strings.TrimSuffix(r.URL.Path, "/")+"/"+created.Name] = body
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(body)
		return
	}

	body, ok := s.objects[r.URL.Path]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		notFound, _ := json.Marshal(&metav1.Status{
			TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"},
			Status:   metav1.StatusFailure,
			Code:     http.StatusNotFound,
			Reason:   metav1.StatusReasonNotFound,
			Message:  "not found: " + r.URL.Path,
		})
		_, _ = w.Write(notFound)
		return
	}
	_, _ = w.Write(body)
}

// tierClientset builds a real clientset against the given handler. ContentType has to be pinned to
// JSON because client-go negotiates protobuf for writes by default, and QPS is disabled so the
// client-side rate limiter cannot slow a multi-request test down.
func tierClientset(t *testing.T, handler http.Handler) *kubernetes.Clientset {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	clientset, err := kubernetes.NewForConfig(&rest.Config{
		Host:          server.URL,
		ContentConfig: rest.ContentConfig{ContentType: "application/json"},
		QPS:           -1,
	})
	require.NoError(t, err)
	return clientset
}

func tierProSecret(data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosProSecretName,
			Namespace: tierNamespace,
		},
		Data: data,
	}
}

func TestGetCurrentOdigosTier(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string][]byte
		absent   bool
		expected common.OdigosTier
	}{
		{
			name:     "no pro secret at all is a community install",
			absent:   true,
			expected: common.CommunityOdigosTier,
		},
		{
			name:     "a pro secret with no recognised key is a community install",
			data:     map[string][]byte{"some-other-key": []byte("value")},
			expected: common.CommunityOdigosTier,
		},
		{
			name:     "the cloud api key marks a cloud install",
			data:     map[string][]byte{k8sconsts.OdigosCloudApiKeySecretKey: []byte("cloud-key")},
			expected: common.CloudOdigosTier,
		},
		{
			name:     "the onprem token marks an on-prem install",
			data:     map[string][]byte{k8sconsts.OdigosOnpremTokenSecretKey: []byte("onprem-token")},
			expected: common.OnPremOdigosTier,
		},
		{
			name: "cloud wins when both keys are present",
			data: map[string][]byte{
				k8sconsts.OdigosCloudApiKeySecretKey: []byte("cloud-key"),
				k8sconsts.OdigosOnpremTokenSecretKey: []byte("onprem-token"),
			},
			expected: common.CloudOdigosTier,
		},
		{
			// A key present with an empty value is still a key: Helm renders the secret before the
			// value is known, and treating that as community would silently downgrade the install.
			name:     "an empty onprem token value still marks an on-prem install",
			data:     map[string][]byte{k8sconsts.OdigosOnpremTokenSecretKey: {}},
			expected: common.OnPremOdigosTier,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTierAPIServer()
			if !tt.absent {
				server.put(t, server.secretPath(tierNamespace, k8sconsts.OdigosProSecretName), tierProSecret(tt.data))
			}

			tier, err := GetCurrentOdigosTier(context.Background(), tierNamespace, tierClientset(t, server))

			require.NoError(t, err)
			assert.Equal(t, tt.expected, tier)
		})
	}
}

func TestGetCurrentOdigosTierReadsTheNamespaceItWasGiven(t *testing.T) {
	server := newTierAPIServer()
	server.put(t, server.secretPath("some-other-namespace", k8sconsts.OdigosProSecretName),
		tierProSecret(map[string][]byte{k8sconsts.OdigosOnpremTokenSecretKey: []byte("onprem-token")}))

	tier, err := GetCurrentOdigosTier(context.Background(), tierNamespace, tierClientset(t, server))

	require.NoError(t, err)
	assert.Equal(t, common.CommunityOdigosTier, tier)
	require.Len(t, server.paths, 1)
	assert.Equal(t, "GET "+server.secretPath(tierNamespace, k8sconsts.OdigosProSecretName), server.paths[0])
}

// An unreachable apiserver must not be reported as a community install: the pro features would be
// disabled on a cluster that has paid for them.
func TestGetCurrentOdigosTierPropagatesAReadError(t *testing.T) {
	clientset := tierClientset(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Failure","code":500,"reason":"InternalError","message":"etcd is down"}`))
	}))

	tier, err := GetCurrentOdigosTier(context.Background(), tierNamespace, clientset)

	require.Error(t, err)
	assert.Empty(t, string(tier))
	assert.NotEqual(t, common.CommunityOdigosTier, tier)
}

// Objects returned by a Get come back with an empty TypeMeta, which the CLI needs when it prints
// the secret back out as yaml.
func TestGetCurrentOdigosProSecretRestoresTheTypeMeta(t *testing.T) {
	server := newTierAPIServer()
	stored := tierProSecret(map[string][]byte{k8sconsts.OdigosOnpremTokenSecretKey: []byte("onprem-token")})
	stored.TypeMeta = metav1.TypeMeta{}
	server.put(t, server.secretPath(tierNamespace, k8sconsts.OdigosProSecretName), stored)

	secret, err := getCurrentOdigosProSecret(context.Background(), tierNamespace, tierClientset(t, server))

	require.NoError(t, err)
	require.NotNil(t, secret)
	assert.Equal(t, metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"}, secret.TypeMeta)
}

func TestGetCurrentOdigosProSecretReturnsNilWhenThereIsNoProSecret(t *testing.T) {
	secret, err := getCurrentOdigosProSecret(context.Background(), tierNamespace, tierClientset(t, newTierAPIServer()))

	require.NoError(t, err)
	assert.Nil(t, secret)
}

// The two secret keys are spelled out again in this package, while everything that writes them
// uses the api/k8sconsts definitions. Nothing links the two copies at compile time, so a rename on
// either side would silently turn a pro install into a community one.
func TestTheTierKeysMatchTheConstantsUsedToWriteTheProSecret(t *testing.T) {
	assert.Equal(t, k8sconsts.OdigosCloudApiKeySecretKey, odigosCloudApiKeySecretKey)
	assert.Equal(t, k8sconsts.OdigosOnpremTokenSecretKey, odigosOnpremTokenSecretKey)
	// Pinned as literals too: these are the data keys of a secret that Helm, the CLI and the UI all
	// write independently of this constant.
	assert.Equal(t, "odigos-cloud-api-key", odigosCloudApiKeySecretKey)
	assert.Equal(t, "odigos-onprem-token", odigosOnpremTokenSecretKey)
	assert.NotEqual(t, odigosCloudApiKeySecretKey, odigosOnpremTokenSecretKey)
	assert.Equal(t, "odigos-pro", k8sconsts.OdigosProSecretName)
}

// The three tiers have to stay distinguishable: every gate in the product is a comparison against
// one of these values.
func TestTheOdigosTiersAreDistinct(t *testing.T) {
	tiers := []common.OdigosTier{common.CommunityOdigosTier, common.CloudOdigosTier, common.OnPremOdigosTier}

	seen := map[common.OdigosTier]bool{}
	for _, tier := range tiers {
		assert.NotEmpty(t, string(tier))
		assert.False(t, seen[tier], "duplicate tier value %q", tier)
		seen[tier] = true
	}
	assert.Len(t, seen, len(tiers))
	assert.False(t, strings.EqualFold(string(common.CloudOdigosTier), string(common.OnPremOdigosTier)))
}
