package env

import (
	"os"
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
)

func TestGetEnvVarOrDefault_Exists(t *testing.T) {
	const envKey = "TEST_ENV_EXISTS"
	const expectedVal = "exists"

	os.Setenv(envKey, expectedVal)
	defer os.Unsetenv(envKey)

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

	expectedNamespace := consts.DefaultOdigosNamespace
	got := GetCurrentNamespace()
	if got != expectedNamespace {
		t.Errorf("GetCurrentNamespace() = %q, want %q", got, expectedNamespace)
	}
}

// This is the only place the tier is normalized: common.OdigosTier.IsEnterprise() treats every
// value other than "community" as enterprise, and enterprise-only collector components crash an
// OSS collector at config load. An unset or unrecognized value must therefore resolve to
// community, not to the empty tier.
func TestGetOdigosTierFromEnv(t *testing.T) {
	testCases := []struct {
		name     string
		envValue string
		setEnv   bool
		expected common.OdigosTier
	}{
		{"unset", "", false, common.CommunityOdigosTier},
		{"empty", "", true, common.CommunityOdigosTier},
		{"unrecognized", "enterprise", true, common.CommunityOdigosTier},
		{"wrong case", "OnPrem", true, common.CommunityOdigosTier},
		{"community", string(common.CommunityOdigosTier), true, common.CommunityOdigosTier},
		{"cloud", string(common.CloudOdigosTier), true, common.CloudOdigosTier},
		{"onprem", string(common.OnPremOdigosTier), true, common.OnPremOdigosTier},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// t.Setenv registers the restore of any pre-existing value, so unsetting after it
			// keeps the case isolated from the ambient environment.
			t.Setenv(consts.OdigosTierEnvVarName, tc.envValue)
			if !tc.setEnv {
				os.Unsetenv(consts.OdigosTierEnvVarName)
			}

			got := GetOdigosTierFromEnv()
			if got != tc.expected {
				t.Errorf("GetOdigosTierFromEnv() = %q, want %q", got, tc.expected)
			}
			if got.IsEnterprise() != (tc.expected != common.CommunityOdigosTier) {
				t.Errorf("GetOdigosTierFromEnv().IsEnterprise() = %v for tier %q", got.IsEnterprise(), got)
			}
		})
	}
}
