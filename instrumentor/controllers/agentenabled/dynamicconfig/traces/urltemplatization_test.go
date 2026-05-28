package traces

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	urltemplatizationactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/actions"
	"github.com/stretchr/testify/require"
)

func TestMergeDefaultTemplatizationSkipPolicyConfigs_nilC1ReturnsC2(t *testing.T) {
	c2 := &actions.DefaultTemplatizationSkipPolicyConfig{
		SkipHttpStatusCodes: []int{404},
	}
	got := mergeDefaultTemplatizationSkipPolicyConfigs(nil, c2)
	require.Same(t, c2, got)
}

func TestMergeDefaultTemplatizationSkipPolicyConfigs_nilC2ReturnsC1(t *testing.T) {
	c1 := &actions.DefaultTemplatizationSkipPolicyConfig{
		SkipHttpStatusCodes: []int{401},
	}
	got := mergeDefaultTemplatizationSkipPolicyConfigs(c1, nil)
	require.Same(t, c1, got)
}

func TestMergeDefaultTemplatizationSkipPolicyConfigs_bothNilReturnsNil(t *testing.T) {
	got := mergeDefaultTemplatizationSkipPolicyConfigs(nil, nil)
	require.Nil(t, got)
}

func TestMergeDefaultTemplatizationSkipPolicyConfigs_skipForNonSuccessCodesInC1(t *testing.T) {
	c1 := &actions.DefaultTemplatizationSkipPolicyConfig{SkipForNonSuccessCodes: true}
	c2 := &actions.DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{404, 500}}

	got := mergeDefaultTemplatizationSkipPolicyConfigs(c1, c2)

	require.NotNil(t, got)
	require.True(t, got.SkipForNonSuccessCodes)
	require.Nil(t, got.SkipHttpStatusCodes)
}

func TestMergeDefaultTemplatizationSkipPolicyConfigs_skipForNonSuccessCodesInC2(t *testing.T) {
	c1 := &actions.DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{401}}
	c2 := &actions.DefaultTemplatizationSkipPolicyConfig{SkipForNonSuccessCodes: true}

	got := mergeDefaultTemplatizationSkipPolicyConfigs(c1, c2)

	require.NotNil(t, got)
	require.True(t, got.SkipForNonSuccessCodes)
	require.Nil(t, got.SkipHttpStatusCodes)
}

func TestMergeDefaultTemplatizationSkipPolicyConfigs_mergesStatusCodes(t *testing.T) {
	c1 := &actions.DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{404}}
	c2 := &actions.DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{401, 500}}

	got := mergeDefaultTemplatizationSkipPolicyConfigs(c1, c2)

	require.NotNil(t, got)
	require.False(t, got.SkipForNonSuccessCodes)
	require.Equal(t, []int{404, 401, 500}, got.SkipHttpStatusCodes)
}

func TestMergeDefaultTemplatizationSkipPolicyConfigs_keepsDuplicateStatusCodes(t *testing.T) {
	c1 := &actions.DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{404, 401}}
	c2 := &actions.DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{404, 500}}

	got := mergeDefaultTemplatizationSkipPolicyConfigs(c1, c2)

	require.NotNil(t, got)
	require.False(t, got.SkipForNonSuccessCodes)
	require.Equal(t, []int{404, 401, 404, 500}, got.SkipHttpStatusCodes)
}

func TestMergeDefaultTemplatizationConfigs_nilC1ReturnsC2(t *testing.T) {
	c2 := &actions.DefaultTemplatizationConfig{
		SkipPolicy: &actions.DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{404}},
	}
	got := mergeDefaultTemplatizationConfigs(nil, c2)
	require.Same(t, c2, got)
}

func TestMergeDefaultTemplatizationConfigs_nilC2ReturnsC1(t *testing.T) {
	c1 := &actions.DefaultTemplatizationConfig{
		SkipPolicy: &actions.DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{401}},
	}
	got := mergeDefaultTemplatizationConfigs(c1, nil)
	require.Same(t, c1, got)
}

func TestMergeDefaultTemplatizationConfigs_bothNilReturnsNil(t *testing.T) {
	got := mergeDefaultTemplatizationConfigs(nil, nil)
	require.Nil(t, got)
}

func TestMergeDefaultTemplatizationConfigs_disabledInC1(t *testing.T) {
	c1 := &actions.DefaultTemplatizationConfig{Disabled: true}
	c2 := &actions.DefaultTemplatizationConfig{
		SkipPolicy: &actions.DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{404}},
	}

	got := mergeDefaultTemplatizationConfigs(c1, c2)

	require.NotNil(t, got)
	require.True(t, got.Disabled)
	require.Nil(t, got.SkipPolicy)
}

func TestMergeDefaultTemplatizationConfigs_disabledInC2(t *testing.T) {
	c1 := &actions.DefaultTemplatizationConfig{
		SkipPolicy: &actions.DefaultTemplatizationSkipPolicyConfig{SkipForNonSuccessCodes: true},
	}
	c2 := &actions.DefaultTemplatizationConfig{Disabled: true}

	got := mergeDefaultTemplatizationConfigs(c1, c2)

	require.NotNil(t, got)
	require.True(t, got.Disabled)
	require.Nil(t, got.SkipPolicy)
}

func TestMergeDefaultTemplatizationConfigs_mergesSkipPolicy(t *testing.T) {
	c1 := &actions.DefaultTemplatizationConfig{
		SkipPolicy: &actions.DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{404}},
	}
	c2 := &actions.DefaultTemplatizationConfig{
		SkipPolicy: &actions.DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{401}},
	}

	got := mergeDefaultTemplatizationConfigs(c1, c2)

	require.NotNil(t, got)
	require.False(t, got.Disabled)
	require.NotNil(t, got.SkipPolicy)
	require.Equal(t, []int{404, 401}, got.SkipPolicy.SkipHttpStatusCodes)
}

func TestMergeDefaultTemplatizationConfigs_enabledWithNilSkipPolicy(t *testing.T) {
	c1 := &actions.DefaultTemplatizationConfig{}
	c2 := &actions.DefaultTemplatizationConfig{
		SkipPolicy: &actions.DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{500}},
	}

	got := mergeDefaultTemplatizationConfigs(c1, c2)

	require.NotNil(t, got)
	require.False(t, got.Disabled)
	require.NotNil(t, got.SkipPolicy)
	require.Equal(t, []int{500}, got.SkipPolicy.SkipHttpStatusCodes)
}

func TestCalculateUrlTemplatizationConfig_dedupesStatusCodes(t *testing.T) {
	agentLevelActions := []odigosv1.Action{{
		Spec: odigosv1.ActionSpec{
			URLTemplatization: &urltemplatizationactions.URLTemplatizationConfig{
				DefaultTemplatizations: []urltemplatizationactions.URLTemplatizationDefaultTemplatizationGroup{
					{
						Config: actions.DefaultTemplatizationConfig{
							SkipPolicy: &actions.DefaultTemplatizationSkipPolicyConfig{
								SkipHttpStatusCodes: []int{404, 401},
							},
						},
					},
					{
						Config: actions.DefaultTemplatizationConfig{
							SkipPolicy: &actions.DefaultTemplatizationSkipPolicyConfig{
								SkipHttpStatusCodes: []int{404, 500},
							},
						},
					},
				},
			},
		},
	}}
	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}

	got := CalculateUrlTemplatizationConfig(&agentLevelActions, "container", common.JavaProgrammingLanguage, pw)

	require.NotNil(t, got)
	require.NotNil(t, got.DefaultTemplatization)
	require.NotNil(t, got.DefaultTemplatization.SkipPolicy)
	require.Equal(t, []int{404, 401, 500}, got.DefaultTemplatization.SkipPolicy.SkipHttpStatusCodes)
}
