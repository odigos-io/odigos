package utils

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

// clusterVersionServer serves the /version endpoint the discovery client queries, and points
// $KUBECONFIG at itself so ClusterVersion resolves to this server.
func clusterVersionServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	path := filepath.Join(t.TempDir(), "config")
	require.NoError(t, os.WriteFile(path, []byte(`apiVersion: v1
kind: Config
current-context: test-context
clusters:
- name: test-cluster
  cluster:
    server: `+server.URL+`
contexts:
- name: test-context
  context:
    cluster: test-cluster
    user: test-user
users:
- name: test-user
  user: {}
`), 0o600))

	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	require.NoError(t, os.Unsetenv("KUBERNETES_SERVICE_HOST"))
	t.Setenv(env.KUBECONFIG, path)
}

func serveVersionInfo(t *testing.T, body string) {
	t.Helper()

	clusterVersionServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/version" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})
}

func TestClusterVersionParsesTheReportedGitVersion(t *testing.T) {
	tests := []struct {
		name       string
		gitVersion string
		major      uint
		minor      uint
		patch      uint
	}{
		{name: "plain release", gitVersion: "v1.31.7", major: 1, minor: 31, patch: 7},
		// Distributions append their own build suffix; a parser that chokes on it would take the
		// instrumentor down on every k3s and GKE cluster.
		{name: "k3s build suffix", gitVersion: "v1.29.3+k3s2", major: 1, minor: 29, patch: 3},
		{name: "gke build suffix", gitVersion: "v1.30.5-gke.1443001", major: 1, minor: 30, patch: 5},
		{name: "no leading v", gitVersion: "1.28.15", major: 1, minor: 28, patch: 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serveVersionInfo(t, `{"major":"1","minor":"22","gitVersion":"`+tt.gitVersion+`"}`)

			version, err := ClusterVersion()

			require.NoError(t, err)
			require.NotNil(t, version)
			assert.Equal(t, tt.major, version.Major())
			// The major/minor fields served alongside gitVersion are deliberately stale: managed
			// distributions round them, so only gitVersion carries the real patch level.
			assert.Equal(t, tt.minor, version.Minor())
			assert.Equal(t, tt.patch, version.Patch())
		})
	}
}

func TestClusterVersionRejectsAnUnparsableGitVersion(t *testing.T) {
	serveVersionInfo(t, `{"major":"1","minor":"29","gitVersion":"not-a-version"}`)

	version, err := ClusterVersion()

	require.Error(t, err)
	assert.Nil(t, version)
	assert.ErrorContains(t, err, `parse "not-a-version"`)
}

func TestClusterVersionReportsAFailedVersionQuery(t *testing.T) {
	clusterVersionServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	version, err := ClusterVersion()

	require.Error(t, err)
	assert.Nil(t, version)
	assert.ErrorContains(t, err, "query /version")
}

func TestClusterVersionReportsAnUnusableKubeConfig(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	require.NoError(t, os.Unsetenv("KUBERNETES_SERVICE_HOST"))
	t.Setenv(env.KUBECONFIG, filepath.Join(t.TempDir(), "absent"))

	version, err := ClusterVersion()

	require.Error(t, err)
	assert.Nil(t, version)
	assert.ErrorContains(t, err, "build kube config")
}

// The three failure points are reported with different prefixes, so a support bundle says whether
// the kubeconfig, the client or the apiserver was the problem.
func TestClusterVersionFailureStagesAreDistinguishable(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	require.NoError(t, os.Unsetenv("KUBERNETES_SERVICE_HOST"))
	t.Setenv(env.KUBECONFIG, filepath.Join(t.TempDir(), "absent"))
	_, buildErr := ClusterVersion()
	require.Error(t, buildErr)

	serveVersionInfo(t, `{"major":"1","minor":"29","gitVersion":"not-a-version"}`)
	_, parseErr := ClusterVersion()
	require.Error(t, parseErr)

	clusterVersionServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	_, queryErr := ClusterVersion()
	require.Error(t, queryErr)

	assert.NotEqual(t, buildErr.Error(), queryErr.Error())
	assert.NotEqual(t, buildErr.Error(), parseErr.Error())
	assert.NotEqual(t, queryErr.Error(), parseErr.Error())
	assert.NotContains(t, buildErr.Error(), "query /version")
	assert.NotContains(t, queryErr.Error(), "build kube config")
	assert.NotContains(t, parseErr.Error(), "query /version")
}
