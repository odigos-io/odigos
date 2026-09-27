package podswebhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestMountDirectory(t *testing.T) {
	t.Run("resolves the agents dir placeholder and mounts the sub path read only", func(t *testing.T) {
		container := &corev1.Container{Name: "test"}

		MountDirectory(container, "{{ODIGOS_AGENTS_DIR}}/nodejs-community")

		require.Len(t, container.VolumeMounts, 1)
		assert.Equal(t, corev1.VolumeMount{
			Name:      "odigos-agent",
			SubPath:   "nodejs-community",
			MountPath: "/var/odigos/nodejs-community",
			ReadOnly:  true,
		}, container.VolumeMounts[0])
	})

	t.Run("nested directory uses only the last path element as sub path", func(t *testing.T) {
		container := &corev1.Container{Name: "test"}

		MountDirectory(container, "{{ODIGOS_AGENTS_DIR}}/loader")

		require.Len(t, container.VolumeMounts, 1)
		assert.Equal(t, "loader", container.VolumeMounts[0].SubPath)
		assert.Equal(t, "/var/odigos/loader", container.VolumeMounts[0].MountPath)
	})

	t.Run("is idempotent for the same directory", func(t *testing.T) {
		container := &corev1.Container{Name: "test"}

		MountDirectory(container, "{{ODIGOS_AGENTS_DIR}}/nodejs-community")
		MountDirectory(container, "{{ODIGOS_AGENTS_DIR}}/nodejs-community")

		assert.Len(t, container.VolumeMounts, 1)
	})

	t.Run("an absolute path is treated the same as the placeholder path", func(t *testing.T) {
		container := &corev1.Container{Name: "test"}

		MountDirectory(container, "{{ODIGOS_AGENTS_DIR}}/nodejs-community")
		MountDirectory(container, "/var/odigos/nodejs-community")

		assert.Len(t, container.VolumeMounts, 1)
	})

	t.Run("different directories are mounted side by side", func(t *testing.T) {
		container := &corev1.Container{Name: "test"}

		MountDirectory(container, "{{ODIGOS_AGENTS_DIR}}/nodejs-community")
		MountDirectory(container, "{{ODIGOS_AGENTS_DIR}}/opentelemetry-node")

		require.Len(t, container.VolumeMounts, 2)
		assert.Equal(t, "/var/odigos/nodejs-community", container.VolumeMounts[0].MountPath)
		assert.Equal(t, "/var/odigos/opentelemetry-node", container.VolumeMounts[1].MountPath)
	})

	t.Run("user volume mounts are preserved", func(t *testing.T) {
		container := &corev1.Container{
			Name:         "test",
			VolumeMounts: []corev1.VolumeMount{{Name: "user-data", MountPath: "/data"}},
		}

		MountDirectory(container, "{{ODIGOS_AGENTS_DIR}}/nodejs-community")

		require.Len(t, container.VolumeMounts, 2)
		assert.Equal(t, "user-data", container.VolumeMounts[0].Name)
	})
}

func TestMountPodVolume(t *testing.T) {
	t.Run("host path", func(t *testing.T) {
		pod := &corev1.Pod{}

		MountPodVolumeToHostPath(pod)

		require.Len(t, pod.Spec.Volumes, 1)
		assert.Equal(t, "odigos-agent", pod.Spec.Volumes[0].Name)
		require.NotNil(t, pod.Spec.Volumes[0].HostPath)
		assert.Equal(t, "/var/odigos", pod.Spec.Volumes[0].HostPath.Path)
		assert.Nil(t, pod.Spec.Volumes[0].EmptyDir)
		assert.Nil(t, pod.Spec.Volumes[0].CSI)
	})

	t.Run("empty dir", func(t *testing.T) {
		pod := &corev1.Pod{}

		MountPodVolumeToEmptyDir(pod)

		require.Len(t, pod.Spec.Volumes, 1)
		assert.Equal(t, "odigos-agent", pod.Spec.Volumes[0].Name)
		assert.NotNil(t, pod.Spec.Volumes[0].EmptyDir)
		assert.Nil(t, pod.Spec.Volumes[0].HostPath)
	})

	t.Run("csi driver", func(t *testing.T) {
		pod := &corev1.Pod{}

		MountPodVolumeToCSI(pod)

		require.Len(t, pod.Spec.Volumes, 1)
		assert.Equal(t, "odigos-agent", pod.Spec.Volumes[0].Name)
		require.NotNil(t, pod.Spec.Volumes[0].CSI)
		assert.Equal(t, "odigos.csi.driver", pod.Spec.Volumes[0].CSI.Driver)
		assert.Equal(t, map[string]string{"type": "instrumentation"}, pod.Spec.Volumes[0].CSI.VolumeAttributes)
	})

	t.Run("the odigos volume is added at most once", func(t *testing.T) {
		pod := &corev1.Pod{}

		MountPodVolumeToHostPath(pod)
		MountPodVolumeToHostPath(pod)
		// a second mount method must not add a second volume with the same name,
		// which the api server would reject as a duplicate.
		MountPodVolumeToEmptyDir(pod)
		MountPodVolumeToCSI(pod)

		require.Len(t, pod.Spec.Volumes, 1)
		assert.NotNil(t, pod.Spec.Volumes[0].HostPath)
	})

	t.Run("user volumes are preserved", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{{Name: "user-data"}},
			},
		}

		MountPodVolumeToEmptyDir(pod)

		require.Len(t, pod.Spec.Volumes, 2)
		assert.Equal(t, "user-data", pod.Spec.Volumes[0].Name)
		assert.Equal(t, "odigos-agent", pod.Spec.Volumes[1].Name)
	})
}
