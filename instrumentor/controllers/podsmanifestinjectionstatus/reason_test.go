package podsmanifestinjectionstatus

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	podsManifestInjection "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The observable pod injection states the reason tree branches on.
var (
	injectionStateNoPods      = odigosv1.PodsManifestInjectionStatus{}
	injectionStateAllUpToDate = odigosv1.PodsManifestInjectionStatus{
		HasInjectedUpToDatePods: true,
	}
	// every pod is injected, but some run a stale agents hash
	injectionStateSomeOutOfDate = odigosv1.PodsManifestInjectionStatus{
		HasInjectedUpToDatePods: true, HasInjectedOutOfDatePods: true,
	}
	// some pods were never injected
	injectionStateSomeUninjected = odigosv1.PodsManifestInjectionStatus{
		HasInjectedUpToDatePods: true, HasUninjectedPods: true,
	}
	// no pod is injected, which is the desired state once the agent is disabled
	injectionStateNoneInjected = odigosv1.PodsManifestInjectionStatus{
		HasUninjectedPods: true,
	}
)

func newReasonInstrumentationConfig(agentInjectionEnabled bool) *odigosv1.InstrumentationConfig {
	return &odigosv1.InstrumentationConfig{
		Spec: odigosv1.InstrumentationConfigSpec{
			AgentInjectionEnabled: agentInjectionEnabled,
			AgentsMetaHash:        injectionCurrentHash,
		},
	}
}

func withRolloutReason(ic *odigosv1.InstrumentationConfig,
	reason odigosv1.WorkloadRolloutReason) *odigosv1.InstrumentationConfig {
	meta.SetStatusCondition(&ic.Status.Conditions, metav1.Condition{
		Type:   odigosv1.WorkloadRolloutStatusConditionType,
		Status: metav1.ConditionTrue,
		Reason: string(reason),
	})
	return ic
}

func withRolloutHash(ic *odigosv1.InstrumentationConfig, hash string) *odigosv1.InstrumentationConfig {
	ic.Status.WorkloadRolloutHash = hash
	return ic
}

func withOptionalPodManifestInjection(ic *odigosv1.InstrumentationConfig) *odigosv1.InstrumentationConfig {
	ic.Spec.PodManifestInjectionOptional = true
	return ic
}

func automaticRolloutConfig(disabled bool) *common.OdigosConfiguration {
	return &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{AutomaticRolloutDisabled: &disabled},
	}
}

// TestCalculatePodsManifestInjectionReason pins the reason reported on the PodsManifestInjection
// condition for each observable state. The reason is what the UI and the CLI show the user, and
// several of these branches only differ in which of two near-identical reasons they select.
func TestCalculatePodsManifestInjectionReason(t *testing.T) {
	tests := []struct {
		name         string
		status       odigosv1.PodsManifestInjectionStatus
		ic           *odigosv1.InstrumentationConfig
		config       *common.OdigosConfiguration
		workloadKind k8sconsts.WorkloadKind
		expected     podsManifestInjection.PodsManifestInjectionReason
	}{
		// no pods is reported before the agent enabled/disabled split
		{
			name:         "no pods while enabled",
			status:       injectionStateNoPods,
			ic:           newReasonInstrumentationConfig(true),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonNoPods,
		},
		{
			name:         "no pods while disabled",
			status:       injectionStateNoPods,
			ic:           newReasonInstrumentationConfig(false),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonNoPods,
		},

		// agent enabled
		{
			name:         "every pod runs the current agents hash",
			status:       injectionStateAllUpToDate,
			ic:           newReasonInstrumentationConfig(true),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonPodsAppliedSuccessfully_Enabled,
		},
		{
			name:         "a distro that does not need pod manifest injection",
			status:       injectionStateSomeUninjected,
			ic:           withOptionalPodManifestInjection(newReasonInstrumentationConfig(true)),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonPodsManifestInjectionNotRequired_Enabled,
		},
		{
			name:         "optional pod manifest injection takes precedence over the workload kind",
			status:       injectionStateSomeUninjected,
			ic:           withOptionalPodManifestInjection(newReasonInstrumentationConfig(true)),
			workloadKind: k8sconsts.WorkloadKindStaticPod,
			expected:     podsManifestInjection.PodsManifestInjectionReasonPodsManifestInjectionNotRequired_Enabled,
		},
		{
			name:         "a static pod cannot be rolled out",
			status:       injectionStateSomeUninjected,
			ic:           newReasonInstrumentationConfig(true),
			workloadKind: k8sconsts.WorkloadKindStaticPod,
			expected:     podsManifestInjection.PodsManifestInjectionReasonRolloutNotSupportedForStaticPods_Enabled,
		},
		{
			name:         "a static pod with only out of date pods reports the same reason",
			status:       injectionStateSomeOutOfDate,
			ic:           newReasonInstrumentationConfig(true),
			workloadKind: k8sconsts.WorkloadKindStaticPod,
			expected:     podsManifestInjection.PodsManifestInjectionReasonRolloutNotSupportedForStaticPods_Enabled,
		},
		{
			name:         "a job picks up the agent on its next run",
			status:       injectionStateSomeUninjected,
			ic:           newReasonInstrumentationConfig(true),
			workloadKind: k8sconsts.WorkloadKindJob,
			expected:     podsManifestInjection.PodsManifestInjectionReasonWaitingForNextJobRun_Enabled,
		},
		{
			name:         "a cronjob with every pod injected but out of date",
			status:       injectionStateSomeOutOfDate,
			ic:           newReasonInstrumentationConfig(true),
			workloadKind: k8sconsts.WorkloadKindCronJob,
			expected:     podsManifestInjection.PodsManifestInjectionReasonWaitingForNextJobRun_UpToDate,
		},
		{
			name:         "a cronjob waits for its next run even when automatic rollout is disabled",
			status:       injectionStateSomeUninjected,
			ic:           newReasonInstrumentationConfig(true),
			config:       automaticRolloutConfig(true),
			workloadKind: k8sconsts.WorkloadKindCronJob,
			expected:     podsManifestInjection.PodsManifestInjectionReasonWaitingForNextJobRun_Enabled,
		},
		{
			name:         "automatic rollout is disabled cluster wide",
			status:       injectionStateSomeUninjected,
			ic:           newReasonInstrumentationConfig(true),
			config:       automaticRolloutConfig(true),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonRestartRequiredAutoRolloutDisabled_Enabled,
		},
		{
			name:         "automatic rollout is disabled and every pod is already injected",
			status:       injectionStateSomeOutOfDate,
			ic:           newReasonInstrumentationConfig(true),
			config:       automaticRolloutConfig(true),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonRestartRequiredAutoRolloutDisabled_UpToDate,
		},
		{
			name:         "automatic rollout is explicitly enabled",
			status:       injectionStateSomeUninjected,
			ic:           newReasonInstrumentationConfig(true),
			config:       automaticRolloutConfig(false),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonWaitingForAutomaticRollout_Enabled,
		},
		{
			name:         "the rollout is queued behind other workloads",
			status:       injectionStateSomeUninjected,
			ic:           withRolloutReason(newReasonInstrumentationConfig(true), odigosv1.WorkloadRolloutReasonWaitingInQueue),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonWaitingInRolloutQueue_Enabled,
		},
		{
			name:         "the rollout is queued and every pod is already injected",
			status:       injectionStateSomeOutOfDate,
			ic:           withRolloutReason(newReasonInstrumentationConfig(true), odigosv1.WorkloadRolloutReasonWaitingInQueue),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonWaitingInRolloutQueue_UpToDate,
		},
		{
			name:         "the rollout was triggered and is still progressing",
			status:       injectionStateSomeUninjected,
			ic:           withRolloutReason(newReasonInstrumentationConfig(true), odigosv1.WorkloadRolloutReasonTriggeredSuccessfully),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonRolloutInProgress_Enabled,
		},
		{
			name:         "a previous rollout is still ongoing",
			status:       injectionStateSomeOutOfDate,
			ic:           withRolloutReason(newReasonInstrumentationConfig(true), odigosv1.WorkloadRolloutReasonPreviousRolloutOngoing),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonRolloutInProgress_UpToDate,
		},
		{
			name:         "odigos failed to patch the workload",
			status:       injectionStateSomeUninjected,
			ic:           withRolloutReason(newReasonInstrumentationConfig(true), odigosv1.WorkloadRolloutReasonFailedToPatch),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonRestartRequiredAutoRolloutFailed_Enabled,
		},
		{
			name:         "odigos failed to patch the workload and every pod is already injected",
			status:       injectionStateSomeOutOfDate,
			ic:           withRolloutReason(newReasonInstrumentationConfig(true), odigosv1.WorkloadRolloutReasonFailedToPatch),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonRestartRequiredAutoRolloutFailed_UpToDate,
		},
		{
			name:         "a disabled automatic rollout takes precedence over the rollout condition",
			status:       injectionStateSomeUninjected,
			ic:           withRolloutReason(newReasonInstrumentationConfig(true), odigosv1.WorkloadRolloutReasonWaitingInQueue),
			config:       automaticRolloutConfig(true),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonRestartRequiredAutoRolloutDisabled_Enabled,
		},
		{
			name:         "pods bypassed the webhook after a rollout for the current hash",
			status:       injectionStateSomeUninjected,
			ic:           withRolloutHash(newReasonInstrumentationConfig(true), injectionCurrentHash),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonRestartRequiredWebhookMissed_Enabled,
		},
		{
			name:         "pods bypassed the webhook and every pod is injected",
			status:       injectionStateSomeOutOfDate,
			ic:           withRolloutHash(newReasonInstrumentationConfig(true), injectionCurrentHash),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonRestartRequiredWebhookMissed_UpToDate,
		},
		{
			name:         "the recorded rollout is for an older agents hash",
			status:       injectionStateSomeUninjected,
			ic:           withRolloutHash(newReasonInstrumentationConfig(true), injectionStaleHash),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonWaitingForAutomaticRollout_Enabled,
		},
		{
			name:         "no rollout has been recorded yet",
			status:       injectionStateSomeOutOfDate,
			ic:           newReasonInstrumentationConfig(true),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonWaitingForAutomaticRollout_UpToDate,
		},
		{
			// before the agents hash is computed both hashes are empty, which must not be read
			// as "a rollout already ran for this hash"
			name:   "neither the agents hash nor the rollout hash is set yet",
			status: injectionStateSomeUninjected,
			ic: func() *odigosv1.InstrumentationConfig {
				ic := newReasonInstrumentationConfig(true)
				ic.Spec.AgentsMetaHash = ""
				return ic
			}(),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonWaitingForAutomaticRollout_Enabled,
		},
		{
			name:   "the rollout condition takes precedence over the recorded rollout hash",
			status: injectionStateSomeUninjected,
			ic: withRolloutHash(withRolloutReason(newReasonInstrumentationConfig(true),
				odigosv1.WorkloadRolloutReasonFailedToPatch), injectionCurrentHash),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonRestartRequiredAutoRolloutFailed_Enabled,
		},
		{
			name:   "a finished rollout falls through to the recorded rollout hash",
			status: injectionStateSomeUninjected,
			ic: withRolloutHash(withRolloutReason(newReasonInstrumentationConfig(true),
				odigosv1.WorkloadRolloutReasonRolloutFinished), injectionCurrentHash),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonRestartRequiredWebhookMissed_Enabled,
		},

		// agent disabled
		{
			name:         "no pod is injected any more",
			status:       injectionStateNoneInjected,
			ic:           newReasonInstrumentationConfig(false),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonPodsAppliedSuccessfully_Disabled,
		},
		{
			name:         "some pods are still injected while others are not",
			status:       injectionStateSomeUninjected,
			ic:           newReasonInstrumentationConfig(false),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonWaitingForAutomaticRollout_Disabled,
		},
		{
			name:         "a static pod still carrying the agent cannot be rolled out",
			status:       injectionStateAllUpToDate,
			ic:           newReasonInstrumentationConfig(false),
			workloadKind: k8sconsts.WorkloadKindStaticPod,
			expected:     podsManifestInjection.PodsManifestInjectionReasonRolloutNotSupportedForStaticPods_Enabled,
		},
		{
			name:         "a job drops the agent on its next run",
			status:       injectionStateAllUpToDate,
			ic:           newReasonInstrumentationConfig(false),
			workloadKind: k8sconsts.WorkloadKindJob,
			expected:     podsManifestInjection.PodsManifestInjectionReasonWaitingForNextJobRun_Disabled,
		},
		{
			name:         "a cronjob drops the agent on its next run",
			status:       injectionStateAllUpToDate,
			ic:           newReasonInstrumentationConfig(false),
			workloadKind: k8sconsts.WorkloadKindCronJob,
			expected:     podsManifestInjection.PodsManifestInjectionReasonWaitingForNextJobRun_Disabled,
		},
		{
			name:         "automatic rollout is disabled cluster wide",
			status:       injectionStateAllUpToDate,
			ic:           newReasonInstrumentationConfig(false),
			config:       automaticRolloutConfig(true),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonRestartRequiredAutoRolloutDisabled_Disabled,
		},
		{
			name:         "the rollout is queued behind other workloads",
			status:       injectionStateAllUpToDate,
			ic:           withRolloutReason(newReasonInstrumentationConfig(false), odigosv1.WorkloadRolloutReasonWaitingInQueue),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonWaitingInRolloutQueue_Disabled,
		},
		{
			name:         "the rollout was triggered and is still progressing",
			status:       injectionStateAllUpToDate,
			ic:           withRolloutReason(newReasonInstrumentationConfig(false), odigosv1.WorkloadRolloutReasonTriggeredSuccessfully),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonRolloutInProgress_Disabled,
		},
		{
			name:         "a previous rollout is still ongoing",
			status:       injectionStateAllUpToDate,
			ic:           withRolloutReason(newReasonInstrumentationConfig(false), odigosv1.WorkloadRolloutReasonPreviousRolloutOngoing),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonRolloutInProgress_Disabled,
		},
		{
			name:         "odigos failed to patch the workload",
			status:       injectionStateAllUpToDate,
			ic:           withRolloutReason(newReasonInstrumentationConfig(false), odigosv1.WorkloadRolloutReasonFailedToPatch),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonRestartRequiredAutoRolloutFailed_Disabled,
		},
		{
			name:         "the rollout finished but injected pods remain",
			status:       injectionStateAllUpToDate,
			ic:           withRolloutReason(newReasonInstrumentationConfig(false), odigosv1.WorkloadRolloutReasonRolloutFinished),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonRestartRequiredWebhookMissed_Enabled,
		},
		{
			name:         "optional pod manifest injection does not short circuit the disabled path",
			status:       injectionStateAllUpToDate,
			ic:           withOptionalPodManifestInjection(newReasonInstrumentationConfig(false)),
			workloadKind: k8sconsts.WorkloadKindDeployment,
			expected:     podsManifestInjection.PodsManifestInjectionReasonWaitingForAutomaticRollout_Disabled,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.expected)+"/"+tt.name, func(t *testing.T) {
			config := tt.config
			if config == nil {
				config = &common.OdigosConfiguration{}
			}

			reason := calculatePodsManifestInjectionReason(tt.status, tt.ic, config, tt.workloadKind)

			assert.Equal(t, string(tt.expected), reason.Name)
			assert.NotEmpty(t, reason.K8sConditionStatus,
				"every reported reason must carry a condition status")
		})
	}
}

// selectEnabledOrUpToDateReason distinguishes "some pods were never injected" from "every pod is
// injected but some are stale", which is the difference between the two reasons the user sees.
func TestSelectEnabledOrUpToDateReason(t *testing.T) {
	enabled := podsManifestInjection.PodsManifestInjectionWaitingInRolloutQueue_Enabled
	upToDate := podsManifestInjection.PodsManifestInjectionWaitingInRolloutQueue_UpToDate

	assert.Equal(t, enabled.Name,
		selectEnabledOrUpToDateReason(injectionStateSomeUninjected, enabled, upToDate).Name)
	assert.Equal(t, upToDate.Name,
		selectEnabledOrUpToDateReason(injectionStateSomeOutOfDate, enabled, upToDate).Name)
	assert.Equal(t, upToDate.Name,
		selectEnabledOrUpToDateReason(injectionStateAllUpToDate, enabled, upToDate).Name)
}
