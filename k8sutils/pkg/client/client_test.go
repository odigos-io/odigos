package client

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	stagingContextName = "staging-context"
	stagingClusterName = "staging-cluster"
	stagingServerURL   = "https://staging.example.test:6443"

	prodContextName = "prod-context"
	prodClusterName = "prod-cluster"
	prodServerURL   = "https://prod.example.test:8443"
)

// twoClusterKubeConfig writes a kubeconfig with two fully distinct contexts so that a swap
// between the context name, the cluster name and the server endpoint is observable.
func twoClusterKubeConfig(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config")
	contents := `apiVersion: v1
kind: Config
current-context: ` + stagingContextName + `
clusters:
- name: ` + stagingClusterName + `
  cluster:
    server: ` + stagingServerURL + `
    insecure-skip-tls-verify: true
- name: ` + prodClusterName + `
  cluster:
    server: ` + prodServerURL + `
    insecure-skip-tls-verify: true
contexts:
- name: ` + stagingContextName + `
  context:
    cluster: ` + stagingClusterName + `
    user: staging-user
- name: ` + prodContextName + `
  context:
    cluster: ` + prodClusterName + `
    user: prod-user
users:
- name: staging-user
  user:
    token: staging-token
- name: prod-user
  user:
    token: prod-token
`
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return path
}

func writeKubeConfig(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return path
}

// outsideKubernetes makes IsRunningInKubernetes report false regardless of how the test process
// was started.
func outsideKubernetes(t *testing.T) {
	t.Helper()
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	require.NoError(t, os.Unsetenv("KUBERNETES_SERVICE_HOST"))
}

func TestIsRunningInKubernetes(t *testing.T) {
	tests := []struct {
		name     string
		set      bool
		value    string
		expected bool
	}{
		{name: "variable is absent", set: false, expected: false},
		// An empty value is what a pod spec with an unresolved variable produces; it is not a
		// cluster, and treating presence alone as "in cluster" would send in-cluster config down
		// a path with no apiserver address.
		{name: "variable is present but empty", set: true, value: "", expected: false},
		{name: "variable holds an apiserver address", set: true, value: "10.96.0.1", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.set {
				t.Setenv("KUBERNETES_SERVICE_HOST", tt.value)
			} else {
				outsideKubernetes(t)
			}
			assert.Equal(t, tt.expected, IsRunningInKubernetes())
		})
	}
}

func TestGetClientConfigWithContextUsesTheCurrentContextWhenNoneIsRequested(t *testing.T) {
	outsideKubernetes(t)

	cfg, err := GetClientConfigWithContext(twoClusterKubeConfig(t), "")

	require.NoError(t, err)
	assert.Equal(t, stagingServerURL, cfg.Host)
}

func TestGetClientConfigWithContextHonoursTheRequestedContext(t *testing.T) {
	outsideKubernetes(t)

	cfg, err := GetClientConfigWithContext(twoClusterKubeConfig(t), prodContextName)

	require.NoError(t, err)
	assert.Equal(t, prodServerURL, cfg.Host)
	assert.Equal(t, "prod-token", cfg.BearerToken)
}

func TestGetClientConfigWithContextRejectsAnUnknownContext(t *testing.T) {
	outsideKubernetes(t)

	cfg, err := GetClientConfigWithContext(twoClusterKubeConfig(t), "context-that-does-not-exist")

	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.ErrorContains(t, err, "context-that-does-not-exist")
}

func TestGetClientConfigWithContextRejectsAMissingKubeConfigFile(t *testing.T) {
	outsideKubernetes(t)

	cfg, err := GetClientConfigWithContext(filepath.Join(t.TempDir(), "absent"), "")

	require.Error(t, err)
	assert.Nil(t, cfg)
}

// The explicit path is the only source of kubeconfig this helper consults; a caller that leaves it
// empty does not silently fall back to $KUBECONFIG.
func TestGetClientConfigWithContextDoesNotFallBackToTheKubeConfigEnvVar(t *testing.T) {
	outsideKubernetes(t)
	t.Setenv("KUBECONFIG", twoClusterKubeConfig(t))

	cfg, err := GetClientConfigWithContext("", "")

	require.Error(t, err)
	assert.Nil(t, cfg)
}

// Inside a pod the kubeconfig must be ignored entirely, otherwise a stray kubeconfig baked into an
// image would point a controller at someone else's cluster.
func TestGetClientConfigWithContextIgnoresTheKubeConfigWhenRunningInKubernetes(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "10.96.0.1")
	t.Setenv("KUBERNETES_SERVICE_PORT", "443")

	cfg, err := GetClientConfigWithContext(twoClusterKubeConfig(t), prodContextName)

	// There is no service account token mounted in this process, so in-cluster config fails - the
	// point of the assertion is that the perfectly valid kubeconfig was not used as a fallback.
	require.Error(t, err)
	assert.Nil(t, cfg)
}

func TestGetCurrentClusterDetails(t *testing.T) {
	tests := []struct {
		name     string
		kContext string
		expected ClusterDetails
	}{
		{
			name:     "no context requested falls back to the kubeconfig current-context",
			kContext: "",
			expected: ClusterDetails{
				CurrentContext: stagingContextName,
				ClusterName:    stagingClusterName,
				ServerEndpoint: stagingServerURL,
			},
		},
		{
			name:     "an explicit context overrides current-context",
			kContext: prodContextName,
			expected: ClusterDetails{
				CurrentContext: prodContextName,
				ClusterName:    prodClusterName,
				ServerEndpoint: prodServerURL,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetCurrentClusterDetails(twoClusterKubeConfig(t), tt.kContext))
		})
	}
}

func TestGetCurrentClusterDetailsReturnsNothingForAnUnreadableKubeConfig(t *testing.T) {
	tests := []struct {
		name string
		path func(t *testing.T) string
	}{
		{
			name: "file does not exist",
			path: func(t *testing.T) string { return filepath.Join(t.TempDir(), "absent") },
		},
		{
			name: "file is not valid kubeconfig yaml",
			path: func(t *testing.T) string { return writeKubeConfig(t, "\tthis is not yaml: [") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, ClusterDetails{}, GetCurrentClusterDetails(tt.path(t), ""))
		})
	}
}

// current-context naming a context that was deleted from the file is not fatal: the CLI still
// reports the context name it was asked about, with nothing resolved behind it.
func TestGetCurrentClusterDetailsWithADanglingCurrentContext(t *testing.T) {
	path := writeKubeConfig(t, `apiVersion: v1
kind: Config
current-context: `+stagingContextName+`
clusters:
- name: `+stagingClusterName+`
  cluster:
    server: `+stagingServerURL+`
contexts: []
users: []
`)

	assert.Equal(t, ClusterDetails{CurrentContext: stagingContextName}, GetCurrentClusterDetails(path, ""))
}

// A context whose cluster entry is missing resolves the cluster name but leaves the endpoint
// empty, rather than reporting some other cluster's server.
func TestGetCurrentClusterDetailsWithAContextPointingAtAMissingCluster(t *testing.T) {
	path := writeKubeConfig(t, `apiVersion: v1
kind: Config
current-context: `+stagingContextName+`
clusters:
- name: `+prodClusterName+`
  cluster:
    server: `+prodServerURL+`
contexts:
- name: `+stagingContextName+`
  context:
    cluster: `+stagingClusterName+`
    user: staging-user
users: []
`)

	assert.Equal(t, ClusterDetails{
		CurrentContext: stagingContextName,
		ClusterName:    stagingClusterName,
	}, GetCurrentClusterDetails(path, ""))
}

// GetCurrentClusterDetails terminates the process when an explicitly requested context is missing,
// so the behaviour can only be observed from a child process.
func TestGetCurrentClusterDetailsExitsForAnUnknownRequestedContext(t *testing.T) {
	if os.Getenv("ODIGOS_TEST_EXIT_ON_UNKNOWN_CONTEXT") == "1" {
		GetCurrentClusterDetails(os.Getenv("ODIGOS_TEST_KUBECONFIG"), "context-that-does-not-exist")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestGetCurrentClusterDetailsExitsForAnUnknownRequestedContext", "-test.v")
	cmd.Env = append(os.Environ(),
		"ODIGOS_TEST_EXIT_ON_UNKNOWN_CONTEXT=1",
		"ODIGOS_TEST_KUBECONFIG="+twoClusterKubeConfig(t),
	)
	output, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr, "expected a non-zero exit, got output: %s", output)
	assert.Equal(t, 1, exitErr.ExitCode())
	assert.Contains(t, string(output), "Context context-that-does-not-exist not found in kubeconfig, bailing")
}

// The same missing context is tolerated when it came from current-context rather than from the
// caller, so an explicit --context typo is the only case that aborts.
func TestGetCurrentClusterDetailsDoesNotExitForADanglingCurrentContext(t *testing.T) {
	path := writeKubeConfig(t, `apiVersion: v1
kind: Config
current-context: `+prodContextName+`
clusters: []
contexts: []
users: []
`)

	assert.Equal(t, ClusterDetails{CurrentContext: prodContextName}, GetCurrentClusterDetails(path, ""))
}

func TestGetK8sClientsetFailsOutsideOfACluster(t *testing.T) {
	outsideKubernetes(t)

	clientset, err := GetK8sClientset()

	require.Error(t, err)
	assert.Nil(t, clientset)
}

func TestIsResourceAvailable(t *testing.T) {
	appsV1 := schema.GroupVersion{Group: "apps", Version: "v1"}
	odigosV1 := schema.GroupVersion{Group: "odigos.io", Version: "v1alpha1"}

	mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{appsV1, odigosV1})
	mapper.Add(appsV1.WithKind("Deployment"), meta.RESTScopeNamespace)
	mapper.Add(odigosV1.WithKind("Source"), meta.RESTScopeNamespace)

	tests := []struct {
		name     string
		gvk      schema.GroupVersionKind
		expected bool
	}{
		{name: "registered kind", gvk: appsV1.WithKind("Deployment"), expected: true},
		{name: "registered kind in another group", gvk: odigosV1.WithKind("Source"), expected: true},
		// The CRD is not installed - the whole point of the RESTMapper check is that callers can
		// skip the resource instead of erroring out on every cluster that lacks it.
		{name: "kind that is not installed", gvk: schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "Rollout"}, expected: false},
		{name: "known kind at an unserved version", gvk: schema.GroupVersionKind{Group: "apps", Version: "v1beta1", Kind: "Deployment"}, expected: false},
		{name: "known group and version but unknown kind", gvk: appsV1.WithKind("NotAKind"), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsResourceAvailable(mapper, tt.gvk))
		})
	}
}
