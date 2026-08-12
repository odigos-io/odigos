package recommendations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	recommendationcatalog "github.com/odigos-io/odigos/recommendations"
)

// ****************
// isRelevantForTier
// ****************

func TestIsRelevantForTier(t *testing.T) {
	for _, tc := range []struct {
		name string
		tier common.OdigosTier
		oss  bool
		want bool
	}{
		{"oss recommendation in community", common.CommunityOdigosTier, true, true},
		{"oss recommendation in onprem", common.OnPremOdigosTier, true, true},
		{"enterprise recommendation in community", common.CommunityOdigosTier, false, false},
		{"enterprise recommendation in onprem", common.OnPremOdigosTier, false, true},
		{"enterprise recommendation in cloud", common.CloudOdigosTier, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			useTier(t, tc.tier)

			assert.Equal(t, tc.want, isRelevantForTier(recommendationcatalog.Recommendation{OSS: tc.oss}))
		})
	}
}

// ****************
// conditions
// ****************

func instrumentationConfigWithDistros(name string, distroNames ...string) *odigosv1.InstrumentationConfig {
	containers := make([]odigosv1.ContainerAgentConfig, 0, len(distroNames))
	for i, distroName := range distroNames {
		containers = append(containers, odigosv1.ContainerAgentConfig{
			ContainerName:  "container-" + string(rune('a'+i)),
			AgentEnabled:   true,
			OtelDistroName: distroName,
		})
	}
	return &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec:       odigosv1.InstrumentationConfigSpec{Containers: containers},
	}
}

func TestHasGoEnterpriseSource(t *testing.T) {
	for _, tc := range []struct {
		name    string
		objects []client.Object
		want    bool
	}{
		{
			name: "no instrumented sources",
			want: false,
		},
		{
			name:    "no golang enterprise distro",
			objects: []client.Object{instrumentationConfigWithDistros("deployment-nodejs", "nodejs-community")},
			want:    false,
		},
		{
			name: "golang enterprise in a container of the second source",
			objects: []client.Object{
				instrumentationConfigWithDistros("deployment-nodejs", "nodejs-community"),
				instrumentationConfigWithDistros("deployment-go", "nodejs-community", golangEnterpriseDistroName),
			},
			want: true,
		},
		{
			name:    "community golang distro is not the enterprise one",
			objects: []client.Object{instrumentationConfigWithDistros("deployment-go", "golang-community")},
			want:    false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newFakeClient(t, tc.objects...)

			got, err := hasGoEnterpriseSource(testContext(), c)

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestEvaluateConditions(t *testing.T) {
	goEnterpriseCondition := recommendationcatalog.Condition{Type: recommendationcatalog.ConditionTypeGoEnterpriseSources}

	t.Run("no conditions always hold", func(t *testing.T) {
		c := newFakeClient(t)

		met, err := evaluateConditions(testContext(), c, recommendationcatalog.Recommendation{})

		require.NoError(t, err)
		assert.True(t, met)
	})

	t.Run("condition that does not hold", func(t *testing.T) {
		c := newFakeClient(t)

		met, err := evaluateConditions(testContext(), c, recommendationcatalog.Recommendation{
			Conditions: []recommendationcatalog.Condition{goEnterpriseCondition},
		})

		require.NoError(t, err)
		assert.False(t, met)
	})

	t.Run("condition that holds", func(t *testing.T) {
		c := newFakeClient(t, instrumentationConfigWithDistros("deployment-go", golangEnterpriseDistroName))

		met, err := evaluateConditions(testContext(), c, recommendationcatalog.Recommendation{
			Conditions: []recommendationcatalog.Condition{goEnterpriseCondition},
		})

		require.NoError(t, err)
		assert.True(t, met)
	})

	// an unknown condition type comes from a newer manifest than this controller, treating it as
	// met would advertise a recommendation whose prerequisites were never checked.
	t.Run("unknown condition type is not met and does not fail the sync", func(t *testing.T) {
		c := newFakeClient(t, instrumentationConfigWithDistros("deployment-go", golangEnterpriseDistroName))

		met, err := evaluateConditions(testContext(), c, recommendationcatalog.Recommendation{
			Conditions: []recommendationcatalog.Condition{{Type: "NoSuchConditionType"}},
		})

		require.NoError(t, err)
		assert.False(t, met)
	})

	t.Run("all conditions must hold", func(t *testing.T) {
		c := newFakeClient(t, instrumentationConfigWithDistros("deployment-go", golangEnterpriseDistroName))

		met, err := evaluateConditions(testContext(), c, recommendationcatalog.Recommendation{
			Conditions: []recommendationcatalog.Condition{goEnterpriseCondition, {Type: "NoSuchConditionType"}},
		})

		require.NoError(t, err)
		assert.False(t, met)
	})
}

// ****************
// syncRecommendation
// ****************

// healthProbesRecommendation is a catalog-shaped recommendation with one effective config check.
func healthProbesRecommendation() recommendationcatalog.Recommendation {
	return recommendationcatalog.Recommendation{
		Type:          common.RecommendationTypeSampleHealthProbes,
		K8sObjectName: "sample-health-probes",
		OSS:           true,
		AppliedWhen: []recommendationcatalog.AppliedWhenCheck{{
			Type:       recommendationcatalog.AppliedWhenTypeEffectiveConfig,
			Expression: "sampling.k8sHealthProbesSampling.enabled == `true`",
		}},
	}
}

func TestSyncRecommendation(t *testing.T) {
	useTestNamespace(t)

	t.Run("applies the recommendation with the evaluated flags", func(t *testing.T) {
		c := newRecordingClient(t, effectiveConfigMap(t, &common.OdigosConfiguration{
			Sampling: &common.SamplingConfiguration{
				K8sHealthProbesSampling: &common.K8sHealthProbesSamplingConfiguration{Enabled: ptr.To(true)},
			},
		}))

		require.NoError(t, syncRecommendation(testContext(), c, healthProbesRecommendation(), testNamespace))

		applied := c.single(t)
		assert.Equal(t, "sample-health-probes", applied.name)
		assert.Equal(t, testNamespace, applied.namespace)
		assert.Equal(t, common.RecommendationTypeSampleHealthProbes, ptr.Deref(applied.spec.Type, ""))
		assert.Equal(t, ptr.To(true), applied.spec.Applied)
		assert.Equal(t, ptr.To(true), applied.spec.ConditionsMet)

		// without these options a recommendation owned by an older field manager can never be
		// updated by the autoscaler.
		assert.Equal(t, fieldOwner, applied.opts.FieldManager)
		assert.Equal(t, ptr.To(true), applied.opts.Force)
	})

	// the recommendation object is the single source of truth for the ui, so a recommendation that
	// was applied and then undone must be reported as not applied again. an omitted field in a
	// server side apply keeps the previous value, so the flags must be sent explicitly.
	t.Run("a recommendation that is no longer applied sends applied false", func(t *testing.T) {
		rec := healthProbesRecommendation()
		c := newRecordingClient(t,
			effectiveConfigMap(t, &common.OdigosConfiguration{}),
			&odigosv1.Recommendation{
				ObjectMeta: metav1.ObjectMeta{Name: rec.K8sObjectName, Namespace: testNamespace},
				Spec: odigosv1.RecommendationSpec{
					Type:          rec.Type,
					Applied:       true,
					ConditionsMet: true,
				},
			})

		require.NoError(t, syncRecommendation(testContext(), c, rec, testNamespace))

		assert.Equal(t, ptr.To(false), c.single(t).spec.Applied)
	})

	t.Run("conditions that do not hold do not prevent the recommendation from being applied", func(t *testing.T) {
		c := newRecordingClient(t, effectiveConfigMap(t, &common.OdigosConfiguration{}))
		rec := healthProbesRecommendation()
		rec.Conditions = []recommendationcatalog.Condition{{Type: recommendationcatalog.ConditionTypeGoEnterpriseSources}}

		require.NoError(t, syncRecommendation(testContext(), c, rec, testNamespace))

		applied := c.single(t)
		assert.Equal(t, ptr.To(false), applied.spec.ConditionsMet)
		assert.Equal(t, ptr.To(false), applied.spec.Applied)
	})

	t.Run("an enabled action makes an ActionExists recommendation applied", func(t *testing.T) {
		action := inferDbAttributesAction("infer-db")
		c := newRecordingClient(t, effectiveConfigMap(t, &common.OdigosConfiguration{}), &action)
		rec := recommendationcatalog.Recommendation{
			Type:          common.RecommendationTypeInferDBAttributes,
			K8sObjectName: "infer-db-attributes",
			OSS:           true,
			AppliedWhen: []recommendationcatalog.AppliedWhenCheck{{
				Type:       recommendationcatalog.AppliedWhenTypeActionExists,
				ActionType: "InferDbAttributes",
			}},
		}

		require.NoError(t, syncRecommendation(testContext(), c, rec, testNamespace))

		assert.Equal(t, ptr.To(true), c.single(t).spec.Applied)
	})

	t.Run("missing effective config is reported as ErrOdigosEffectiveConfigNotFound", func(t *testing.T) {
		c := newRecordingClient(t)

		err := syncRecommendation(testContext(), c, healthProbesRecommendation(), testNamespace)

		assert.ErrorIs(t, err, k8sutils.ErrOdigosEffectiveConfigNotFound)
		assert.Empty(t, c.applies, "no recommendation should be written when the config is missing")
	})

	t.Run("a manifest with an invalid appliedWhen check fails the sync", func(t *testing.T) {
		c := newRecordingClient(t, effectiveConfigMap(t, &common.OdigosConfiguration{}))
		rec := healthProbesRecommendation()
		rec.AppliedWhen = []recommendationcatalog.AppliedWhenCheck{{Type: "NoSuchCheckType"}}

		err := syncRecommendation(testContext(), c, rec, testNamespace)

		require.Error(t, err)
		assert.NotErrorIs(t, err, k8sutils.ErrOdigosEffectiveConfigNotFound)
		assert.Empty(t, c.applies)
	})
}

// ****************
// syncRecommendations
// ****************

func TestSyncRecommendations_TierFiltering(t *testing.T) {
	useTestNamespace(t)
	catalog := loadCatalog(t)

	var ossNames, enterpriseOnlyNames []string
	for _, rec := range catalog {
		if rec.OSS {
			ossNames = append(ossNames, rec.K8sObjectName)
		} else {
			enterpriseOnlyNames = append(enterpriseOnlyNames, rec.K8sObjectName)
		}
	}
	require.NotEmpty(t, ossNames, "catalog should have oss recommendations")
	require.NotEmpty(t, enterpriseOnlyNames, "catalog should have enterprise only recommendations")

	syncedNames := func(t *testing.T) []string {
		t.Helper()
		c := newFakeClient(t, effectiveConfigMap(t, &common.OdigosConfiguration{}))
		require.NoError(t, syncRecommendations(testContext(), c))

		var list odigosv1.RecommendationList
		require.NoError(t, c.List(testContext(), &list))
		names := make([]string, 0, len(list.Items))
		for i := range list.Items {
			assert.Equal(t, testNamespace, list.Items[i].Namespace)
			names = append(names, list.Items[i].Name)
		}
		return names
	}

	t.Run("community tier syncs only the oss recommendations", func(t *testing.T) {
		useTier(t, common.CommunityOdigosTier)

		assert.ElementsMatch(t, ossNames, syncedNames(t))
	})

	t.Run("enterprise tier syncs the whole catalog", func(t *testing.T) {
		useTier(t, common.OnPremOdigosTier)

		assert.ElementsMatch(t, append(append([]string{}, ossNames...), enterpriseOnlyNames...), syncedNames(t))
	})
}
