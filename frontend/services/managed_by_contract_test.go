package services

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/require"
)

// mbRecognizedValues pairs every odigos.io/managed-by value the mapper recognizes with
// the enum value it must produce. Being a map literal keyed by the constants, it also
// collapses if two of them ever hold the same string.
var mbRecognizedValues = map[string]model.ManagedBy{
	k8sconsts.OdigosProfilesManagedByValue:          model.ManagedByProfile,
	k8sconsts.OdigosUIManagedByValue:                model.ManagedByOdigosUI,
	k8sconsts.OdigosInterrogationLoopManagedByValue: model.ManagedByInterrogationLoop,
}

// The label key and its values are persisted on cluster objects by one component and
// read by another: the scheduler's profile reconciler stamps "profile" and deletes
// resources by matching that exact value, while the UI stamps "odigos-ui" on the
// Actions and InstrumentationRules it creates. Pinning the literals here means a
// rename shows up as a test failure instead of as resources silently changing owner
// (or, for the "profile" value, silently becoming eligible for deletion).
func TestManagedByLabelContractIsPinned(t *testing.T) {
	require.Equal(t, "odigos.io/managed-by", k8sconsts.OdigosProfilesManagedByLabel)
	require.Equal(t, "profile", k8sconsts.OdigosProfilesManagedByValue)
	require.Equal(t, "odigos-ui", k8sconsts.OdigosUIManagedByValue)
	require.Equal(t, "interrogation-loop", k8sconsts.OdigosInterrogationLoopManagedByValue)

	require.Len(t, mbRecognizedValues, len(model.AllManagedBy)-1,
		"every ManagedBy member except Unknown needs its own distinct label value")
}

func TestManagedByFromLabelsCoversEveryEnumMember(t *testing.T) {
	sourceOf := map[model.ManagedBy]string{}

	for value, want := range mbRecognizedValues {
		got := managedByFromLabels(map[string]string{k8sconsts.OdigosProfilesManagedByLabel: value})
		require.Equal(t, want, got, "label value %q", value)
		require.True(t, got.IsValid(), "%s is not a member of the GraphQL enum", got)

		previous, duplicated := sourceOf[got]
		require.False(t, duplicated, "label values %q and %q both resolve to %s", previous, value, got)
		sourceOf[got] = value
	}
	sourceOf[managedByFromLabels(nil)] = "<no label>"

	// A member added to the schema without a case in managedByFromLabels would render
	// in the UI as Unknown, which reads as "applied by hand" rather than "owned by a
	// surface the user cannot edit".
	for _, member := range model.AllManagedBy {
		require.Contains(t, sourceOf, member, "no managed-by label value resolves to %s", member)
	}
	require.Len(t, sourceOf, len(model.AllManagedBy))
}

func TestManagedByFromLabelsRejectsLookalikeValues(t *testing.T) {
	for _, tc := range []struct {
		name   string
		labels map[string]string
	}{
		{name: "nil labels", labels: nil},
		{name: "no labels", labels: map[string]string{}},
		{name: "empty value", labels: map[string]string{k8sconsts.OdigosProfilesManagedByLabel: ""}},
		{name: "wrong case", labels: map[string]string{k8sconsts.OdigosProfilesManagedByLabel: "Profile"}},
		{name: "trailing space", labels: map[string]string{k8sconsts.OdigosProfilesManagedByLabel: "odigos-ui "}},
		{name: "plural", labels: map[string]string{k8sconsts.OdigosProfilesManagedByLabel: "profiles"}},
		{name: "underscored", labels: map[string]string{k8sconsts.OdigosProfilesManagedByLabel: "interrogation_loop"}},
		{
			name:   "recognized value under the upstream key",
			labels: map[string]string{"app.kubernetes.io/managed-by": k8sconsts.OdigosUIManagedByValue},
		},
		{
			name:   "other odigos labels only",
			labels: map[string]string{k8sconsts.OdigosProfilesHashLabel: "8f14e45fceea167a"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, model.ManagedByUnknown, managedByFromLabels(tc.labels))
		})
	}
}
