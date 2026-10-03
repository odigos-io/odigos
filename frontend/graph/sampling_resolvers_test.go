package graph

import (
	"context"
	"testing"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/yaml"
)

const samplingResolverNamespace = "odigos-test"

type samplingResolverHarness struct {
	t        *testing.T
	resolver *Resolver
	cache    client.WithWatch
	live     *odigosfake.Clientset
}

// newSamplingResolverHarness wires everything the sampling resolvers reach for: the
// resolver's own injected cache client, the package-level kube.CacheClient the sampling
// rule service reads through, and the live odigos clientset it writes through.
func newSamplingResolverHarness(t *testing.T, objects ...client.Object) *samplingResolverHarness {
	return newSamplingResolverHarnessWithInterceptor(t, interceptor.Funcs{}, objects...)
}

func newSamplingResolverHarnessWithInterceptor(t *testing.T, funcs interceptor.Funcs, objects ...client.Object) *samplingResolverHarness {
	t.Helper()
	t.Setenv(consts.CurrentNamespaceEnvVar, samplingResolverNamespace)

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	cache := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		WithInterceptorFuncs(funcs).
		Build()

	liveObjects := make([]runtime.Object, 0, len(objects))
	for _, object := range objects {
		if sampling, ok := object.(*v1alpha1.Sampling); ok {
			liveObjects = append(liveObjects, sampling.DeepCopy())
		}
	}
	live := odigosfake.NewSimpleClientset(liveObjects...)

	previousCache := kube.CacheClient
	previousDefault := kube.DefaultClient
	kube.CacheClient = cache
	kube.SetDefaultClient(&kube.Client{OdigosClient: live.OdigosV1alpha1()})
	t.Cleanup(func() {
		kube.CacheClient = previousCache
		kube.SetDefaultClient(previousDefault)
	})

	return &samplingResolverHarness{
		t:        t,
		resolver: &Resolver{K8sCacheClient: cache},
		cache:    cache,
		live:     live,
	}
}

func (h *samplingResolverHarness) storedSampling(name string) *v1alpha1.Sampling {
	h.t.Helper()
	cr, err := h.live.OdigosV1alpha1().Samplings(samplingResolverNamespace).
		Get(context.Background(), name, metav1.GetOptions{})
	require.NoError(h.t, err)
	return cr
}

func (h *samplingResolverHarness) storedLocalUIConfig() *common.OdigosConfiguration {
	h.t.Helper()
	var cm corev1.ConfigMap
	require.NoError(h.t, h.cache.Get(context.Background(),
		client.ObjectKey{Namespace: samplingResolverNamespace, Name: consts.OdigosLocalUiConfigName}, &cm))
	cfg := &common.OdigosConfiguration{}
	require.NoError(h.t, yaml.Unmarshal([]byte(cm.Data[consts.OdigosConfigurationFileName]), cfg))
	return cfg
}

// samplingConfigMap renders an odigos configuration document into one of the four
// ConfigMaps the sampling settings screen reads.
func samplingConfigMap(t *testing.T, name string, cfg *common.OdigosConfiguration) *corev1.ConfigMap {
	t.Helper()
	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: samplingResolverNamespace, UID: "owner-uid"},
		Data:       map[string]string{consts.OdigosConfigurationFileName: string(data)},
	}
}

func samplingConfigWithWaitDuration(duration string) *common.OdigosConfiguration {
	return &common.OdigosConfiguration{
		Sampling: &common.SamplingConfiguration{
			TailSampling: &commonapisampling.TailSamplingConfiguration{
				TraceAggregationWaitDuration: srcStr(duration),
			},
		},
	}
}

// The settings screen shows four sampling configurations side by side so a user can see
// what helm shipped, what the central backend pushed, what the UI overrode and what the
// three merge into. The four resolvers are the same five lines pointing at four different
// ConfigMaps, so each fixture carries a distinguishable value: with one shared fixture any
// two of them could be swapped and every assertion would still pass.
func TestTheFourSamplingConfigResolversReadFourDifferentConfigMaps(t *testing.T) {
	h := newSamplingResolverHarness(t,
		samplingConfigMap(t, consts.OdigosEffectiveConfigName, samplingConfigWithWaitDuration("1s")),
		samplingConfigMap(t, consts.OdigosConfigurationName, samplingConfigWithWaitDuration("2s")),
		samplingConfigMap(t, consts.OdigosRemoteConfigName, samplingConfigWithWaitDuration("3s")),
		samplingConfigMap(t, consts.OdigosLocalUiConfigName, samplingConfigWithWaitDuration("4s")),
	)

	configs := h.resolver.SamplingConfigs()
	ctx := context.Background()

	readers := []struct {
		name     string
		read     func() (*model.SamplingConfig, error)
		expected string
	}{
		{"effective", func() (*model.SamplingConfig, error) { return configs.Effective(ctx, nil) }, "1s"},
		{"helmDeployment", func() (*model.SamplingConfig, error) { return configs.HelmDeployment(ctx, nil) }, "2s"},
		{"remoteConfigFromCentral", func() (*model.SamplingConfig, error) { return configs.RemoteConfigFromCentral(ctx, nil) }, "3s"},
		{"localUiConfig", func() (*model.SamplingConfig, error) { return configs.LocalUIConfig(ctx, nil) }, "4s"},
	}

	for _, reader := range readers {
		t.Run(reader.name, func(t *testing.T) {
			got, err := reader.read()
			require.NoError(t, err)
			require.NotNil(t, got)
			require.NotNil(t, got.TailSampling)
			assert.Equal(t, srcStr(reader.expected), got.TailSampling.TraceAggregationWaitDuration,
				"%s must read its own ConfigMap", reader.name)
		})
	}
}

// A ConfigMap that does not exist yet is a normal state on a fresh install, and every one
// of the four fields is nullable in the schema, so it has to read as "nothing configured"
// rather than failing the whole settings query.
func TestTheSamplingConfigResolversReportAMissingConfigMapAsNoConfig(t *testing.T) {
	h := newSamplingResolverHarness(t)
	configs := h.resolver.SamplingConfigs()
	ctx := context.Background()

	for name, read := range map[string]func() (*model.SamplingConfig, error){
		"effective":               func() (*model.SamplingConfig, error) { return configs.Effective(ctx, nil) },
		"helmDeployment":          func() (*model.SamplingConfig, error) { return configs.HelmDeployment(ctx, nil) },
		"remoteConfigFromCentral": func() (*model.SamplingConfig, error) { return configs.RemoteConfigFromCentral(ctx, nil) },
		"localUiConfig":           func() (*model.SamplingConfig, error) { return configs.LocalUIConfig(ctx, nil) },
	} {
		t.Run(name, func(t *testing.T) {
			got, err := read()
			assert.NoError(t, err, "a missing ConfigMap must not fail the settings query")
			assert.Nil(t, got)
		})
	}
}

func TestTheSamplingConfigResolversSurfaceAReadError(t *testing.T) {
	h := newSamplingResolverHarnessWithInterceptor(t, interceptor.Funcs{
		Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
			return apierrors.NewInternalError(assert.AnError)
		},
	})
	configs := h.resolver.SamplingConfigs()
	ctx := context.Background()

	for name, read := range map[string]func() (*model.SamplingConfig, error){
		"effective":               func() (*model.SamplingConfig, error) { return configs.Effective(ctx, nil) },
		"helmDeployment":          func() (*model.SamplingConfig, error) { return configs.HelmDeployment(ctx, nil) },
		"remoteConfigFromCentral": func() (*model.SamplingConfig, error) { return configs.RemoteConfigFromCentral(ctx, nil) },
		"localUiConfig":           func() (*model.SamplingConfig, error) { return configs.LocalUIConfig(ctx, nil) },
	} {
		t.Run(name, func(t *testing.T) {
			got, err := read()
			require.Error(t, err, "a real read failure must not be reported as an empty config")
			assert.Nil(t, got)
		})
	}
}

// The sampling query field is non-null and its configs child is what the four config
// resolvers hang off, so returning a nil configs block would make the whole settings
// screen unqueryable.
func TestTheSamplingQueryReturnsAConfigsBlockToResolveAgainst(t *testing.T) {
	h := newSamplingResolverHarness(t)

	got, err := h.resolver.Query().Sampling(context.Background())

	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.Configs, "the configs block must exist for the nested resolvers to run")
}

func TestTheSamplingRulesQueryReturnsEveryGroup(t *testing.T) {
	h := newSamplingResolverHarness(t,
		&v1alpha1.Sampling{
			ObjectMeta: metav1.ObjectMeta{Name: "payments", Namespace: samplingResolverNamespace},
			Spec: v1alpha1.SamplingSpec{
				Name: "payments",
				NoisyOperations: []v1alpha1.NoisyOperation{
					{Name: "health", Operation: srcHeadServerMatcher("/healthz")},
				},
			},
		},
		&v1alpha1.Sampling{
			ObjectMeta: metav1.ObjectMeta{Name: "orders", Namespace: samplingResolverNamespace},
			Spec:       v1alpha1.SamplingSpec{Name: "orders"},
		},
	)

	groups, err := h.resolver.Sampling().Rules(context.Background(), &model.Sampling{})

	require.NoError(t, err)
	require.Len(t, groups, 2)
	ids := []string{groups[0].ID, groups[1].ID}
	assert.ElementsMatch(t, []string{"payments", "orders"}, ids)
}

func TestTheSamplingRulesQuerySurfacesAListError(t *testing.T) {
	h := newSamplingResolverHarnessWithInterceptor(t, interceptor.Funcs{
		List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
			return apierrors.NewInternalError(assert.AnError)
		},
	})

	groups, err := h.resolver.Sampling().Rules(context.Background(), &model.Sampling{})

	require.Error(t, err)
	assert.Nil(t, groups)
}

// The three rule-list resolvers are one-line pass-throughs of three same-typed fields on
// the already-resolved group. Three distinguishable fixtures are what make a
// cross-wired field visible.
func TestTheRuleListResolversReturnTheirOwnFamily(t *testing.T) {
	h := newSamplingResolverHarness(t)
	group := &model.SamplingRules{
		ID:                       "group",
		NoisyOperations:          []*model.NoisyOperationRule{{RuleID: "noisy-a"}},
		HighlyRelevantOperations: []*model.HighlyRelevantOperationRule{{RuleID: "relevant-a"}, {RuleID: "relevant-b"}},
		CostReductionRules:       []*model.CostReductionRule{{RuleID: "cost-a"}, {RuleID: "cost-b"}, {RuleID: "cost-c"}},
	}
	rules := h.resolver.SamplingRules()
	ctx := context.Background()

	noisy, err := rules.NoisyOperations(ctx, group)
	require.NoError(t, err)
	require.Len(t, noisy, 1)
	assert.Equal(t, "noisy-a", noisy[0].RuleID)

	relevant, err := rules.HighlyRelevantOperations(ctx, group)
	require.NoError(t, err)
	require.Len(t, relevant, 2)
	assert.Equal(t, "relevant-a", relevant[0].RuleID)

	cost, err := rules.CostReductionRules(ctx, group)
	require.NoError(t, err)
	require.Len(t, cost, 3)
	assert.Equal(t, "cost-a", cost[0].RuleID)
}

func TestTheRuleListResolversPassThroughAnEmptyGroup(t *testing.T) {
	h := newSamplingResolverHarness(t)
	rules := h.resolver.SamplingRules()
	ctx := context.Background()

	noisy, err := rules.NoisyOperations(ctx, &model.SamplingRules{})
	require.NoError(t, err)
	assert.Empty(t, noisy)

	relevant, err := rules.HighlyRelevantOperations(ctx, &model.SamplingRules{})
	require.NoError(t, err)
	assert.Empty(t, relevant)

	cost, err := rules.CostReductionRules(ctx, &model.SamplingRules{})
	require.NoError(t, err)
	assert.Empty(t, cost)
}

// ---- mutations ----

// The sampling settings mutation writes into the local UI overlay, which is also where
// the component log levels and the MCP access mode live. Persisting a tail sampling change
// must not drop the health probe block a previous save put there, or turning tail sampling
// off would silently re-enable health probe tracing.
func TestUpdateLocalUISamplingConfigMergesIntoTheExistingOverlay(t *testing.T) {
	existing := &common.OdigosConfiguration{
		Sampling: &common.SamplingConfiguration{
			DryRun: srcBool(true),
			K8sHealthProbesSampling: &common.K8sHealthProbesSamplingConfiguration{
				Enabled:        srcBool(true),
				KeepPercentage: srcFloat(1),
			},
		},
	}
	h := newSamplingResolverHarness(t,
		samplingConfigMap(t, consts.OdigosLocalUiConfigName, existing),
		samplingConfigMap(t, consts.OdigosConfigurationName, &common.OdigosConfiguration{}),
	)

	ok, err := h.resolver.Mutation().UpdateLocalUISamplingConfig(context.Background(), &model.SamplingConfigInput{
		TailSampling: &model.TailSamplingConfigInput{
			Disabled:                     srcBool(true),
			TraceAggregationWaitDuration: srcStr("9s"),
		},
	})
	require.NoError(t, err)
	assert.True(t, ok)

	stored := h.storedLocalUIConfig()
	require.NotNil(t, stored.Sampling)
	require.NotNil(t, stored.Sampling.TailSampling)
	assert.Equal(t, srcBool(true), stored.Sampling.TailSampling.Disabled)
	assert.Equal(t, srcStr("9s"), stored.Sampling.TailSampling.TraceAggregationWaitDuration)
	require.NotNil(t, stored.Sampling.K8sHealthProbesSampling,
		"the health probe block another save wrote must survive a tail sampling change")
	assert.Equal(t, srcBool(true), stored.Sampling.K8sHealthProbesSampling.Enabled)
	assert.Equal(t, srcFloat(1), stored.Sampling.K8sHealthProbesSampling.KeepPercentage)
	assert.Equal(t, srcBool(true), stored.Sampling.DryRun, "dryRun is not expressible in the input and must be preserved")
}

// The value the mutation writes has to be the value the read resolver then reports, or the
// settings screen shows the user something other than what they just saved.
func TestUpdateLocalUISamplingConfigIsVisibleToTheLocalUIConfigResolver(t *testing.T) {
	h := newSamplingResolverHarness(t,
		samplingConfigMap(t, consts.OdigosConfigurationName, &common.OdigosConfiguration{}),
	)

	_, err := h.resolver.Mutation().UpdateLocalUISamplingConfig(context.Background(), &model.SamplingConfigInput{
		K8sHealthProbesSampling: &model.K8sHealthProbesSamplingConfigInput{
			Enabled:        srcBool(true),
			KeepPercentage: srcFloat(2.5),
		},
	})
	require.NoError(t, err)

	got, err := h.resolver.SamplingConfigs().LocalUIConfig(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.K8sHealthProbesSampling)
	assert.Equal(t, srcBool(true), got.K8sHealthProbesSampling.Enabled)
	assert.Equal(t, srcFloat(2.5), got.K8sHealthProbesSampling.KeepPercentage)
}

func TestUpdateLocalUISamplingConfigSurfacesAWriteError(t *testing.T) {
	h := newSamplingResolverHarnessWithInterceptor(t, interceptor.Funcs{
		Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
			return apierrors.NewInternalError(assert.AnError)
		},
	})

	ok, err := h.resolver.Mutation().UpdateLocalUISamplingConfig(context.Background(), &model.SamplingConfigInput{
		TailSampling: &model.TailSamplingConfigInput{Disabled: srcBool(true)},
	})

	require.Error(t, err)
	assert.False(t, ok, "the boolean result must not claim success when the write failed")
}

// The nine rule mutations are nine one-line delegations that differ only in which service
// function they name and which family they address. Driving all nine against one group and
// asserting the other two families are untouched is what catches a delegation pointed at
// the wrong family.
func TestEverySamplingRuleMutationResolverReachesItsOwnRuleFamily(t *testing.T) {
	group := func() *v1alpha1.Sampling {
		return &v1alpha1.Sampling{
			ObjectMeta: metav1.ObjectMeta{Name: "group", Namespace: samplingResolverNamespace},
			Spec: v1alpha1.SamplingSpec{
				Name:                     "group",
				NoisyOperations:          []v1alpha1.NoisyOperation{{Name: "noisy", Operation: srcHeadServerMatcher("/noisy")}},
				HighlyRelevantOperations: []v1alpha1.HighlyRelevantOperation{{Name: "relevant", Operation: srcTailServerMatcher("/relevant")}},
				CostReductionRules:       []v1alpha1.CostReductionRule{{Name: "cost", Operation: srcTailServerMatcher("/cost")}},
			},
		}
	}
	seed := group().Spec
	noisyID := v1alpha1.ComputeNoisyOperationHash(&seed.NoisyOperations[0])
	relevantID := v1alpha1.ComputeHighlyRelevantOperationHash(&seed.HighlyRelevantOperations[0])
	costID := v1alpha1.ComputeCostReductionRuleHash(&seed.CostReductionRules[0])

	type expectation struct {
		noisy    []string
		relevant []string
		cost     []string
	}

	cases := map[string]struct {
		run    func(context.Context, MutationResolver) error
		expect expectation
	}{
		"createNoisyOperationRule": {
			run: func(ctx context.Context, m MutationResolver) error {
				_, err := m.CreateNoisyOperationRule(ctx, "group", model.NoisyOperationRuleInput{Name: srcStr("added")})
				return err
			},
			expect: expectation{noisy: []string{"noisy", "added"}, relevant: []string{"relevant"}, cost: []string{"cost"}},
		},
		"updateNoisyOperationRule": {
			run: func(ctx context.Context, m MutationResolver) error {
				_, err := m.UpdateNoisyOperationRule(ctx, "group", noisyID, model.NoisyOperationRuleInput{Name: srcStr("renamed")})
				return err
			},
			expect: expectation{noisy: []string{"renamed"}, relevant: []string{"relevant"}, cost: []string{"cost"}},
		},
		"deleteNoisyOperationRule": {
			run: func(ctx context.Context, m MutationResolver) error {
				_, err := m.DeleteNoisyOperationRule(ctx, "group", noisyID)
				return err
			},
			expect: expectation{noisy: []string{}, relevant: []string{"relevant"}, cost: []string{"cost"}},
		},
		"createHighlyRelevantOperationRule": {
			run: func(ctx context.Context, m MutationResolver) error {
				_, err := m.CreateHighlyRelevantOperationRule(ctx, "group", model.HighlyRelevantOperationRuleInput{Name: srcStr("added")})
				return err
			},
			expect: expectation{noisy: []string{"noisy"}, relevant: []string{"relevant", "added"}, cost: []string{"cost"}},
		},
		"updateHighlyRelevantOperationRule": {
			run: func(ctx context.Context, m MutationResolver) error {
				_, err := m.UpdateHighlyRelevantOperationRule(ctx, "group", relevantID, model.HighlyRelevantOperationRuleInput{Name: srcStr("renamed")})
				return err
			},
			expect: expectation{noisy: []string{"noisy"}, relevant: []string{"renamed"}, cost: []string{"cost"}},
		},
		"deleteHighlyRelevantOperationRule": {
			run: func(ctx context.Context, m MutationResolver) error {
				_, err := m.DeleteHighlyRelevantOperationRule(ctx, "group", relevantID)
				return err
			},
			expect: expectation{noisy: []string{"noisy"}, relevant: []string{}, cost: []string{"cost"}},
		},
		"createCostReductionRule": {
			run: func(ctx context.Context, m MutationResolver) error {
				_, err := m.CreateCostReductionRule(ctx, "group", model.CostReductionRuleInput{Name: srcStr("added")})
				return err
			},
			expect: expectation{noisy: []string{"noisy"}, relevant: []string{"relevant"}, cost: []string{"cost", "added"}},
		},
		"updateCostReductionRule": {
			run: func(ctx context.Context, m MutationResolver) error {
				_, err := m.UpdateCostReductionRule(ctx, "group", costID, model.CostReductionRuleInput{Name: srcStr("renamed")})
				return err
			},
			expect: expectation{noisy: []string{"noisy"}, relevant: []string{"relevant"}, cost: []string{"renamed"}},
		},
		"deleteCostReductionRule": {
			run: func(ctx context.Context, m MutationResolver) error {
				_, err := m.DeleteCostReductionRule(ctx, "group", costID)
				return err
			},
			expect: expectation{noisy: []string{"noisy"}, relevant: []string{"relevant"}, cost: []string{}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			h := newSamplingResolverHarness(t, group())

			require.NoError(t, tc.run(context.Background(), h.resolver.Mutation()))

			stored := h.storedSampling("group").Spec
			assert.Equal(t, tc.expect.noisy, samplingRuleNames(stored.NoisyOperations))
			assert.Equal(t, tc.expect.relevant, samplingRuleNames(stored.HighlyRelevantOperations))
			assert.Equal(t, tc.expect.cost, samplingRuleNames(stored.CostReductionRules))
		})
	}
}

// Each rule mutation returns the rule it just wrote, and the UI keys its list on the
// returned rule id. A resolver that dropped the result would leave the new rule
// unaddressable until a full refetch.
func TestTheSamplingRuleMutationResolversReturnTheWrittenRule(t *testing.T) {
	h := newSamplingResolverHarness(t)
	mutation := h.resolver.Mutation()
	ctx := context.Background()

	noisy, err := mutation.CreateNoisyOperationRule(ctx, "group", model.NoisyOperationRuleInput{
		Name:      srcStr("health"),
		Operation: &model.HeadSamplingOperationMatcherInput{HTTPServer: &model.HeadSamplingHTTPServerMatcherInput{Route: srcStr("/healthz")}},
	})
	require.NoError(t, err)
	require.NotNil(t, noisy)
	assert.Equal(t, srcStr("health"), noisy.Name)
	assert.NotEmpty(t, noisy.RuleID)

	relevant, err := mutation.CreateHighlyRelevantOperationRule(ctx, "group", model.HighlyRelevantOperationRuleInput{
		Name:  srcStr("charges"),
		Error: srcBool(true),
	})
	require.NoError(t, err)
	require.NotNil(t, relevant)
	assert.Equal(t, srcStr("charges"), relevant.Name)
	assert.True(t, relevant.Error)

	cost, err := mutation.CreateCostReductionRule(ctx, "group", model.CostReductionRuleInput{
		Name:             srcStr("orders"),
		PercentageAtMost: 30,
	})
	require.NoError(t, err)
	require.NotNil(t, cost)
	assert.Equal(t, srcStr("orders"), cost.Name)
	assert.Equal(t, 30.0, cost.PercentageAtMost)
}

func TestTheSamplingRuleMutationResolversSurfaceServiceErrors(t *testing.T) {
	h := newSamplingResolverHarness(t)
	mutation := h.resolver.Mutation()
	ctx := context.Background()

	failures := map[string]func() error{
		"updateNoisyOperationRule": func() error {
			_, err := mutation.UpdateNoisyOperationRule(ctx, "missing", "id", model.NoisyOperationRuleInput{})
			return err
		},
		"deleteNoisyOperationRule": func() error {
			ok, err := mutation.DeleteNoisyOperationRule(ctx, "missing", "id")
			assert.False(t, ok)
			return err
		},
		"updateHighlyRelevantOperationRule": func() error {
			_, err := mutation.UpdateHighlyRelevantOperationRule(ctx, "missing", "id", model.HighlyRelevantOperationRuleInput{})
			return err
		},
		"deleteHighlyRelevantOperationRule": func() error {
			ok, err := mutation.DeleteHighlyRelevantOperationRule(ctx, "missing", "id")
			assert.False(t, ok)
			return err
		},
		"updateCostReductionRule": func() error {
			_, err := mutation.UpdateCostReductionRule(ctx, "missing", "id", model.CostReductionRuleInput{})
			return err
		},
		"deleteCostReductionRule": func() error {
			ok, err := mutation.DeleteCostReductionRule(ctx, "missing", "id")
			assert.False(t, ok)
			return err
		},
	}

	for name, fail := range failures {
		t.Run(name, func(t *testing.T) {
			assert.Error(t, fail(), "a service failure must reach the GraphQL layer")
		})
	}
}

// The three resolver constructors must hand back resolvers bound to the same Resolver, or
// the injected cache client is nil and every nested read nil-panics at request time.
func TestTheSamplingResolverConstructorsCarryTheInjectedDependencies(t *testing.T) {
	h := newSamplingResolverHarness(t,
		samplingConfigMap(t, consts.OdigosEffectiveConfigName, samplingConfigWithWaitDuration("7s")),
	)

	require.NotNil(t, h.resolver.Sampling())
	require.NotNil(t, h.resolver.SamplingConfigs())
	require.NotNil(t, h.resolver.SamplingRules())

	effective, err := h.resolver.SamplingConfigs().Effective(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, effective)
	require.NotNil(t, effective.TailSampling)
	assert.Equal(t, srcStr("7s"), effective.TailSampling.TraceAggregationWaitDuration,
		"the configs resolver must be able to reach the resolver's cache client")
}

// samplingRuleNames reads the Name of every rule in any of the three rule slices, so one
// assertion shape works for all three families.
func samplingRuleNames[T v1alpha1.NoisyOperation | v1alpha1.HighlyRelevantOperation | v1alpha1.CostReductionRule](rules []T) []string {
	names := make([]string, 0, len(rules))
	for i := range rules {
		switch rule := any(rules[i]).(type) {
		case v1alpha1.NoisyOperation:
			names = append(names, rule.Name)
		case v1alpha1.HighlyRelevantOperation:
			names = append(names, rule.Name)
		case v1alpha1.CostReductionRule:
			names = append(names, rule.Name)
		}
	}
	return names
}
