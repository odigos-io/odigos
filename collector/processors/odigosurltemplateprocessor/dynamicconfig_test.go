package odigosurltemplateprocessor

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processortest"
	semconv "go.opentelemetry.io/collector/semconv/v1.27.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	commonapi "github.com/odigos-io/odigos/common/api"
	commonactionsapi "github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/common/collector"
	"github.com/odigos-io/odigos/common/odigosattributes"
)

var extensionComponentID = component.MustNewID("odigos_config_k8s")

// dynamicConfigExtension stands in for odigos_config_k8s. The cache key is the resource's
// service.name so a single batch can carry spans of several workloads, and a resource without
// that attribute exercises the GetWorkloadCacheKey error path.
type dynamicConfigExtension struct {
	registered     []collector.WorkloadConfigCacheCallback
	unregistered   []collector.WorkloadConfigCacheCallback
	cacheSynced    bool
	cacheSyncCalls int
}

var errNoWorkloadIdentity = errors.New("no workload identity on resource")

func (e *dynamicConfigExtension) Start(context.Context, component.Host) error { return nil }

func (e *dynamicConfigExtension) Shutdown(context.Context) error { return nil }

func (e *dynamicConfigExtension) GetFromResource(pcommon.Resource) (*commonapi.ContainerCollectorConfig, bool) {
	return nil, false
}

func (e *dynamicConfigExtension) IsActiveSource(pcommon.Resource) bool { return true }

func (e *dynamicConfigExtension) GetWorkloadCacheKey(res pcommon.Resource) (string, error) {
	key, found := res.Attributes().Get(semconv.AttributeServiceName)
	if !found {
		return "", errNoWorkloadIdentity
	}
	return key.Str(), nil
}

func (e *dynamicConfigExtension) GetWorkloadIdentityFromResource(res pcommon.Resource) (string, pcommon.Map, error) {
	key, err := e.GetWorkloadCacheKey(res)
	return key, res.Attributes(), err
}

func (e *dynamicConfigExtension) RegisterWorkloadConfigCacheCallback(cb collector.WorkloadConfigCacheCallback) {
	e.registered = append(e.registered, cb)
}

func (e *dynamicConfigExtension) UnregisterWorkloadConfigCacheCallback(cb collector.WorkloadConfigCacheCallback) {
	e.unregistered = append(e.unregistered, cb)
}

func (e *dynamicConfigExtension) WaitForCacheSync(context.Context) bool {
	e.cacheSyncCalls++
	return e.cacheSynced
}

func (e *dynamicConfigExtension) GetDataStreamsForWorkload(pcommon.Resource) ([]string, bool) {
	return nil, false
}

// plainExtension is a component that does not implement collector.OdigosConfigExtension.
type plainExtension struct{}

func (plainExtension) Start(context.Context, component.Host) error { return nil }

func (plainExtension) Shutdown(context.Context) error { return nil }

type extensionHost struct {
	extensions map[component.ID]component.Component
}

func (h extensionHost) GetExtensions() map[component.ID]component.Component { return h.extensions }

func newDynamicConfigSettings(t *testing.T) (processor.Settings, *observer.ObservedLogs) {
	t.Helper()
	core, logs := observer.New(zap.DebugLevel)
	set := processortest.NewNopSettings(processortest.NopType)
	set.Logger = zap.New(core)
	return set, logs
}

// newStartedProcessor builds a processor wired to the extension, exactly as Start() does in a
// running collector, and returns both so tests can drive the cache callbacks.
func newStartedProcessor(t *testing.T) (*urlTemplateProcessor, *dynamicConfigExtension) {
	t.Helper()
	proc, ext, _ := newStartedProcessorWithLogs(t)
	return proc, ext
}

func newStartedProcessorWithLogs(t *testing.T) (*urlTemplateProcessor, *dynamicConfigExtension, *observer.ObservedLogs) {
	t.Helper()
	set, logs := newDynamicConfigSettings(t)
	ext := &dynamicConfigExtension{cacheSynced: true}
	proc, err := newUrlTemplateProcessor(set, &Config{OdigosConfigExtension: &extensionComponentID})
	require.NoError(t, err)
	require.NoError(t, proc.Start(context.Background(), extensionHost{
		extensions: map[component.ID]component.Component{extensionComponentID: ext},
	}))
	return proc, ext, logs
}

// workloadSpan describes one span to place under the resource of the given workload.
type workloadSpan struct {
	workload string
	kind     ptrace.SpanKind
	attrs    map[string]any
}

// tracesForWorkloads builds one resource span per entry, each with a single span. A workload name
// of "" leaves service.name unset so the resource has no cache key.
func tracesForWorkloads(t *testing.T, spans ...workloadSpan) ptrace.Traces {
	t.Helper()
	td := ptrace.NewTraces()
	for _, s := range spans {
		rs := td.ResourceSpans().AppendEmpty()
		if s.workload != "" {
			rs.Resource().Attributes().PutStr(semconv.AttributeServiceName, s.workload)
		}
		span := rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
		require.NoError(t, span.Attributes().FromRaw(s.attrs))
		span.SetName("GET")
		span.SetKind(s.kind)
	}
	return td
}

func serverSpanForWorkload(t *testing.T, workload string, attrs map[string]any) ptrace.Traces {
	t.Helper()
	return tracesForWorkloads(t, workloadSpan{workload: workload, kind: ptrace.SpanKindServer, attrs: attrs})
}

func processedSpanAt(t *testing.T, td ptrace.Traces, resourceIdx int) ptrace.Span {
	t.Helper()
	return td.ResourceSpans().At(resourceIdx).ScopeSpans().At(0).Spans().At(0)
}

func requireAttr(t *testing.T, span ptrace.Span, key string, want string) {
	t.Helper()
	val, found := span.Attributes().Get(key)
	require.True(t, found, "expected attribute %q on span", key)
	require.Equal(t, want, val.AsString())
}

func requireNoAttr(t *testing.T, span ptrace.Span, key string) {
	t.Helper()
	_, found := span.Attributes().Get(key)
	require.False(t, found, "expected no attribute %q on span", key)
}

func templatizationConfig(templates []string, def *commonactionsapi.DefaultTemplatizationConfig) *commonapi.ContainerCollectorConfig {
	return &commonapi.ContainerCollectorConfig{
		ContainerName: "app",
		UrlTemplatization: &commonactionsapi.UrlTemplatizationConfig{
			Templates: templates,
			Default:   def,
		},
	}
}

// ******************************************************
// Start / Shutdown: wiring to the odigos config extension
// ******************************************************

func TestStart_NoExtensionConfiguredKeepsStaticMode(t *testing.T) {
	set, logs := newDynamicConfigSettings(t)
	proc, err := newUrlTemplateProcessor(set, &Config{})
	require.NoError(t, err)

	// a host with the extension present must still be ignored when the config does not reference it
	require.NoError(t, proc.Start(context.Background(), extensionHost{
		extensions: map[component.ID]component.Component{extensionComponentID: &dynamicConfigExtension{cacheSynced: true}},
	}))
	require.Nil(t, proc.provider)
	require.Equal(t, 1, logs.FilterMessage("odigos_config_extension unset, ensure processor contains the templatization rules").Len())
}

func TestStart_RegistersCallbackAndWaitsForCacheSync(t *testing.T) {
	set, _ := newDynamicConfigSettings(t)
	ext := &dynamicConfigExtension{cacheSynced: true}
	proc, err := newUrlTemplateProcessor(set, &Config{OdigosConfigExtension: &extensionComponentID})
	require.NoError(t, err)

	require.NoError(t, proc.Start(context.Background(), extensionHost{
		extensions: map[component.ID]component.Component{extensionComponentID: ext},
	}))

	require.Same(t, ext, proc.provider)
	require.Equal(t, []collector.WorkloadConfigCacheCallback{proc}, ext.registered)
	require.Equal(t, 1, ext.cacheSyncCalls)
}

func TestStart_ExtensionMissingFromHost(t *testing.T) {
	set, _ := newDynamicConfigSettings(t)
	proc, err := newUrlTemplateProcessor(set, &Config{OdigosConfigExtension: &extensionComponentID})
	require.NoError(t, err)

	err = proc.Start(context.Background(), extensionHost{extensions: map[component.ID]component.Component{}})
	require.ErrorContains(t, err, `odigos config extension "odigos_config_k8s" not found`)
	require.Nil(t, proc.provider)
}

func TestStart_ExtensionDoesNotImplementInterface(t *testing.T) {
	set, _ := newDynamicConfigSettings(t)
	proc, err := newUrlTemplateProcessor(set, &Config{OdigosConfigExtension: &extensionComponentID})
	require.NoError(t, err)

	err = proc.Start(context.Background(), extensionHost{
		extensions: map[component.ID]component.Component{extensionComponentID: plainExtension{}},
	})
	require.ErrorContains(t, err, `extension "odigos_config_k8s" is not an OdigosConfigExtension`)
	require.Nil(t, proc.provider)
}

// A cache that has not synced must not fail collector startup, otherwise a slow API server takes
// down the whole node collector instead of degrading templatization.
func TestStart_CacheSyncFailureDoesNotFailStartup(t *testing.T) {
	set, logs := newDynamicConfigSettings(t)
	ext := &dynamicConfigExtension{cacheSynced: false}
	proc, err := newUrlTemplateProcessor(set, &Config{OdigosConfigExtension: &extensionComponentID})
	require.NoError(t, err)

	require.NoError(t, proc.Start(context.Background(), extensionHost{
		extensions: map[component.ID]component.Component{extensionComponentID: ext},
	}))

	require.Same(t, ext, proc.provider)
	require.Equal(t, []collector.WorkloadConfigCacheCallback{proc}, ext.registered)
	require.Equal(t, 1, logs.FilterMessage("odigos config extension cache sync did not complete; some spans may be missed on startup").Len())
}

func TestShutdown_UnregistersAndClearsCache(t *testing.T) {
	proc, ext := newStartedProcessor(t)
	proc.OnSet("checkout", templatizationConfig([]string{"/user/{user-id}"}, nil))

	require.NoError(t, proc.Shutdown(context.Background()))

	require.Equal(t, []collector.WorkloadConfigCacheCallback{proc}, ext.unregistered)
	require.Nil(t, proc.provider)
	_, found := proc.parsedRulesCache.get("checkout")
	require.False(t, found)

	// shutting down twice must not unregister twice (the extension would hold a stale reference)
	require.NoError(t, proc.Shutdown(context.Background()))
	require.Len(t, ext.unregistered, 1)
}

// ******************************************************
// OnSet / OnDeleteKey: per-workload rule cache
// ******************************************************

func TestOnSet_RulesAreScopedToTheirWorkload(t *testing.T) {
	proc, _ := newStartedProcessor(t)
	proc.OnSet("checkout", templatizationConfig([]string{"/user/{user-id}"}, nil))
	proc.OnSet("catalog", templatizationConfig([]string{"/user/{shopper}"}, nil))

	td := tracesForWorkloads(t,
		workloadSpan{workload: "checkout", kind: ptrace.SpanKindServer, attrs: map[string]any{
			"http.request.method": "GET",
			"url.path":            "/user/john",
		}},
		workloadSpan{workload: "catalog", kind: ptrace.SpanKindServer, attrs: map[string]any{
			"http.request.method": "GET",
			"url.path":            "/user/john",
		}},
	)

	out, err := proc.processTraces(context.Background(), td)
	require.NoError(t, err)

	requireAttr(t, processedSpanAt(t, out, 0), semconv.AttributeHTTPRoute, "/user/{user-id}")
	requireAttr(t, processedSpanAt(t, out, 1), semconv.AttributeHTTPRoute, "/user/{shopper}")
}

func TestOnSet_RulesGroupedBySegmentCount(t *testing.T) {
	proc, _ := newStartedProcessor(t)
	proc.OnSet("checkout", templatizationConfig([]string{
		"/user/{user-id}",
		"/user/{user-id}/friends/{friend-id}",
	}, nil))

	entry, found := proc.parsedRulesCache.get("checkout")
	require.True(t, found)
	require.Len(t, entry.parsedRules[2], 1)
	require.Len(t, entry.parsedRules[4], 1)

	for _, tc := range []struct{ path, want string }{
		{path: "/user/john", want: "/user/{user-id}"},
		{path: "/user/john/friends/jane", want: "/user/{user-id}/friends/{friend-id}"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			out, err := proc.processTraces(context.Background(), serverSpanForWorkload(t, "checkout", map[string]any{
				"http.request.method": "GET",
				"url.path":            tc.path,
			}))
			require.NoError(t, err)
			requireAttr(t, processedSpanAt(t, out, 0), semconv.AttributeHTTPRoute, tc.want)
		})
	}
}

// A workload whose templatization action is edited must not keep applying the removed rule.
func TestOnSet_ReplacesPreviousRules(t *testing.T) {
	proc, _ := newStartedProcessor(t)
	proc.OnSet("checkout", templatizationConfig([]string{"/user/{user-id}"}, nil))
	proc.OnSet("checkout", templatizationConfig([]string{"/order/{order-id}"}, nil))

	out, err := proc.processTraces(context.Background(), serverSpanForWorkload(t, "checkout", map[string]any{
		"http.request.method": "GET",
		"url.path":            "/user/john",
	}))
	require.NoError(t, err)
	// the old rule is gone and the new one does not match, so with no default config nothing is set
	requireNoAttr(t, processedSpanAt(t, out, 0), semconv.AttributeHTTPRoute)

	out, err = proc.processTraces(context.Background(), serverSpanForWorkload(t, "checkout", map[string]any{
		"http.request.method": "GET",
		"url.path":            "/order/john",
	}))
	require.NoError(t, err)
	requireAttr(t, processedSpanAt(t, out, 0), semconv.AttributeHTTPRoute, "/order/{order-id}")
}

// No UrlTemplatization on the config means the feature is off for that workload: the entry has to
// be dropped, not left behind with the rules of the previous revision.
func TestOnSet_WithoutUrlTemplatizationDeletesEntry(t *testing.T) {
	proc, _ := newStartedProcessor(t)
	proc.OnSet("checkout", templatizationConfig([]string{"/user/{user-id}"}, nil))
	proc.OnSet("checkout", &commonapi.ContainerCollectorConfig{ContainerName: "app"})

	_, found := proc.parsedRulesCache.get("checkout")
	require.False(t, found)

	out, err := proc.processTraces(context.Background(), serverSpanForWorkload(t, "checkout", map[string]any{
		"http.request.method": "GET",
		"url.path":            "/user/john",
	}))
	require.NoError(t, err)
	span := processedSpanAt(t, out, 0)
	requireNoAttr(t, span, semconv.AttributeHTTPRoute)
	require.Equal(t, "GET", span.Name())
}

func TestOnSet_NoRulesFallsBackToDefaultHeuristic(t *testing.T) {
	proc, _ := newStartedProcessor(t)
	proc.OnSet("checkout", templatizationConfig(nil, &commonactionsapi.DefaultTemplatizationConfig{}))

	entry, found := proc.parsedRulesCache.get("checkout")
	require.True(t, found)
	require.Empty(t, entry.parsedRules)

	out, err := proc.processTraces(context.Background(), serverSpanForWorkload(t, "checkout", map[string]any{
		"http.request.method": "GET",
		"url.path":            "/user/1234",
	}))
	require.NoError(t, err)
	span := processedSpanAt(t, out, 0)
	requireAttr(t, span, semconv.AttributeHTTPRoute, "/user/{id}")
	requireAttr(t, span, odigosattributes.UrlTemplatizationResultAttribute, string(odigosattributes.UrlTemplatizationResultDefaultHeuristic))
	require.Equal(t, "GET /user/{id}", span.Name())
}

func TestOnDeleteKey_StopsTemplatizingTheWorkload(t *testing.T) {
	proc, _ := newStartedProcessor(t)
	proc.OnSet("checkout", templatizationConfig([]string{"/user/{user-id}"}, nil))
	proc.OnSet("catalog", templatizationConfig([]string{"/user/{shopper}"}, nil))

	proc.OnDeleteKey("checkout")

	_, found := proc.parsedRulesCache.get("checkout")
	require.False(t, found)
	_, found = proc.parsedRulesCache.get("catalog")
	require.True(t, found, "deleting one workload must not evict the others")

	out, err := proc.processTraces(context.Background(), serverSpanForWorkload(t, "checkout", map[string]any{
		"http.request.method": "GET",
		"url.path":            "/user/john",
	}))
	require.NoError(t, err)
	requireNoAttr(t, processedSpanAt(t, out, 0), semconv.AttributeHTTPRoute)
}

// ******************************************************
// processTraces in extension mode
// ******************************************************

func TestProcessTraces_ResourceWithoutCacheEntryIsSkipped(t *testing.T) {
	proc, _ := newStartedProcessor(t)

	out, err := proc.processTraces(context.Background(), serverSpanForWorkload(t, "checkout", map[string]any{
		"http.request.method": "GET",
		"url.path":            "/user/1234",
	}))
	require.NoError(t, err)
	span := processedSpanAt(t, out, 0)
	requireNoAttr(t, span, semconv.AttributeHTTPRoute)
	require.Equal(t, "GET", span.Name())
}

// A resource we cannot identify is skipped, but it must not stop the rest of the batch.
func TestProcessTraces_UnidentifiedResourceDoesNotSkipTheBatch(t *testing.T) {
	proc, _, logs := newStartedProcessorWithLogs(t)
	proc.OnSet("checkout", templatizationConfig([]string{"/user/{user-id}"}, nil))

	td := tracesForWorkloads(t,
		workloadSpan{workload: "", kind: ptrace.SpanKindServer, attrs: map[string]any{
			"http.request.method": "GET",
			"url.path":            "/user/john",
		}},
		workloadSpan{workload: "checkout", kind: ptrace.SpanKindServer, attrs: map[string]any{
			"http.request.method": "GET",
			"url.path":            "/user/john",
		}},
	)

	out, err := proc.processTraces(context.Background(), td)
	require.NoError(t, err)
	requireNoAttr(t, processedSpanAt(t, out, 0), semconv.AttributeHTTPRoute)
	requireAttr(t, processedSpanAt(t, out, 1), semconv.AttributeHTTPRoute, "/user/{user-id}")
	// the identity error is what skips the resource; today the extension also returns an empty
	// key on error, so the log is the only way to tell the guard is still honored
	require.Equal(t, 1, logs.FilterMessage("processTraces skip resource: GetWorkloadCacheKey failed").Len())
}

func TestProcessTraces_AllScopesAndSpansOfAResource(t *testing.T) {
	proc, _ := newStartedProcessor(t)
	proc.OnSet("checkout", templatizationConfig([]string{"/user/{user-id}"}, nil))

	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr(semconv.AttributeServiceName, "checkout")
	for scope := 0; scope < 2; scope++ {
		ss := rs.ScopeSpans().AppendEmpty()
		for i := 0; i < 2; i++ {
			span := ss.Spans().AppendEmpty()
			span.SetKind(ptrace.SpanKindServer)
			span.SetName("GET")
			span.Attributes().PutStr("http.request.method", "GET")
			span.Attributes().PutStr("url.path", "/user/john")
		}
	}

	out, err := proc.processTraces(context.Background(), td)
	require.NoError(t, err)

	outRS := out.ResourceSpans().At(0)
	require.Equal(t, 2, outRS.ScopeSpans().Len())
	for scope := 0; scope < outRS.ScopeSpans().Len(); scope++ {
		spans := outRS.ScopeSpans().At(scope).Spans()
		require.Equal(t, 2, spans.Len())
		for i := 0; i < spans.Len(); i++ {
			requireAttr(t, spans.At(i), semconv.AttributeHTTPRoute, "/user/{user-id}")
		}
	}
}

// ******************************************************
// Default templatization skip policy
// ******************************************************

func TestDefaultTemplatizationSkipPolicy(t *testing.T) {
	tt := []struct {
		name              string
		skipPolicy        *commonactionsapi.DefaultTemplatizationSkipPolicyConfig
		spanKind          ptrace.SpanKind
		statusAttrKey     string
		statusAttrValue   any
		expectedRoute     string
		expectedSpanName  string
		expectedResultTag string
	}{
		{
			name:              "no skip policy templatizes any status code",
			spanKind:          ptrace.SpanKindServer,
			statusAttrKey:     semconv.AttributeHTTPResponseStatusCode,
			statusAttrValue:   404,
			expectedRoute:     "/user/{id}",
			expectedSpanName:  "GET /user/{id}",
			expectedResultTag: string(odigosattributes.UrlTemplatizationResultDefaultHeuristic),
		},
		{
			name:             "non success codes skipped",
			skipPolicy:       &commonactionsapi.DefaultTemplatizationSkipPolicyConfig{SkipForNonSuccessCodes: true},
			spanKind:         ptrace.SpanKindServer,
			statusAttrKey:    semconv.AttributeHTTPResponseStatusCode,
			statusAttrValue:  404,
			expectedSpanName: "GET",
		},
		{
			name:              "success code templatized",
			skipPolicy:        &commonactionsapi.DefaultTemplatizationSkipPolicyConfig{SkipForNonSuccessCodes: true},
			spanKind:          ptrace.SpanKindServer,
			statusAttrKey:     semconv.AttributeHTTPResponseStatusCode,
			statusAttrValue:   200,
			expectedRoute:     "/user/{id}",
			expectedSpanName:  "GET /user/{id}",
			expectedResultTag: string(odigosattributes.UrlTemplatizationResultDefaultHeuristic),
		},
		{
			name:              "299 is the last success code",
			skipPolicy:        &commonactionsapi.DefaultTemplatizationSkipPolicyConfig{SkipForNonSuccessCodes: true},
			spanKind:          ptrace.SpanKindServer,
			statusAttrKey:     semconv.AttributeHTTPResponseStatusCode,
			statusAttrValue:   299,
			expectedRoute:     "/user/{id}",
			expectedSpanName:  "GET /user/{id}",
			expectedResultTag: string(odigosattributes.UrlTemplatizationResultDefaultHeuristic),
		},
		{
			name:             "300 is not a success code",
			skipPolicy:       &commonactionsapi.DefaultTemplatizationSkipPolicyConfig{SkipForNonSuccessCodes: true},
			spanKind:         ptrace.SpanKindServer,
			statusAttrKey:    semconv.AttributeHTTPResponseStatusCode,
			statusAttrValue:  300,
			expectedSpanName: "GET",
		},
		{
			name:             "199 is not a success code",
			skipPolicy:       &commonactionsapi.DefaultTemplatizationSkipPolicyConfig{SkipForNonSuccessCodes: true},
			spanKind:         ptrace.SpanKindServer,
			statusAttrKey:    semconv.AttributeHTTPResponseStatusCode,
			statusAttrValue:  199,
			expectedSpanName: "GET",
		},
		{
			name:             "deprecated http.status_code attribute is honored",
			skipPolicy:       &commonactionsapi.DefaultTemplatizationSkipPolicyConfig{SkipForNonSuccessCodes: true},
			spanKind:         ptrace.SpanKindServer,
			statusAttrKey:    "http.status_code",
			statusAttrValue:  404,
			expectedSpanName: "GET",
		},
		{
			name:              "non integer status code is ignored",
			skipPolicy:        &commonactionsapi.DefaultTemplatizationSkipPolicyConfig{SkipForNonSuccessCodes: true},
			spanKind:          ptrace.SpanKindServer,
			statusAttrKey:     semconv.AttributeHTTPResponseStatusCode,
			statusAttrValue:   "404",
			expectedRoute:     "/user/{id}",
			expectedSpanName:  "GET /user/{id}",
			expectedResultTag: string(odigosattributes.UrlTemplatizationResultDefaultHeuristic),
		},
		{
			name:              "non integer deprecated status code is ignored",
			skipPolicy:        &commonactionsapi.DefaultTemplatizationSkipPolicyConfig{SkipForNonSuccessCodes: true},
			spanKind:          ptrace.SpanKindServer,
			statusAttrKey:     "http.status_code",
			statusAttrValue:   "404",
			expectedRoute:     "/user/{id}",
			expectedSpanName:  "GET /user/{id}",
			expectedResultTag: string(odigosattributes.UrlTemplatizationResultDefaultHeuristic),
		},
		{
			name:              "no status code attribute is not skipped",
			skipPolicy:        &commonactionsapi.DefaultTemplatizationSkipPolicyConfig{SkipForNonSuccessCodes: true},
			spanKind:          ptrace.SpanKindServer,
			expectedRoute:     "/user/{id}",
			expectedSpanName:  "GET /user/{id}",
			expectedResultTag: string(odigosattributes.UrlTemplatizationResultDefaultHeuristic),
		},
		{
			name:             "explicit status code list",
			skipPolicy:       &commonactionsapi.DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{401, 404}},
			spanKind:         ptrace.SpanKindServer,
			statusAttrKey:    semconv.AttributeHTTPResponseStatusCode,
			statusAttrValue:  404,
			expectedSpanName: "GET",
		},
		{
			name:              "status code outside the list is templatized",
			skipPolicy:        &commonactionsapi.DefaultTemplatizationSkipPolicyConfig{SkipHttpStatusCodes: []int{401, 404}},
			spanKind:          ptrace.SpanKindServer,
			statusAttrKey:     semconv.AttributeHTTPResponseStatusCode,
			statusAttrValue:   500,
			expectedRoute:     "/user/{id}",
			expectedSpanName:  "GET /user/{id}",
			expectedResultTag: string(odigosattributes.UrlTemplatizationResultDefaultHeuristic),
		},
		{
			// the policy exists to filter garbage requests hitting a public service, outgoing
			// requests are expected to be legitimate so client spans are never skipped
			name:              "client spans are never skipped",
			skipPolicy:        &commonactionsapi.DefaultTemplatizationSkipPolicyConfig{SkipForNonSuccessCodes: true},
			spanKind:          ptrace.SpanKindClient,
			statusAttrKey:     semconv.AttributeHTTPResponseStatusCode,
			statusAttrValue:   404,
			expectedRoute:     "/user/{id}",
			expectedSpanName:  "GET /user/{id}",
			expectedResultTag: string(odigosattributes.UrlTemplatizationResultDefaultHeuristic),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			proc, _ := newStartedProcessor(t)
			proc.OnSet("checkout", templatizationConfig(nil, &commonactionsapi.DefaultTemplatizationConfig{
				SkipPolicy: tc.skipPolicy,
			}))

			attrs := map[string]any{
				"http.request.method": "GET",
				"url.path":            "/user/1234",
			}
			if tc.statusAttrKey != "" {
				attrs[tc.statusAttrKey] = tc.statusAttrValue
			}

			out, err := proc.processTraces(context.Background(),
				tracesForWorkloads(t, workloadSpan{workload: "checkout", kind: tc.spanKind, attrs: attrs}))
			require.NoError(t, err)

			span := processedSpanAt(t, out, 0)
			targetAttribute := semconv.AttributeHTTPRoute
			if tc.spanKind == ptrace.SpanKindClient {
				targetAttribute = semconv.AttributeURLTemplate
			}
			if tc.expectedRoute == "" {
				requireNoAttr(t, span, targetAttribute)
				requireNoAttr(t, span, odigosattributes.UrlTemplatizationResultAttribute)
			} else {
				requireAttr(t, span, targetAttribute, tc.expectedRoute)
				requireAttr(t, span, odigosattributes.UrlTemplatizationResultAttribute, tc.expectedResultTag)
			}
			require.Equal(t, tc.expectedSpanName, span.Name())
		})
	}
}

// Custom rules are evaluated before the skip policy so a route the user explicitly declared is
// still reported for error responses.
func TestSkipPolicy_CustomRuleStillMatchesOnErrorResponse(t *testing.T) {
	proc, _ := newStartedProcessor(t)
	proc.OnSet("checkout", templatizationConfig([]string{"/user/{user-id}"}, &commonactionsapi.DefaultTemplatizationConfig{
		SkipPolicy: &commonactionsapi.DefaultTemplatizationSkipPolicyConfig{SkipForNonSuccessCodes: true},
	}))

	out, err := proc.processTraces(context.Background(), serverSpanForWorkload(t, "checkout", map[string]any{
		"http.request.method":       "GET",
		"url.path":                  "/user/john",
		"http.response.status_code": 404,
	}))
	require.NoError(t, err)

	span := processedSpanAt(t, out, 0)
	requireAttr(t, span, semconv.AttributeHTTPRoute, "/user/{user-id}")
	requireAttr(t, span, odigosattributes.UrlTemplatizationResultAttribute, string(odigosattributes.UrlTemplatizationResultCustomRule))
}

func TestDefaultTemplatization_DisabledKeepsCustomRulesOnly(t *testing.T) {
	proc, _ := newStartedProcessor(t)
	proc.OnSet("checkout", templatizationConfig([]string{"/user/{user-id}"}, &commonactionsapi.DefaultTemplatizationConfig{
		Disabled: true,
	}))

	out, err := proc.processTraces(context.Background(), serverSpanForWorkload(t, "checkout", map[string]any{
		"http.request.method": "GET",
		"url.path":            "/user/john",
	}))
	require.NoError(t, err)
	requireAttr(t, processedSpanAt(t, out, 0), semconv.AttributeHTTPRoute, "/user/{user-id}")

	// a path no rule matches is left alone instead of falling back to the heuristic
	out, err = proc.processTraces(context.Background(), serverSpanForWorkload(t, "checkout", map[string]any{
		"http.request.method": "GET",
		"url.path":            "/order/1234",
	}))
	require.NoError(t, err)
	span := processedSpanAt(t, out, 0)
	requireNoAttr(t, span, semconv.AttributeHTTPRoute)
	require.Equal(t, "GET", span.Name())
}

// A path made only of slashes is normalized before the rules and before the skip policy, so it is
// reported even when default templatization is disabled.
func TestDefaultTemplatization_AllSlashesPathNormalized(t *testing.T) {
	proc, _ := newStartedProcessor(t)
	proc.OnSet("checkout", templatizationConfig(nil, &commonactionsapi.DefaultTemplatizationConfig{Disabled: true}))

	out, err := proc.processTraces(context.Background(), serverSpanForWorkload(t, "checkout", map[string]any{
		"http.request.method": "GET",
		"url.path":            "///",
	}))
	require.NoError(t, err)

	span := processedSpanAt(t, out, 0)
	requireAttr(t, span, semconv.AttributeHTTPRoute, "/")
	requireAttr(t, span, odigosattributes.UrlTemplatizationResultAttribute, string(odigosattributes.UrlTemplatizationResultPathNormalization))
	require.Equal(t, "GET /", span.Name())
}

// ******************************************************
// Existing target attribute on the span
// ******************************************************

func TestEnhanceSpan_ExistingRouteAttributeIsNotOverridden(t *testing.T) {
	proc, _ := newStartedProcessor(t)
	proc.OnSet("checkout", templatizationConfig(nil, &commonactionsapi.DefaultTemplatizationConfig{}))

	out, err := proc.processTraces(context.Background(), serverSpanForWorkload(t, "checkout", map[string]any{
		"http.request.method": "GET",
		"url.path":            "/user/1234",
		"http.route":          "/user/:id",
	}))
	require.NoError(t, err)

	span := processedSpanAt(t, out, 0)
	requireAttr(t, span, semconv.AttributeHTTPRoute, "/user/:id")
	requireNoAttr(t, span, odigosattributes.UrlTemplatizationResultAttribute)
	require.Equal(t, "GET", span.Name())
}

func TestEnhanceSpan_EmptyRouteAttributeNamesSpanWithRoot(t *testing.T) {
	proc, _ := newStartedProcessor(t)
	proc.OnSet("checkout", templatizationConfig(nil, &commonactionsapi.DefaultTemplatizationConfig{}))

	out, err := proc.processTraces(context.Background(), serverSpanForWorkload(t, "checkout", map[string]any{
		"http.request.method": "GET",
		"url.path":            "/user/1234",
		"http.route":          "",
	}))
	require.NoError(t, err)

	span := processedSpanAt(t, out, 0)
	requireAttr(t, span, semconv.AttributeHTTPRoute, "")
	require.Equal(t, "GET /", span.Name())
}

func TestEnhanceSpan_NonStringRouteAttributeIsLeftAlone(t *testing.T) {
	proc, _ := newStartedProcessor(t)
	proc.OnSet("checkout", templatizationConfig(nil, &commonactionsapi.DefaultTemplatizationConfig{}))

	out, err := proc.processTraces(context.Background(), serverSpanForWorkload(t, "checkout", map[string]any{
		"http.request.method": "GET",
		"url.path":            "/user/1234",
		"http.route":          42,
	}))
	require.NoError(t, err)

	span := processedSpanAt(t, out, 0)
	requireAttr(t, span, semconv.AttributeHTTPRoute, "42")
	requireNoAttr(t, span, odigosattributes.UrlTemplatizationResultAttribute)
	require.Equal(t, "GET", span.Name())
}

// ******************************************************
// Config validation and construction
// ******************************************************

func TestConfigValidate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := Config{
			TemplatizationConfig: TemplatizationConfig{
				TemplatizationRules: []string{"/user/{user-id}"},
				CustomIds:           []CustomIdConfig{{Regexp: `^ap\d+$`, TemplateName: "appId"}},
			},
			OdigosConfigExtension: &extensionComponentID,
		}
		require.NoError(t, cfg.Validate())
	})

	t.Run("invalid custom id regexp", func(t *testing.T) {
		cfg := Config{TemplatizationConfig: TemplatizationConfig{
			CustomIds: []CustomIdConfig{{Regexp: "("}},
		}}
		require.ErrorContains(t, cfg.Validate(), "invalid custom id regexp")
	})

	t.Run("extension id without a type", func(t *testing.T) {
		emptyID := component.ID{}
		cfg := Config{OdigosConfigExtension: &emptyID}
		require.ErrorContains(t, cfg.Validate(), `invalid odigos_config_extension type ""`)
	})
}

func TestNewUrlTemplateProcessor_InvalidCustomIdRegexp(t *testing.T) {
	set, _ := newDynamicConfigSettings(t)
	proc, err := newUrlTemplateProcessor(set, &Config{TemplatizationConfig: TemplatizationConfig{
		CustomIds: []CustomIdConfig{{Regexp: "("}},
	}})
	require.ErrorContains(t, err, "invalid custom id regex")
	require.Nil(t, proc)
}

// The component built by the factory must run Start/Shutdown, otherwise the processor never
// registers with the extension and per-workload templatization silently stops working.
func TestCreateTracesProcessor_WiresStartAndShutdown(t *testing.T) {
	ext := &dynamicConfigExtension{cacheSynced: true}
	sink := new(consumertest.TracesSink)
	factory := NewFactory()

	tp, err := factory.CreateTraces(context.Background(), processortest.NewNopSettings(factory.Type()),
		&Config{OdigosConfigExtension: &extensionComponentID}, sink)
	require.NoError(t, err)
	require.NoError(t, tp.Start(context.Background(), extensionHost{
		extensions: map[component.ID]component.Component{extensionComponentID: ext},
	}))
	require.Len(t, ext.registered, 1)

	cb, ok := ext.registered[0].(collector.WorkloadConfigCacheCallback)
	require.True(t, ok)
	cb.OnSet("checkout", templatizationConfig([]string{"/user/{user-id}"}, nil))

	require.NoError(t, tp.ConsumeTraces(context.Background(), serverSpanForWorkload(t, "checkout", map[string]any{
		"http.request.method": "GET",
		"url.path":            "/user/john",
	})))
	require.Len(t, sink.AllTraces(), 1)
	requireAttr(t, processedSpanAt(t, sink.AllTraces()[0], 0), semconv.AttributeHTTPRoute, "/user/{user-id}")

	require.NoError(t, tp.Shutdown(context.Background()))
	require.Len(t, ext.unregistered, 1)
}

func TestCreateTracesProcessor_InvalidCustomIdRegexp(t *testing.T) {
	factory := NewFactory()
	_, err := factory.CreateTraces(context.Background(), processortest.NewNopSettings(factory.Type()),
		&Config{TemplatizationConfig: TemplatizationConfig{
			CustomIds: []CustomIdConfig{{Regexp: "("}},
		}}, nil)
	require.ErrorContains(t, err, "invalid custom id regex")
}
