package dynamicconfig

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonapi "github.com/odigos-io/odigos/common/api"
	"github.com/odigos-io/odigos/common/api/agentsignalconfig"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/stretchr/testify/require"
)

// The literals below are spelled out instead of being taken from the production constants,
// so that renaming a constant's VALUE (which is persisted in the odigos config and read back
// by the agent) cannot pass unnoticed.
const (
	nfAutoLiteral   = "auto"
	nfAlwaysLiteral = "always"
)

func nfEnforcement(value string) *commonapisampling.NoisyOperationsEnforcement {
	enforcement := commonapisampling.NoisyOperationsEnforcement(value)
	return &enforcement
}

func nfDistro(headSamplingSupported bool) *distro.OtelDistro {
	return &distro.OtelDistro{
		Name:     "test-distro",
		Language: common.JavaProgrammingLanguage,
		Traces: &distro.Traces{
			HeadSampling: &distro.HeadSampling{Supported: headSamplingSupported},
		},
	}
}

// nfConfig builds the effective config with only the tail sampling enforcement set.
// A nil enforcement still produces the full Sampling/TailSampling chain; the shallower
// chains are written out inline by the cases that cover them.
func nfConfig(enforcement *commonapisampling.NoisyOperationsEnforcement) *common.OdigosConfiguration {
	return &common.OdigosConfiguration{
		Sampling: &common.SamplingConfiguration{
			TailSampling: &commonapisampling.TailSamplingConfiguration{
				NoisyOperationsEnforcement: enforcement,
			},
		},
	}
}

func nfSamplingRules(spec odigosv1.SamplingSpec) *[]odigosv1.Sampling {
	return &[]odigosv1.Sampling{{Spec: spec}}
}

// nfNoisySpec is a rule that matches every source (no source scopes).
func nfNoisySpec(names ...string) odigosv1.SamplingSpec {
	spec := odigosv1.SamplingSpec{}
	for _, name := range names {
		spec.NoisyOperations = append(spec.NoisyOperations, odigosv1.NoisyOperation{Name: name})
	}
	return spec
}

func nfCalculate(t *testing.T, d *distro.OtelDistro, config *common.OdigosConfiguration, rules *[]odigosv1.Sampling) (*agentsignalconfig.AgentTracesConfig, *commonapi.ContainerCollectorConfig) {
	t.Helper()

	actions := []odigosv1.Action{}
	irls := []odigosv1.InstrumentationRule{}
	runtimeDetails := &odigosv1.RuntimeDetailsByContainer{
		ContainerName: "app",
		Language:      common.JavaProgrammingLanguage,
	}
	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "ns", Kind: k8sconsts.WorkloadKindDeployment}

	agentConfig, collectorConfig, disabledInfo := calculateTracesConfig(
		&actions, "app", runtimeDetails, pw, d, nil, config, rules, &irls, nil)
	require.Nil(t, disabledInfo)
	require.NotNil(t, agentConfig)
	return agentConfig, collectorConfig
}

// nil means "no noisy operation is enforced here", regardless of whether the enclosing
// config was left unset or was created with an empty operation list.
func nfHeadNoisyOperationNames(agentConfig *agentsignalconfig.AgentTracesConfig) []string {
	if agentConfig == nil || agentConfig.HeadSampling == nil || len(agentConfig.HeadSampling.NoisyOperations) == 0 {
		return nil
	}
	names := make([]string, 0, len(agentConfig.HeadSampling.NoisyOperations))
	for _, op := range agentConfig.HeadSampling.NoisyOperations {
		names = append(names, op.Name)
	}
	return names
}

func nfTailNoisyOperationNames(collectorConfig *commonapi.ContainerCollectorConfig) []string {
	if collectorConfig == nil || collectorConfig.TailSampling == nil || len(collectorConfig.TailSampling.NoisyOperations) == 0 {
		return nil
	}
	names := make([]string, 0, len(collectorConfig.TailSampling.NoisyOperations))
	for _, op := range collectorConfig.TailSampling.NoisyOperations {
		names = append(names, op.Name)
	}
	return names
}

// The decision matrix CORE-1719 introduced: noisy operations are enforced at the agent (head)
// whenever the distro supports head sampling, and additionally at the collector (tail) when
// either the distro cannot head-sample, or the operator opted in with "always".
func TestNoisyOperationsEnforcementRoutesNoisyOperationsBetweenHeadAndTail(t *testing.T) {
	t.Parallel()

	noisyOps := []string{"healthz", "metrics-scrape"}

	tests := []struct {
		name                  string
		headSamplingSupported bool
		config                *common.OdigosConfiguration
		wantHeadNoisyOps      []string
		wantTailNoisyOps      []string
	}{
		{
			name:                  "head capable distro, enforcement unset, enforces at head only",
			headSamplingSupported: true,
			config:                nfConfig(nil),
			wantHeadNoisyOps:      noisyOps,
			wantTailNoisyOps:      nil,
		},
		{
			name:                  "head capable distro, auto, enforces at head only",
			headSamplingSupported: true,
			config:                nfConfig(nfEnforcement(nfAutoLiteral)),
			wantHeadNoisyOps:      noisyOps,
			wantTailNoisyOps:      nil,
		},
		{
			name:                  "head capable distro, always, enforces at head and tail",
			headSamplingSupported: true,
			config:                nfConfig(nfEnforcement(nfAlwaysLiteral)),
			wantHeadNoisyOps:      noisyOps,
			wantTailNoisyOps:      noisyOps,
		},
		{
			name:                  "head incapable distro, enforcement unset, enforces at tail only",
			headSamplingSupported: false,
			config:                nfConfig(nil),
			wantHeadNoisyOps:      nil,
			wantTailNoisyOps:      noisyOps,
		},
		{
			name:                  "head incapable distro, auto, enforces at tail only",
			headSamplingSupported: false,
			config:                nfConfig(nfEnforcement(nfAutoLiteral)),
			wantHeadNoisyOps:      nil,
			wantTailNoisyOps:      noisyOps,
		},
		{
			name:                  "head incapable distro, always, enforces at tail only",
			headSamplingSupported: false,
			config:                nfConfig(nfEnforcement(nfAlwaysLiteral)),
			wantHeadNoisyOps:      nil,
			wantTailNoisyOps:      noisyOps,
		},
	}

	headEnforcedCount := 0
	tailEnforcedCount := 0
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agentConfig, collectorConfig := nfCalculate(t, nfDistro(tt.headSamplingSupported), tt.config, nfSamplingRules(nfNoisySpec(noisyOps...)))

			require.Equal(t, tt.wantHeadNoisyOps, nfHeadNoisyOperationNames(agentConfig))
			require.Equal(t, tt.wantTailNoisyOps, nfTailNoisyOperationNames(collectorConfig))
		})

		if tt.wantHeadNoisyOps != nil {
			headEnforcedCount++
		}
		if tt.wantTailNoisyOps != nil {
			tailEnforcedCount++
		}
	}

	// guard against a table that only ever expects one side of the decision
	require.Equal(t, 3, headEnforcedCount)
	require.Equal(t, 4, tailEnforcedCount)
}

// Each level of the four-deep nil chain guarding the enforcement lookup must independently
// fall back to the "auto" behaviour, and only the exact "always" value may opt in.
func TestNoisyOperationsEnforcementOptsInOnlyOnTheExactAlwaysValue(t *testing.T) {
	t.Parallel()

	noisyOps := []string{"healthz"}

	tests := []struct {
		name            string
		config          *common.OdigosConfiguration
		wantTailEnabled bool
	}{
		{
			name:   "sampling config absent",
			config: &common.OdigosConfiguration{},
		},
		{
			name:   "tail sampling config absent",
			config: &common.OdigosConfiguration{Sampling: &common.SamplingConfiguration{}},
		},
		{
			name:   "enforcement absent",
			config: nfConfig(nil),
		},
		{
			name:   "enforcement empty string",
			config: nfConfig(nfEnforcement("")),
		},
		{
			name:   "enforcement auto",
			config: nfConfig(nfEnforcement(nfAutoLiteral)),
		},
		{
			name:   "enforcement capitalized always",
			config: nfConfig(nfEnforcement("Always")),
		},
		{
			name:   "enforcement upper case always",
			config: nfConfig(nfEnforcement("ALWAYS")),
		},
		{
			name:   "enforcement always with trailing space",
			config: nfConfig(nfEnforcement(nfAlwaysLiteral + " ")),
		},
		{
			name:   "enforcement plural always",
			config: nfConfig(nfEnforcement(nfAlwaysLiteral + "s")),
		},
		{
			name:            "enforcement always",
			config:          nfConfig(nfEnforcement(nfAlwaysLiteral)),
			wantTailEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// a head sampling capable distro is the only one where the enforcement value matters
			agentConfig, collectorConfig := nfCalculate(t, nfDistro(true), tt.config, nfSamplingRules(nfNoisySpec(noisyOps...)))

			require.Equal(t, noisyOps, nfHeadNoisyOperationNames(agentConfig))
			if tt.wantTailEnabled {
				require.Equal(t, noisyOps, nfTailNoisyOperationNames(collectorConfig))
			} else {
				require.Nil(t, nfTailNoisyOperationNames(collectorConfig))
			}
		})
	}
}

// "always" must not fabricate a tail sampling config for a source that has no noisy
// operations - that would add tail sampling cost to every workload in the cluster.
func TestNoisyOperationsEnforcementAlwaysWithoutNoisyOperationsLeavesTailSamplingUnset(t *testing.T) {
	t.Parallel()

	for _, headSamplingSupported := range []bool{true, false} {
		agentConfig, collectorConfig := nfCalculate(t,
			nfDistro(headSamplingSupported),
			nfConfig(nfEnforcement(nfAlwaysLiteral)),
			&[]odigosv1.Sampling{})

		require.Nil(t, agentConfig.HeadSampling)
		if collectorConfig != nil {
			require.Nil(t, collectorConfig.TailSampling)
		}
	}
}

// Noisy operations and the highly-relevant / cost-reduction rules are written into the same
// TailSamplingSourceConfig by two separate blocks, so each combination has to prove that
// neither block drops what the other wrote.
func TestTailSamplingCombinesNoisyOperationsWithHighlyRelevantAndCostRules(t *testing.T) {
	t.Parallel()

	spec := nfNoisySpec("healthz")
	spec.HighlyRelevantOperations = []odigosv1.HighlyRelevantOperation{{Name: "checkout"}}
	spec.CostReductionRules = []odigosv1.CostReductionRule{{Name: "static-assets"}}

	tests := []struct {
		name             string
		config           *common.OdigosConfiguration
		wantTailNoisyOps []string
	}{
		{
			name:             "auto keeps noisy operations out of tail sampling",
			config:           nfConfig(nfEnforcement(nfAutoLiteral)),
			wantTailNoisyOps: nil,
		},
		{
			name:             "always adds noisy operations alongside the other tail rules",
			config:           nfConfig(nfEnforcement(nfAlwaysLiteral)),
			wantTailNoisyOps: []string{"healthz"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, collectorConfig := nfCalculate(t, nfDistro(true), tt.config, nfSamplingRules(spec))

			require.NotNil(t, collectorConfig)
			require.NotNil(t, collectorConfig.TailSampling)
			require.Equal(t, tt.wantTailNoisyOps, nfTailNoisyOperationNames(collectorConfig))
			require.Len(t, collectorConfig.TailSampling.HighlyRelevantOperations, 1)
			require.Equal(t, "checkout", collectorConfig.TailSampling.HighlyRelevantOperations[0].Name)
			require.Len(t, collectorConfig.TailSampling.CostReductionRules, 1)
			require.Equal(t, "static-assets", collectorConfig.TailSampling.CostReductionRules[0].Name)
		})
	}
}

// The enforcement value is persisted in the odigos config and consumed here by string
// comparison, so the constants must keep the exact values the helm chart and the settings
// screen offer.
func TestNoisyOperationsEnforcementConstantsMatchTheirWireValues(t *testing.T) {
	t.Parallel()

	require.Equal(t, nfAutoLiteral, string(commonapisampling.NoisyOperationsEnforcementAuto))
	require.Equal(t, nfAlwaysLiteral, string(commonapisampling.NoisyOperationsEnforcementAlways))
	require.NotEqual(t, commonapisampling.NoisyOperationsEnforcementAuto, commonapisampling.NoisyOperationsEnforcementAlways)
}
