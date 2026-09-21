package sampling

import (
	"context"
	"fmt"
	"testing"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const samplingRulesNamespace = "odigos-test"

// samplingRulesFixture describes the two object stores sampling_rules.go talks to.
// cached is what the informer-backed kube.CacheClient serves; when it is nil the cache
// mirrors live. Keeping them separate is what makes the cache-lag branches reachable.
type samplingRulesFixture struct {
	// namespace is the odigos namespace the code under test resolves from the
	// environment; it defaults to samplingRulesNamespace.
	namespace  string
	cached     []*v1alpha1.Sampling
	live       []*v1alpha1.Sampling
	cacheFuncs interceptor.Funcs
}

type samplingRulesHarness struct {
	t     *testing.T
	ns    string
	live  *odigosfake.Clientset
	cache client.WithWatch
}

func newSamplingRulesHarness(t *testing.T, stored ...*v1alpha1.Sampling) *samplingRulesHarness {
	return newSamplingRules(t, samplingRulesFixture{live: stored})
}

func newSamplingRules(t *testing.T, f samplingRulesFixture) *samplingRulesHarness {
	t.Helper()
	ns := f.namespace
	if ns == "" {
		ns = samplingRulesNamespace
	}
	t.Setenv(consts.CurrentNamespaceEnvVar, ns)

	cached := f.cached
	if cached == nil {
		cached = f.live
	}

	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	builder := fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(f.cacheFuncs)
	for _, cr := range cached {
		builder = builder.WithObjects(cr.DeepCopy())
	}
	cache := builder.Build()

	liveObjs := make([]runtime.Object, 0, len(f.live))
	for _, cr := range f.live {
		liveObjs = append(liveObjs, cr.DeepCopy())
	}
	live := odigosfake.NewSimpleClientset(liveObjs...)

	previousDefault := kube.DefaultClient
	previousCache := kube.CacheClient
	kube.SetDefaultClient(&kube.Client{OdigosClient: live.OdigosV1alpha1()})
	kube.CacheClient = cache
	t.Cleanup(func() {
		kube.SetDefaultClient(previousDefault)
		kube.CacheClient = previousCache
	})

	return &samplingRulesHarness{t: t, ns: ns, live: live, cache: cache}
}

// stored reads a Sampling CR back from the API server, which is where the mutations land.
func (h *samplingRulesHarness) stored(name string) *v1alpha1.Sampling {
	h.t.Helper()
	cr, err := h.live.OdigosV1alpha1().Samplings(h.ns).
		Get(context.Background(), name, metav1.GetOptions{})
	require.NoError(h.t, err)
	return cr
}

func (h *samplingRulesHarness) storedNames() []string {
	h.t.Helper()
	list, err := h.live.OdigosV1alpha1().Samplings(h.ns).
		List(context.Background(), metav1.ListOptions{})
	require.NoError(h.t, err)
	names := make([]string, 0, len(list.Items))
	for i := range list.Items {
		names = append(names, list.Items[i].Name)
	}
	return names
}

// refreshCache models the informer catching up with the API server.
func (h *samplingRulesHarness) refreshCache() {
	h.t.Helper()
	ctx := context.Background()
	list, err := h.live.OdigosV1alpha1().Samplings(h.ns).List(ctx, metav1.ListOptions{})
	require.NoError(h.t, err)
	for i := range list.Items {
		cr := list.Items[i].DeepCopy()
		existing := &v1alpha1.Sampling{}
		getErr := h.cache.Get(ctx, client.ObjectKeyFromObject(cr), existing)
		if apierrors.IsNotFound(getErr) {
			cr.ResourceVersion = ""
			require.NoError(h.t, h.cache.Create(ctx, cr))
			continue
		}
		require.NoError(h.t, getErr)
		cr.ResourceVersion = existing.ResourceVersion
		require.NoError(h.t, h.cache.Update(ctx, cr))
	}
}

// countLiveWrites counts every create/update the code under test sends to the API server,
// so a test can assert that a rejected mutation wrote nothing at all.
func (h *samplingRulesHarness) countLiveWrites() *int {
	writes := 0
	h.live.PrependReactor("*", "samplings", func(action k8stesting.Action) (bool, runtime.Object, error) {
		switch action.GetVerb() {
		case "create", "update", "patch", "delete":
			writes++
		}
		return false, nil, nil
	})
	return &writes
}

// conflictOnFirstUpdates makes the API server reject the first n updates with a Conflict,
// the error retry.RetryOnConflict is expected to absorb.
func (h *samplingRulesHarness) conflictOnFirstUpdates(n int) *int {
	attempts := 0
	h.live.PrependReactor("update", "samplings", func(action k8stesting.Action) (bool, runtime.Object, error) {
		attempts++
		if attempts <= n {
			return true, nil, apierrors.NewConflict(
				v1alpha1.Resource("samplings"),
				action.(k8stesting.UpdateAction).GetObject().(*v1alpha1.Sampling).Name,
				fmt.Errorf("the object has been modified; please apply your changes to the latest version"))
		}
		return false, nil, nil
	})
	return &attempts
}

func samplingCR(name string, mutate ...func(*v1alpha1.Sampling)) *v1alpha1.Sampling {
	cr := &v1alpha1.Sampling{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: samplingRulesNamespace},
		Spec:       v1alpha1.SamplingSpec{Name: name},
	}
	for _, m := range mutate {
		m(cr)
	}
	return cr
}

func headServerMatcher(route string) *commonapisampling.HeadSamplingOperationMatcher {
	return &commonapisampling.HeadSamplingOperationMatcher{
		HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{Route: route},
	}
}

func tailServerMatcher(route string) *commonapisampling.TailSamplingOperationMatcher {
	return &commonapisampling.TailSamplingOperationMatcher{
		HttpServer: &commonapisampling.TailSamplingHttpServerOperationMatcher{Route: route},
	}
}

func namespaceScope(namespaces ...string) *k8sconsts.SourcesScopes {
	return &k8sconsts.SourcesScopes{Namespaces: namespaces}
}

func storedNoisyOperation(name, route string) v1alpha1.NoisyOperation {
	return v1alpha1.NoisyOperation{
		Name:             name,
		Operation:        headServerMatcher(route),
		PercentageAtMost: float64Ptr(5),
	}
}

func storedHighlyRelevantOperation(name, route string) v1alpha1.HighlyRelevantOperation {
	return v1alpha1.HighlyRelevantOperation{
		Name:              name,
		Operation:         tailServerMatcher(route),
		PercentageAtLeast: float64Ptr(100),
	}
}

func storedCostReductionRule(name, route string) v1alpha1.CostReductionRule {
	return v1alpha1.CostReductionRule{
		Name:             name,
		Operation:        tailServerMatcher(route),
		PercentageAtMost: 10,
	}
}

func noisyOperationNames(rules []v1alpha1.NoisyOperation) []string {
	names := make([]string, 0, len(rules))
	for i := range rules {
		names = append(names, rules[i].Name)
	}
	return names
}

func highlyRelevantOperationNames(rules []v1alpha1.HighlyRelevantOperation) []string {
	names := make([]string, 0, len(rules))
	for i := range rules {
		names = append(names, rules[i].Name)
	}
	return names
}

func costReductionRuleNames(rules []v1alpha1.CostReductionRule) []string {
	names := make([]string, 0, len(rules))
	for i := range rules {
		names = append(names, rules[i].Name)
	}
	return names
}
