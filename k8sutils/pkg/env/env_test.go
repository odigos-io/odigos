package env

import (
	"os"
	"testing"

	"github.com/odigos-io/odigos/common/consts"
)

func TestGetEnvVarOrDefault_Exists(t *testing.T) {
	const envKey = "TEST_ENV_EXISTS"
	const expectedVal = "exists"

	t.Setenv(envKey, expectedVal)

	if got := GetEnvVarOrDefault(envKey, "default"); got != expectedVal {
		t.Errorf("GetEnvVarOrDefault(%q, %q) = %q, want %q", envKey, "default", got, expectedVal)
	}
}

func TestGetEnvVarOrDefault_NotExists(t *testing.T) {
	const envKey = "TEST_ENV_NOT_EXISTS"
	const defaultVal = "default"

	if got := GetEnvVarOrDefault(envKey, defaultVal); got != defaultVal {
		t.Errorf("GetEnvVarOrDefault(%q, %q) = %q, want %q", envKey, defaultVal, got, defaultVal)
	}
}

// TestGetCurrentNamespace_EnvVarExists tests GetCurrentNamespace when the environment variable is set.
func TestGetCurrentNamespace_EnvVarExists(t *testing.T) {
	expectedNamespace := "test-namespace"
	t.Setenv(consts.CurrentNamespaceEnvVar, expectedNamespace)

	got := GetCurrentNamespace()
	if got != expectedNamespace {
		t.Errorf("GetCurrentNamespace() = %q, want %q", got, expectedNamespace)
	}
}

// TestGetCurrentNamespace_EnvVarNotExists tests GetCurrentNamespace when the environment variable is not set.
func TestGetCurrentNamespace_EnvVarNotExists(t *testing.T) {
	os.Unsetenv(consts.CurrentNamespaceEnvVar) // Ensure the environment variable is not set

	expectedNamespace := consts.DefaultOdigosNamespace
	got := GetCurrentNamespace()
	if got != expectedNamespace {
		t.Errorf("GetCurrentNamespace() = %q, want %q", got, expectedNamespace)
	}
}
