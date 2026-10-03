package predicate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

// collectorRoleObject builds an object carrying the collector role label. The label and its values
// are spelled out rather than taken from k8sconsts, because they have to keep matching the labels
// on collector pods that are already running.
func collectorRoleObject(role string) client.Object {
	object := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "odigos-gateway-1", Namespace: "odigos-system"}}
	if role != "" {
		object.Labels = map[string]string{"odigos.io/collector-role": role}
	}
	return object
}

func TestClusterCollectorsPredicate(t *testing.T) {
	filter := &ClusterCollectorsPredicate{}

	for _, tc := range []struct {
		name     string
		object   client.Object
		expected bool
	}{
		{
			name:     "a cluster gateway collector",
			object:   collectorRoleObject("CLUSTER_GATEWAY"),
			expected: true,
		},
		{
			// the node collectors are reconciled by a different manager
			name:     "a node collector",
			object:   collectorRoleObject("NODE_COLLECTOR"),
			expected: false,
		},
		{
			name:     "an object with no collector role",
			object:   collectorRoleObject(""),
			expected: false,
		},
		{
			name:     "no object",
			object:   nil,
			expected: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, filter.Create(event.CreateEvent{Object: tc.object}), "create")
			assert.Equal(t, tc.expected, filter.Delete(event.DeleteEvent{Object: tc.object}), "delete")
			assert.Equal(t, tc.expected, filter.Generic(event.GenericEvent{Object: tc.object}), "generic")
			assert.Equal(t, tc.expected, filter.Update(event.UpdateEvent{
				ObjectOld: tc.object,
				ObjectNew: tc.object,
			}), "update")
		})
	}
}

func TestClusterCollectorsPredicate_UpdateFollowsTheNewObject(t *testing.T) {
	filter := &ClusterCollectorsPredicate{}
	gateway := collectorRoleObject("CLUSTER_GATEWAY")
	unlabeled := collectorRoleObject("")

	// the label was just added, so the object became this manager's business
	assert.True(t, filter.Update(event.UpdateEvent{ObjectOld: unlabeled, ObjectNew: gateway}))
	assert.False(t, filter.Update(event.UpdateEvent{ObjectOld: gateway, ObjectNew: unlabeled}))
}

func TestClusterCollectorsPredicate_UpdateWithAHalfPopulatedEvent(t *testing.T) {
	filter := &ClusterCollectorsPredicate{}
	gateway := collectorRoleObject("CLUSTER_GATEWAY")

	assert.False(t, filter.Update(event.UpdateEvent{ObjectNew: gateway}))
	assert.False(t, filter.Update(event.UpdateEvent{ObjectOld: gateway}))
}
