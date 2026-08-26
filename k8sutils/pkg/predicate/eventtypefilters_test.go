package predicate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	cr_predicate "sigs.k8s.io/controller-runtime/pkg/predicate"
)

// Several controllers wire these predicates by value rather than by pointer
// (e.g. `WithEventFilter(odigospredicate.ExistencePredicate{})`), so switching any
// of them to pointer receivers would break those call sites in other modules.
var (
	_ cr_predicate.Predicate = CreationPredicate{}
	_ cr_predicate.Predicate = DeletionPredicate{}
	_ cr_predicate.Predicate = ExistencePredicate{}
	_ cr_predicate.Predicate = OnlyUpdatesPredicate{}
	_ cr_predicate.Predicate = ObjectNamePredicate{}
)

func predicateTestObject() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "some-object", Namespace: "odigos-system"},
	}
}

// TestEventTypeFiltersAllowMatrix pins which event types each of the four
// event-type-only predicates lets through. These decide whether a controller
// reconciles at all, and a wrong value here is silent: the controller simply
// never runs for that event.
func TestEventTypeFiltersAllowMatrix(t *testing.T) {
	tests := []struct {
		name      string
		predicate cr_predicate.Predicate
		create    bool
		update    bool
		delete    bool
		generic   bool
	}{
		{
			name:      "CreationPredicate",
			predicate: CreationPredicate{},
			create:    true,
			update:    false,
			delete:    false,
			generic:   false,
		},
		{
			name:      "DeletionPredicate",
			predicate: DeletionPredicate{},
			create:    false,
			update:    false,
			delete:    true,
			generic:   false,
		},
		{
			name:      "ExistencePredicate",
			predicate: ExistencePredicate{},
			create:    true,
			update:    false,
			delete:    true,
			generic:   false,
		},
		{
			name:      "OnlyUpdatesPredicate",
			predicate: OnlyUpdatesPredicate{},
			create:    false,
			update:    true,
			delete:    false,
			generic:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := predicateTestObject()

			assert.Equal(t, tt.create, tt.predicate.Create(event.CreateEvent{Object: obj}), "create")
			assert.Equal(t, tt.update, tt.predicate.Update(event.UpdateEvent{ObjectOld: obj, ObjectNew: obj}), "update")
			assert.Equal(t, tt.delete, tt.predicate.Delete(event.DeleteEvent{Object: obj}), "delete")
			assert.Equal(t, tt.generic, tt.predicate.Generic(event.GenericEvent{Object: obj}), "generic")
		})
	}
}

// These predicates only look at the event type, so the object carried by the
// event must not change the verdict.
func TestEventTypeFiltersIgnoreTheObject(t *testing.T) {
	predicates := map[string]cr_predicate.Predicate{
		"CreationPredicate":    CreationPredicate{},
		"DeletionPredicate":    DeletionPredicate{},
		"ExistencePredicate":   ExistencePredicate{},
		"OnlyUpdatesPredicate": OnlyUpdatesPredicate{},
	}

	for name, p := range predicates {
		t.Run(name, func(t *testing.T) {
			obj := predicateTestObject()
			other := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "another-object"}}

			assert.Equal(t, p.Create(event.CreateEvent{Object: obj}), p.Create(event.CreateEvent{Object: other}))
			assert.Equal(t,
				p.Update(event.UpdateEvent{ObjectOld: obj, ObjectNew: obj}),
				p.Update(event.UpdateEvent{ObjectOld: other, ObjectNew: other}))
			assert.Equal(t, p.Delete(event.DeleteEvent{Object: obj}), p.Delete(event.DeleteEvent{Object: other}))
			assert.Equal(t, p.Generic(event.GenericEvent{Object: obj}), p.Generic(event.GenericEvent{Object: other}))
		})
	}
}
