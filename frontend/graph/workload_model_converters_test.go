package graph

import (
	"reflect"
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/agentsignalconfig"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyStrToNil(t *testing.T) {
	assert.Nil(t, emptyStrToNil(""))

	got := emptyStrToNil("golang-community")
	require.NotNil(t, got)
	assert.Equal(t, "golang-community", *got)

	// whitespace is a value, not an absence
	blank := emptyStrToNil(" ")
	require.NotNil(t, blank)
	assert.Equal(t, " ", *blank)
}

func TestDistroParamsToModel(t *testing.T) {
	got := distroParamsToModel(map[string]string{"LIBC_TYPE": "musl", "RUNTIME_VERSION": "1.24.0"})

	require.Len(t, got, 2)
	byName := map[string]string{}
	for _, p := range got {
		byName[p.Name] = p.Value
	}
	assert.Equal(t, map[string]string{"LIBC_TYPE": "musl", "RUNTIME_VERSION": "1.24.0"}, byName)
}

func TestDistroParamsToModelTurnsNoParamsIntoAnEmptyList(t *testing.T) {
	got := distroParamsToModel(nil)
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

// workload.graphqls declares `processEnvVars: [EnvVar!]!`, so a nil slice here would serialise as
// null and fail the whole workload query for any container with no env vars.
func TestEnvVarsToModelTurnsNoEnvVarsIntoAnEmptyListRatherThanNull(t *testing.T) {
	got := envVarsToModel(nil)
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

func TestEnvVarsToModelPreservesOrderAndPairing(t *testing.T) {
	got := envVarsToModel([]v1alpha1.EnvVar{
		{Name: "PYTHONPATH", Value: "/var/odigos/python"},
		{Name: "OTEL_SERVICE_NAME", Value: "checkout"},
	})

	require.Len(t, got, 2)
	assert.Equal(t, "PYTHONPATH", got[0].Name)
	assert.Equal(t, "/var/odigos/python", got[0].Value)
	assert.Equal(t, "OTEL_SERVICE_NAME", got[1].Name)
	assert.Equal(t, "checkout", got[1].Value)
}

func wuFullRuntimeDetails() *v1alpha1.RuntimeDetailsByContainer {
	libc := common.Musl
	criError := "failed to read env from the container runtime"
	secureExecution := true
	return &v1alpha1.RuntimeDetailsByContainer{
		ContainerName:  "app",
		Language:       common.PythonProgrammingLanguage,
		RuntimeVersion: "3.12.1",
		// the two env var lists are same-typed siblings, so they must hold distinguishable
		// values or a cross-wiring between them is invisible.
		EnvVars:                 []v1alpha1.EnvVar{{Name: "PYTHONPATH", Value: "/app/vendor"}},
		EnvFromContainerRuntime: []v1alpha1.EnvVar{{Name: "NODE_OPTIONS", Value: "--require /cri/agent"}},
		OtherAgents:             []v1alpha1.OtherAgent{{Name: "datadog"}, {Name: "newrelic"}},
		LibCType:                &libc,
		SecureExecutionMode:     &secureExecution,
		CriErrorMessage:         &criError,
	}
}

func TestRuntimeDetailsContainersToModelCarriesEveryFieldToItsOwnDestination(t *testing.T) {
	got := runtimeDetailsContainersToModel(wuFullRuntimeDetails())

	require.NotNil(t, got)
	assert.Equal(t, "app", got.ContainerName)
	assert.Equal(t, model.ProgrammingLanguage(common.PythonProgrammingLanguage), got.Language)
	require.NotNil(t, got.RuntimeVersion)
	assert.Equal(t, "3.12.1", *got.RuntimeVersion)

	require.Len(t, got.ProcessEnvVars, 1)
	assert.Equal(t, "PYTHONPATH", got.ProcessEnvVars[0].Name)
	assert.Equal(t, "/app/vendor", got.ProcessEnvVars[0].Value)

	require.Len(t, got.ContainerRuntimeEnvVars, 1)
	assert.Equal(t, "NODE_OPTIONS", got.ContainerRuntimeEnvVars[0].Name)
	assert.Equal(t, "--require /cri/agent", got.ContainerRuntimeEnvVars[0].Value)

	assert.Equal(t, []string{"datadog", "newrelic"}, got.OtherAgentNames)
	require.NotNil(t, got.LibcType)
	assert.Equal(t, string(common.Musl), *got.LibcType)
	require.NotNil(t, got.SecureExecutionMode)
	assert.True(t, *got.SecureExecutionMode)
	require.NotNil(t, got.CriErrorMessage)
	assert.Equal(t, "failed to read env from the container runtime", *got.CriErrorMessage)
}

// The field-by-field test above cannot see a field that is missing from BOTH the fixture and the
// expectation, so pin the invariant by reflection: a fully populated input must leave no output
// field at its zero value.
func TestEveryRuntimeInfoModelFieldIsPopulatedFromAFullRuntimeDetails(t *testing.T) {
	got := runtimeDetailsContainersToModel(wuFullRuntimeDetails())

	value := reflect.ValueOf(*got)
	require.Positive(t, value.NumField())
	for i := 0; i < value.NumField(); i++ {
		assert.False(t, value.Field(i).IsZero(), "%s was not populated from a fully populated RuntimeDetailsByContainer", value.Type().Field(i).Name)
	}
}

func TestRuntimeDetailsContainersToModelLeavesAbsentOptionalsAbsent(t *testing.T) {
	got := runtimeDetailsContainersToModel(&v1alpha1.RuntimeDetailsByContainer{
		ContainerName: "app",
		Language:      common.GoProgrammingLanguage,
	})

	require.NotNil(t, got)
	assert.Nil(t, got.RuntimeVersion, "an empty runtime version must be reported as unknown, not as the empty string")
	assert.Nil(t, got.LibcType)
	assert.Nil(t, got.SecureExecutionMode)
	assert.Nil(t, got.CriErrorMessage)
	assert.NotNil(t, got.ProcessEnvVars)
	assert.Empty(t, got.ProcessEnvVars)
	assert.NotNil(t, got.OtherAgentNames)
	assert.Empty(t, got.OtherAgentNames)
}

func TestContainerOverrideToModelCarriesEveryField(t *testing.T) {
	distro := "python-enterprise"
	allowConcurrent := true
	got := containerOverrideToModel(&v1alpha1.ContainerOverride{
		ContainerName:         "app",
		OtelDistroName:        &distro,
		AllowConcurrentAgents: &allowConcurrent,
		RuntimeInfo:           wuFullRuntimeDetails(),
	})

	require.NotNil(t, got)
	assert.Equal(t, "app", got.ContainerName)
	require.NotNil(t, got.OtelDistroName)
	assert.Equal(t, "python-enterprise", *got.OtelDistroName)
	require.NotNil(t, got.AllowConcurrentAgents)
	assert.True(t, *got.AllowConcurrentAgents)
	require.NotNil(t, got.RuntimeInfo)
	assert.Equal(t, "3.12.1", *got.RuntimeInfo.RuntimeVersion)
}

func TestContainerOverrideToModelLeavesAnAbsentRuntimeInfoAbsent(t *testing.T) {
	got := containerOverrideToModel(&v1alpha1.ContainerOverride{ContainerName: "app"})

	require.NotNil(t, got)
	assert.Equal(t, "app", got.ContainerName)
	assert.Nil(t, got.RuntimeInfo)
	assert.Nil(t, got.OtelDistroName)
	assert.Nil(t, got.AllowConcurrentAgents)
}

func TestAgentEnabledContainersToModelCarriesEveryField(t *testing.T) {
	envInjection := common.EnvInjectionDecisionLoader
	got := agentEnabledContainersToModel(&v1alpha1.ContainerAgentConfig{
		ContainerName:      "app",
		AgentEnabled:       true,
		AgentEnabledReason: v1alpha1.AgentEnabledReasonEnabledSuccessfully,
		OtelDistroName:     "python-enterprise",
		EnvInjectionMethod: &envInjection,
		DistroParams:       map[string]string{"LIBC_TYPE": "musl"},
		Traces:             &agentsignalconfig.AgentTracesConfig{},
		Metrics:            &agentsignalconfig.AgentMetricsConfig{},
		Logs:               &agentsignalconfig.AgentLogsConfig{},
	})

	require.NotNil(t, got)
	assert.Equal(t, "app", got.ContainerName)
	assert.True(t, got.AgentEnabled)
	require.NotNil(t, got.AgentEnabledStatus)
	require.NotNil(t, got.OtelDistroName)
	assert.Equal(t, "python-enterprise", *got.OtelDistroName)
	require.NotNil(t, got.EnvInjectionMethod)
	assert.Equal(t, string(common.EnvInjectionDecisionLoader), *got.EnvInjectionMethod)
	require.Len(t, got.DistroParams, 1)
	assert.Equal(t, "LIBC_TYPE", got.DistroParams[0].Name)
}

func TestAgentEnabledContainersToModelReportsAnUnsetDistroAsAbsent(t *testing.T) {
	got := agentEnabledContainersToModel(&v1alpha1.ContainerAgentConfig{ContainerName: "app"})

	require.NotNil(t, got)
	assert.Nil(t, got.OtelDistroName, "an empty distro name means the container is not instrumented")
	assert.Nil(t, got.EnvInjectionMethod)
	assert.Nil(t, got.Traces)
	assert.Nil(t, got.Metrics)
	assert.Nil(t, got.Logs)
}

// Traces, metrics and logs are three independent nil checks that all produce the identical
// `Enabled: true` value, so a fully populated fixture cannot prove any of them is wired to its own
// source field. Enable exactly one signal at a time instead.
func TestEachEnabledSignalIsReportedOnItsOwnField(t *testing.T) {
	tests := []struct {
		name    string
		config  v1alpha1.ContainerAgentConfig
		traces  bool
		metrics bool
		logs    bool
	}{
		{name: "traces only", config: v1alpha1.ContainerAgentConfig{Traces: &agentsignalconfig.AgentTracesConfig{}}, traces: true},
		{name: "metrics only", config: v1alpha1.ContainerAgentConfig{Metrics: &agentsignalconfig.AgentMetricsConfig{}}, metrics: true},
		{name: "logs only", config: v1alpha1.ContainerAgentConfig{Logs: &agentsignalconfig.AgentLogsConfig{}}, logs: true},
		{name: "no signals", config: v1alpha1.ContainerAgentConfig{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.config
			config.ContainerName = "app"
			got := agentEnabledContainersToModel(&config)

			require.NotNil(t, got)
			assert.Equal(t, tt.traces, got.Traces != nil, "traces")
			assert.Equal(t, tt.metrics, got.Metrics != nil, "metrics")
			assert.Equal(t, tt.logs, got.Logs != nil, "logs")

			if got.Traces != nil {
				assert.True(t, got.Traces.Enabled)
			}
			if got.Metrics != nil {
				assert.True(t, got.Metrics.Enabled)
			}
			if got.Logs != nil {
				assert.True(t, got.Logs.Enabled)
			}
		})
	}
}

func TestGetContainerNamesWithOptionalPodManifestInjection(t *testing.T) {
	// the workload-level flag and the per-container flag are two independent operands of the same
	// AND, so each needs a case where it is the only one true.
	tests := []struct {
		name string
		ic   *v1alpha1.InstrumentationConfig
		want []string
	}{
		{
			name: "no instrumentation config",
			ic:   nil,
		},
		{
			name: "workload opted in, no container opted in",
			ic: &v1alpha1.InstrumentationConfig{Spec: v1alpha1.InstrumentationConfigSpec{
				PodManifestInjectionOptional: true,
				Containers:                   []v1alpha1.ContainerAgentConfig{{ContainerName: "app"}},
			}},
		},
		{
			name: "container opted in, workload did not",
			ic: &v1alpha1.InstrumentationConfig{Spec: v1alpha1.InstrumentationConfigSpec{
				Containers: []v1alpha1.ContainerAgentConfig{{ContainerName: "app", PodManifestInjectionOptional: true}},
			}},
		},
		{
			name: "both opted in",
			ic: &v1alpha1.InstrumentationConfig{Spec: v1alpha1.InstrumentationConfigSpec{
				PodManifestInjectionOptional: true,
				Containers:                   []v1alpha1.ContainerAgentConfig{{ContainerName: "app", PodManifestInjectionOptional: true}},
			}},
			want: []string{"app"},
		},
		{
			name: "only the opted-in containers of an opted-in workload",
			ic: &v1alpha1.InstrumentationConfig{Spec: v1alpha1.InstrumentationConfigSpec{
				PodManifestInjectionOptional: true,
				Containers: []v1alpha1.ContainerAgentConfig{
					{ContainerName: "app", PodManifestInjectionOptional: true},
					{ContainerName: "sidecar"},
					{ContainerName: "worker", PodManifestInjectionOptional: true},
				},
			}},
			want: []string{"app", "worker"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getContainerNamesWithOptionalPodManifestInjection(tt.ic)

			// the health aggregation indexes into this map, so it must never be nil
			require.NotNil(t, got)
			names := make([]string, 0, len(got))
			for name := range got {
				names = append(names, name)
			}
			assert.ElementsMatch(t, tt.want, names)
		})
	}
}

func TestGetContainerConfigByNameReturnsTheNamedContainerNotTheFirst(t *testing.T) {
	ic := &v1alpha1.InstrumentationConfig{Spec: v1alpha1.InstrumentationConfigSpec{
		Containers: []v1alpha1.ContainerAgentConfig{
			{ContainerName: "sidecar", OtelDistroName: "nodejs-community"},
			{ContainerName: "app", OtelDistroName: "python-enterprise"},
		},
	}}

	got := getContainerConfigByName(ic, "app")

	require.NotNil(t, got)
	assert.Equal(t, "app", got.ContainerName)
	assert.Equal(t, "python-enterprise", got.OtelDistroName)
}

func TestGetContainerConfigByNameReportsAnUnknownContainerAsAbsent(t *testing.T) {
	ic := &v1alpha1.InstrumentationConfig{Spec: v1alpha1.InstrumentationConfigSpec{
		Containers: []v1alpha1.ContainerAgentConfig{{ContainerName: "app"}},
	}}

	assert.Nil(t, getContainerConfigByName(ic, "not-a-container"))
	assert.Nil(t, getContainerConfigByName(nil, "app"))
	assert.Nil(t, getContainerConfigByName(&v1alpha1.InstrumentationConfig{}, "app"))
}
