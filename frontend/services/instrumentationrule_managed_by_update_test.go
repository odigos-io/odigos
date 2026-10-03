package services

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateInstrumentationRule stamps odigos.io/managed-by=odigos-ui, but the update path
// must not: a rule the profile reconciler owns has to keep both its managed-by value
// and its profiles-hash (losing either orphans it from profile garbage collection),
// and a rule applied from YAML has to stay unlabeled instead of being adopted by the UI.
func TestUpdateInstrumentationRulePreservesManagedBy(t *testing.T) {
	for _, tc := range []struct {
		name   string
		labels map[string]string
		want   model.ManagedBy
	}{
		{
			name: "profile managed",
			labels: map[string]string{
				k8sconsts.OdigosProfilesManagedByLabel: k8sconsts.OdigosProfilesManagedByValue,
				k8sconsts.OdigosProfilesHashLabel:      "8f14e45fceea167a",
			},
			want: model.ManagedByProfile,
		},
		{
			name:   "ui managed",
			labels: map[string]string{k8sconsts.OdigosProfilesManagedByLabel: k8sconsts.OdigosUIManagedByValue},
			want:   model.ManagedByOdigosUI,
		},
		{
			name:   "interrogation loop managed",
			labels: map[string]string{k8sconsts.OdigosProfilesManagedByLabel: k8sconsts.OdigosInterrogationLoopManagedByValue},
			want:   model.ManagedByInterrogationLoop,
		},
		{
			name:   "applied from yaml",
			labels: nil,
			want:   model.ManagedByUnknown,
		},
		{
			name:   "unrecognized owner",
			labels: map[string]string{k8sconsts.OdigosProfilesManagedByLabel: "helm"},
			want:   model.ManagedByUnknown,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			const ruleID = "rule-under-update"
			renamedTo := "renamed by the ui"
			notes := ""
			disabled := false

			useFakeRuleClient(t, &v1alpha1.InstrumentationRule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ruleID,
					Namespace: consts.DefaultOdigosNamespace,
					Labels:    tc.labels,
				},
				Spec: v1alpha1.InstrumentationRuleSpec{RuleName: "original"},
			})

			updated, err := UpdateInstrumentationRule(ctx, ruleID, model.InstrumentationRuleInput{
				RuleName: &renamedTo,
				Notes:    &notes,
				Disabled: &disabled,
			})
			require.NoError(t, err)
			require.Equal(t, renamedTo, *updated.RuleName, "the update must have actually happened")
			require.Equal(t, tc.want, updated.ManagedBy)

			require.Equal(t, tc.labels, storedRule(t, ruleID).Labels)

			// The list and single-item read paths build the model separately, so they
			// each have their own managedByFromLabels call to get wrong.
			fetched, err := GetInstrumentationRule(ctx, ruleID)
			require.NoError(t, err)
			require.Equal(t, tc.want, fetched.ManagedBy)

			listed, err := GetInstrumentationRules(ctx)
			require.NoError(t, err)
			require.Len(t, listed, 1)
			require.Equal(t, tc.want, listed[0].ManagedBy)
		})
	}
}
