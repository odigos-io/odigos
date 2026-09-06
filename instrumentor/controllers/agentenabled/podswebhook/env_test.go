package podswebhook

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/k8sutils/pkg/service"
)

func TestInjectConstEnvVarToPodContainer(t *testing.T) {
	t.Run("injects and records the name", func(t *testing.T) {
		container := &corev1.Container{Name: "test"}

		existing := InjectConstEnvVarToPodContainer(GetEnvVarNamesSet(container), container, "MY_VAR", "my-value")

		assert.Equal(t, []corev1.EnvVar{{Name: "MY_VAR", Value: "my-value"}}, container.Env)
		assert.Contains(t, existing, "MY_VAR")
	})

	t.Run("a value already set by the user is never overwritten", func(t *testing.T) {
		container := &corev1.Container{Name: "test", Env: []corev1.EnvVar{{Name: "MY_VAR", Value: "user-value"}}}

		InjectConstEnvVarToPodContainer(GetEnvVarNamesSet(container), container, "MY_VAR", "odigos-value")

		assert.Equal(t, []corev1.EnvVar{{Name: "MY_VAR", Value: "user-value"}}, container.Env)
	})
}

func TestInjectStaticEnvVarsToPodContainer(t *testing.T) {
	t.Run("plain and templated values", func(t *testing.T) {
		container := &corev1.Container{Name: "test"}
		envVars := []distro.StaticEnvironmentVariable{
			{EnvName: "PLAIN", EnvValue: "plain-value"},
			{
				EnvName:  "TEMPLATED",
				EnvValue: "/lib/{{.LIBC_TYPE}}/agent.so",
				Template: template.Must(template.New("main").Option("missingkey=error").Parse("/lib/{{.LIBC_TYPE}}/agent.so")),
			},
		}

		_, err := InjectStaticEnvVarsToPodContainer(GetEnvVarNamesSet(container), container, envVars,
			map[string]string{"LIBC_TYPE": "musl"})

		require.NoError(t, err)
		assert.Equal(t, []corev1.EnvVar{
			{Name: "PLAIN", Value: "plain-value"},
			{Name: "TEMPLATED", Value: "/lib/musl/agent.so"},
		}, container.Env)
	})

	t.Run("a missing distro param fails instead of injecting a bogus value", func(t *testing.T) {
		container := &corev1.Container{Name: "test"}
		envVars := []distro.StaticEnvironmentVariable{{
			EnvName:  "TEMPLATED",
			EnvValue: "/lib/{{.LIBC_TYPE}}/agent.so",
			Template: template.Must(template.New("main").Option("missingkey=error").Parse("/lib/{{.LIBC_TYPE}}/agent.so")),
		}}

		_, err := InjectStaticEnvVarsToPodContainer(GetEnvVarNamesSet(container), container, envVars, map[string]string{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "TEMPLATED")
		assert.Empty(t, container.Env)
	})

	t.Run("existing env vars are not templated over", func(t *testing.T) {
		container := &corev1.Container{Name: "test", Env: []corev1.EnvVar{{Name: "TEMPLATED", Value: "user-value"}}}
		envVars := []distro.StaticEnvironmentVariable{{
			EnvName:  "TEMPLATED",
			EnvValue: "/lib/{{.LIBC_TYPE}}/agent.so",
			Template: template.Must(template.New("main").Option("missingkey=error").Parse("/lib/{{.LIBC_TYPE}}/agent.so")),
		}}

		// no distro params at all: the template would fail if it was executed
		_, err := InjectStaticEnvVarsToPodContainer(GetEnvVarNamesSet(container), container, envVars, nil)

		require.NoError(t, err)
		assert.Equal(t, []corev1.EnvVar{{Name: "TEMPLATED", Value: "user-value"}}, container.Env)
	})
}

func TestInjectOdigosK8sEnvVars(t *testing.T) {
	container := &corev1.Container{Name: "my-container"}

	InjectOdigosK8sEnvVars(GetEnvVarNamesSet(container), container, "nodejs-community", "my-ns")

	assert.Equal(t, "my-container", findEnvVar(t, container, "ODIGOS_CONTAINER_NAME").Value)
	assert.Equal(t, "nodejs-community", findEnvVar(t, container, "ODIGOS_DISTRO_NAME").Value)
	assert.Equal(t, "my-ns", findEnvVar(t, container, "ODIGOS_WORKLOAD_NAMESPACE").Value)

	podName := findEnvVar(t, container, "ODIGOS_POD_NAME")
	require.NotNil(t, podName.ValueFrom)
	require.NotNil(t, podName.ValueFrom.FieldRef)
	assert.Equal(t, "metadata.name", podName.ValueFrom.FieldRef.FieldPath)
}

func TestInjectOpampAndOtlpEnvVars(t *testing.T) {
	t.Run("opamp over http injects the node ip and the server host", func(t *testing.T) {
		container := &corev1.Container{Name: "test"}

		InjectOpampServerEnvVar(GetEnvVarNamesSet(container), container)

		nodeIP := findEnvVar(t, container, "NODE_IP")
		require.NotNil(t, nodeIP.ValueFrom)
		require.NotNil(t, nodeIP.ValueFrom.FieldRef)
		assert.Equal(t, "status.hostIP", nodeIP.ValueFrom.FieldRef.FieldPath)
		assert.Equal(t, service.LocalTrafficOpAmpOdigletEndpoint("$(NODE_IP)"),
			findEnvVar(t, container, "ODIGOS_OPAMP_SERVER_HOST").Value)
	})

	t.Run("opamp over unix socket does not need the node ip", func(t *testing.T) {
		container := &corev1.Container{Name: "test"}

		InjectOpampUnixSocketEnvVar(GetEnvVarNamesSet(container), container)

		assert.Equal(t, []corev1.EnvVar{
			{Name: "ODIGOS_OPAMP_UNIX_SOCKET", Value: "/var/odigos/exchange/exchange.sock"},
		}, container.Env)
	})

	t.Run("the node ip env var is injected only once", func(t *testing.T) {
		container := &corev1.Container{Name: "test"}

		existing := InjectOpampServerEnvVar(GetEnvVarNamesSet(container), container)
		InjectOtlpHttpEndpointEnvVar(existing, container)

		nodeIPCount := 0
		for _, envVar := range container.Env {
			if envVar.Name == "NODE_IP" {
				nodeIPCount++
			}
		}
		assert.Equal(t, 1, nodeIPCount)
		assert.Equal(t, service.LocalTrafficOTLPHttpDataCollectionEndpoint("$(NODE_IP)"),
			findEnvVar(t, container, "OTEL_EXPORTER_OTLP_ENDPOINT").Value)
	})
}

func TestInjectSignalsAsStaticOtelEnvVars(t *testing.T) {
	tests := []struct {
		name                                     string
		traces, metrics, logs                    bool
		expectedTraces, expectedMetrics, expLogs string
	}{
		{"all signals enabled", true, true, true, "otlp", "otlp", "otlp"},
		{"all signals disabled", false, false, false, "none", "none", "none"},
		{"only traces enabled", true, false, false, "otlp", "none", "none"},
		{"only logs enabled", false, false, true, "none", "none", "otlp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := &corev1.Container{Name: "test"}

			InjectSignalsAsStaticOtelEnvVars(GetEnvVarNamesSet(container), container, tt.traces, tt.metrics, tt.logs)

			assert.Equal(t, tt.expectedTraces, findEnvVar(t, container, "OTEL_TRACES_EXPORTER").Value)
			assert.Equal(t, tt.expectedMetrics, findEnvVar(t, container, "OTEL_METRICS_EXPORTER").Value)
			assert.Equal(t, tt.expLogs, findEnvVar(t, container, "OTEL_LOGS_EXPORTER").Value)
		})
	}
}

func TestInjectAgentDiagnosticsEnvVars(t *testing.T) {
	debugLevel := common.LogLevelDebug
	warnLevel := common.LogLevelWarn

	t.Run("nil diagnostics injects nothing", func(t *testing.T) {
		container := &corev1.Container{Name: "test"}

		InjectAgentDiagnosticsEnvVars(GetEnvVarNamesSet(container), container, nil)

		assert.Empty(t, container.Env)
	})

	t.Run("only the configured levels are injected", func(t *testing.T) {
		container := &corev1.Container{Name: "test"}

		InjectAgentDiagnosticsEnvVars(GetEnvVarNamesSet(container), container,
			&instrumentationrules.AgentDiagnostics{OdigosLogLevel: &debugLevel})

		assert.Equal(t, []corev1.EnvVar{{Name: "ODIGOS_LOG_LEVEL", Value: "debug"}}, container.Env)
	})

	t.Run("both levels are injected", func(t *testing.T) {
		container := &corev1.Container{Name: "test"}

		InjectAgentDiagnosticsEnvVars(GetEnvVarNamesSet(container), container, &instrumentationrules.AgentDiagnostics{
			OdigosLogLevel:                  &debugLevel,
			OpenTelemetryComponentsLogLevel: &warnLevel,
		})

		assert.Equal(t, []corev1.EnvVar{
			{Name: "ODIGOS_LOG_LEVEL", Value: "debug"},
			{Name: "OTEL_LOG_LEVEL", Value: "warn"},
		}, container.Env)
	})
}

func TestInjectUserEnvForLang(t *testing.T) {
	newConfig := func(language common.ProgrammingLanguage, enabled bool, envVars map[string]string) *common.OdigosConfiguration {
		return &common.OdigosConfiguration{
			UserInstrumentationEnvs: &common.UserInstrumentationEnvs{
				Languages: map[common.ProgrammingLanguage]common.LanguageConfig{
					language: {Enabled: enabled, EnvVars: envVars},
				},
			},
		}
	}

	newPodWithContainers := func(names ...string) *corev1.Pod {
		containers := make([]corev1.Container, 0, len(names))
		for _, name := range names {
			containers = append(containers, corev1.Container{Name: name})
		}
		return &corev1.Pod{Spec: corev1.PodSpec{Containers: containers}}
	}

	newIC := func(containerName string, language common.ProgrammingLanguage) *odigosv1.InstrumentationConfig {
		return &odigosv1.InstrumentationConfig{
			Status: odigosv1.InstrumentationConfigStatus{
				RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{
					{ContainerName: containerName, Language: language},
				},
			},
		}
	}

	t.Run("injects the env vars of the container language", func(t *testing.T) {
		pod := newPodWithContainers("my-container")
		config := newConfig(common.JavascriptProgrammingLanguage, true, map[string]string{"NODE_DEBUG": "1"})

		InjectUserEnvForLang(config, pod, newIC("my-container", common.JavascriptProgrammingLanguage))

		assert.Equal(t, []corev1.EnvVar{{Name: "NODE_DEBUG", Value: "1"}}, pod.Spec.Containers[0].Env)
	})

	t.Run("a disabled language is skipped", func(t *testing.T) {
		pod := newPodWithContainers("my-container")
		config := newConfig(common.JavascriptProgrammingLanguage, false, map[string]string{"NODE_DEBUG": "1"})

		InjectUserEnvForLang(config, pod, newIC("my-container", common.JavascriptProgrammingLanguage))

		assert.Empty(t, pod.Spec.Containers[0].Env)
	})

	t.Run("a language without configuration is skipped", func(t *testing.T) {
		pod := newPodWithContainers("my-container")
		config := newConfig(common.JavascriptProgrammingLanguage, true, map[string]string{"NODE_DEBUG": "1"})

		InjectUserEnvForLang(config, pod, newIC("my-container", common.PythonProgrammingLanguage))

		assert.Empty(t, pod.Spec.Containers[0].Env)
	})

	t.Run("a container that is not in the pod is skipped", func(t *testing.T) {
		pod := newPodWithContainers("other-container")
		config := newConfig(common.JavascriptProgrammingLanguage, true, map[string]string{"NODE_DEBUG": "1"})

		InjectUserEnvForLang(config, pod, newIC("my-container", common.JavascriptProgrammingLanguage))

		assert.Empty(t, pod.Spec.Containers[0].Env)
	})

	t.Run("values already set on the container win", func(t *testing.T) {
		pod := newPodWithContainers("my-container")
		pod.Spec.Containers[0].Env = []corev1.EnvVar{{Name: "NODE_DEBUG", Value: "user-value"}}
		config := newConfig(common.JavascriptProgrammingLanguage, true, map[string]string{"NODE_DEBUG": "1"})

		InjectUserEnvForLang(config, pod, newIC("my-container", common.JavascriptProgrammingLanguage))

		assert.Equal(t, []corev1.EnvVar{{Name: "NODE_DEBUG", Value: "user-value"}}, pod.Spec.Containers[0].Env)
	})
}
