package webhookenvinjector

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonconsts "github.com/odigos-io/odigos/common/consts"
	distroTypes "github.com/odigos-io/odigos/distros/distro"
	corev1 "k8s.io/api/core/v1"
)

func ldPreloadValue(c *corev1.Container) string {
	for _, e := range c.Env {
		if e.Name == commonconsts.LdPreloadEnvVarName {
			return e.Value
		}
	}
	return ""
}

func loaderInjectorFixtures() (*distroTypes.OtelDistro, *common.OdigosConfiguration, *odigosv1.RuntimeDetailsByContainer) {
	loaderMethod := common.LoaderEnvInjectionMethod
	secure := false
	distro := &distroTypes.OtelDistro{
		// non-empty so InjectOdigosAgentEnvVars does not early-return before the loader block
		EnvironmentVariables: distroTypes.EnvironmentVariables{
			AppendOdigosVariables: []distroTypes.AppendOdigosEnvironmentVariable{{EnvName: "PYTHONPATH"}},
		},
	}
	config := &common.OdigosConfiguration{AgentEnvVarsInjectionMethod: &loaderMethod}
	rd := &odigosv1.RuntimeDetailsByContainer{SecureExecutionMode: &secure}
	return distro, config, rd
}

// TestLoaderComposesWithMemprofLdPreload guards the Python case: memory profiling sets
// LD_PRELOAD to the odigos interposer earlier in the webhook, and the tracing-loader
// injection must PREPEND its loader rather than bail out — so both survive as
// <loader>:<libmemsample>.
func TestLoaderComposesWithMemprofLdPreload(t *testing.T) {
	distro, config, rd := loaderInjectorFixtures()
	container := &corev1.Container{
		Name: "app",
		Env: []corev1.EnvVar{
			{Name: commonconsts.LdPreloadEnvVarName, Value: "/var/odigos/memprof/libmemsample.so"},
		},
	}

	if err := InjectOdigosAgentEnvVars(context.Background(), logr.Discard(), container, distro, rd, config); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := ldPreloadValue(container)
	want := "/var/odigos/loader/loader.so:/var/odigos/memprof/libmemsample.so"
	if got != want {
		t.Fatalf("LD_PRELOAD compose wrong:\n got=%q\nwant=%q", got, want)
	}
}

// TestLoaderDoesNotComposeWithUserLdPreload guards the safety gate: a user-defined
// LD_PRELOAD (not under /var/odigos) must never be modified by the compose path.
func TestLoaderDoesNotComposeWithUserLdPreload(t *testing.T) {
	distro, config, rd := loaderInjectorFixtures()
	container := &corev1.Container{
		Name: "app",
		Env:  []corev1.EnvVar{{Name: commonconsts.LdPreloadEnvVarName, Value: "/opt/user/mylib.so"}},
	}

	err := InjectOdigosAgentEnvVars(context.Background(), logr.Discard(), container, distro, rd, config)
	if err == nil {
		t.Fatal("expected error for user-defined LD_PRELOAD with strict loader method, got nil")
	}
	if got := ldPreloadValue(container); got != "/opt/user/mylib.so" {
		t.Fatalf("user LD_PRELOAD must be untouched, got %q", got)
	}
}
