package predicates

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func icWithRuntimeDetails(details ...odigosv1.RuntimeDetailsByContainer) *odigosv1.InstrumentationConfig {
	return &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "deployment-testapp", Namespace: "default"},
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: details,
		},
	}
}

func containerDetails(containerName string, language common.ProgrammingLanguage) odigosv1.RuntimeDetailsByContainer {
	return odigosv1.RuntimeDetailsByContainer{
		ContainerName:  containerName,
		Language:       language,
		RuntimeVersion: "1.22.0",
	}
}

func ldPreloadEnvVar() odigosv1.EnvVar {
	return odigosv1.EnvVar{Name: consts.LdPreloadEnvVarName, Value: "/odigos/loader.so"}
}

func annotatedObject(annotations map[string]string) *corev1.Pod {
	return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "testapp-abc", Namespace: "default", Annotations: annotations}}
}

// The predicate is registered by value in agentenabled/manager.go, so the exported
// package-level vars must behave identically to the zero value.
func TestPredicateVarsAreUsableZeroValues(t *testing.T) {
	assert.Equal(t, RuntimeDetailsChangedPredicate{}, InstrumentationConfigRuntimeDetailsChangedPredicate)
	assert.Equal(t, ContainerOverridesChangedPredicate{}, InstrumentationConfigContainerOverridesChangedPredicate)
}

func TestRuntimeDetailsChangedPredicate_Create(t *testing.T) {
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
			name:  "object that is not an InstrumentationConfig is ignored",
			event: event.CreateEvent{Object: &odigosv1.Source{ObjectMeta: metav1.ObjectMeta{Name: "src"}}},
			want:  false,
		},
		{
			name:  "InstrumentationConfig without runtime details is ignored",
			event: event.CreateEvent{Object: icWithRuntimeDetails()},
			want:  false,
		},
		{
			name:  "InstrumentationConfig with runtime details is reconciled",
			event: event.CreateEvent{Object: icWithRuntimeDetails(containerDetails("app", common.GoProgrammingLanguage))},
			want:  true,
		},
		{
			name: "InstrumentationConfig with several containers is reconciled",
			event: event.CreateEvent{Object: icWithRuntimeDetails(
				containerDetails("app", common.GoProgrammingLanguage),
				containerDetails("sidecar", common.JavaProgrammingLanguage),
			)},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, RuntimeDetailsChangedPredicate{}.Create(tt.event))
		})
	}
}

func TestRuntimeDetailsChangedPredicate_Update(t *testing.T) {
	goDetails := containerDetails("app", common.GoProgrammingLanguage)

	withLanguage := func(details odigosv1.RuntimeDetailsByContainer, language common.ProgrammingLanguage) odigosv1.RuntimeDetailsByContainer {
		details.Language = language
		return details
	}
	withRuntimeVersion := func(details odigosv1.RuntimeDetailsByContainer, version string) odigosv1.RuntimeDetailsByContainer {
		details.RuntimeVersion = version
		return details
	}
	withOtherAgents := func(details odigosv1.RuntimeDetailsByContainer, names ...string) odigosv1.RuntimeDetailsByContainer {
		details.OtherAgents = nil
		for _, name := range names {
			details.OtherAgents = append(details.OtherAgents, odigosv1.OtherAgent{Name: name})
		}
		return details
	}
	withEnvVars := func(details odigosv1.RuntimeDetailsByContainer, envVars ...odigosv1.EnvVar) odigosv1.RuntimeDetailsByContainer {
		details.EnvVars = envVars
		return details
	}

	tests := []struct {
		name  string
		event event.UpdateEvent
		want  bool
	}{
		{
			name:  "nil old object is ignored",
			event: event.UpdateEvent{ObjectOld: nil, ObjectNew: icWithRuntimeDetails(goDetails)},
			want:  false,
		},
		{
			name:  "nil new object is ignored",
			event: event.UpdateEvent{ObjectOld: icWithRuntimeDetails(goDetails), ObjectNew: nil},
			want:  false,
		},
		{
			name: "old object that is not an InstrumentationConfig is ignored",
			event: event.UpdateEvent{
				ObjectOld: &odigosv1.Source{ObjectMeta: metav1.ObjectMeta{Name: "src"}},
				ObjectNew: icWithRuntimeDetails(goDetails),
			},
			want: false,
		},
		{
			name: "new object that is not an InstrumentationConfig is ignored",
			event: event.UpdateEvent{
				ObjectOld: icWithRuntimeDetails(goDetails),
				ObjectNew: &odigosv1.Source{ObjectMeta: metav1.ObjectMeta{Name: "src"}},
			},
			want: false,
		},
		{
			name: "runtime details detected for the first time",
			event: event.UpdateEvent{
				ObjectOld: icWithRuntimeDetails(),
				ObjectNew: icWithRuntimeDetails(goDetails),
			},
			want: true,
		},
		{
			name: "a container was removed from the runtime details",
			event: event.UpdateEvent{
				ObjectOld: icWithRuntimeDetails(goDetails, containerDetails("sidecar", common.JavaProgrammingLanguage)),
				ObjectNew: icWithRuntimeDetails(goDetails),
			},
			want: true,
		},
		{
			name: "identical runtime details are not reconciled",
			event: event.UpdateEvent{
				ObjectOld: icWithRuntimeDetails(goDetails),
				ObjectNew: icWithRuntimeDetails(goDetails),
			},
			want: false,
		},
		{
			name: "language changed",
			event: event.UpdateEvent{
				ObjectOld: icWithRuntimeDetails(goDetails),
				ObjectNew: icWithRuntimeDetails(withLanguage(goDetails, common.JavaProgrammingLanguage)),
			},
			want: true,
		},
		{
			name: "runtime version changed",
			event: event.UpdateEvent{
				ObjectOld: icWithRuntimeDetails(goDetails),
				ObjectNew: icWithRuntimeDetails(withRuntimeVersion(goDetails, "1.23.0")),
			},
			want: true,
		},
		{
			name: "other agent detected",
			event: event.UpdateEvent{
				ObjectOld: icWithRuntimeDetails(goDetails),
				ObjectNew: icWithRuntimeDetails(withOtherAgents(goDetails, "datadog")),
			},
			want: true,
		},
		{
			name: "other agent renamed",
			event: event.UpdateEvent{
				ObjectOld: icWithRuntimeDetails(withOtherAgents(goDetails, "datadog")),
				ObjectNew: icWithRuntimeDetails(withOtherAgents(goDetails, "newrelic")),
			},
			want: true,
		},
		{
			name: "a second other agent detected",
			event: event.UpdateEvent{
				ObjectOld: icWithRuntimeDetails(withOtherAgents(goDetails, "datadog")),
				ObjectNew: icWithRuntimeDetails(withOtherAgents(goDetails, "datadog", "newrelic")),
			},
			want: true,
		},
		{
			name: "identical other agents are not reconciled",
			event: event.UpdateEvent{
				ObjectOld: icWithRuntimeDetails(withOtherAgents(goDetails, "datadog", "newrelic")),
				ObjectNew: icWithRuntimeDetails(withOtherAgents(goDetails, "datadog", "newrelic")),
			},
			want: false,
		},
		{
			name: "LD_PRELOAD appeared in the container env",
			event: event.UpdateEvent{
				ObjectOld: icWithRuntimeDetails(goDetails),
				ObjectNew: icWithRuntimeDetails(withEnvVars(goDetails, ldPreloadEnvVar())),
			},
			want: true,
		},
		{
			name: "LD_PRELOAD disappeared from the container env",
			event: event.UpdateEvent{
				ObjectOld: icWithRuntimeDetails(withEnvVars(goDetails, ldPreloadEnvVar())),
				ObjectNew: icWithRuntimeDetails(goDetails),
			},
			want: true,
		},
		{
			name: "an LD_PRELOAD value change is not reconciled",
			event: event.UpdateEvent{
				ObjectOld: icWithRuntimeDetails(withEnvVars(goDetails, odigosv1.EnvVar{Name: consts.LdPreloadEnvVarName, Value: "/a.so"})),
				ObjectNew: icWithRuntimeDetails(withEnvVars(goDetails, odigosv1.EnvVar{Name: consts.LdPreloadEnvVarName, Value: "/b.so"})),
			},
			want: false,
		},
		{
			name: "an unrelated env var change is not reconciled",
			event: event.UpdateEvent{
				ObjectOld: icWithRuntimeDetails(withEnvVars(goDetails, odigosv1.EnvVar{Name: "PYTHONPATH", Value: "/a"})),
				ObjectNew: icWithRuntimeDetails(withEnvVars(goDetails, odigosv1.EnvVar{Name: "PYTHONPATH", Value: "/b"})),
			},
			want: false,
		},
		{
			// The predicate compares containers pairwise by index, so a change that is only
			// visible past the first container must still be detected.
			name: "only the second container changed",
			event: event.UpdateEvent{
				ObjectOld: icWithRuntimeDetails(goDetails, containerDetails("sidecar", common.JavaProgrammingLanguage)),
				ObjectNew: icWithRuntimeDetails(goDetails, containerDetails("sidecar", common.PythonProgrammingLanguage)),
			},
			want: true,
		},
		{
			name: "only the second container gained LD_PRELOAD",
			event: event.UpdateEvent{
				ObjectOld: icWithRuntimeDetails(goDetails, containerDetails("sidecar", common.JavaProgrammingLanguage)),
				ObjectNew: icWithRuntimeDetails(goDetails, withEnvVars(containerDetails("sidecar", common.JavaProgrammingLanguage), ldPreloadEnvVar())),
			},
			want: true,
		},
		{
			name: "several unchanged containers are not reconciled",
			event: event.UpdateEvent{
				ObjectOld: icWithRuntimeDetails(goDetails, containerDetails("sidecar", common.JavaProgrammingLanguage)),
				ObjectNew: icWithRuntimeDetails(goDetails, containerDetails("sidecar", common.JavaProgrammingLanguage)),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, RuntimeDetailsChangedPredicate{}.Update(tt.event))
		})
	}
}

func TestRuntimeDetailsChangedPredicate_DeleteAndGenericAreIgnored(t *testing.T) {
	ic := icWithRuntimeDetails(containerDetails("app", common.GoProgrammingLanguage))

	assert.False(t, RuntimeDetailsChangedPredicate{}.Delete(event.DeleteEvent{Object: ic}))
	assert.False(t, RuntimeDetailsChangedPredicate{}.Generic(event.GenericEvent{Object: ic}))
}

func TestRecoveredFromRollbackAtChangedPredicate_Create(t *testing.T) {
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
			name:  "object without annotations is ignored",
			event: event.CreateEvent{Object: annotatedObject(nil)},
			want:  false,
		},
		{
			name:  "empty recovery annotation is ignored",
			event: event.CreateEvent{Object: annotatedObject(map[string]string{k8sconsts.RollbackRecoveryAtAnnotation: ""})},
			want:  false,
		},
		{
			name:  "an unrelated annotation is ignored",
			event: event.CreateEvent{Object: annotatedObject(map[string]string{"odigos.io/some-other-annotation": "2026-09-05T10:00:00Z"})},
			want:  false,
		},
		{
			name:  "recovery annotation is set",
			event: event.CreateEvent{Object: annotatedObject(map[string]string{k8sconsts.RollbackRecoveryAtAnnotation: "2026-09-05T10:00:00Z"})},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, RecoveredFromRollbackAtChangedPredicate{}.Create(tt.event))
		})
	}
}

func TestRecoveredFromRollbackAtChangedPredicate_Update(t *testing.T) {
	recoveryAt := func(value string) *corev1.Pod {
		return annotatedObject(map[string]string{k8sconsts.RollbackRecoveryAtAnnotation: value})
	}

	tests := []struct {
		name  string
		event event.UpdateEvent
		want  bool
	}{
		{
			name:  "nil old object is ignored",
			event: event.UpdateEvent{ObjectOld: nil, ObjectNew: recoveryAt("2026-09-05T10:00:00Z")},
			want:  false,
		},
		{
			name:  "nil new object is ignored",
			event: event.UpdateEvent{ObjectOld: recoveryAt("2026-09-05T10:00:00Z"), ObjectNew: nil},
			want:  false,
		},
		{
			name:  "annotation absent on both sides is ignored",
			event: event.UpdateEvent{ObjectOld: annotatedObject(nil), ObjectNew: annotatedObject(nil)},
			want:  false,
		},
		{
			name:  "unchanged annotation is ignored",
			event: event.UpdateEvent{ObjectOld: recoveryAt("2026-09-05T10:00:00Z"), ObjectNew: recoveryAt("2026-09-05T10:00:00Z")},
			want:  false,
		},
		{
			name: "an unrelated annotation change is ignored",
			event: event.UpdateEvent{
				ObjectOld: annotatedObject(map[string]string{"odigos.io/some-other-annotation": "before"}),
				ObjectNew: annotatedObject(map[string]string{"odigos.io/some-other-annotation": "after"}),
			},
			want: false,
		},
		{
			name:  "recovery requested",
			event: event.UpdateEvent{ObjectOld: annotatedObject(nil), ObjectNew: recoveryAt("2026-09-05T10:00:00Z")},
			want:  true,
		},
		{
			name:  "recovery annotation removed",
			event: event.UpdateEvent{ObjectOld: recoveryAt("2026-09-05T10:00:00Z"), ObjectNew: annotatedObject(nil)},
			want:  true,
		},
		{
			name:  "recovery requested again with a new timestamp",
			event: event.UpdateEvent{ObjectOld: recoveryAt("2026-09-05T10:00:00Z"), ObjectNew: recoveryAt("2026-09-05T11:00:00Z")},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, RecoveredFromRollbackAtChangedPredicate{}.Update(tt.event))
		})
	}
}

func TestRecoveredFromRollbackAtChangedPredicate_DeleteAndGenericAreIgnored(t *testing.T) {
	object := annotatedObject(map[string]string{k8sconsts.RollbackRecoveryAtAnnotation: "2026-09-05T10:00:00Z"})

	assert.False(t, RecoveredFromRollbackAtChangedPredicate{}.Delete(event.DeleteEvent{Object: object}))
	assert.False(t, RecoveredFromRollbackAtChangedPredicate{}.Generic(event.GenericEvent{Object: object}))
}

func TestContainerOverridesChangedPredicate_Create(t *testing.T) {
	// Create is unconditional: the agentenabled controller must see every new
	// InstrumentationConfig regardless of its overrides hash.
	assert.True(t, ContainerOverridesChangedPredicate{}.Create(event.CreateEvent{Object: icWithOverridesHash("")}))
	assert.True(t, ContainerOverridesChangedPredicate{}.Create(event.CreateEvent{Object: icWithOverridesHash("abc123")}))
	assert.True(t, ContainerOverridesChangedPredicate{}.Create(event.CreateEvent{Object: &odigosv1.Source{}}))
}

func TestContainerOverridesChangedPredicate_Update(t *testing.T) {
	tests := []struct {
		name  string
		event event.UpdateEvent
		want  bool
	}{
		{
			name:  "nil old object is ignored",
			event: event.UpdateEvent{ObjectOld: nil, ObjectNew: icWithOverridesHash("abc123")},
			want:  false,
		},
		{
			name:  "nil new object is ignored",
			event: event.UpdateEvent{ObjectOld: icWithOverridesHash("abc123"), ObjectNew: nil},
			want:  false,
		},
		{
			name: "old object that is not an InstrumentationConfig is ignored",
			event: event.UpdateEvent{
				ObjectOld: &odigosv1.Source{ObjectMeta: metav1.ObjectMeta{Name: "src"}},
				ObjectNew: icWithOverridesHash("abc123"),
			},
			want: false,
		},
		{
			name: "new object that is not an InstrumentationConfig is ignored",
			event: event.UpdateEvent{
				ObjectOld: icWithOverridesHash("abc123"),
				ObjectNew: &odigosv1.Source{ObjectMeta: metav1.ObjectMeta{Name: "src"}},
			},
			want: false,
		},
		{
			name: "unchanged overrides hash is ignored",
			event: event.UpdateEvent{
				ObjectOld: icWithOverridesHash("abc123"),
				ObjectNew: icWithOverridesHash("abc123"),
			},
			want: false,
		},
		{
			name: "overrides hash changed",
			event: event.UpdateEvent{
				ObjectOld: icWithOverridesHash("abc123"),
				ObjectNew: icWithOverridesHash("def456"),
			},
			want: true,
		},
		{
			name: "overrides set for the first time",
			event: event.UpdateEvent{
				ObjectOld: icWithOverridesHash(""),
				ObjectNew: icWithOverridesHash("abc123"),
			},
			want: true,
		},
		{
			// Runtime details live in the status, so a runtime-details-only change must not
			// look like an overrides change to this predicate.
			name: "a runtime details change is not an overrides change",
			event: event.UpdateEvent{
				ObjectOld: icWithOverridesHash("abc123"),
				ObjectNew: func() *odigosv1.InstrumentationConfig {
					ic := icWithOverridesHash("abc123")
					ic.Status.RuntimeDetailsByContainer = []odigosv1.RuntimeDetailsByContainer{containerDetails("app", common.GoProgrammingLanguage)}
					return ic
				}(),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ContainerOverridesChangedPredicate{}.Update(tt.event))
		})
	}
}

func TestContainerOverridesChangedPredicate_DeleteAndGenericAreIgnored(t *testing.T) {
	ic := icWithOverridesHash("abc123")

	assert.False(t, ContainerOverridesChangedPredicate{}.Delete(event.DeleteEvent{Object: ic}))
	assert.False(t, ContainerOverridesChangedPredicate{}.Generic(event.GenericEvent{Object: ic}))
}

func icWithOverridesHash(hash string) *odigosv1.InstrumentationConfig {
	ic := icWithRuntimeDetails()
	ic.Spec.ContainerOverridesHash = hash
	return ic
}
