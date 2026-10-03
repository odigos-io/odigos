package graph

import (
	"reflect"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonapi "github.com/odigos-io/odigos/common/api"
	"github.com/odigos-io/odigos/common/api/agentsignalconfig"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func srcStr(value string) *string     { return &value }
func srcBool(value bool) *bool        { return &value }
func srcInt(value int) *int           { return &value }
func srcFloat(value float64) *float64 { return &value }

func srcHeadServerMatcher(route string) *commonapisampling.HeadSamplingOperationMatcher {
	return &commonapisampling.HeadSamplingOperationMatcher{
		HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{Route: route},
	}
}

func srcTailServerMatcher(route string) *commonapisampling.TailSamplingOperationMatcher {
	return &commonapisampling.TailSamplingOperationMatcher{
		HttpServer: &commonapisampling.TailSamplingHttpServerOperationMatcher{Route: route},
	}
}

// assertEveryModelFieldSet is the invariant that catches a per-source converter dropping a
// field. Asserting a hand-written expected struct cannot: a field absent from both the
// converter and the expectation passes.
func assertEveryModelFieldSet(t *testing.T, converted any, describe string) {
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

// ---- rule id pass-through: the contract with the instrumentor ----

// The per-source sampling view is the second place a sampling rule id is surfaced. The
// instrumentor stamps every rule it copies into an InstrumentationConfig with
// ComputeNoisyOperationHash over the user-authored CR rule, and the sampling rules screen
// derives the very same hash from the CR. Both screens address the same rule, so this
// layer must pass the stored id straight through - deriving its own would hand the UI an
// id that the update and delete mutations cannot resolve.
func TestPerSourceSamplingRuleIDsAreThePassedThroughInstrumentorIDs(t *testing.T) {
	authoredNoisy := v1alpha1.NoisyOperation{
		Name:             "health checks",
		SourceScopes:     &k8sconsts.SourcesScopes{Namespaces: []string{"payments"}},
		Operation:        srcHeadServerMatcher("/healthz"),
		PercentageAtMost: srcFloat(1),
	}
	authoredRelevant := v1alpha1.HighlyRelevantOperation{
		Name:         "keep charges",
		SourceScopes: &k8sconsts.SourcesScopes{Namespaces: []string{"payments"}},
		Error:        true,
		Operation:    srcTailServerMatcher("/charge"),
	}
	authoredCost := v1alpha1.CostReductionRule{
		Name:         "drop orders",
		SourceScopes: &k8sconsts.SourcesScopes{Namespaces: []string{"orders"}},
		Operation:    srcTailServerMatcher("/orders"),
	}

	noisyID := v1alpha1.ComputeNoisyOperationHash(&authoredNoisy)
	relevantID := v1alpha1.ComputeHighlyRelevantOperationHash(&authoredRelevant)
	costID := v1alpha1.ComputeCostReductionRuleHash(&authoredCost)

	head := headSamplingNoisyOperationToModel(&commonapisampling.NoisyOperation{
		Id:        noisyID,
		Name:      authoredNoisy.Name,
		Operation: authoredNoisy.Operation,
	})
	require.NotNil(t, head)
	assert.Equal(t, noisyID, head.RuleID,
		"the head sampling view must report the rule id the instrumentor stamped")

	tailNoisy := tailSamplingNoisyOperationToModel(&commonapisampling.NoisyOperation{
		Id:        noisyID,
		Name:      authoredNoisy.Name,
		Operation: authoredNoisy.Operation,
	})
	require.NotNil(t, tailNoisy)
	assert.Equal(t, noisyID, tailNoisy.RuleID)

	tailRelevant := tailSamplingHighlyRelevantOperationToModel(&commonapisampling.HighlyRelevantOperation{
		Id:        relevantID,
		Name:      authoredRelevant.Name,
		Error:     authoredRelevant.Error,
		Operation: authoredRelevant.Operation,
	})
	require.NotNil(t, tailRelevant)
	assert.Equal(t, relevantID, tailRelevant.RuleID)

	tailCost := tailSamplingCostReductionRuleToModel(&commonapisampling.CostReductionRule{
		Id:        costID,
		Name:      authoredCost.Name,
		Operation: authoredCost.Operation,
	})
	require.NotNil(t, tailCost)
	assert.Equal(t, costID, tailCost.RuleID)

	assert.NotEqual(t, noisyID, relevantID, "the fixture must not make the three ids coincide")
	assert.NotEqual(t, relevantID, costID)
}

// TestPerSourceSamplingFallbackIDMatchesAnUnscopedRule covers the branch taken when a rule
// arrives without an id. The fallback hashes the fields the container-level rule still
// carries, which for a rule the user never scoped is exactly the set the sampling rules
// screen hashes - so the two screens agree.
//
// A scoped rule is a different story and is reported as a defect in the PR that adds this
// test: the scope is resolved away before the rule reaches the container config, so the
// fallback can only ever reproduce the id of an unscoped rule.
func TestPerSourceSamplingFallbackIDMatchesAnUnscopedRule(t *testing.T) {
	unscoped := v1alpha1.NoisyOperation{
		Name:             "health checks",
		Operation:        srcHeadServerMatcher("/healthz"),
		PercentageAtMost: srcFloat(1),
	}

	got := headSamplingNoisyOperationToModel(&commonapisampling.NoisyOperation{
		Name:             unscoped.Name,
		Operation:        unscoped.Operation,
		PercentageAtMost: unscoped.PercentageAtMost,
	})

	require.NotNil(t, got)
	assert.Equal(t, v1alpha1.ComputeNoisyOperationHash(&unscoped), got.RuleID,
		"a rule with no stored id falls back to the same hash the sampling rules screen uses")
	assert.NotEmpty(t, got.RuleID)
}

func TestPerSourceSamplingPrefersTheStoredIDOverTheFallback(t *testing.T) {
	rule := &commonapisampling.NoisyOperation{
		Id:        "stamped-by-the-instrumentor",
		Name:      "health checks",
		Operation: srcHeadServerMatcher("/healthz"),
	}

	assert.Equal(t, "stamped-by-the-instrumentor", headSamplingNoisyOperationToModel(rule).RuleID)
	assert.Equal(t, "stamped-by-the-instrumentor", tailSamplingNoisyOperationToModel(rule).RuleID)
	assert.Equal(t, "stamped-by-the-instrumentor",
		tailSamplingHighlyRelevantOperationToModel(&commonapisampling.HighlyRelevantOperation{
			Id: "stamped-by-the-instrumentor",
		}).RuleID)
	assert.Equal(t, "stamped-by-the-instrumentor",
		tailSamplingCostReductionRuleToModel(&commonapisampling.CostReductionRule{
			Id: "stamped-by-the-instrumentor",
		}).RuleID)
}

// A tail-sampling noisy operation reaches the same rule from the collector side, so its
// fallback has to agree with the head sampling one: the same rule seen from the agent and
// from the collector must carry the same id, or the two views of one source disagree about
// which rule the user is looking at.
func TestTheHeadAndTailViewsOfOneNoisyOperationAgreeOnItsFallbackID(t *testing.T) {
	rule := &commonapisampling.NoisyOperation{
		Name:             "health checks",
		Operation:        srcHeadServerMatcher("/healthz"),
		PercentageAtMost: srcFloat(1),
	}

	head := headSamplingNoisyOperationToModel(rule)
	tail := tailSamplingNoisyOperationToModel(rule)

	require.NotNil(t, head)
	require.NotNil(t, tail)
	assert.NotEmpty(t, tail.RuleID)
	assert.Equal(t, head.RuleID, tail.RuleID)
	assert.Equal(t, v1alpha1.ComputeNoisyOperationHash(&v1alpha1.NoisyOperation{
		Name:             rule.Name,
		Operation:        rule.Operation,
		PercentageAtMost: rule.PercentageAtMost,
	}), tail.RuleID)
}

// The three fallbacks hash three different structs. A fallback that hashed the wrong
// family's struct would return an id that resolves to nothing, so each one is checked
// against its own producer over a fixture that makes the three hashes differ.
func TestEachPerSourceSamplingFallbackHashesItsOwnRuleFamily(t *testing.T) {
	matcher := srcTailServerMatcher("/charge")

	relevant := tailSamplingHighlyRelevantOperationToModel(&commonapisampling.HighlyRelevantOperation{
		Error:             true,
		DurationAtLeastMs: srcInt(500),
		Operation:         matcher,
	})
	require.NotNil(t, relevant)
	assert.Equal(t, v1alpha1.ComputeHighlyRelevantOperationHash(&v1alpha1.HighlyRelevantOperation{
		Error:             true,
		DurationAtLeastMs: srcInt(500),
		Operation:         matcher,
	}), relevant.RuleID)

	cost := tailSamplingCostReductionRuleToModel(&commonapisampling.CostReductionRule{
		Operation: matcher,
	})
	require.NotNil(t, cost)
	assert.Equal(t, v1alpha1.ComputeCostReductionRuleHash(&v1alpha1.CostReductionRule{
		Operation: matcher,
	}), cost.RuleID)

	assert.NotEqual(t, relevant.RuleID, cost.RuleID,
		"two rule families sharing a matcher must not share an id")
}

// ---- container agent config (head sampling) ----

func TestContainerAgentConfigToAgentConfigModelCarriesTheWholeHeadSamplingTree(t *testing.T) {
	got := containerAgentConfigToAgentConfigModel(&v1alpha1.ContainerAgentConfig{
		ContainerName: "app",
		Traces: &agentsignalconfig.AgentTracesConfig{
			HeadSampling: &commonapisampling.HeadSamplingConfig{
				DryRun:          true,
				SpanMetricsMode: commonapisampling.SpanMetricsModeAllSpans,
				NoisyOperations: []commonapisampling.NoisyOperation{
					{Id: "rule-a", Name: "health", Disabled: true, Operation: srcHeadServerMatcher("/healthz"), PercentageAtMost: srcFloat(1)},
					{Id: "rule-b", Name: "metrics", Operation: srcHeadServerMatcher("/metrics")},
				},
			},
		},
	})

	require.NotNil(t, got)
	require.NotNil(t, got.Traces)
	require.NotNil(t, got.Traces.HeadSampling)
	head := got.Traces.HeadSampling
	assert.Equal(t, srcBool(true), head.DryRun)
	require.NotNil(t, head.SpanMetricsMode)
	assert.Equal(t, model.K8sWorkloadContainerAgentConfigTracesHeadSamplingSpanMetricsModeAllSpans, *head.SpanMetricsMode)
	require.Len(t, head.NoisyOperations, 2)
	assert.Equal(t, "rule-a", head.NoisyOperations[0].RuleID)
	assert.Equal(t, srcStr("health"), head.NoisyOperations[0].Name)
	assert.True(t, head.NoisyOperations[0].Disabled)
	assert.Equal(t, srcFloat(1), head.NoisyOperations[0].PercentageAtMost)
	assert.Equal(t, "rule-b", head.NoisyOperations[1].RuleID,
		"the rules must keep the order the instrumentor wrote them in")
	assert.False(t, head.NoisyOperations[1].Disabled)
}

// "this container has traces configured but no head sampling" and "this container has no
// traces at all" are different answers for the UI, and the converter reports them as an
// empty traces block versus nothing at all.
func TestContainerAgentConfigToAgentConfigModelSeparatesNoTracesFromNoHeadSampling(t *testing.T) {
	assert.Nil(t, containerAgentConfigToAgentConfigModel(nil),
		"no container config at all is reported as absent")
	assert.Nil(t, containerAgentConfigToAgentConfigModel(&v1alpha1.ContainerAgentConfig{ContainerName: "app"}),
		"a container with no traces config is reported as absent")

	noHeadSampling := containerAgentConfigToAgentConfigModel(&v1alpha1.ContainerAgentConfig{
		ContainerName: "app",
		Traces:        &agentsignalconfig.AgentTracesConfig{},
	})
	require.NotNil(t, noHeadSampling, "traces configured without head sampling is still a traces config")
	require.NotNil(t, noHeadSampling.Traces)
	assert.Nil(t, noHeadSampling.Traces.HeadSampling)
}

// dryRun is only surfaced when it is on, so the UI shows "not in dry run" as an absent
// field rather than an explicit false. Reporting ptrBool(false) instead would light up the
// dry-run banner for every source.
func TestHeadSamplingConfigOmitsEveryUnsetField(t *testing.T) {
	got := headSamplingConfigToModel(&commonapisampling.HeadSamplingConfig{})

	require.NotNil(t, got)
	assert.Nil(t, got.DryRun, "dry run off is reported as absent, never as an explicit false")
	assert.Nil(t, got.SpanMetricsMode, "an unset span metrics mode must not be defaulted here")
	assert.Nil(t, got.NoisyOperations)
}

func TestHeadSamplingConfigToModelReportsAnAbsentConfigAsAbsent(t *testing.T) {
	assert.Nil(t, headSamplingConfigToModel(nil))
}

func TestHeadSamplingConfigToModelSetsOneFieldAtATime(t *testing.T) {
	cases := map[string]struct {
		config *commonapisampling.HeadSamplingConfig
		verify func(*testing.T, *model.K8sWorkloadContainerAgentConfigTracesHeadSampling)
	}{
		"onlyDryRun": {
			config: &commonapisampling.HeadSamplingConfig{DryRun: true},
			verify: func(t *testing.T, got *model.K8sWorkloadContainerAgentConfigTracesHeadSampling) {
				assert.Equal(t, srcBool(true), got.DryRun)
				assert.Nil(t, got.SpanMetricsMode)
				assert.Nil(t, got.NoisyOperations)
			},
		},
		"onlySpanMetricsMode": {
			config: &commonapisampling.HeadSamplingConfig{SpanMetricsMode: commonapisampling.SpanMetricsModeAllSpans},
			verify: func(t *testing.T, got *model.K8sWorkloadContainerAgentConfigTracesHeadSampling) {
				assert.Nil(t, got.DryRun)
				require.NotNil(t, got.SpanMetricsMode)
				assert.Nil(t, got.NoisyOperations)
			},
		},
		"onlyNoisyOperations": {
			config: &commonapisampling.HeadSamplingConfig{
				NoisyOperations: []commonapisampling.NoisyOperation{{Id: "rule", Operation: srcHeadServerMatcher("/healthz")}},
			},
			verify: func(t *testing.T, got *model.K8sWorkloadContainerAgentConfigTracesHeadSampling) {
				assert.Nil(t, got.DryRun)
				assert.Nil(t, got.SpanMetricsMode)
				require.Len(t, got.NoisyOperations, 1)
				assert.Equal(t, "rule", got.NoisyOperations[0].RuleID)
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := headSamplingConfigToModel(tc.config)
			require.NotNil(t, got)
			tc.verify(t, got)
		})
	}
}

// The span metrics mode decides whether unsampled spans are forwarded for metric
// computation, so the two wire values the collector accepts are a contract and are pinned
// as literals. Anything the collector does not know maps to the cheaper of the two rather
// than surfacing as an unknown enum the UI cannot render.
func TestHeadSamplingSpanMetricsModeMapsEveryCollectorValue(t *testing.T) {
	assert.Equal(t, commonapisampling.SpanMetricsMode("all-spans"), commonapisampling.SpanMetricsModeAllSpans)
	assert.Equal(t, commonapisampling.SpanMetricsMode("sampled-spans-only"), commonapisampling.SpanMetricsModeSampledSpansOnly)

	assert.Equal(t,
		model.K8sWorkloadContainerAgentConfigTracesHeadSamplingSpanMetricsModeAllSpans,
		headSamplingSpanMetricsModeToModel(commonapisampling.SpanMetricsModeAllSpans))
	assert.Equal(t,
		model.K8sWorkloadContainerAgentConfigTracesHeadSamplingSpanMetricsModeSampledSpansOnly,
		headSamplingSpanMetricsModeToModel(commonapisampling.SpanMetricsModeSampledSpansOnly))
	assert.Equal(t,
		model.K8sWorkloadContainerAgentConfigTracesHeadSamplingSpanMetricsModeSampledSpansOnly,
		headSamplingSpanMetricsModeToModel(commonapisampling.SpanMetricsMode("statistical")),
		"an unrecognised mode falls back to sampled-spans-only rather than an unrenderable value")
}

func TestHeadSamplingNoisyOperationCarriesEveryField(t *testing.T) {
	got := headSamplingNoisyOperationToModel(&commonapisampling.NoisyOperation{
		Id:               "rule-a",
		Name:             "health checks",
		Disabled:         true,
		Operation:        srcHeadServerMatcher("/healthz"),
		PercentageAtMost: srcFloat(1),
	})

	assertEveryModelFieldSet(t, got, "K8sWorkloadContainerAgentConfigTracesHeadSamplingNoisyOperation")
	assert.Equal(t, "rule-a", got.RuleID)
	assert.Equal(t, srcStr("health checks"), got.Name)
	require.NotNil(t, got.Operation)
	require.NotNil(t, got.Operation.HTTPServer)
	assert.Equal(t, srcStr("/healthz"), got.Operation.HTTPServer.Route)
}

func TestHeadSamplingNoisyOperationOmitsAnEmptyName(t *testing.T) {
	got := headSamplingNoisyOperationToModel(&commonapisampling.NoisyOperation{Id: "rule-a"})

	require.NotNil(t, got)
	assert.Nil(t, got.Name, "an unnamed rule is reported as absent, not as an empty string")
	assert.Nil(t, got.Operation)
	assert.Nil(t, got.PercentageAtMost)
}

func TestHeadSamplingNoisyOperationReportsAnAbsentRuleAsAbsent(t *testing.T) {
	assert.Nil(t, headSamplingNoisyOperationToModel(nil))
}

// ---- container collector config (tail sampling) ----

func TestContainerCollectorConfigToModelCarriesAllThreeTailSamplingFamilies(t *testing.T) {
	got := containerCollectorConfigToModel(&commonapi.ContainerCollectorConfig{
		ContainerName: "app",
		TailSampling: &commonapisampling.TailSamplingSourceConfig{
			NoisyOperations: []commonapisampling.NoisyOperation{
				{Id: "noisy-a", Name: "health", Operation: srcHeadServerMatcher("/healthz")},
			},
			HighlyRelevantOperations: []commonapisampling.HighlyRelevantOperation{
				{Id: "relevant-a", Name: "charges", Error: true, Operation: srcTailServerMatcher("/charge")},
				{Id: "relevant-b", Name: "refunds", Operation: srcTailServerMatcher("/refund")},
			},
			CostReductionRules: []commonapisampling.CostReductionRule{
				{Id: "cost-a", Name: "orders", Operation: srcTailServerMatcher("/orders"), PercentageAtMost: 10},
			},
		},
	})

	require.NotNil(t, got)
	require.NotNil(t, got.TailSampling)
	tail := got.TailSampling
	require.Len(t, tail.NoisyOperations, 1)
	assert.Equal(t, "noisy-a", tail.NoisyOperations[0].RuleID)
	require.Len(t, tail.HighlyRelevantOperations, 2)
	assert.Equal(t, "relevant-a", tail.HighlyRelevantOperations[0].RuleID)
	assert.Equal(t, "relevant-b", tail.HighlyRelevantOperations[1].RuleID,
		"the rules must keep their stored order")
	require.Len(t, tail.CostReductionRules, 1)
	assert.Equal(t, "cost-a", tail.CostReductionRules[0].RuleID)
	assert.Equal(t, 10.0, tail.CostReductionRules[0].PercentageAtMost)
}

func TestContainerCollectorConfigToModelSeparatesNoConfigFromNoTailSampling(t *testing.T) {
	assert.Nil(t, containerCollectorConfigToModel(nil))

	noTailSampling := containerCollectorConfigToModel(&commonapi.ContainerCollectorConfig{ContainerName: "app"})
	require.NotNil(t, noTailSampling, "a collector config without tail sampling is still a collector config")
	assert.Nil(t, noTailSampling.TailSampling)
}

func TestTailSamplingSourceConfigToModelReportsAnAbsentConfigAsAbsent(t *testing.T) {
	assert.Nil(t, tailSamplingSourceConfigToModel(nil))
}

// The three rule families are built by three near-identical loops over three slices. A
// fixture carrying only one family at a time is what makes a loop reading the wrong slice
// visible.
func TestTailSamplingSourceConfigToModelBuildsEachFamilyIndependently(t *testing.T) {
	cases := map[string]struct {
		config *commonapisampling.TailSamplingSourceConfig
		verify func(*testing.T, *model.K8sWorkloadContainerCollectorConfigTailSampling)
	}{
		"onlyNoisyOperations": {
			config: &commonapisampling.TailSamplingSourceConfig{
				NoisyOperations: []commonapisampling.NoisyOperation{{Id: "noisy"}},
			},
			verify: func(t *testing.T, got *model.K8sWorkloadContainerCollectorConfigTailSampling) {
				require.Len(t, got.NoisyOperations, 1)
				assert.Equal(t, "noisy", got.NoisyOperations[0].RuleID)
				assert.Nil(t, got.HighlyRelevantOperations)
				assert.Nil(t, got.CostReductionRules)
			},
		},
		"onlyHighlyRelevantOperations": {
			config: &commonapisampling.TailSamplingSourceConfig{
				HighlyRelevantOperations: []commonapisampling.HighlyRelevantOperation{{Id: "relevant"}},
			},
			verify: func(t *testing.T, got *model.K8sWorkloadContainerCollectorConfigTailSampling) {
				assert.Nil(t, got.NoisyOperations)
				require.Len(t, got.HighlyRelevantOperations, 1)
				assert.Equal(t, "relevant", got.HighlyRelevantOperations[0].RuleID)
				assert.Nil(t, got.CostReductionRules)
			},
		},
		"onlyCostReductionRules": {
			config: &commonapisampling.TailSamplingSourceConfig{
				CostReductionRules: []commonapisampling.CostReductionRule{{Id: "cost"}},
			},
			verify: func(t *testing.T, got *model.K8sWorkloadContainerCollectorConfigTailSampling) {
				assert.Nil(t, got.NoisyOperations)
				assert.Nil(t, got.HighlyRelevantOperations)
				require.Len(t, got.CostReductionRules, 1)
				assert.Equal(t, "cost", got.CostReductionRules[0].RuleID)
			},
		},
		"noRulesAtAll": {
			config: &commonapisampling.TailSamplingSourceConfig{},
			verify: func(t *testing.T, got *model.K8sWorkloadContainerCollectorConfigTailSampling) {
				assert.Nil(t, got.NoisyOperations)
				assert.Nil(t, got.HighlyRelevantOperations)
				assert.Nil(t, got.CostReductionRules)
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := tailSamplingSourceConfigToModel(tc.config)
			require.NotNil(t, got)
			tc.verify(t, got)
		})
	}
}

func TestTailSamplingNoisyOperationCarriesEveryField(t *testing.T) {
	got := tailSamplingNoisyOperationToModel(&commonapisampling.NoisyOperation{
		Id:               "noisy-a",
		Name:             "health checks",
		Disabled:         true,
		Operation:        srcHeadServerMatcher("/healthz"),
		PercentageAtMost: srcFloat(1),
	})

	assertEveryModelFieldSet(t, got, "K8sWorkloadContainerCollectorConfigTailSamplingNoisyOperation")
	assert.Equal(t, "noisy-a", got.RuleID)
}

// A tail-sampling noisy operation carries a head sampling matcher, so it is the only tail
// rule whose matcher can express an http client call. Its converter has to be the head
// matcher one; the tail matcher cannot represent httpClient at all and would drop it.
func TestTailSamplingNoisyOperationUsesTheHeadMatcherSoHTTPClientSurvives(t *testing.T) {
	got := tailSamplingNoisyOperationToModel(&commonapisampling.NoisyOperation{
		Id: "noisy-a",
		Operation: &commonapisampling.HeadSamplingOperationMatcher{
			HttpClient: &commonapisampling.HeadSamplingHttpClientOperationMatcher{
				ServerAddress: "payments.internal:8443",
				TemplatedPath: "/v1/charge/{id}",
			},
		},
	})

	require.NotNil(t, got)
	require.NotNil(t, got.Operation)
	require.NotNil(t, got.Operation.HTTPClient)
	assert.Equal(t, srcStr("payments.internal:8443"), got.Operation.HTTPClient.ServerAddress)
	assert.Equal(t, srcStr("/v1/charge/{id}"), got.Operation.HTTPClient.TemplatedPath)
}

func TestTailSamplingHighlyRelevantOperationCarriesEveryField(t *testing.T) {
	got := tailSamplingHighlyRelevantOperationToModel(&commonapisampling.HighlyRelevantOperation{
		Id:                "relevant-a",
		Name:              "keep failing charges",
		Disabled:          true,
		Error:             true,
		DurationAtLeastMs: srcInt(500),
		Operation:         srcTailServerMatcher("/charge"),
		PercentageAtLeast: srcFloat(100),
	})

	assertEveryModelFieldSet(t, got, "K8sWorkloadContainerCollectorConfigTailSamplingHighlyRelevantOperation")
	assert.Equal(t, "relevant-a", got.RuleID)
	assert.Equal(t, srcInt(500), got.DurationAtLeastMs)
	assert.Equal(t, srcFloat(100), got.PercentageAtLeast)
}

func TestTailSamplingCostReductionRuleCarriesEveryField(t *testing.T) {
	got := tailSamplingCostReductionRuleToModel(&commonapisampling.CostReductionRule{
		Id:               "cost-a",
		Name:             "drop order noise",
		Disabled:         true,
		Operation:        srcTailServerMatcher("/orders"),
		PercentageAtMost: 25,
	})

	assertEveryModelFieldSet(t, got, "K8sWorkloadContainerCollectorConfigTailSamplingCostReductionRule")
	assert.Equal(t, "cost-a", got.RuleID)
	assert.Equal(t, 25.0, got.PercentageAtMost)
}

func TestTailSamplingRuleConvertersReportAnAbsentRuleAsAbsent(t *testing.T) {
	assert.Nil(t, tailSamplingNoisyOperationToModel(nil))
	assert.Nil(t, tailSamplingHighlyRelevantOperationToModel(nil))
	assert.Nil(t, tailSamplingCostReductionRuleToModel(nil))
}

func TestTailSamplingRuleConvertersOmitEmptyNames(t *testing.T) {
	assert.Nil(t, tailSamplingNoisyOperationToModel(&commonapisampling.NoisyOperation{Id: "a"}).Name)
	assert.Nil(t, tailSamplingHighlyRelevantOperationToModel(&commonapisampling.HighlyRelevantOperation{Id: "a"}).Name)
	assert.Nil(t, tailSamplingCostReductionRuleToModel(&commonapisampling.CostReductionRule{Id: "a"}).Name)
}

// kafkaConsumer and kafkaProducer are the same one-field shape, set one line apart.
// Distinct topics are what pins each to its own field.
func TestTailSamplingOperationMatcherKeepsConsumerProducerAndServerApart(t *testing.T) {
	got := tailSamplingOperationMatcherToModel(&commonapisampling.TailSamplingOperationMatcher{
		HttpServer: &commonapisampling.TailSamplingHttpServerOperationMatcher{
			Route:       "/api/orders/{id}",
			RoutePrefix: "/api/orders",
			Method:      "DELETE",
		},
		KafkaConsumer: &commonapisampling.TailSamplingKafkaOperationMatcher{KafkaTopic: "orders-consumed"},
		KafkaProducer: &commonapisampling.TailSamplingKafkaOperationMatcher{KafkaTopic: "orders-produced"},
	})

	require.NotNil(t, got)
	require.NotNil(t, got.HTTPServer)
	assert.Equal(t, srcStr("/api/orders/{id}"), got.HTTPServer.Route)
	assert.Equal(t, srcStr("/api/orders"), got.HTTPServer.RoutePrefix)
	assert.Equal(t, srcStr("DELETE"), got.HTTPServer.Method)
	require.NotNil(t, got.KafkaConsumer)
	assert.Equal(t, srcStr("orders-consumed"), got.KafkaConsumer.KafkaTopic)
	require.NotNil(t, got.KafkaProducer)
	assert.Equal(t, srcStr("orders-produced"), got.KafkaProducer.KafkaTopic)
}

func TestTailSamplingOperationMatcherConvertsEachBranchAlone(t *testing.T) {
	serverOnly := tailSamplingOperationMatcherToModel(srcTailServerMatcher("/only"))
	require.NotNil(t, serverOnly.HTTPServer)
	assert.Nil(t, serverOnly.KafkaConsumer)
	assert.Nil(t, serverOnly.KafkaProducer)

	consumerOnly := tailSamplingOperationMatcherToModel(&commonapisampling.TailSamplingOperationMatcher{
		KafkaConsumer: &commonapisampling.TailSamplingKafkaOperationMatcher{KafkaTopic: "only"},
	})
	assert.Nil(t, consumerOnly.HTTPServer)
	require.NotNil(t, consumerOnly.KafkaConsumer)
	assert.Nil(t, consumerOnly.KafkaProducer)

	producerOnly := tailSamplingOperationMatcherToModel(&commonapisampling.TailSamplingOperationMatcher{
		KafkaProducer: &commonapisampling.TailSamplingKafkaOperationMatcher{KafkaTopic: "only"},
	})
	assert.Nil(t, producerOnly.HTTPServer)
	assert.Nil(t, producerOnly.KafkaConsumer)
	require.NotNil(t, producerOnly.KafkaProducer)

	empty := tailSamplingOperationMatcherToModel(&commonapisampling.TailSamplingOperationMatcher{})
	require.NotNil(t, empty)
	assert.Nil(t, empty.HTTPServer)
	assert.Nil(t, empty.KafkaConsumer)
	assert.Nil(t, empty.KafkaProducer)

	assert.Nil(t, tailSamplingOperationMatcherToModel(nil),
		"an omitted matcher means every operation and must stay omitted")
}

func TestHeadSamplingOperationMatcherKeepsTheServerAndClientBranchesApart(t *testing.T) {
	got := headSamplingOperationMatcherToModel(&commonapisampling.HeadSamplingOperationMatcher{
		HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{
			Route:       "/api/orders/{id}",
			RoutePrefix: "/api/orders",
			Method:      "GET",
		},
		HttpClient: &commonapisampling.HeadSamplingHttpClientOperationMatcher{
			ServerAddress:       "payments.internal:8443",
			TemplatedPath:       "/v1/charge/{id}",
			TemplatedPathPrefix: "/v1/charge",
			Method:              "PUT",
		},
	})

	require.NotNil(t, got)
	require.NotNil(t, got.HTTPServer)
	assert.Equal(t, srcStr("/api/orders/{id}"), got.HTTPServer.Route)
	assert.Equal(t, srcStr("/api/orders"), got.HTTPServer.RoutePrefix)
	assert.Equal(t, srcStr("GET"), got.HTTPServer.Method)
	require.NotNil(t, got.HTTPClient)
	assert.Equal(t, srcStr("payments.internal:8443"), got.HTTPClient.ServerAddress)
	assert.Equal(t, srcStr("/v1/charge/{id}"), got.HTTPClient.TemplatedPath)
	assert.Equal(t, srcStr("/v1/charge"), got.HTTPClient.TemplatedPathPrefix)
	assert.Equal(t, srcStr("PUT"), got.HTTPClient.Method)
}

func TestHeadSamplingOperationMatcherReportsEmptyFieldsAsAbsent(t *testing.T) {
	assert.Nil(t, headSamplingOperationMatcherToModel(nil))

	empty := headSamplingOperationMatcherToModel(&commonapisampling.HeadSamplingOperationMatcher{
		HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{Route: "/kept"},
	})
	require.NotNil(t, empty.HTTPServer)
	assert.Equal(t, srcStr("/kept"), empty.HTTPServer.Route)
	assert.Nil(t, empty.HTTPServer.RoutePrefix)
	assert.Nil(t, empty.HTTPServer.Method)
	assert.Nil(t, empty.HTTPServer.QueryParams)
	assert.Nil(t, empty.HTTPClient)
}

func TestHeadSamplingQueryParamsToModelReportsAnEmptyListAsAbsent(t *testing.T) {
	assert.Nil(t, headSamplingQueryParamsToModel(nil))
	assert.Nil(t, headSamplingQueryParamsToModel([]commonapisampling.QueryParamMatcher{}))
}

// ---- cluster-wide sampling configuration ----

func TestConvertOdigosConfigToSamplingConfigCarriesEveryBlock(t *testing.T) {
	got := convertOdigosConfigToSamplingConfig(&common.OdigosConfiguration{
		Sampling: &common.SamplingConfiguration{
			DryRun: srcBool(true),
			SpanSamplingAttributes: &commonapisampling.SpanSamplingAttributesConfiguration{
				Disabled:                       srcBool(true),
				SamplingCategoryDisabled:       srcBool(true),
				TraceDecidingRuleDisabled:      srcBool(true),
				SpanDecisionAttributesDisabled: srcBool(true),
			},
			TailSampling: &commonapisampling.TailSamplingConfiguration{
				Disabled:                     srcBool(true),
				TraceAggregationWaitDuration: srcStr("10s"),
			},
			K8sHealthProbesSampling: &common.K8sHealthProbesSamplingConfiguration{
				Enabled:        srcBool(true),
				KeepPercentage: srcFloat(1),
			},
		},
	})

	require.NotNil(t, got)
	assertEveryModelFieldSet(t, got, "SamplingConfig")
	assertEveryModelFieldSet(t, got.SpanSamplingAttributes, "SpanSamplingAttributesConfig")
	assertEveryModelFieldSet(t, got.TailSampling, "TailSamplingConfig")
	assertEveryModelFieldSet(t, got.K8sHealthProbesSampling, "K8sHealthProbesSamplingConfig")
	assert.Equal(t, srcStr("10s"), got.TailSampling.TraceAggregationWaitDuration)
	assert.Equal(t, srcFloat(1), got.K8sHealthProbesSampling.KeepPercentage)
}

func TestConvertOdigosConfigToSamplingConfigReportsAnUnconfiguredClusterAsAbsent(t *testing.T) {
	assert.Nil(t, convertOdigosConfigToSamplingConfig(nil))
	assert.Nil(t, convertOdigosConfigToSamplingConfig(&common.OdigosConfiguration{}),
		"a cluster with no sampling block has no sampling config to show")
}

// The four span sampling attribute switches are all *bool and sit in four consecutive
// lines. One fully populated fixture cannot prove each is wired to its own field, so each
// case sets exactly one switch and requires the other three to stay absent.
func TestConvertOdigosConfigToSamplingConfigWiresEachSwitchToItsOwnField(t *testing.T) {
	attributeSwitches := map[string]struct {
		set    func(*commonapisampling.SpanSamplingAttributesConfiguration)
		expect func(*model.SpanSamplingAttributesConfig) *bool
	}{
		"disabled": {
			set:    func(c *commonapisampling.SpanSamplingAttributesConfiguration) { c.Disabled = srcBool(true) },
			expect: func(m *model.SpanSamplingAttributesConfig) *bool { return m.Disabled },
		},
		"samplingCategoryDisabled": {
			set: func(c *commonapisampling.SpanSamplingAttributesConfiguration) {
				c.SamplingCategoryDisabled = srcBool(true)
			},
			expect: func(m *model.SpanSamplingAttributesConfig) *bool { return m.SamplingCategoryDisabled },
		},
		"traceDecidingRuleDisabled": {
			set: func(c *commonapisampling.SpanSamplingAttributesConfiguration) {
				c.TraceDecidingRuleDisabled = srcBool(true)
			},
			expect: func(m *model.SpanSamplingAttributesConfig) *bool { return m.TraceDecidingRuleDisabled },
		},
		"spanDecisionAttributesDisabled": {
			set: func(c *commonapisampling.SpanSamplingAttributesConfiguration) {
				c.SpanDecisionAttributesDisabled = srcBool(true)
			},
			expect: func(m *model.SpanSamplingAttributesConfig) *bool { return m.SpanDecisionAttributesDisabled },
		},
	}

	for name, tc := range attributeSwitches {
		t.Run(name, func(t *testing.T) {
			attributes := &commonapisampling.SpanSamplingAttributesConfiguration{}
			tc.set(attributes)

			got := convertOdigosConfigToSamplingConfig(&common.OdigosConfiguration{
				Sampling: &common.SamplingConfiguration{SpanSamplingAttributes: attributes},
			})

			require.NotNil(t, got)
			require.NotNil(t, got.SpanSamplingAttributes)
			assert.Equal(t, srcBool(true), tc.expect(got.SpanSamplingAttributes),
				"%s must be surfaced on the matching model field", name)

			set := 0
			surfaced := reflect.ValueOf(*got.SpanSamplingAttributes)
			for i := 0; i < surfaced.NumField(); i++ {
				if !surfaced.Field(i).IsZero() {
					set++
				}
			}
			assert.Equal(t, 1, set, "setting one switch must surface exactly one model field")
		})
	}
}

func TestConvertOdigosConfigToSamplingConfigKeepsUnsetBlocksAbsent(t *testing.T) {
	got := convertOdigosConfigToSamplingConfig(&common.OdigosConfiguration{
		Sampling: &common.SamplingConfiguration{
			TailSampling: &commonapisampling.TailSamplingConfiguration{Disabled: srcBool(true)},
		},
	})

	require.NotNil(t, got)
	assert.Nil(t, got.DryRun)
	assert.Nil(t, got.SpanSamplingAttributes)
	require.NotNil(t, got.TailSampling)
	assert.Equal(t, srcBool(true), got.TailSampling.Disabled)
	assert.Nil(t, got.TailSampling.TraceAggregationWaitDuration)
	assert.Nil(t, got.K8sHealthProbesSampling)
}

func TestConvertSamplingConfigInputToOdigosConfigCarriesTheWritableBlocks(t *testing.T) {
	got := convertSamplingConfigInputToOdigosConfig(&model.SamplingConfigInput{
		TailSampling: &model.TailSamplingConfigInput{
			Disabled:                     srcBool(true),
			TraceAggregationWaitDuration: srcStr("15s"),
		},
		K8sHealthProbesSampling: &model.K8sHealthProbesSamplingConfigInput{
			Enabled:        srcBool(true),
			KeepPercentage: srcFloat(2),
		},
	})

	require.NotNil(t, got)
	require.NotNil(t, got.TailSampling)
	assert.Equal(t, srcBool(true), got.TailSampling.Disabled)
	assert.Equal(t, srcStr("15s"), got.TailSampling.TraceAggregationWaitDuration)
	require.NotNil(t, got.K8sHealthProbesSampling)
	assert.Equal(t, srcBool(true), got.K8sHealthProbesSampling.Enabled)
	assert.Equal(t, srcFloat(2), got.K8sHealthProbesSampling.KeepPercentage)
	assert.Nil(t, got.DryRun, "the input cannot express dryRun, so it must be left unset rather than defaulted")
	assert.Nil(t, got.SpanSamplingAttributes)
}

func TestConvertSamplingConfigInputToOdigosConfigConvertsEachBlockAlone(t *testing.T) {
	assert.Nil(t, convertSamplingConfigInputToOdigosConfig(nil))

	empty := convertSamplingConfigInputToOdigosConfig(&model.SamplingConfigInput{})
	require.NotNil(t, empty, "an empty input still produces a sampling block to persist")
	assert.Nil(t, empty.TailSampling)
	assert.Nil(t, empty.K8sHealthProbesSampling)

	tailOnly := convertSamplingConfigInputToOdigosConfig(&model.SamplingConfigInput{
		TailSampling: &model.TailSamplingConfigInput{Disabled: srcBool(true)},
	})
	require.NotNil(t, tailOnly.TailSampling)
	assert.Nil(t, tailOnly.K8sHealthProbesSampling)

	probesOnly := convertSamplingConfigInputToOdigosConfig(&model.SamplingConfigInput{
		K8sHealthProbesSampling: &model.K8sHealthProbesSamplingConfigInput{Enabled: srcBool(true)},
	})
	assert.Nil(t, probesOnly.TailSampling)
	require.NotNil(t, probesOnly.K8sHealthProbesSampling)
}

// The sampling screen writes through convertSamplingConfigInputToOdigosConfig and reads
// back through convertOdigosConfigToSamplingConfig. The two are separate hand-written field
// lists over the same document, so a field added to one and not the other means the UI
// silently discards what the user just saved.
func TestTheSamplingConfigWrittenByTheUIIsTheSamplingConfigReadBack(t *testing.T) {
	input := &model.SamplingConfigInput{
		TailSampling: &model.TailSamplingConfigInput{
			Disabled:                     srcBool(true),
			TraceAggregationWaitDuration: srcStr("12s"),
		},
		K8sHealthProbesSampling: &model.K8sHealthProbesSamplingConfigInput{
			Enabled:        srcBool(false),
			KeepPercentage: srcFloat(3.5),
		},
	}

	readBack := convertOdigosConfigToSamplingConfig(&common.OdigosConfiguration{
		Sampling: convertSamplingConfigInputToOdigosConfig(input),
	})

	require.NotNil(t, readBack)
	require.NotNil(t, readBack.TailSampling)
	assert.Equal(t, input.TailSampling.Disabled, readBack.TailSampling.Disabled)
	assert.Equal(t, input.TailSampling.TraceAggregationWaitDuration, readBack.TailSampling.TraceAggregationWaitDuration)
	require.NotNil(t, readBack.K8sHealthProbesSampling)
	assert.Equal(t, input.K8sHealthProbesSampling.Enabled, readBack.K8sHealthProbesSampling.Enabled)
	assert.Equal(t, input.K8sHealthProbesSampling.KeepPercentage, readBack.K8sHealthProbesSampling.KeepPercentage)
}

// TestEverySamplingConfigInputFieldIsPersisted is the completeness half of the round trip
// above: it fails when the input type grows a field that
// convertSamplingConfigInputToOdigosConfig does not copy, which would otherwise be a
// silently ignored setting rather than a test failure.
func TestEverySamplingConfigInputFieldIsPersisted(t *testing.T) {
	persistedByInputField := map[string]func(*common.SamplingConfiguration) bool{
		"TailSampling": func(c *common.SamplingConfiguration) bool {
			return c.TailSampling != nil &&
				c.TailSampling.Disabled != nil &&
				c.TailSampling.TraceAggregationWaitDuration != nil
		},
		"K8sHealthProbesSampling": func(c *common.SamplingConfiguration) bool {
			return c.K8sHealthProbesSampling != nil &&
				c.K8sHealthProbesSampling.Enabled != nil &&
				c.K8sHealthProbesSampling.KeepPercentage != nil
		},
	}

	fullyPopulated := &model.SamplingConfigInput{
		TailSampling: &model.TailSamplingConfigInput{
			Disabled:                     srcBool(true),
			TraceAggregationWaitDuration: srcStr("12s"),
		},
		K8sHealthProbesSampling: &model.K8sHealthProbesSamplingConfigInput{
			Enabled:        srcBool(true),
			KeepPercentage: srcFloat(3.5),
		},
	}

	inputType := reflect.TypeOf(*fullyPopulated)
	require.Equal(t, inputType.NumField(), len(persistedByInputField),
		"every field of SamplingConfigInput needs an entry asserting it reaches the persisted config")

	persisted := convertSamplingConfigInputToOdigosConfig(fullyPopulated)
	require.NotNil(t, persisted)

	for i := 0; i < inputType.NumField(); i++ {
		fieldName := inputType.Field(i).Name
		check, declared := persistedByInputField[fieldName]
		require.True(t, declared, "SamplingConfigInput.%s is not covered by this test", fieldName)
		assert.True(t, check(persisted),
			"SamplingConfigInput.%s is accepted by the GraphQL API but never persisted", fieldName)
	}

	assertEveryModelFieldSet(t, fullyPopulated, "SamplingConfigInput")
}
