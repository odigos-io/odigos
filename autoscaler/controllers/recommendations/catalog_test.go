package recommendations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	recommendationcatalog "github.com/odigos-io/odigos/recommendations"
)

// The checks below run against the recommendation manifests that ship with odigos, so a new or
// edited manifest is validated against the types it is evaluated with.

func TestCatalogAppliedWhenChecksAreEvaluable(t *testing.T) {
	useTestNamespace(t)
	c := newFakeClient(t, effectiveConfigMap(t, &common.OdigosConfiguration{}))
	cfgData, err := getEffectiveConfigData(testContext(), c)
	require.NoError(t, err)

	for _, rec := range loadCatalog(t) {
		t.Run(rec.K8sObjectName, func(t *testing.T) {
			// a recommendation without checks is reported as not applied forever, even after the
			// user applies one of its remediations.
			require.NotEmpty(t, rec.AppliedWhen, "recommendation has no appliedWhen checks")

			// unknown check types, empty action types and expressions that are invalid or do not
			// evaluate to a bool are all reported as errors, which fail the whole sync.
			_, err := evaluateAppliedWhen(cfgData, nil, rec)
			assert.NoError(t, err, "appliedWhen checks must be evaluable against an empty cluster")
		})
	}
}

// A remediation is what the ui applies on the user's behalf, and appliedWhen is how the result is
// reported back. If they drift apart the ui applies the remediation and keeps recommending it.
func TestCatalogRemediationsSatisfyAppliedWhen(t *testing.T) {
	useTestNamespace(t)

	for _, rec := range loadCatalog(t) {
		for _, remediation := range rec.Remediations {
			if len(remediation.Steps) == 0 {
				continue
			}

			t.Run(rec.K8sObjectName+"/"+remediation.Type, func(t *testing.T) {
				cfg := &common.OdigosConfiguration{}
				require.NoError(t, recommendationcatalog.ApplyStepsToConfig(cfg, recommendationcatalog.ConfigSteps(remediation.Steps)))

				var clusterActions []odigosv1.Action
				if len(recommendationcatalog.ActionSteps(remediation.Steps)) > 0 {
					clusterActions = append(clusterActions, actionFromApplyExample(t, remediation))
				}

				c := newFakeClient(t, effectiveConfigMap(t, cfg))
				cfgData, err := getEffectiveConfigData(testContext(), c)
				require.NoError(t, err)

				applied, err := evaluateAppliedWhen(cfgData, clusterActions, rec)

				require.NoError(t, err)
				assert.True(t, applied, "applying the remediation should mark the recommendation as applied")
			})
		}
	}
}

// actionFromApplyExample parses the Action manifest the ui creates for an ApplyOdigosAction step.
func actionFromApplyExample(t *testing.T, remediation recommendationcatalog.Remediation) odigosv1.Action {
	t.Helper()
	example, ok := remediation.OdigosActionExample()
	require.True(t, ok, "ApplyOdigosAction step requires an applyExamples entry of type %q",
		recommendationcatalog.ApplyExampleTypeOdigosAction)

	content := strings.TrimSpace(example.Content)
	require.NotEmpty(t, content, "OdigosAction applyExamples content is empty")

	var action odigosv1.Action
	require.NoError(t, yaml.Unmarshal([]byte(content), &action))
	require.NotEmpty(t, action.Name, "OdigosAction applyExamples content has no metadata.name")
	return action
}
