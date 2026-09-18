package sampling

import (
	"context"
	"reflect"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestGetAllSamplingRuleGroupsReturnsEveryGroupWithItsRulesPopulated(t *testing.T) {
	newSamplingRulesHarness(t,
		samplingCR("payments", func(cr *v1alpha1.Sampling) {
			cr.Spec.NoisyOperations = []v1alpha1.NoisyOperation{
				storedNoisyOperation("health", "/healthz"),
				storedNoisyOperation("metrics", "/metrics"),
			}
			cr.Spec.HighlyRelevantOperations = []v1alpha1.HighlyRelevantOperation{
				storedHighlyRelevantOperation("charges", "/charge"),
			}
		}),
		samplingCR("orders", func(cr *v1alpha1.Sampling) {
			cr.Spec.CostReductionRules = []v1alpha1.CostReductionRule{
				storedCostReductionRule("kafka", "/orders"),
			}
		}),
	)

	groups, err := GetAllSamplingRuleGroups(context.Background())
	require.NoError(t, err)
	require.Len(t, groups, 2)

	byID := map[string]*model.SamplingRules{}
	for _, group := range groups {
		byID[group.ID] = group
	}
	require.Contains(t, byID, "payments")
	require.Contains(t, byID, "orders")

	payments := byID["payments"]
	require.NotNil(t, payments.Name)
	assert.Equal(t, "payments", *payments.Name)
	require.Len(t, payments.NoisyOperations, 2)
	assert.Equal(t, stringPtr("health"), payments.NoisyOperations[0].Name)
	assert.Equal(t, stringPtr("metrics"), payments.NoisyOperations[1].Name,
		"the rules must keep the order they are stored in")
	require.Len(t, payments.HighlyRelevantOperations, 1)
	assert.Empty(t, payments.CostReductionRules)

	orders := byID["orders"]
	require.Len(t, orders.CostReductionRules, 1)
	assert.Equal(t, stringPtr("kafka"), orders.CostReductionRules[0].Name)
	assert.Empty(t, orders.NoisyOperations,
		"rules must not leak between groups")
}

func TestGetAllSamplingRuleGroupsIsScopedToTheOdigosNamespace(t *testing.T) {
	elsewhere := samplingCR("tenant-group")
	elsewhere.Namespace = "another-namespace"

	newSamplingRules(t, samplingRulesFixture{
		cached: []*v1alpha1.Sampling{samplingCR("odigos-group"), elsewhere},
	})

	groups, err := GetAllSamplingRuleGroups(context.Background())
	require.NoError(t, err)

	require.Len(t, groups, 1, "only sampling groups in the odigos namespace are the UI's")
	assert.Equal(t, "odigos-group", groups[0].ID)
}

func TestGetAllSamplingRuleGroupsSurfacesAListError(t *testing.T) {
	newSamplingRules(t, samplingRulesFixture{
		cacheFuncs: interceptor.Funcs{
			List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
				return apierrors.NewInternalError(assert.AnError)
			},
		},
	})

	groups, err := GetAllSamplingRuleGroups(context.Background())

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to list sampling CRs")
	assert.Nil(t, groups)
}

// The GraphQL schema declares sampling.rules and each rule list as non-null
// ([NoisyOperationRule!]!), so a nil slice anywhere here serialises as null and fails the
// whole query for every user - including the empty-state screen a fresh install shows.
func TestSamplingRuleGroupsAndTheirRuleListsAreNeverNil(t *testing.T) {
	newSamplingRulesHarness(t, samplingCR("empty-group"))

	groups, err := GetAllSamplingRuleGroups(context.Background())
	require.NoError(t, err)
	require.Len(t, groups, 1)

	assert.NotNil(t, groups[0].NoisyOperations)
	assert.Empty(t, groups[0].NoisyOperations)
	assert.NotNil(t, groups[0].HighlyRelevantOperations)
	assert.Empty(t, groups[0].HighlyRelevantOperations)
	assert.NotNil(t, groups[0].CostReductionRules)
	assert.Empty(t, groups[0].CostReductionRules)
}

func TestGetAllSamplingRuleGroupsReturnsAnEmptyListWhenNoGroupExists(t *testing.T) {
	newSamplingRulesHarness(t)

	groups, err := GetAllSamplingRuleGroups(context.Background())

	require.NoError(t, err)
	assert.NotNil(t, groups)
	assert.Empty(t, groups)
}

// The group id the UI addresses mutations with is the CR name, while the display name is
// the spec field. Collapsing the two would make every mutation from the UI target a
// nonexistent CR as soon as a group is renamed.
func TestSamplingCRToModelSeparatesTheCRNameFromTheDisplayName(t *testing.T) {
	cr := samplingCR("group-object-name", func(cr *v1alpha1.Sampling) {
		cr.Spec.Name = "Payments team rules"
	})

	got := samplingCRToModel(cr)

	assert.Equal(t, "group-object-name", got.ID)
	require.NotNil(t, got.Name)
	assert.Equal(t, "Payments team rules", *got.Name)
}

func TestSamplingCRToModelOmitsAnUnsetDisplayName(t *testing.T) {
	cr := samplingCR("group-object-name", func(cr *v1alpha1.Sampling) {
		cr.Spec.Name = ""
	})

	got := samplingCRToModel(cr)

	assert.Equal(t, "group-object-name", got.ID)
	assert.Nil(t, got.Name, "an unset display name is reported as absent, not as an empty string")
}

// fullyPopulatedNoisyOperation and its siblings deliberately set every field to a
// non-default value, so the "no output field is left zero" invariants below can tell a
// dropped assignment from a field the fixture simply never set.
func fullyPopulatedNoisyOperation() v1alpha1.NoisyOperation {
	return v1alpha1.NoisyOperation{
		Name:     "health checks",
		Disabled: true,
		SourceScopes: &k8sconsts.SourcesScopes{
			Namespaces: []string{"payments"},
			Languages:  []common.ProgrammingLanguage{common.JavaProgrammingLanguage},
			Sources: []k8sconsts.PodWorkload{
				{Name: "checkout", Namespace: "payments", Kind: k8sconsts.WorkloadKindDeployment},
			},
		},
		Operation:        headServerMatcher("/healthz"),
		PercentageAtMost: float64Ptr(1),
		Notes:            "raised by the payments team",
	}
}

func fullyPopulatedHighlyRelevantOperation() v1alpha1.HighlyRelevantOperation {
	return v1alpha1.HighlyRelevantOperation{
		Name:              "keep failing charges",
		Disabled:          true,
		SourceScopes:      &k8sconsts.SourcesScopes{Namespaces: []string{"payments"}},
		Error:             true,
		DurationAtLeastMs: intPtr(500),
		Operation:         tailServerMatcher("/charge"),
		PercentageAtLeast: float64Ptr(100),
		Notes:             "SLO relevant",
	}
}

func fullyPopulatedCostReductionRule() v1alpha1.CostReductionRule {
	return v1alpha1.CostReductionRule{
		Name:             "drop order noise",
		Disabled:         true,
		SourceScopes:     &k8sconsts.SourcesScopes{Namespaces: []string{"orders"}},
		Operation:        tailServerMatcher("/orders"),
		PercentageAtMost: 25,
		Notes:            "agreed with the orders team",
	}
}

// assertNoZeroFields is the invariant that catches a conversion dropping a field: a single
// assert.Equal against a hand-written expectation passes happily when a field is missing
// from both the expectation and the converter.
func assertNoZeroFields(t *testing.T, converted any, describe string) {
	t.Helper()
	value := reflect.ValueOf(converted)
	if value.Kind() == reflect.Ptr {
		require.False(t, value.IsNil(), "%s must not be nil", describe)
		value = value.Elem()
	}
	valueType := value.Type()
	require.Positive(t, valueType.NumField(), "%s has no fields to check", describe)
	for i := 0; i < valueType.NumField(); i++ {
		assert.False(t, value.Field(i).IsZero(),
			"%s.%s was left at its zero value although the input set it",
			describe, valueType.Field(i).Name)
	}
}

func TestConvertNoisyOperationToModelCarriesEveryField(t *testing.T) {
	rule := fullyPopulatedNoisyOperation()

	got := convertNoisyOperationToModel(&rule)

	assertNoZeroFields(t, got, "NoisyOperationRule")
	assert.Equal(t, v1alpha1.ComputeNoisyOperationHash(&rule), got.RuleID)
	assert.Equal(t, stringPtr("health checks"), got.Name)
	assert.True(t, got.Disabled)
	assert.Equal(t, float64Ptr(1), got.PercentageAtMost)
	assert.Equal(t, stringPtr("raised by the payments team"), got.Notes)
	require.NotNil(t, got.SourceScopes)
	assert.Equal(t, []string{"payments"}, got.SourceScopes.Namespaces)
	require.Len(t, got.SourceScopes.Sources, 1)
	assert.Equal(t, "checkout", got.SourceScopes.Sources[0].Name)
	require.NotNil(t, got.Operation)
	require.NotNil(t, got.Operation.HTTPServer)
	assert.Equal(t, stringPtr("/healthz"), got.Operation.HTTPServer.Route)
}

func TestConvertHighlyRelevantOperationToModelCarriesEveryField(t *testing.T) {
	rule := fullyPopulatedHighlyRelevantOperation()

	got := convertHighlyRelevantOperationToModel(&rule)

	assertNoZeroFields(t, got, "HighlyRelevantOperationRule")
	assert.Equal(t, v1alpha1.ComputeHighlyRelevantOperationHash(&rule), got.RuleID)
	assert.True(t, got.Error)
	assert.Equal(t, intPtr(500), got.DurationAtLeastMs)
	assert.Equal(t, float64Ptr(100), got.PercentageAtLeast)
	require.NotNil(t, got.Operation)
	require.NotNil(t, got.Operation.HTTPServer)
	assert.Equal(t, stringPtr("/charge"), got.Operation.HTTPServer.Route)
}

func TestConvertCostReductionRuleToModelCarriesEveryField(t *testing.T) {
	rule := fullyPopulatedCostReductionRule()

	got := convertCostReductionRuleToModel(&rule)

	assertNoZeroFields(t, got, "CostReductionRule")
	assert.Equal(t, v1alpha1.ComputeCostReductionRuleHash(&rule), got.RuleID)
	assert.Equal(t, 25.0, got.PercentageAtMost)
	require.NotNil(t, got.Operation)
	require.NotNil(t, got.Operation.HTTPServer)
	assert.Equal(t, stringPtr("/orders"), got.Operation.HTTPServer.Route)
}

func TestSamplingRuleConvertersReportUnsetOptionalFieldsAsAbsent(t *testing.T) {
	noisy := convertNoisyOperationToModel(&v1alpha1.NoisyOperation{})
	assert.Nil(t, noisy.Name)
	assert.Nil(t, noisy.Notes)
	assert.Nil(t, noisy.SourceScopes, "an unscoped rule applies to every source")
	assert.Nil(t, noisy.Operation, "a rule with no matcher applies to every operation")
	assert.NotEmpty(t, noisy.RuleID, "even an empty rule gets an addressable id")

	relevant := convertHighlyRelevantOperationToModel(&v1alpha1.HighlyRelevantOperation{})
	assert.Nil(t, relevant.Name)
	assert.Nil(t, relevant.DurationAtLeastMs)
	assert.Nil(t, relevant.Operation)

	cost := convertCostReductionRuleToModel(&v1alpha1.CostReductionRule{})
	assert.Nil(t, cost.Name)
	assert.Nil(t, cost.Operation)
}

// TailSamplingOperationMatcher's kafkaConsumer and kafkaProducer are the same shape with
// the same single field, one line apart in both converters. Distinct topics on both sides
// are what makes a swapped assignment visible.
func TestTailSamplingOperationMatcherKeepsTheKafkaConsumerAndProducerApart(t *testing.T) {
	input := &model.TailSamplingOperationMatcherInput{
		KafkaConsumer: &model.TailSamplingKafkaMatcherInput{KafkaTopic: stringPtr("orders-consumed")},
		KafkaProducer: &model.TailSamplingKafkaMatcherInput{KafkaTopic: stringPtr("orders-produced")},
	}

	crd := tailSamplingOperationMatcherInputToCRD(input)
	require.NotNil(t, crd)
	require.NotNil(t, crd.KafkaConsumer)
	require.NotNil(t, crd.KafkaProducer)
	assert.Equal(t, "orders-consumed", crd.KafkaConsumer.KafkaTopic)
	assert.Equal(t, "orders-produced", crd.KafkaProducer.KafkaTopic)

	back := tailSamplingOperationMatcherCRDToModel(crd)
	require.NotNil(t, back)
	require.NotNil(t, back.KafkaConsumer)
	require.NotNil(t, back.KafkaProducer)
	assert.Equal(t, stringPtr("orders-consumed"), back.KafkaConsumer.KafkaTopic)
	assert.Equal(t, stringPtr("orders-produced"), back.KafkaProducer.KafkaTopic)
}

func TestTailSamplingOperationMatcherKeepsRouteAndRoutePrefixApart(t *testing.T) {
	crd := tailSamplingOperationMatcherInputToCRD(&model.TailSamplingOperationMatcherInput{
		HTTPServer: &model.TailSamplingHTTPServerMatcherInput{
			Route:       stringPtr("/api/orders/{id}"),
			RoutePrefix: stringPtr("/api/orders"),
			Method:      stringPtr("DELETE"),
		},
	})
	require.NotNil(t, crd)
	require.NotNil(t, crd.HttpServer)
	assert.Equal(t, "/api/orders/{id}", crd.HttpServer.Route)
	assert.Equal(t, "/api/orders", crd.HttpServer.RoutePrefix)
	assert.Equal(t, "DELETE", crd.HttpServer.Method)

	back := tailSamplingOperationMatcherCRDToModel(crd)
	require.NotNil(t, back.HTTPServer)
	assert.Equal(t, stringPtr("/api/orders/{id}"), back.HTTPServer.Route)
	assert.Equal(t, stringPtr("/api/orders"), back.HTTPServer.RoutePrefix)
	assert.Equal(t, stringPtr("DELETE"), back.HTTPServer.Method)
}

// Each matcher branch has to survive on its own: an && where the code means two
// independent ifs only shows up when exactly one branch is present.
func TestTailSamplingOperationMatcherConvertsEachBranchOnItsOwn(t *testing.T) {
	cases := map[string]struct {
		input  *model.TailSamplingOperationMatcherInput
		verify func(*testing.T, *commonapisampling.TailSamplingOperationMatcher, *model.TailSamplingOperationMatcher)
	}{
		"httpServerOnly": {
			input: &model.TailSamplingOperationMatcherInput{
				HTTPServer: &model.TailSamplingHTTPServerMatcherInput{Route: stringPtr("/only")},
			},
			verify: func(t *testing.T, crd *commonapisampling.TailSamplingOperationMatcher, back *model.TailSamplingOperationMatcher) {
				require.NotNil(t, crd.HttpServer)
				assert.Equal(t, "/only", crd.HttpServer.Route)
				assert.Nil(t, crd.KafkaConsumer)
				assert.Nil(t, crd.KafkaProducer)
				require.NotNil(t, back.HTTPServer)
				assert.Nil(t, back.KafkaConsumer)
				assert.Nil(t, back.KafkaProducer)
			},
		},
		"kafkaConsumerOnly": {
			input: &model.TailSamplingOperationMatcherInput{
				KafkaConsumer: &model.TailSamplingKafkaMatcherInput{KafkaTopic: stringPtr("only")},
			},
			verify: func(t *testing.T, crd *commonapisampling.TailSamplingOperationMatcher, back *model.TailSamplingOperationMatcher) {
				require.NotNil(t, crd.KafkaConsumer)
				assert.Equal(t, "only", crd.KafkaConsumer.KafkaTopic)
				assert.Nil(t, crd.HttpServer)
				assert.Nil(t, crd.KafkaProducer)
				require.NotNil(t, back.KafkaConsumer)
				assert.Nil(t, back.HTTPServer)
				assert.Nil(t, back.KafkaProducer)
			},
		},
		"kafkaProducerOnly": {
			input: &model.TailSamplingOperationMatcherInput{
				KafkaProducer: &model.TailSamplingKafkaMatcherInput{KafkaTopic: stringPtr("only")},
			},
			verify: func(t *testing.T, crd *commonapisampling.TailSamplingOperationMatcher, back *model.TailSamplingOperationMatcher) {
				require.NotNil(t, crd.KafkaProducer)
				assert.Equal(t, "only", crd.KafkaProducer.KafkaTopic)
				assert.Nil(t, crd.HttpServer)
				assert.Nil(t, crd.KafkaConsumer)
				require.NotNil(t, back.KafkaProducer)
				assert.Nil(t, back.HTTPServer)
				assert.Nil(t, back.KafkaConsumer)
			},
		},
		"noBranchAtAll": {
			input: &model.TailSamplingOperationMatcherInput{},
			verify: func(t *testing.T, crd *commonapisampling.TailSamplingOperationMatcher, back *model.TailSamplingOperationMatcher) {
				assert.Nil(t, crd.HttpServer)
				assert.Nil(t, crd.KafkaConsumer)
				assert.Nil(t, crd.KafkaProducer)
				assert.Nil(t, back.HTTPServer)
				assert.Nil(t, back.KafkaConsumer)
				assert.Nil(t, back.KafkaProducer)
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			crd := tailSamplingOperationMatcherInputToCRD(tc.input)
			require.NotNil(t, crd, "a present matcher input must produce a matcher")
			back := tailSamplingOperationMatcherCRDToModel(crd)
			require.NotNil(t, back)
			tc.verify(t, crd, back)
		})
	}
}

// The head sampling http client matcher has four independently meaningful string fields
// set from four pointers in a row. Four distinct values are what pins each assignment to
// its own field; a single shared value would let any two of them be swapped.
func TestHeadSamplingOperationMatcherKeepsTheFourHTTPClientFieldsApart(t *testing.T) {
	crd := headSamplingOperationMatcherInputToCRD(&model.HeadSamplingOperationMatcherInput{
		HTTPClient: &model.HeadSamplingHTTPClientMatcherInput{
			ServerAddress:       stringPtr("payments.internal:8443"),
			TemplatedPath:       stringPtr("/v1/charge/{id}"),
			TemplatedPathPrefix: stringPtr("/v1/charge"),
			Method:              stringPtr("PUT"),
		},
	})
	require.NotNil(t, crd)
	require.NotNil(t, crd.HttpClient)
	assert.Nil(t, crd.HttpServer, "an http client matcher must not also produce a server matcher")
	assert.Equal(t, "payments.internal:8443", crd.HttpClient.ServerAddress)
	assert.Equal(t, "/v1/charge/{id}", crd.HttpClient.TemplatedPath)
	assert.Equal(t, "/v1/charge", crd.HttpClient.TemplatedPathPrefix)
	assert.Equal(t, "PUT", crd.HttpClient.Method)

	back := headSamplingOperationMatcherCRDToModel(crd)
	require.NotNil(t, back)
	require.NotNil(t, back.HTTPClient)
	assert.Nil(t, back.HTTPServer)
	assert.Equal(t, stringPtr("payments.internal:8443"), back.HTTPClient.ServerAddress)
	assert.Equal(t, stringPtr("/v1/charge/{id}"), back.HTTPClient.TemplatedPath)
	assert.Equal(t, stringPtr("/v1/charge"), back.HTTPClient.TemplatedPathPrefix)
	assert.Equal(t, stringPtr("PUT"), back.HTTPClient.Method)
}

func TestHeadSamplingQueryParamsInputSkipsNilEntriesWithoutLosingTheRest(t *testing.T) {
	got := headSamplingQueryParamsInputToCRD([]*model.HeadSamplingQueryParamMatcherInput{
		nil,
		{Name: "query", ValueExact: stringPtr("{ health }")},
		nil,
		{Name: "operationName"},
	})

	require.Len(t, got, 2, "a nil entry is skipped, the surrounding entries are kept")
	assert.Equal(t, "query", got[0].Name)
	assert.Equal(t, stringPtr("{ health }"), got[0].ValueExact)
	assert.Equal(t, "operationName", got[1].Name)
	assert.Nil(t, got[1].ValueExact)
}

// An omitted matcher means "match every operation" and must stay omitted all the way
// through; turning it into an empty matcher would change what the collector matches.
func TestAnOmittedTailSamplingMatcherStaysOmitted(t *testing.T) {
	assert.Nil(t, tailSamplingOperationMatcherInputToCRD(nil))
	assert.Nil(t, tailSamplingOperationMatcherCRDToModel(nil))
}

func TestTailSamplingOperationMatcherCRDToModelReportsEmptyFieldsAsAbsent(t *testing.T) {
	back := tailSamplingOperationMatcherCRDToModel(&commonapisampling.TailSamplingOperationMatcher{
		HttpServer:    &commonapisampling.TailSamplingHttpServerOperationMatcher{Route: "/kept"},
		KafkaConsumer: &commonapisampling.TailSamplingKafkaOperationMatcher{},
	})

	require.NotNil(t, back.HTTPServer)
	assert.Equal(t, stringPtr("/kept"), back.HTTPServer.Route)
	assert.Nil(t, back.HTTPServer.RoutePrefix, "an unset prefix must not surface as an empty string")
	assert.Nil(t, back.HTTPServer.Method)
	require.NotNil(t, back.KafkaConsumer)
	assert.Nil(t, back.KafkaConsumer.KafkaTopic)
}

// A rule written by the UI has to come back out of the CR as the same rule; the two
// conversions are separate hand-written field lists and only a round trip ties them.
func TestASamplingRuleSurvivesTheInputToCRDToModelRoundTrip(t *testing.T) {
	input := model.NoisyOperationRuleInput{
		Name:     stringPtr("health checks"),
		Disabled: boolPtr(true),
		SourceScopes: &model.SourcesScopesInput{
			Namespaces: []string{"payments"},
			Languages:  []model.SamplingWorkloadLanguage{model.SamplingWorkloadLanguage(common.JavaProgrammingLanguage)},
		},
		Operation: &model.HeadSamplingOperationMatcherInput{
			HTTPServer: &model.HeadSamplingHTTPServerMatcherInput{
				Route:  stringPtr("/healthz"),
				Method: stringPtr("GET"),
			},
		},
		PercentageAtMost: float64Ptr(1),
		Notes:            stringPtr("agreed with payments"),
	}

	stored := noisyOperationFromInput(input)
	got := convertNoisyOperationToModel(&stored)

	assert.Equal(t, input.Name, got.Name)
	assert.Equal(t, *input.Disabled, got.Disabled)
	assert.Equal(t, input.PercentageAtMost, got.PercentageAtMost)
	assert.Equal(t, input.Notes, got.Notes)
	require.NotNil(t, got.SourceScopes)
	assert.Equal(t, []string{"payments"}, got.SourceScopes.Namespaces)
	assert.Equal(t, []model.SamplingWorkloadLanguage{model.SamplingWorkloadLanguage(common.JavaProgrammingLanguage)},
		got.SourceScopes.Languages)
	require.NotNil(t, got.Operation)
	require.NotNil(t, got.Operation.HTTPServer)
	assert.Equal(t, stringPtr("/healthz"), got.Operation.HTTPServer.Route)
	assert.Equal(t, stringPtr("GET"), got.Operation.HTTPServer.Method)
}
