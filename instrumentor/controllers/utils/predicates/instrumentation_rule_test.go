package predicates

import (
	"reflect"
	"testing"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// The production file asserts this for its siblings but not for this predicate, which is
// registered as an event filter in agentenabled/manager.go.
var _ predicate.Predicate = AgentInjectionRelevantRulesPredicate{}

// agentInjectionRelevantSpecFields lists the InstrumentationRuleSpec fields that
// isRuleRelevantForAgentInjection must treat as relevant, and
// agentInjectionIrrelevantSpecFields the ones it must ignore. Together they must cover
// every field of the spec: a new rule config that is not classified here fails
// TestInstrumentationRuleSpecFieldsAreAllClassified, which is the only thing that would
// otherwise flag a new agent-affecting config that was never added to the predicate
// (the agentenabled controller would then never reconcile on it).
var agentInjectionRelevantSpecFields = []string{
	"OtelDistros",
	"HeadersCollection",
	"TraceConfig",
	"PayloadCollection",
	"TraceVerbosity",
	"CustomInstrumentations",
	"CodeAttributes",
	"EbpfLogCapture",
	"AgentDiagnostics",
	"NetworkMetrics",
}

var agentInjectionIrrelevantSpecFields = []string{
	"RuleName",
	"Notes",
	"Disabled",
	"Scopes",
	"InstrumentationLibraries",
	"HeadSamplingFallbackFraction",
}

func specFieldNames() []string {
	specType := reflect.TypeOf(odigosv1alpha1.InstrumentationRuleSpec{})
	names := make([]string, 0, specType.NumField())
	for i := 0; i < specType.NumField(); i++ {
		names = append(names, specType.Field(i).Name)
	}
	return names
}

// specWithOnlyField returns a spec in which exactly one field is set to a non-zero value.
func specWithOnlyField(t *testing.T, fieldName string) *odigosv1alpha1.InstrumentationRuleSpec {
	t.Helper()

	spec := &odigosv1alpha1.InstrumentationRuleSpec{}
	field := reflect.ValueOf(spec).Elem().FieldByName(fieldName)
	require.True(t, field.IsValid(), "InstrumentationRuleSpec has no field %s", fieldName)

	switch field.Kind() {
	case reflect.Ptr:
		field.Set(reflect.New(field.Type().Elem()))
	case reflect.Bool:
		field.SetBool(true)
	case reflect.String:
		field.SetString("some-value")
	default:
		t.Fatalf("field %s has unsupported kind %s, extend specWithOnlyField", fieldName, field.Kind())
	}

	return spec
}

func TestInstrumentationRuleSpecFieldsAreAllClassified(t *testing.T) {
	classification := map[string]int{}
	for _, name := range agentInjectionRelevantSpecFields {
		classification[name]++
	}
	for _, name := range agentInjectionIrrelevantSpecFields {
		classification[name]++
	}

	for _, name := range specFieldNames() {
		assert.Equal(t, 1, classification[name],
			"InstrumentationRuleSpec.%s must be listed in exactly one of agentInjectionRelevantSpecFields / "+
				"agentInjectionIrrelevantSpecFields; if it affects agent injection it also has to be added to "+
				"isRuleRelevantForAgentInjection", name)
		delete(classification, name)
	}

	assert.Empty(t, classification, "these names are classified but are not fields of InstrumentationRuleSpec")
}

func TestIsRuleRelevantForAgentInjection_PerSpecField(t *testing.T) {
	relevant := map[string]bool{}
	for _, name := range agentInjectionRelevantSpecFields {
		relevant[name] = true
	}

	assert.False(t, isRuleRelevantForAgentInjection(&odigosv1alpha1.InstrumentationRuleSpec{}),
		"an empty rule spec configures nothing, so it must not trigger agent injection reconciliation")

	for _, fieldName := range specFieldNames() {
		t.Run(fieldName, func(t *testing.T) {
			assert.Equal(t, relevant[fieldName], isRuleRelevantForAgentInjection(specWithOnlyField(t, fieldName)))
		})
	}
}

func TestIsRuleRelevantForAgentInjection_DisabledRuleIsStillRelevant(t *testing.T) {
	// The predicate deliberately ignores Disabled: disabling a rule that used to configure
	// agents must still reach the controller so the agents can be reconfigured.
	spec := specWithOnlyField(t, "OtelDistros")
	spec.Disabled = true

	assert.True(t, isRuleRelevantForAgentInjection(spec))
}

func agentInjectionRule(spec *odigosv1alpha1.InstrumentationRuleSpec) *odigosv1alpha1.InstrumentationRule {
	return &odigosv1alpha1.InstrumentationRule{
		ObjectMeta: metav1.ObjectMeta{Name: "some-rule", Namespace: "odigos-system"},
		Spec:       *spec,
	}
}

func TestAgentInjectionRelevantRulesPredicate_Create(t *testing.T) {
	tests := []struct {
		name  string
		event event.CreateEvent
		want  bool
	}{
		{
			name:  "nil object is ignored",
			event: event.CreateEvent{Object: nil},
			want:  false,
		},
		{
			name:  "object that is not an InstrumentationRule is ignored",
			event: event.CreateEvent{Object: &odigosv1alpha1.Source{ObjectMeta: metav1.ObjectMeta{Name: "src"}}},
			want:  false,
		},
		{
			name:  "rule that configures nothing relevant is ignored",
			event: event.CreateEvent{Object: agentInjectionRule(specWithOnlyField(t, "HeadSamplingFallbackFraction"))},
			want:  false,
		},
		{
			name:  "rule that configures agents is reconciled",
			event: event.CreateEvent{Object: agentInjectionRule(specWithOnlyField(t, "PayloadCollection"))},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, AgentInjectionRelevantRulesPredicate{}.Create(tt.event))
		})
	}
}

func TestAgentInjectionRelevantRulesPredicate_Update(t *testing.T) {
	relevantSpec := specWithOnlyField(t, "CodeAttributes")
	irrelevantSpec := specWithOnlyField(t, "Notes")

	tests := []struct {
		name  string
		event event.UpdateEvent
		want  bool
	}{
		{
			name:  "nil old object is ignored",
			event: event.UpdateEvent{ObjectOld: nil, ObjectNew: agentInjectionRule(relevantSpec)},
			want:  false,
		},
		{
			name:  "nil new object is ignored",
			event: event.UpdateEvent{ObjectOld: agentInjectionRule(relevantSpec), ObjectNew: nil},
			want:  false,
		},
		{
			name: "old object that is not an InstrumentationRule is ignored",
			event: event.UpdateEvent{
				ObjectOld: &odigosv1alpha1.Source{ObjectMeta: metav1.ObjectMeta{Name: "src"}},
				ObjectNew: agentInjectionRule(relevantSpec),
			},
			want: false,
		},
		{
			name: "new object that is not an InstrumentationRule is ignored",
			event: event.UpdateEvent{
				ObjectOld: agentInjectionRule(relevantSpec),
				ObjectNew: &odigosv1alpha1.Source{ObjectMeta: metav1.ObjectMeta{Name: "src"}},
			},
			want: false,
		},
		{
			name: "neither revision configures agents",
			event: event.UpdateEvent{
				ObjectOld: agentInjectionRule(irrelevantSpec),
				ObjectNew: agentInjectionRule(irrelevantSpec),
			},
			want: false,
		},
		{
			// The relevant config was removed, so the agents still have to be reconfigured.
			name: "only the old revision configures agents",
			event: event.UpdateEvent{
				ObjectOld: agentInjectionRule(relevantSpec),
				ObjectNew: agentInjectionRule(irrelevantSpec),
			},
			want: true,
		},
		{
			name: "only the new revision configures agents",
			event: event.UpdateEvent{
				ObjectOld: agentInjectionRule(irrelevantSpec),
				ObjectNew: agentInjectionRule(relevantSpec),
			},
			want: true,
		},
		{
			name: "both revisions configure agents",
			event: event.UpdateEvent{
				ObjectOld: agentInjectionRule(relevantSpec),
				ObjectNew: agentInjectionRule(specWithOnlyField(t, "TraceVerbosity")),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, AgentInjectionRelevantRulesPredicate{}.Update(tt.event))
		})
	}
}

func TestAgentInjectionRelevantRulesPredicate_Delete(t *testing.T) {
	tests := []struct {
		name  string
		event event.DeleteEvent
		want  bool
	}{
		{
			name:  "nil object is ignored",
			event: event.DeleteEvent{Object: nil},
			want:  false,
		},
		{
			name:  "object that is not an InstrumentationRule is ignored",
			event: event.DeleteEvent{Object: &odigosv1alpha1.Source{ObjectMeta: metav1.ObjectMeta{Name: "src"}}},
			want:  false,
		},
		{
			name:  "deleting a rule that configures nothing relevant is ignored",
			event: event.DeleteEvent{Object: agentInjectionRule(specWithOnlyField(t, "Scopes"))},
			want:  false,
		},
		{
			// Deleting a rule that configured agents must reach the controller so the
			// configuration it applied is rolled back.
			name:  "deleting a rule that configures agents is reconciled",
			event: event.DeleteEvent{Object: agentInjectionRule(specWithOnlyField(t, "EbpfLogCapture"))},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, AgentInjectionRelevantRulesPredicate{}.Delete(tt.event))
		})
	}
}

func TestAgentInjectionRelevantRulesPredicate_GenericIsIgnored(t *testing.T) {
	rule := agentInjectionRule(specWithOnlyField(t, "OtelDistros"))

	assert.False(t, AgentInjectionRelevantRulesPredicate{}.Generic(event.GenericEvent{Object: rule}))
}
