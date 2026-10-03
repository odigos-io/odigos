package predicate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
)

// urlTemplatizationAction builds an Action that either carries a URLTemplatization
// config or does not, at the given generation.
func urlTemplatizationAction(hasConfig bool, disabled bool, generation int64) *odigosv1.Action {
	action := &odigosv1.Action{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "url-templatization",
			Namespace:  "odigos-system",
			Generation: generation,
		},
		Spec: odigosv1.ActionSpec{Disabled: disabled},
	}
	if hasConfig {
		action.Spec.URLTemplatization = &actions.URLTemplatizationConfig{}
	}
	return action
}

// An action only counts as a URL-templatization action when it carries the config
// and is not disabled, so a disabled action is invisible to this filter.
func TestURLTemplatizationActionPredicateCreateAndDelete(t *testing.T) {
	p := URLTemplatizationActionPredicate{}

	tests := []struct {
		name      string
		hasConfig bool
		disabled  bool
		want      bool
	}{
		{name: "enabled action with the config", hasConfig: true, disabled: false, want: true},
		{name: "disabled action with the config", hasConfig: true, disabled: true, want: false},
		{name: "enabled action without the config", hasConfig: false, disabled: false, want: false},
		{name: "disabled action without the config", hasConfig: false, disabled: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := urlTemplatizationAction(tt.hasConfig, tt.disabled, 1)

			assert.Equal(t, tt.want, p.Create(event.CreateEvent{Object: action}), "create")
			assert.Equal(t, tt.want, p.Delete(event.DeleteEvent{Object: action}), "delete")
		})
	}
}

func TestURLTemplatizationActionPredicateUpdate(t *testing.T) {
	p := URLTemplatizationActionPredicate{}

	tests := []struct {
		name         string
		oldHasConfig bool
		oldDisabled  bool
		oldGen       int64
		newHasConfig bool
		newDisabled  bool
		newGen       int64
		want         bool
	}{
		{
			name:         "config was added",
			oldHasConfig: false, oldDisabled: false, oldGen: 1,
			newHasConfig: true, newDisabled: false, newGen: 2,
			want: true,
		},
		{
			name:         "config was removed",
			oldHasConfig: true, oldDisabled: false, oldGen: 1,
			newHasConfig: false, newDisabled: false, newGen: 2,
			want: true,
		},
		{
			name:         "action was disabled",
			oldHasConfig: true, oldDisabled: false, oldGen: 1,
			newHasConfig: true, newDisabled: true, newGen: 2,
			want: true,
		},
		{
			name:         "action was re-enabled",
			oldHasConfig: true, oldDisabled: true, oldGen: 1,
			newHasConfig: true, newDisabled: false, newGen: 2,
			want: true,
		},
		{
			// a spec edit shows up as a generation bump.
			name:         "config changed while enabled",
			oldHasConfig: true, oldDisabled: false, oldGen: 1,
			newHasConfig: true, newDisabled: false, newGen: 2,
			want: true,
		},
		{
			// a status-only write leaves the generation alone and must be filtered
			// out, or every status update would re-trigger a config rebuild.
			name:         "nothing relevant changed",
			oldHasConfig: true, oldDisabled: false, oldGen: 3,
			newHasConfig: true, newDisabled: false, newGen: 3,
			want: false,
		},
		{
			name:         "generation bump on an action without the config",
			oldHasConfig: false, oldDisabled: false, oldGen: 1,
			newHasConfig: false, newDisabled: false, newGen: 2,
			want: false,
		},
		{
			// both sides are invisible to this filter, so the disabled flag flipping
			// on an action that has no config is not interesting either.
			name:         "disabled flag flips on an action without the config",
			oldHasConfig: false, oldDisabled: false, oldGen: 1,
			newHasConfig: false, newDisabled: true, newGen: 2,
			want: false,
		},
		{
			name:         "config added to an action that stays disabled",
			oldHasConfig: false, oldDisabled: true, oldGen: 1,
			newHasConfig: true, newDisabled: true, newGen: 2,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.Update(event.UpdateEvent{
				ObjectOld: urlTemplatizationAction(tt.oldHasConfig, tt.oldDisabled, tt.oldGen),
				ObjectNew: urlTemplatizationAction(tt.newHasConfig, tt.newDisabled, tt.newGen),
			})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestURLTemplatizationActionPredicateRejectsUnusableEvents(t *testing.T) {
	p := URLTemplatizationActionPredicate{}
	action := urlTemplatizationAction(true, false, 1)
	notAnAction := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "url-templatization"}}

	assert.False(t, p.Create(event.CreateEvent{Object: nil}))
	assert.False(t, p.Create(event.CreateEvent{Object: notAnAction}))
	assert.False(t, p.Delete(event.DeleteEvent{Object: nil}))
	assert.False(t, p.Delete(event.DeleteEvent{Object: notAnAction}))
	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: nil, ObjectNew: action}))
	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: action, ObjectNew: nil}))
	assert.False(t, p.Update(event.UpdateEvent{}))
	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: notAnAction, ObjectNew: action}))
	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: action, ObjectNew: notAnAction}))
}

func TestURLTemplatizationActionPredicateBlocksGeneric(t *testing.T) {
	p := URLTemplatizationActionPredicate{}

	assert.False(t, p.Generic(event.GenericEvent{Object: urlTemplatizationAction(true, false, 1)}))
}
