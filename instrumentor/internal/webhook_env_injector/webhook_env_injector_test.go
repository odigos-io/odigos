package webhookenvinjector

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonconsts "github.com/odigos-io/odigos/common/consts"
	distroTypes "github.com/odigos-io/odigos/distros/distro"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

const (
	javaToolOptionsEnvName = "JAVA_TOOL_OPTIONS"
	// mirrors the replacePattern of distros/yamls/java-community.yaml
	javaToolOptionsPattern = "{{ORIGINAL_ENV_VALUE}} -javaagent:{{ODIGOS_AGENTS_DIR}}/java/javaagent.jar"
	// the value the pattern above resolves to when the env var has no pre-existing value
	javaAgentOnly    = "-javaagent:/var/odigos/java/javaagent.jar"
	odigosLoaderPath = "/var/odigos/loader/loader.so"
)

func javaDistro() *distroTypes.OtelDistro {
	return &distroTypes.OtelDistro{
		Name: "java-community",
		EnvironmentVariables: distroTypes.EnvironmentVariables{
			AppendOdigosVariables: []distroTypes.AppendOdigosEnvironmentVariable{
				{EnvName: javaToolOptionsEnvName, ReplacePattern: javaToolOptionsPattern},
			},
		},
	}
}

func configWithInjectionMethod(method common.EnvInjectionMethod) *common.OdigosConfiguration {
	return &common.OdigosConfiguration{AgentEnvVarsInjectionMethod: &method}
}

func boolPtr(b bool) *bool {
	return &b
}

func strPtr(s string) *string {
	return &s
}

func inject(t *testing.T, container *corev1.Container, runtimeDetails *odigosv1.RuntimeDetailsByContainer, config *common.OdigosConfiguration) error {
	t.Helper()
	return InjectOdigosAgentEnvVars(context.Background(), logr.Discard(), container, javaDistro(), runtimeDetails, config)
}

// A distro with nothing to append is a no-op, even before the injection method is validated
// (golang and the OBI distro have no append env vars at all).
func TestInjectOdigosAgentEnvVars_NoAppendEnvVarsIsNoop(t *testing.T) {
	container := &corev1.Container{Name: "test", Env: []corev1.EnvVar{{Name: "USER_ENV", Value: "user-value"}}}

	err := InjectOdigosAgentEnvVars(context.Background(), logr.Discard(), container,
		&distroTypes.OtelDistro{Name: "golang-community"},
		&odigosv1.RuntimeDetailsByContainer{ContainerName: "test"},
		&common.OdigosConfiguration{})

	require.NoError(t, err)
	assert.Equal(t, []corev1.EnvVar{{Name: "USER_ENV", Value: "user-value"}}, container.Env)
}

// The injector reads the effective config, where the injection method is always resolved or
// defaulted. A missing method must fail loudly instead of silently picking one.
func TestInjectOdigosAgentEnvVars_MissingInjectionMethodIsAnError(t *testing.T) {
	container := &corev1.Container{Name: "test"}

	err := inject(t, container, &odigosv1.RuntimeDetailsByContainer{ContainerName: "test"}, &common.OdigosConfiguration{})

	require.Error(t, err)
	assert.Empty(t, container.Env)
}

// The loader is injected by setting LD_PRELOAD, which is only safe when the process is not running
// in secure-execution mode and no other LD_PRELOAD value would be overwritten. With the strict
// "loader" method a conflict is an error; with "loader-fallback-to-pod-manifest" it falls back to
// appending the agent to the distro env var instead.
func TestInjectOdigosAgentEnvVars_LoaderMethod(t *testing.T) {
	for _, tc := range []struct {
		name           string
		method         common.EnvInjectionMethod
		containerEnv   []corev1.EnvVar
		runtimeDetails odigosv1.RuntimeDetailsByContainer
		wantErr        bool
		wantEnv        []corev1.EnvVar
	}{
		{
			name:           "loader is used when nothing conflicts",
			method:         common.LoaderEnvInjectionMethod,
			runtimeDetails: odigosv1.RuntimeDetailsByContainer{SecureExecutionMode: boolPtr(false)},
			wantEnv:        []corev1.EnvVar{{Name: commonconsts.LdPreloadEnvVarName, Value: odigosLoaderPath}},
		},
		{
			name:           "fallback method also prefers the loader when nothing conflicts",
			method:         common.LoaderFallbackToPodManifestInjectionMethod,
			runtimeDetails: odigosv1.RuntimeDetailsByContainer{SecureExecutionMode: boolPtr(false)},
			wantEnv:        []corev1.EnvVar{{Name: commonconsts.LdPreloadEnvVarName, Value: odigosLoaderPath}},
		},
		{
			// a leftover from a previous installation or a terminating pod, not a user value
			name:   "an inspected LD_PRELOAD pointing at the odigos loader is not a conflict",
			method: common.LoaderEnvInjectionMethod,
			runtimeDetails: odigosv1.RuntimeDetailsByContainer{
				SecureExecutionMode: boolPtr(false),
				EnvVars:             []odigosv1.EnvVar{{Name: commonconsts.LdPreloadEnvVarName, Value: odigosLoaderPath}},
			},
			wantEnv: []corev1.EnvVar{{Name: commonconsts.LdPreloadEnvVarName, Value: odigosLoaderPath}},
		},
		{
			// LD_PRELOAD is ignored by the dynamic loader in secure-execution mode, so injecting it
			// would silently not instrument the process.
			name:           "secure execution mode blocks the loader",
			method:         common.LoaderEnvInjectionMethod,
			runtimeDetails: odigosv1.RuntimeDetailsByContainer{SecureExecutionMode: boolPtr(true)},
			wantErr:        true,
		},
		{
			// nil means the inspection could not determine it - assume the worst.
			name:           "unknown secure execution mode blocks the loader",
			method:         common.LoaderEnvInjectionMethod,
			runtimeDetails: odigosv1.RuntimeDetailsByContainer{},
			wantErr:        true,
		},
		{
			name:           "unknown secure execution mode falls back to appending the agent",
			method:         common.LoaderFallbackToPodManifestInjectionMethod,
			runtimeDetails: odigosv1.RuntimeDetailsByContainer{},
			wantEnv:        []corev1.EnvVar{{Name: javaToolOptionsEnvName, Value: javaAgentOnly}},
		},
		{
			name:           "a user LD_PRELOAD in the manifest blocks the loader",
			method:         common.LoaderEnvInjectionMethod,
			containerEnv:   []corev1.EnvVar{{Name: commonconsts.LdPreloadEnvVarName, Value: "/lib/user.so"}},
			runtimeDetails: odigosv1.RuntimeDetailsByContainer{SecureExecutionMode: boolPtr(false)},
			wantErr:        true,
		},
		{
			name:           "a user LD_PRELOAD in the manifest is preserved when falling back",
			method:         common.LoaderFallbackToPodManifestInjectionMethod,
			containerEnv:   []corev1.EnvVar{{Name: commonconsts.LdPreloadEnvVarName, Value: "/lib/user.so"}},
			runtimeDetails: odigosv1.RuntimeDetailsByContainer{SecureExecutionMode: boolPtr(false)},
			wantEnv: []corev1.EnvVar{
				{Name: commonconsts.LdPreloadEnvVarName, Value: "/lib/user.so"},
				{Name: javaToolOptionsEnvName, Value: javaAgentOnly},
			},
		},
		{
			name:   "an inspected user LD_PRELOAD blocks the loader",
			method: common.LoaderEnvInjectionMethod,
			runtimeDetails: odigosv1.RuntimeDetailsByContainer{
				SecureExecutionMode: boolPtr(false),
				EnvVars:             []odigosv1.EnvVar{{Name: commonconsts.LdPreloadEnvVarName, Value: "/lib/user.so"}},
			},
			wantErr: true,
		},
		{
			name:   "an inspected user LD_PRELOAD is not injected when falling back",
			method: common.LoaderFallbackToPodManifestInjectionMethod,
			runtimeDetails: odigosv1.RuntimeDetailsByContainer{
				SecureExecutionMode: boolPtr(false),
				EnvVars:             []odigosv1.EnvVar{{Name: commonconsts.LdPreloadEnvVarName, Value: "/lib/user.so"}},
			},
			wantEnv: []corev1.EnvVar{{Name: javaToolOptionsEnvName, Value: javaAgentOnly}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			container := &corev1.Container{Name: "test", Env: tc.containerEnv}
			runtimeDetails := tc.runtimeDetails

			err := inject(t, container, &runtimeDetails, configWithInjectionMethod(tc.method))

			if tc.wantErr {
				require.Error(t, err)
				assert.Equal(t, tc.containerEnv, container.Env, "container env must not be modified when injection fails")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantEnv, container.Env)
		})
	}
}

// With the pod manifest method the agent is appended to the distro env var, using the value already
// defined in the manifest, the value observed in the container runtime, or the odigos default.
func TestInjectOdigosAgentEnvVars_PodManifestMethod(t *testing.T) {
	for _, tc := range []struct {
		name           string
		containerEnv   []corev1.EnvVar
		runtimeDetails odigosv1.RuntimeDetailsByContainer
		wantEnv        []corev1.EnvVar
	}{
		{
			name:    "no existing value gets the odigos default",
			wantEnv: []corev1.EnvVar{{Name: javaToolOptionsEnvName, Value: javaAgentOnly}},
		},
		{
			name:         "a manifest value is appended to",
			containerEnv: []corev1.EnvVar{{Name: javaToolOptionsEnvName, Value: "-Xmx1g"}},
			wantEnv:      []corev1.EnvVar{{Name: javaToolOptionsEnvName, Value: "-Xmx1g " + javaAgentOnly}},
		},
		{
			// the webhook runs on every pod creation, so injecting into an already-injected manifest
			// (e.g. a rollout of a workload whose template odigos previously patched) must not
			// append the agent twice.
			name:         "a manifest value that already includes odigos is left as is",
			containerEnv: []corev1.EnvVar{{Name: javaToolOptionsEnvName, Value: "-Xmx1g " + javaAgentOnly}},
			wantEnv:      []corev1.EnvVar{{Name: javaToolOptionsEnvName, Value: "-Xmx1g " + javaAgentOnly}},
		},
		{
			name:         "an empty manifest value is replaced in place with the odigos default",
			containerEnv: []corev1.EnvVar{{Name: javaToolOptionsEnvName}},
			wantEnv:      []corev1.EnvVar{{Name: javaToolOptionsEnvName, Value: javaAgentOnly}},
		},
		{
			// a valueFrom cannot be appended to, so the reference is renamed and expanded into the
			// new value to keep the user value intact.
			name: "a manifest valueFrom is renamed and referenced",
			containerEnv: []corev1.EnvVar{{
				Name: javaToolOptionsEnvName,
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "cm"},
						Key:                  "opts",
					},
				},
			}},
			wantEnv: []corev1.EnvVar{
				{
					Name: "ORIGINAL_" + javaToolOptionsEnvName,
					ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "cm"},
							Key:                  "opts",
						},
					},
				},
				{Name: javaToolOptionsEnvName, Value: "$(ORIGINAL_" + javaToolOptionsEnvName + ") " + javaAgentOnly},
			},
		},
		{
			name: "a container runtime value is appended to",
			runtimeDetails: odigosv1.RuntimeDetailsByContainer{
				EnvFromContainerRuntime: []odigosv1.EnvVar{{Name: javaToolOptionsEnvName, Value: "-Xmx1g"}},
			},
			wantEnv: []corev1.EnvVar{{Name: javaToolOptionsEnvName, Value: "-Xmx1g " + javaAgentOnly}},
		},
		{
			name: "an empty container runtime value gets the odigos default",
			runtimeDetails: odigosv1.RuntimeDetailsByContainer{
				EnvFromContainerRuntime: []odigosv1.EnvVar{{Name: javaToolOptionsEnvName}},
			},
			wantEnv: []corev1.EnvVar{{Name: javaToolOptionsEnvName, Value: javaAgentOnly}},
		},
		{
			// an env var declared with no value in both places must end up declared once, or the
			// container would get two entries for the same name.
			name:         "an empty value in both the manifest and the container runtime is set once",
			containerEnv: []corev1.EnvVar{{Name: javaToolOptionsEnvName}},
			runtimeDetails: odigosv1.RuntimeDetailsByContainer{
				EnvFromContainerRuntime: []odigosv1.EnvVar{{Name: javaToolOptionsEnvName}},
			},
			wantEnv: []corev1.EnvVar{{Name: javaToolOptionsEnvName, Value: javaAgentOnly}},
		},
		{
			// EnvVars holds every env var observed for the process, which includes values the image
			// or the runtime set for reasons of their own. Only EnvFromContainerRuntime is known to
			// come from the container runtime and is safe to re-declare in the manifest.
			name: "an inspected value that is not from the container runtime is not appended to",
			runtimeDetails: odigosv1.RuntimeDetailsByContainer{
				EnvVars: []odigosv1.EnvVar{{Name: javaToolOptionsEnvName, Value: "-Xmx1g"}},
			},
			wantEnv: []corev1.EnvVar{{Name: javaToolOptionsEnvName, Value: javaAgentOnly}},
		},
		{
			// the CRI lookup failed, so the observed value may be wrong; appending to it could
			// re-declare a value the container never had.
			name: "a CRI error message discards the container runtime value",
			runtimeDetails: odigosv1.RuntimeDetailsByContainer{
				CriErrorMessage:         strPtr("failed to get container runtime env vars"),
				EnvFromContainerRuntime: []odigosv1.EnvVar{{Name: javaToolOptionsEnvName, Value: "-Xmx1g"}},
			},
			wantEnv: []corev1.EnvVar{{Name: javaToolOptionsEnvName, Value: javaAgentOnly}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			container := &corev1.Container{Name: "test", Env: tc.containerEnv}
			runtimeDetails := tc.runtimeDetails

			err := inject(t, container, &runtimeDetails, configWithInjectionMethod(common.PodManifestEnvInjectionMethod))

			require.NoError(t, err)
			assert.Equal(t, tc.wantEnv, container.Env)
		})
	}
}
