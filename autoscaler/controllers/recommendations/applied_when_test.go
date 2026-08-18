package recommendations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common/consts"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	recommendationcatalog "github.com/odigos-io/odigos/recommendations"
)

// ****************
// actionExistsAppliedWhen
// ****************

func newAction(name string, mutate func(spec *odigosv1.ActionSpec)) odigosv1.Action {
	action := odigosv1.Action{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
	}
	mutate(&action.Spec)
	return action
}

func inferDbAttributesAction(name string) odigosv1.Action {
	return newAction(name, func(spec *odigosv1.ActionSpec) {
		spec.InferDbAttributes = &actions.InferDbAttributesConfig{}
	})
}

func TestActionExistsAppliedWhen(t *testing.T) {
	disabledInferDb := inferDbAttributesAction("disabled-infer-db")
	disabledInferDb.Spec.Disabled = true

	urlTemplatization := newAction("url-templatization", func(spec *odigosv1.ActionSpec) {
		spec.URLTemplatization = &actions.URLTemplatizationConfig{}
	})

	for _, tc := range []struct {
		name       string
		actionType string
		actions    []odigosv1.Action
		want       bool
		wantErr    bool
	}{
		{
			name:       "empty actionType is a manifest error",
			actionType: "  ",
			actions:    []odigosv1.Action{inferDbAttributesAction("infer-db")},
			wantErr:    true,
		},
		{
			name:       "no actions in the cluster",
			actionType: "InferDbAttributes",
			want:       false,
		},
		{
			name:       "matching action",
			actionType: "InferDbAttributes",
			actions:    []odigosv1.Action{inferDbAttributesAction("infer-db")},
			want:       true,
		},
		{
			name:       "action of a different type",
			actionType: "InferDbAttributes",
			actions:    []odigosv1.Action{urlTemplatization},
			want:       false,
		},
		{
			name:       "disabled action does not count as applied",
			actionType: "InferDbAttributes",
			actions:    []odigosv1.Action{disabledInferDb},
			want:       false,
		},
		{
			name:       "an enabled action next to a disabled one counts",
			actionType: "InferDbAttributes",
			actions:    []odigosv1.Action{disabledInferDb, inferDbAttributesAction("infer-db")},
			want:       true,
		},
		{
			name:       "unknown field name never matches",
			actionType: "NoSuchAction",
			actions:    []odigosv1.Action{inferDbAttributesAction("infer-db")},
			want:       false,
		},
		{
			// the check reflects over the go field names of ActionSpec, not the json names.
			name:       "json field name never matches",
			actionType: "inferDbAttributes",
			actions:    []odigosv1.Action{inferDbAttributesAction("infer-db")},
			want:       false,
		},
		{
			// Disabled is always addressable and non-nil, so a non pointer field must not
			// be treated as a configured action.
			name:       "non pointer field never matches",
			actionType: "Disabled",
			actions:    []odigosv1.Action{inferDbAttributesAction("infer-db")},
			want:       false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			check := recommendationcatalog.AppliedWhenCheck{
				Type:       recommendationcatalog.AppliedWhenTypeActionExists,
				ActionType: tc.actionType,
			}

			got, err := actionExistsAppliedWhen(tc.actions, check)

			if tc.wantErr {
				require.Error(t, err)
				assert.False(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ****************
// effectiveConfigAppliedWhen
// ****************

func TestEffectiveConfigAppliedWhen(t *testing.T) {
	cfgData := map[string]any{
		"goAutoOffsetsMode": "direct",
		"sampling": map[string]any{
			"k8sHealthProbesSampling": map[string]any{
				"enabled": true,
			},
		},
	}

	for _, tc := range []struct {
		name       string
		expression string
		want       bool
		wantErr    bool
	}{
		{
			name:       "empty expression is a manifest error",
			expression: "   ",
			wantErr:    true,
		},
		{
			name:       "invalid jmespath is an error",
			expression: "sampling[",
			wantErr:    true,
		},
		{
			name:       "expression must evaluate to a bool",
			expression: "goAutoOffsetsMode",
			wantErr:    true,
		},
		{
			name:       "missing path evaluates to null which is not a bool",
			expression: "sampling.noSuchKey",
			wantErr:    true,
		},
		{
			name:       "nested comparison that holds",
			expression: "sampling.k8sHealthProbesSampling.enabled == `true`",
			want:       true,
		},
		{
			name:       "nested comparison that does not hold",
			expression: "sampling.k8sHealthProbesSampling.enabled == `false`",
			want:       false,
		},
		{
			name:       "string comparison that holds",
			expression: "goAutoOffsetsMode != 'off'",
			want:       true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			check := recommendationcatalog.AppliedWhenCheck{
				Type:       recommendationcatalog.AppliedWhenTypeEffectiveConfig,
				Expression: tc.expression,
			}

			got, err := effectiveConfigAppliedWhen(cfgData, check)

			if tc.wantErr {
				require.Error(t, err)
				assert.False(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ****************
// evaluateAppliedWhen
// ****************

func TestEvaluateAppliedWhen(t *testing.T) {
	cfgData := map[string]any{"goAutoOffsetsMode": "direct"}
	clusterActions := []odigosv1.Action{inferDbAttributesAction("infer-db")}

	trueConfigCheck := recommendationcatalog.AppliedWhenCheck{
		Type:       recommendationcatalog.AppliedWhenTypeEffectiveConfig,
		Expression: "goAutoOffsetsMode != 'off'",
	}
	falseConfigCheck := recommendationcatalog.AppliedWhenCheck{
		Type:       recommendationcatalog.AppliedWhenTypeEffectiveConfig,
		Expression: "goAutoOffsetsMode == 'off'",
	}
	trueActionCheck := recommendationcatalog.AppliedWhenCheck{
		Type:       recommendationcatalog.AppliedWhenTypeActionExists,
		ActionType: "InferDbAttributes",
	}
	invalidCheck := recommendationcatalog.AppliedWhenCheck{
		Type:       recommendationcatalog.AppliedWhenTypeEffectiveConfig,
		Expression: "",
	}

	for _, tc := range []struct {
		name        string
		appliedWhen []recommendationcatalog.AppliedWhenCheck
		want        bool
		wantErr     bool
	}{
		{
			// without checks there is nothing that can prove the recommendation is applied,
			// so it must not be reported as applied.
			name:        "no checks is not applied",
			appliedWhen: nil,
			want:        false,
		},
		{
			name:        "single check that holds",
			appliedWhen: []recommendationcatalog.AppliedWhenCheck{trueConfigCheck},
			want:        true,
		},
		{
			name:        "checks of different types are and-ed",
			appliedWhen: []recommendationcatalog.AppliedWhenCheck{trueConfigCheck, trueActionCheck},
			want:        true,
		},
		{
			name:        "one check that does not hold makes the recommendation not applied",
			appliedWhen: []recommendationcatalog.AppliedWhenCheck{trueConfigCheck, falseConfigCheck, trueActionCheck},
			want:        false,
		},
		{
			name:        "unknown check type is an error",
			appliedWhen: []recommendationcatalog.AppliedWhenCheck{{Type: "NoSuchCheckType"}},
			wantErr:     true,
		},
		{
			name:        "an invalid check reports the error instead of a result",
			appliedWhen: []recommendationcatalog.AppliedWhenCheck{trueConfigCheck, invalidCheck},
			wantErr:     true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := recommendationcatalog.Recommendation{AppliedWhen: tc.appliedWhen}

			got, err := evaluateAppliedWhen(cfgData, clusterActions, rec)

			if tc.wantErr {
				require.Error(t, err)
				assert.False(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ****************
// getEffectiveConfigData
// ****************

func TestGetEffectiveConfigData(t *testing.T) {
	useTestNamespace(t)

	t.Run("parsed config is queryable with jmespath", func(t *testing.T) {
		c := newFakeClient(t, effectiveConfigMapWithYAML(testNamespace, "goAutoOffsetsMode: direct\n"))

		data, err := getEffectiveConfigData(testContext(), c)

		require.NoError(t, err)
		applied, err := effectiveConfigAppliedWhen(data, recommendationcatalog.AppliedWhenCheck{
			Expression: "goAutoOffsetsMode == 'direct'",
		})
		require.NoError(t, err)
		assert.True(t, applied)
	})

	// the reconcilers translate this specific error into a requeue, so that recommendations are
	// synced once the effective config is created instead of failing the reconcile.
	t.Run("missing config map reports ErrOdigosEffectiveConfigNotFound", func(t *testing.T) {
		c := newFakeClient(t)

		_, err := getEffectiveConfigData(testContext(), c)

		assert.ErrorIs(t, err, k8sutils.ErrOdigosEffectiveConfigNotFound)
	})

	t.Run("config map in another namespace is not used", func(t *testing.T) {
		c := newFakeClient(t, effectiveConfigMapWithYAML("other-ns", "goAutoOffsetsMode: direct\n"))

		_, err := getEffectiveConfigData(testContext(), c)

		assert.ErrorIs(t, err, k8sutils.ErrOdigosEffectiveConfigNotFound)
	})

	for _, tc := range []struct {
		name      string
		configMap *corev1.ConfigMap
	}{
		{
			name: "config map without data",
			configMap: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Name:      consts.OdigosEffectiveConfigName,
				Namespace: testNamespace,
			}},
		},
		{
			name:      "empty config file",
			configMap: effectiveConfigMapWithYAML(testNamespace, "  \n"),
		},
		{
			name:      "malformed config file",
			configMap: effectiveConfigMapWithYAML(testNamespace, "goAutoOffsetsMode: [unterminated\n"),
		},
	} {
		t.Run(tc.name+" is an error and not an empty config", func(t *testing.T) {
			c := newFakeClient(t, tc.configMap)

			data, err := getEffectiveConfigData(testContext(), c)

			require.Error(t, err)
			assert.NotErrorIs(t, err, k8sutils.ErrOdigosEffectiveConfigNotFound)
			assert.Nil(t, data)
		})
	}
}
