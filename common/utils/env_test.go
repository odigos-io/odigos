package utils

import (
	"os"
	"testing"

	"github.com/odigos-io/odigos/common/consts"
)

func TestGetEnvVarOrDefault_Exists(t *testing.T) {
	const envKey = "TEST_ENV_EXISTS"
	const expectedVal = "exists"

	os.Setenv(envKey, expectedVal)
	defer os.Unsetenv(envKey)

	if got := getEnvVarOrDefault(envKey, "default"); got != expectedVal {
		t.Errorf("getEnvVarOrDefault(%q, %q) = %q, want %q", envKey, "default", got, expectedVal)
	}
}

func TestGetEnvVarOrDefault_NotExists(t *testing.T) {
	const envKey = "TEST_ENV_NOT_EXISTS"
	const defaultVal = "default"

	if got := getEnvVarOrDefault(envKey, defaultVal); got != defaultVal {
		t.Errorf("getEnvVarOrDefault(%q, %q) = %q, want %q", envKey, defaultVal, got, defaultVal)
	}
}

// TestGetCurrentNamespace_EnvVarExists tests GetCurrentNamespace when the environment variable is set.
func TestGetCurrentNamespace_EnvVarExists(t *testing.T) {
	expectedNamespace := "test-namespace"
	os.Setenv(consts.CurrentNamespaceEnvVar, expectedNamespace)
	defer os.Unsetenv(consts.CurrentNamespaceEnvVar)

	got := GetCurrentNamespace()
	if got != expectedNamespace {
		t.Errorf("GetCurrentNamespace() = %q, want %q", got, expectedNamespace)
	}
}

// TestGetCurrentNamespace_EnvVarNotExists tests GetCurrentNamespace when the environment variable is not set.
func TestGetCurrentNamespace_EnvVarNotExists(t *testing.T) {
	os.Unsetenv(consts.CurrentNamespaceEnvVar) // Ensure the environment variable is not set

	expectedNamespace := consts.DefaultNamespace
	got := GetCurrentNamespace()
	if got != expectedNamespace {
		t.Errorf("GetCurrentNamespace() = %q, want %q", got, expectedNamespace)
	}
}
