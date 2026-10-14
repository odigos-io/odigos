package kube

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// writeKubeconfig writes a kubeconfig with one context per entry of servers,
// each pointing at that server, and returns its path.
func writeKubeconfig(t *testing.T, currentContext string, servers map[string]string) string {
	t.Helper()
	cfg := clientcmdapi.NewConfig()
	for name, server := range servers {
		cfg.Clusters[name] = &clientcmdapi.Cluster{Server: server}
		cfg.AuthInfos[name] = &clientcmdapi.AuthInfo{}
		cfg.Contexts[name] = &clientcmdapi.Context{Cluster: name, AuthInfo: name}
	}
	cfg.CurrentContext = currentContext
	path := filepath.Join(t.TempDir(), "kubeconfig")
	require.NoError(t, clientcmd.WriteToFile(*cfg, path))
	return path
}

func TestCacheRestConfig(t *testing.T) {
	// Out of cluster, so k8sutils does not fall back to the in-cluster config.
	t.Setenv("KUBERNETES_SERVICE_HOST", "")

	envPath := writeKubeconfig(t, "env", map[string]string{
		"env": "https://env.example:6443",
	})
	flagPath := writeKubeconfig(t, "flag", map[string]string{
		"flag":  "https://flag.example:6443",
		"other": "https://other.example:6443",
	})
	t.Setenv("KUBECONFIG", envPath)

	tests := []struct {
		name        string
		kubeConfig  string
		kubeContext string
		wantHost    string
	}{
		{
			name:       "kubeconfig path wins over KUBECONFIG",
			kubeConfig: flagPath,
			wantHost:   "https://flag.example:6443",
		},
		{
			name:        "kubeconfig path with context",
			kubeConfig:  flagPath,
			kubeContext: "other",
			wantHost:    "https://other.example:6443",
		},
		{
			name:     "no kubeconfig path reads KUBECONFIG",
			wantHost: "https://env.example:6443",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := cacheRestConfig(tt.kubeConfig, tt.kubeContext)
			require.NoError(t, err)
			require.Equal(t, tt.wantHost, cfg.Host)
			require.Equal(t, float32(-1), cfg.QPS, "client-side rate limiting must stay off for the cache")
		})
	}
}
