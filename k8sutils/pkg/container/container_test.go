package container

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

func agentConfig(containerName, distroName string) odigosv1.ContainerAgentConfig {
	return odigosv1.ContainerAgentConfig{
		ContainerName:  containerName,
		AgentEnabled:   true,
		OtelDistroName: distroName,
	}
}

func waitingStatus(reason string) *corev1.ContainerStatus {
	return &corev1.ContainerStatus{
		Name:  "app",
		Ready: false,
		State: corev1.ContainerState{
			Waiting: &corev1.ContainerStateWaiting{Reason: reason},
		},
	}
}

func TestGetContainerConfigByName(t *testing.T) {
	containers := []odigosv1.ContainerAgentConfig{
		agentConfig("istio-proxy", ""),
		agentConfig("app", "nodejs-community"),
		agentConfig("sidecar", "java-community"),
	}

	t.Run("finds the config for the requested container", func(t *testing.T) {
		found := GetContainerConfigByName(containers, "app")

		require.NotNil(t, found)
		assert.Equal(t, "app", found.ContainerName)
		assert.Equal(t, "nodejs-community", found.OtelDistroName)
	})

	t.Run("returns nil for a container that has no config", func(t *testing.T) {
		assert.Nil(t, GetContainerConfigByName(containers, "not-in-the-workload"))
	})

	t.Run("returns nil for an empty container list", func(t *testing.T) {
		assert.Nil(t, GetContainerConfigByName(nil, "app"))
		assert.Nil(t, GetContainerConfigByName([]odigosv1.ContainerAgentConfig{}, "app"))
	})

	t.Run("container names are matched exactly", func(t *testing.T) {
		assert.Nil(t, GetContainerConfigByName(containers, "App"))
		assert.Nil(t, GetContainerConfigByName(containers, "ap"))
		assert.Nil(t, GetContainerConfigByName(containers, ""))
	})

	// The result is the element of the caller's slice, not a copy of it.
	t.Run("returns the element itself", func(t *testing.T) {
		found := GetContainerConfigByName(containers, "sidecar")

		require.NotNil(t, found)
		assert.Same(t, &containers[2], found)
	})
}

// CrashLoopBackOff and ImagePullBackOff are the literal reasons the kubelet writes into the
// waiting state, and odigos uses them to decide that an injected agent has broken the workload.
func TestContainerBackOffDetection(t *testing.T) {
	tests := []struct {
		name               string
		status             *corev1.ContainerStatus
		expectedCrashLoop  bool
		expectedImagePull  bool
		expectedAnyBackOff bool
	}{
		{
			name:               "crash loop back off",
			status:             waitingStatus("CrashLoopBackOff"),
			expectedCrashLoop:  true,
			expectedImagePull:  false,
			expectedAnyBackOff: true,
		},
		{
			name:               "image pull back off",
			status:             waitingStatus("ImagePullBackOff"),
			expectedCrashLoop:  false,
			expectedImagePull:  true,
			expectedAnyBackOff: true,
		},
		{
			name:               "waiting to start for an unrelated reason",
			status:             waitingStatus("ContainerCreating"),
			expectedCrashLoop:  false,
			expectedImagePull:  false,
			expectedAnyBackOff: false,
		},
		{
			name:               "a failed image pull has not backed off yet",
			status:             waitingStatus("ErrImagePull"),
			expectedCrashLoop:  false,
			expectedImagePull:  false,
			expectedAnyBackOff: false,
		},
		{
			name:               "an empty waiting reason",
			status:             waitingStatus(""),
			expectedCrashLoop:  false,
			expectedImagePull:  false,
			expectedAnyBackOff: false,
		},
		{
			name: "a running container is never backing off",
			status: &corev1.ContainerStatus{
				Name:  "app",
				Ready: true,
				State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
			},
			expectedCrashLoop:  false,
			expectedImagePull:  false,
			expectedAnyBackOff: false,
		},
		{
			name: "a terminated container is never backing off",
			status: &corev1.ContainerStatus{
				Name: "app",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{Reason: "CrashLoopBackOff"},
				},
			},
			expectedCrashLoop:  false,
			expectedImagePull:  false,
			expectedAnyBackOff: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedCrashLoop, IsContainerInCrashLoopBackOff(tt.status))
			assert.Equal(t, tt.expectedImagePull, IsContainerInImagePullBackOff(tt.status))
			assert.Equal(t, tt.expectedAnyBackOff, IsContainerInBackOff(tt.status))
		})
	}
}
