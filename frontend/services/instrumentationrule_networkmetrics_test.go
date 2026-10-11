package services

import (
	"context"
	"testing"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	apirules "github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// useFakeRuleClient points the package-level kube client at a fake odigos
// clientset seeded with the given rules, and pins the namespace the services
// resolve so the tests don't depend on the environment they run in.
func useFakeRuleClient(t *testing.T, rules ...*v1alpha1.InstrumentationRule) {
	t.Helper()

	t.Setenv(consts.CurrentNamespaceEnvVar, consts.DefaultOdigosNamespace)

	objects := make([]runtime.Object, 0, len(rules))
	for _, rule := range rules {
		objects = append(objects, rule)
	}

	clientset := odigosfake.NewSimpleClientset(objects...)
	previousClient := kube.DefaultClient
	kube.SetDefaultClient(&kube.Client{OdigosClient: clientset.OdigosV1alpha1()})
	t.Cleanup(func() {
		kube.SetDefaultClient(previousClient)
	})
}

func storedRule(t *testing.T, id string) *v1alpha1.InstrumentationRule {
	t.Helper()

	rule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(consts.DefaultOdigosNamespace).
		Get(context.Background(), id, metav1.GetOptions{})
	require.NoError(t, err)
	return rule
}

func TestDeriveTypeFromRule(t *testing.T) {
	enabled := true
	disabledMarker := false
	collect := true
	headerKey := "Authorization"

	for _, tc := range []struct {
		name     string
		rule     model.InstrumentationRule
		expected model.InstrumentationRuleType
	}{
		{
			name:     "empty rule",
			rule:     model.InstrumentationRule{},
			expected: model.InstrumentationRuleTypeUnknownType,
		},
		{
			name:     "code attributes",
			rule:     model.InstrumentationRule{CodeAttributes: &model.CodeAttributes{FilePath: &collect}},
			expected: model.InstrumentationRuleTypeCodeAttributes,
		},
		{
			name:     "code attributes with no flag set",
			rule:     model.InstrumentationRule{CodeAttributes: &model.CodeAttributes{}},
			expected: model.InstrumentationRuleTypeUnknownType,
		},
		{
			name:     "headers collection",
			rule:     model.InstrumentationRule{HeadersCollection: &model.HeadersCollection{HeaderKeys: []*string{&headerKey}}},
			expected: model.InstrumentationRuleTypeHeadersCollection,
		},
		{
			name:     "headers collection with no keys",
			rule:     model.InstrumentationRule{HeadersCollection: &model.HeadersCollection{}},
			expected: model.InstrumentationRuleTypeUnknownType,
		},
		{
			name: "payload collection",
			rule: model.InstrumentationRule{PayloadCollection: &model.PayloadCollection{
				HTTPRequest: &model.HTTPPayloadCollection{},
			}},
			expected: model.InstrumentationRuleTypePayloadCollection,
		},
		{
			name:     "payload collection with no section enabled",
			rule:     model.InstrumentationRule{PayloadCollection: &model.PayloadCollection{}},
			expected: model.InstrumentationRuleTypeUnknownType,
		},
		{
			name:     "custom instrumentation",
			rule:     model.InstrumentationRule{CustomInstrumentations: &model.CustomInstrumentations{}},
			expected: model.InstrumentationRuleTypeCustomInstrumentation,
		},
		{
			name:     "network metrics",
			rule:     model.InstrumentationRule{NetworkMetrics: &enabled},
			expected: model.InstrumentationRuleTypeNetworkMetrics,
		},
		{
			// The marker is what identifies the rule's type, so a rule that carries
			// it as `false` is indistinguishable from an empty rule. This is why the
			// catalog must not expose it as a form field.
			name:     "network metrics explicitly off",
			rule:     model.InstrumentationRule{NetworkMetrics: &disabledMarker},
			expected: model.InstrumentationRuleTypeUnknownType,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, deriveTypeFromRule(&tc.rule))
		})
	}
}

func TestNetworkMetricsInputRoundTrip(t *testing.T) {
	enabled := true
	disabledMarker := false

	for _, tc := range []struct {
		name         string
		input        *bool
		expectConfig bool
		expectedType model.InstrumentationRuleType
	}{
		{name: "omitted", input: nil, expectConfig: false, expectedType: model.InstrumentationRuleTypeUnknownType},
		{name: "off", input: &disabledMarker, expectConfig: false, expectedType: model.InstrumentationRuleTypeUnknownType},
		{name: "on", input: &enabled, expectConfig: true, expectedType: model.InstrumentationRuleTypeNetworkMetrics},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := getNetworkMetricsInput(model.InstrumentationRuleInput{NetworkMetrics: tc.input})
			if tc.expectConfig {
				require.Equal(t, &apirules.NetworkMetricsConfig{}, config)
			} else {
				require.Nil(t, config)
			}

			rule := model.InstrumentationRule{NetworkMetrics: networkMetricsEnabledPtr(config)}
			require.Equal(t, tc.expectedType, deriveTypeFromRule(&rule))
		})
	}
}

func TestCreateInstrumentationRuleNetworkMetrics(t *testing.T) {
	ctx := context.Background()
	ruleName := "network metrics"
	notes := ""
	disabled := false
	enabled := true

	useFakeRuleClient(t)

	created, err := CreateInstrumentationRule(ctx, model.InstrumentationRuleInput{
		RuleName:       &ruleName,
		Notes:          &notes,
		Disabled:       &disabled,
		NetworkMetrics: &enabled,
	})
	require.NoError(t, err)
	require.Equal(t, model.InstrumentationRuleTypeNetworkMetrics, created.Type)
	require.NotNil(t, created.NetworkMetrics)
	require.True(t, *created.NetworkMetrics)

	stored := storedRule(t, created.RuleID)
	require.Equal(t, &apirules.NetworkMetricsConfig{}, stored.Spec.NetworkMetrics,
		"the presence marker is the only thing identifying the rule's type")
	require.False(t, stored.Spec.Disabled)
}

func TestCreateInstrumentationRuleWithoutNetworkMetricsMarker(t *testing.T) {
	ctx := context.Background()
	ruleName := "network metrics"
	notes := ""
	disabled := false
	marker := false

	useFakeRuleClient(t)

	// Nothing but the marker distinguishes a network metrics rule, so creating one
	// with the marker off stores an empty spec that reads back as UnknownType and
	// can no longer be opened in the UI.
	created, err := CreateInstrumentationRule(ctx, model.InstrumentationRuleInput{
		RuleName:       &ruleName,
		Notes:          &notes,
		Disabled:       &disabled,
		NetworkMetrics: &marker,
	})
	require.NoError(t, err)
	require.Equal(t, model.InstrumentationRuleTypeUnknownType, created.Type)
	require.Nil(t, created.NetworkMetrics)
	require.Nil(t, storedRule(t, created.RuleID).Spec.NetworkMetrics)
}

func TestGetInstrumentationRuleReportsDisabledNetworkMetricsRule(t *testing.T) {
	ctx := context.Background()
	ruleID := "ui-instrumentation-rule-network"

	useFakeRuleClient(t, &v1alpha1.InstrumentationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleID,
			Namespace: consts.DefaultOdigosNamespace,
		},
		Spec: v1alpha1.InstrumentationRuleSpec{
			RuleName:       "network metrics",
			Disabled:       true,
			NetworkMetrics: &apirules.NetworkMetricsConfig{},
		},
	})

	rule, err := GetInstrumentationRule(ctx, ruleID)
	require.NoError(t, err)
	// A disabled rule keeps its marker, so it still resolves to its own type and
	// the UI can reopen it.
	require.Equal(t, model.InstrumentationRuleTypeNetworkMetrics, rule.Type)
	require.NotNil(t, rule.NetworkMetrics)
	require.True(t, *rule.NetworkMetrics)
	require.NotNil(t, rule.Disabled)
	require.True(t, *rule.Disabled)

	rules, err := GetInstrumentationRules(ctx)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	require.Equal(t, model.InstrumentationRuleTypeNetworkMetrics, rules[0].Type)
}

func TestUpdateInstrumentationRuleDisablingKeepsNetworkMetricsType(t *testing.T) {
	ctx := context.Background()
	ruleID := "ui-instrumentation-rule-network"
	ruleName := "network metrics"
	notes := ""
	disabled := true
	enabled := true

	useFakeRuleClient(t, &v1alpha1.InstrumentationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleID,
			Namespace: consts.DefaultOdigosNamespace,
		},
		Spec: v1alpha1.InstrumentationRuleSpec{
			RuleName:       ruleName,
			NetworkMetrics: &apirules.NetworkMetricsConfig{},
		},
	})

	updated, err := UpdateInstrumentationRule(ctx, ruleID, model.InstrumentationRuleInput{
		RuleName:       &ruleName,
		Notes:          &notes,
		Disabled:       &disabled,
		NetworkMetrics: &enabled,
	})
	require.NoError(t, err)
	require.Equal(t, model.InstrumentationRuleTypeNetworkMetrics, updated.Type)
	require.NotNil(t, updated.Disabled)
	require.True(t, *updated.Disabled)

	stored := storedRule(t, ruleID)
	require.True(t, stored.Spec.Disabled)
	require.Equal(t, &apirules.NetworkMetricsConfig{}, stored.Spec.NetworkMetrics,
		"disabling a rule must not erase the field that identifies its type")
}
