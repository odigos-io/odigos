package services

import (
	"testing"

	"github.com/odigos-io/odigos/recommendations"
	"github.com/stretchr/testify/require"
)

// `canApplyViaUi` is the only thing standing between a recommendation and the UI's
// "Apply via UI" button: the drawer renders the GitOps snippet instead whenever no
// remediation reports it. It is derived — not stored in the catalog — so dropping the
// derivation silently serializes Go's zero value and every recommendation degrades to
// GitOps-only, with nothing failing to say so. These tests exist to make that fail.

func TestToCatalogRemediations_CanApplyViaUIFollowsSteps(t *testing.T) {
	remediations, err := toCatalogRemediations([]recommendations.Remediation{
		{
			Type:  "WithSteps",
			Steps: []recommendations.RemediationStep{{Type: recommendations.RemediationStepTypeApplyOdigosAction}},
		},
		{
			// GitOps-only: a snippet the customer applies themselves (e.g. Helm values,
			// which the backend cannot apply on their behalf).
			Type:  "WithoutSteps",
			Steps: nil,
		},
	})
	require.NoError(t, err)
	require.Len(t, remediations, 2)

	require.True(t, remediations[0].CanApplyViaUI, "a remediation with catalog steps must be applicable from the UI")
	require.False(t, remediations[1].CanApplyViaUI, "a remediation with no catalog steps is GitOps-only")
}

// Guards the real catalog, not just the mapping: every shipped remediation that declares
// steps has to reach the UI as applicable. Written against whatever the catalog holds, so
// a new recommendation is covered the moment it is added.
func TestCatalogRemediationsExposeUIApply(t *testing.T) {
	require.NoError(t, recommendations.Load())

	catalog := recommendations.Get()
	require.NotEmpty(t, catalog, "recommendation catalog is empty")

	applicable := 0
	for _, rec := range catalog {
		mapped, err := toCatalogRemediations(rec.Remediations)
		require.NoErrorf(t, err, "recommendation %q", rec.Type)
		require.Lenf(t, mapped, len(rec.Remediations), "recommendation %q", rec.Type)

		for i, rem := range rec.Remediations {
			require.Equalf(t, len(rem.Steps) > 0, mapped[i].CanApplyViaUI, "recommendation %q remediation %q: canApplyViaUi must follow whether it has catalog steps", rec.Type, rem.Type)
			if len(rem.Steps) > 0 {
				applicable++
			}
		}
	}

	// Without this the suite would still pass if the catalog lost every `steps:` block —
	// the per-remediation assertion above is vacuously true when nothing has steps.
	require.NotZero(t, applicable, "no catalog remediation is applicable from the UI; the Apply via UI button would never render")
}
