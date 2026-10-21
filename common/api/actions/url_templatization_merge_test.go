package actions

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeDefaultTemplatizationSkipPolicyConfigs_nilC1ReturnsC2(t *testing.T) {
	c2 := &DefaultTemplatizationSkipPolicyConfig{
		SkipHttpStatusCodes: []int{404},
	}
	got := mergeDefaultTemplatizationSkipPolicyConfigs(nil, c2)
	require.Same(t, c2, got)
}

func TestMergeDefaultTemplatizationSkipPolicyConfigs_nilC2ReturnsC1(t *testing.T) {
	c1 := &DefaultTemplatizationSkipPolicyConfig{
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
	c1 := &DefaultTemplatizationSkipPolicyConfig{SkipForNonSuccessCodes: true}
	c2 := &DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{404, 500}}

	got := mergeDefaultTemplatizationSkipPolicyConfigs(c1, c2)

	require.NotNil(t, got)
	require.True(t, got.SkipForNonSuccessCodes)
	require.Nil(t, got.SkipHttpStatusCodes)
}

func TestMergeDefaultTemplatizationSkipPolicyConfigs_skipForNonSuccessCodesInC2(t *testing.T) {
	c1 := &DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{401}}
	c2 := &DefaultTemplatizationSkipPolicyConfig{SkipForNonSuccessCodes: true}

	got := mergeDefaultTemplatizationSkipPolicyConfigs(c1, c2)

	require.NotNil(t, got)
	require.True(t, got.SkipForNonSuccessCodes)
	require.Nil(t, got.SkipHttpStatusCodes)
}

func TestMergeDefaultTemplatizationSkipPolicyConfigs_mergesStatusCodes(t *testing.T) {
	c1 := &DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{404}}
	c2 := &DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{401, 500}}

	got := mergeDefaultTemplatizationSkipPolicyConfigs(c1, c2)

	require.NotNil(t, got)
	require.False(t, got.SkipForNonSuccessCodes)
	require.Equal(t, []int{404, 401, 500}, got.SkipHttpStatusCodes)
}

func TestMergeDefaultTemplatizationSkipPolicyConfigs_keepsDuplicateStatusCodes(t *testing.T) {
	c1 := &DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{404, 401}}
	c2 := &DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{404, 500}}

	got := mergeDefaultTemplatizationSkipPolicyConfigs(c1, c2)

	require.NotNil(t, got)
	require.False(t, got.SkipForNonSuccessCodes)
	require.Equal(t, []int{404, 401, 404, 500}, got.SkipHttpStatusCodes)
}

func TestMergeDefaultTemplatizationConfigs_nilC1ReturnsC2(t *testing.T) {
	c2 := &DefaultTemplatizationConfig{
		SkipPolicy: &DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{404}},
	}
	got := MergeDefaultTemplatizationConfigs(nil, c2)
	require.Same(t, c2, got)
}

func TestMergeDefaultTemplatizationConfigs_nilC2ReturnsC1(t *testing.T) {
	c1 := &DefaultTemplatizationConfig{
		SkipPolicy: &DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{401}},
	}
	got := MergeDefaultTemplatizationConfigs(c1, nil)
	require.Same(t, c1, got)
}

func TestMergeDefaultTemplatizationConfigs_bothNilReturnsNil(t *testing.T) {
	got := MergeDefaultTemplatizationConfigs(nil, nil)
	require.Nil(t, got)
}

func TestMergeDefaultTemplatizationConfigs_disabledInC1(t *testing.T) {
	c1 := &DefaultTemplatizationConfig{Disabled: true}
	c2 := &DefaultTemplatizationConfig{
		SkipPolicy: &DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{404}},
	}

	got := MergeDefaultTemplatizationConfigs(c1, c2)

	require.NotNil(t, got)
	require.True(t, got.Disabled)
	require.Nil(t, got.SkipPolicy)
}

func TestMergeDefaultTemplatizationConfigs_disabledInC2(t *testing.T) {
	c1 := &DefaultTemplatizationConfig{
		SkipPolicy: &DefaultTemplatizationSkipPolicyConfig{SkipForNonSuccessCodes: true},
	}
	c2 := &DefaultTemplatizationConfig{Disabled: true}

	got := MergeDefaultTemplatizationConfigs(c1, c2)

	require.NotNil(t, got)
	require.True(t, got.Disabled)
	require.Nil(t, got.SkipPolicy)
}

func TestMergeDefaultTemplatizationConfigs_mergesSkipPolicy(t *testing.T) {
	c1 := &DefaultTemplatizationConfig{
		SkipPolicy: &DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{404}},
	}
	c2 := &DefaultTemplatizationConfig{
		SkipPolicy: &DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{401}},
	}

	got := MergeDefaultTemplatizationConfigs(c1, c2)

	require.NotNil(t, got)
	require.False(t, got.Disabled)
	require.NotNil(t, got.SkipPolicy)
	require.Equal(t, []int{404, 401}, got.SkipPolicy.SkipHttpStatusCodes)
}

func TestMergeDefaultTemplatizationConfigs_enabledWithNilSkipPolicy(t *testing.T) {
	c1 := &DefaultTemplatizationConfig{}
	c2 := &DefaultTemplatizationConfig{
		SkipPolicy: &DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{500}},
	}

	got := MergeDefaultTemplatizationConfigs(c1, c2)

	require.NotNil(t, got)
	require.False(t, got.Disabled)
	require.NotNil(t, got.SkipPolicy)
	require.Equal(t, []int{500}, got.SkipPolicy.SkipHttpStatusCodes)
}

func TestDedupeAndSortTemplates_ordersSegmentsBySpecificity(t *testing.T) {
	got := dedupeAndSortTemplates([]string{
		"/foo/{id}",
		"/foo/*",
		"/foo/bar",
		"/bar/{id}",
		"/foo/bar",
		"/foo/{name}/extra",
		"/foo/*/extra",
		"/foo/baz/extra",
	})

	require.Equal(t, []string{
		"/bar/{id}",
		"/foo/bar",
		"/foo/baz/extra",
		"/foo/*",
		"/foo/*/extra",
		"/foo/{id}",
		"/foo/{name}/extra",
	}, got)
}

func TestBuildUrlTemplatizationConfig_nilDefaultEnablesDefaultTemplatization(t *testing.T) {
	got := BuildUrlTemplatizationConfig([]string{"/b/{id}", "/a/{id}", "/b/{id}"}, nil, true)

	require.NotNil(t, got)
	require.Equal(t, []string{"/a/{id}", "/b/{id}"}, got.Templates)
	require.Equal(t, &DefaultTemplatizationConfig{}, got.Default)
}

func TestBuildUrlTemplatizationConfig_disabledDefaultWithRules(t *testing.T) {
	got := BuildUrlTemplatizationConfig([]string{"/a/{id}"}, &DefaultTemplatizationConfig{Disabled: true}, true)

	require.NotNil(t, got)
	require.Equal(t, []string{"/a/{id}"}, got.Templates)
	require.Nil(t, got.Default)
}

func TestBuildUrlTemplatizationConfig_disabledDefaultWithoutRulesReturnsNil(t *testing.T) {
	got := BuildUrlTemplatizationConfig(nil, &DefaultTemplatizationConfig{Disabled: true}, false)
	require.Nil(t, got)
}

func TestBuildUrlTemplatizationConfig_dedupesStatusCodes(t *testing.T) {
	defaultConfig := &DefaultTemplatizationConfig{
		SkipPolicy: &DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{404, 401, 404, 500}},
	}

	got := BuildUrlTemplatizationConfig(nil, defaultConfig, false)

	require.NotNil(t, got)
	require.Equal(t, []int{401, 404, 500}, got.Default.SkipPolicy.SkipHttpStatusCodes)
}
