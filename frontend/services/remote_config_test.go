package services

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/yaml"
)

func TestUpdateRemoteConfigMergesProfilingCacheLimits(t *testing.T) {
	const namespace = "test-namespace"
	t.Setenv(consts.CurrentNamespaceEnvVar, namespace)

	automaticRolloutDisabled := true
	profilingEnabled := true
	existing := &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{
			AutomaticRolloutDisabled: &automaticRolloutDisabled,
		},
		Profiling: &common.ProfilingConfiguration{
			Enabled: &profilingEnabled,
			Ui: &common.ProfilingUiConfiguration{
				MaxSlots:       10,
				SlotMaxBytes:   1024,
				SlotTTLSeconds: 60,
			},
		},
	}
	existingYAML, err := yaml.Marshal(existing)
	require.NoError(t, err)

	client := k8sfake.NewSimpleClientset(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosRemoteConfigName,
			Namespace: namespace,
		},
		Data: map[string]string{consts.OdigosConfigurationFileName: string(existingYAML)},
	})
	previousClient := kube.DefaultClient
	kube.SetDefaultClient(&kube.Client{Interface: client})
	t.Cleanup(func() {
		kube.SetDefaultClient(previousClient)
	})

	updated, err := UpdateRemoteConfig(context.Background(), &common.OdigosConfiguration{
		Profiling: &common.ProfilingConfiguration{
			Ui: &common.ProfilingUiConfiguration{MaxSlots: 20},
		},
	})
	require.NoError(t, err)

	require.NotNil(t, updated.Rollout)
	require.True(t, *updated.Rollout.AutomaticRolloutDisabled)
	require.NotNil(t, updated.Profiling)
	require.True(t, *updated.Profiling.Enabled)
	require.Equal(t, 20, updated.Profiling.Ui.MaxSlots)
	require.Equal(t, 1024, updated.Profiling.Ui.SlotMaxBytes)
	require.Equal(t, 60, updated.Profiling.Ui.SlotTTLSeconds)
}

func TestUpdateRemoteConfigPreservesProfilingWhenUpdatingRollout(t *testing.T) {
	current := &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{
			MaxConcurrentRollouts: 5,
		},
		Profiling: &common.ProfilingConfiguration{
			Ui: &common.ProfilingUiConfiguration{
				MaxSlots:       10,
				SlotMaxBytes:   1024,
				SlotTTLSeconds: 60,
			},
		},
	}
	automaticRolloutDisabled := true

	mergeRemoteConfigUpdate(current, &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{
			AutomaticRolloutDisabled: &automaticRolloutDisabled,
		},
	})

	require.NotNil(t, current.Rollout)
	require.True(t, *current.Rollout.AutomaticRolloutDisabled)
	require.Equal(t, 5, current.Rollout.MaxConcurrentRollouts)
	require.Equal(t, 10, current.Profiling.Ui.MaxSlots)
	require.Equal(t, 1024, current.Profiling.Ui.SlotMaxBytes)
	require.Equal(t, 60, current.Profiling.Ui.SlotTTLSeconds)
}

func TestUpdateRemoteConfigEmptyUpdatePreservesExistingConfig(t *testing.T) {
	automaticRolloutDisabled := true
	current := &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{
			AutomaticRolloutDisabled: &automaticRolloutDisabled,
		},
		Profiling: &common.ProfilingConfiguration{
			Ui: &common.ProfilingUiConfiguration{MaxSlots: 10},
		},
	}

	mergeRemoteConfigUpdate(current, &common.OdigosConfiguration{})

	require.NotNil(t, current.Rollout)
	require.True(t, *current.Rollout.AutomaticRolloutDisabled)
	require.Equal(t, 10, current.Profiling.Ui.MaxSlots)
}
