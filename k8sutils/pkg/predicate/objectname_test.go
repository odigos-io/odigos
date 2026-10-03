package predicate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func namedObject(name string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "odigos-system"},
	}
}

func TestObjectNamePredicateMatchesOnlyTheAllowedName(t *testing.T) {
	p := ObjectNamePredicate{AllowedObjectName: "allowed"}

	tests := []struct {
		name       string
		objectName string
		want       bool
	}{
		{name: "exact match", objectName: "allowed", want: true},
		{name: "different name", objectName: "not-allowed", want: false},
		{name: "empty name", objectName: "", want: false},
		{name: "name differing by case", objectName: "Allowed", want: false},
		{name: "name with the allowed name as a prefix", objectName: "allowed-2", want: false},
		{name: "name with the allowed name as a suffix", objectName: "my-allowed", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := namedObject(tt.objectName)

			assert.Equal(t, tt.want, p.Create(event.CreateEvent{Object: obj}), "create")
			assert.Equal(t, tt.want, p.Update(event.UpdateEvent{ObjectOld: obj, ObjectNew: obj}), "update")
			assert.Equal(t, tt.want, p.Delete(event.DeleteEvent{Object: obj}), "delete")
			assert.Equal(t, tt.want, p.Generic(event.GenericEvent{Object: obj}), "generic")
		})
	}
}

// The update verdict is taken from the new object, so an object renamed into the
// allowed name is let through and one renamed out of it is filtered.
func TestObjectNamePredicateUpdateLooksAtTheNewObject(t *testing.T) {
	p := ObjectNamePredicate{AllowedObjectName: "allowed"}

	renamedIn := p.Update(event.UpdateEvent{
		ObjectOld: namedObject("something-else"),
		ObjectNew: namedObject("allowed"),
	})
	assert.True(t, renamedIn)

	renamedOut := p.Update(event.UpdateEvent{
		ObjectOld: namedObject("allowed"),
		ObjectNew: namedObject("something-else"),
	})
	assert.False(t, renamedOut)
}

func TestObjectNamePredicateRejectsNilObjects(t *testing.T) {
	p := ObjectNamePredicate{AllowedObjectName: "allowed"}
	allowed := namedObject("allowed")

	assert.False(t, p.Create(event.CreateEvent{Object: nil}))
	assert.False(t, p.Delete(event.DeleteEvent{Object: nil}))
	assert.False(t, p.Generic(event.GenericEvent{Object: nil}))
	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: allowed, ObjectNew: nil}))
	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: nil, ObjectNew: allowed}))
	assert.False(t, p.Update(event.UpdateEvent{}))
}

// The shared predicate values below are what controllers across the autoscaler,
// instrumentor, odiglet and scheduler wire into their event filters. Binding one
// to the wrong object name is silent at runtime - the controller just stops
// reconciling - so the expected names are spelled out here as literals rather
// than taken from the same constants the production code uses.
func TestSharedObjectNamePredicatesAreBoundToTheirObject(t *testing.T) {
	tests := []struct {
		name      string
		predicate ObjectNamePredicate
		wantName  string
	}{
		{name: "OdigosConfigMapPredicate", predicate: OdigosConfigMapPredicate, wantName: "odigos-configuration"},
		{name: "OdigosEffectiveConfigMapPredicate", predicate: OdigosEffectiveConfigMapPredicate, wantName: "effective-config"},
		{name: "OdigosCollectorsGroupNodePredicate", predicate: OdigosCollectorsGroupNodePredicate, wantName: "odigos-data-collection"},
		{name: "OdigosCollectorsGroupClusterPredicate", predicate: OdigosCollectorsGroupClusterPredicate, wantName: "odigos-gateway"},
		{name: "OdigosURLTemplatizationProcessorPredicate", predicate: OdigosURLTemplatizationProcessorPredicate, wantName: "odigos-url-templatization"},
		{name: "OdigosPiiMaskingProcessorPredicate", predicate: OdigosPiiMaskingProcessorPredicate, wantName: "odigos-pii-masking"},
		{name: "OdigosSQLQueryProcessorPredicate", predicate: OdigosSQLQueryProcessorPredicate, wantName: "odigos-sql-query"},
		{name: "OdigosProSecretPredicate", predicate: OdigosProSecretPredicate, wantName: "odigos-pro"},
		{name: "OdigosDeploymentConfigMapPredicate", predicate: OdigosDeploymentConfigMapPredicate, wantName: "odigos-deployment"},
		{name: "OdigosRemoteConfigMapPredicate", predicate: OdigosRemoteConfigMapPredicate, wantName: "odigos-remote-config"},
		{name: "OdigosLocalUiConfigMapPredicate", predicate: OdigosLocalUiConfigMapPredicate, wantName: "odigos-local-ui-config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantName, tt.predicate.AllowedObjectName)
			assert.True(t, tt.predicate.Create(event.CreateEvent{Object: namedObject(tt.wantName)}))
		})
	}

	// The node and cluster collectors group predicates are the pair most at risk
	// of being swapped, and both watch the same CollectorsGroup kind.
	assert.False(t, OdigosCollectorsGroupNodePredicate.Create(event.CreateEvent{Object: namedObject("odigos-gateway")}))
	assert.False(t, OdigosCollectorsGroupClusterPredicate.Create(event.CreateEvent{Object: namedObject("odigos-data-collection")}))
}
