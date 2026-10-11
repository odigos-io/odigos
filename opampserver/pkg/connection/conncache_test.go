package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newTestConnectionInfo(namespace string, podName string, containerName string, pid int64) *ConnectionInfo {
	return &ConnectionInfo{
		Pod: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: namespace,
			},
		},
		ContainerName: containerName,
		Pid:           pid,
	}
}

// Agents report the pid from their own container's pid namespace, so two containers of the same pod
// both commonly report pid 1. Adding the second one must not evict the first.
func TestAddConnection_TwoContainersInSamePodWithSamePid(t *testing.T) {
	cache := NewConnectionsCache()

	cache.AddConnection("uid-nodejs", newTestConnectionInfo("default", "my-pod", "nodejs", 1))
	cache.AddConnection("uid-python", newTestConnectionInfo("default", "my-pod", "python", 1))

	nodejsConn, ok := cache.GetConnection("uid-nodejs")
	require.True(t, ok, "the first container's connection was evicted by the second container")
	assert.Equal(t, "nodejs", nodejsConn.ContainerName)

	pythonConn, ok := cache.GetConnection("uid-python")
	require.True(t, ok)
	assert.Equal(t, "python", pythonConn.ContainerName)
}

// Two pods can carry the same name in different namespaces (a StatefulSet gives deterministic pod
// names), and both can be scheduled on the same node and thus talk to the same opamp server.
func TestAddConnection_SamePodNameInDifferentNamespaces(t *testing.T) {
	cache := NewConnectionsCache()

	cache.AddConnection("uid-ns-a", newTestConnectionInfo("ns-a", "web-0", "app", 1))
	cache.AddConnection("uid-ns-b", newTestConnectionInfo("ns-b", "web-0", "app", 1))

	_, ok := cache.GetConnection("uid-ns-a")
	require.True(t, ok, "a pod in another namespace evicted this connection")
	_, ok = cache.GetConnection("uid-ns-b")
	require.True(t, ok)
}

// The reason removeMatchingConnections exists: a process re-executing itself inside the same
// container reconnects with a new instance uid, and the stale entry must go.
func TestAddConnection_RespawnedProcessInSameContainer(t *testing.T) {
	cache := NewConnectionsCache()

	cache.AddConnection("uid-old", newTestConnectionInfo("default", "my-pod", "python", 1))
	cache.AddConnection("uid-new", newTestConnectionInfo("default", "my-pod", "python", 1))

	_, ok := cache.GetConnection("uid-old")
	assert.False(t, ok, "the stale connection of the replaced process was not removed")
	_, ok = cache.GetConnection("uid-new")
	assert.True(t, ok)
}
