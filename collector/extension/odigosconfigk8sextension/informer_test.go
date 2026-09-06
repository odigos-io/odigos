package odigosconfigk8sextension

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8scache "k8s.io/client-go/tools/cache"

	commonapi "github.com/odigos-io/odigos/common/api"
)

// newInformerTestExtension returns an extension with an empty cache and a logger whose output can
// be inspected. The informer itself is never started; the tests drive the event handlers directly.
func newInformerTestExtension(t *testing.T) (*OdigosWorkloadConfig, *observer.ObservedLogs) {
	t.Helper()
	core, logs := observer.New(zapcore.DebugLevel)
	return &OdigosWorkloadConfig{cache: newCache(), logger: zap.New(core)}, logs
}

// newInstrumentationConfig builds the unstructured InstrumentationConfig the dynamic informer
// hands to the event handlers. runtimeObjectName is the "<kind>-<workload>" object name.
func newInstrumentationConfig(namespace, runtimeObjectName string, containerNames ...string) *unstructured.Unstructured {
	containerConfigs := make([]interface{}, 0, len(containerNames))
	for _, containerName := range containerNames {
		containerConfigs = append(containerConfigs, map[string]interface{}{
			"containerName": containerName,
			"urlTemplatization": map[string]interface{}{
				"templatizationRules": []interface{}{"/users/{id}"},
			},
		})
	}
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"metadata": map[string]interface{}{
			"namespace": namespace,
			"name":      runtimeObjectName,
		},
		"spec": map[string]interface{}{
			"workloadCollectorConfig": containerConfigs,
		},
	}}
}

// recordingCallback records the cache notifications in the order the extension delivered them.
type recordingCallback struct {
	events []string
}

func (r *recordingCallback) OnSet(key string, _ *commonapi.ContainerCollectorConfig) {
	r.events = append(r.events, "set "+key)
}

func (r *recordingCallback) OnDeleteKey(key string) {
	r.events = append(r.events, "delete "+key)
}

// cacheKeys returns every key currently held in the extension cache.
func cacheKeys(c *cache) []string {
	var keys []string
	c.Range(func(key string, _ *commonapi.ContainerCollectorConfig) {
		keys = append(keys, key)
	})
	return keys
}

// TestKindFromInstrumentationConfigName pins the InstrumentationConfig name prefix to Kubernetes
// Kind mapping. This is a local copy of the k8sutils workload-kind parsing, so it cannot drift
// with a compiler error: the Kind it returns is the Kind in the cache key the informer writes,
// and it must match the Kind resolved from resource attributes at lookup time.
func TestKindFromInstrumentationConfigName(t *testing.T) {
	tests := []struct {
		lowercase string
		want      string
	}{
		{"deployment", "Deployment"},
		{"daemonset", "DaemonSet"},
		{"statefulset", "StatefulSet"},
		{"namespace", "Namespace"},
		{"staticpod", "StaticPod"},
		{"cronjob", "CronJob"},
		{"job", "Job"},
		{"deploymentconfig", "DeploymentConfig"},
		{"rollout", "Rollout"},
		{"Deployment", "Deployment"},
		{"replicaset", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.lowercase, func(t *testing.T) {
			require.Equal(t, tt.want, kindFromInstrumentationConfigName(tt.lowercase))
		})
	}
}

func TestWorkloadKeyFromObject(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		objName   string
		want      workloadKey
		wantOK    bool
	}{
		{
			name:      "deployment",
			namespace: "prod",
			objName:   "deployment-checkout",
			want:      workloadKey{Namespace: "prod", Kind: "Deployment", Name: "checkout"},
			wantOK:    true,
		},
		{
			// SplitN keeps the rest of the name intact; a workload name with dashes is common.
			name:      "workload name containing dashes",
			namespace: "prod",
			objName:   "deploymentconfig-my-checkout-service",
			want:      workloadKey{Namespace: "prod", Kind: "DeploymentConfig", Name: "my-checkout-service"},
			wantOK:    true,
		},
		{
			name:      "unsupported kind prefix",
			namespace: "prod",
			objName:   "replicaset-checkout",
			wantOK:    false,
		},
		{
			name:      "no separator",
			namespace: "prod",
			objName:   "checkout",
			wantOK:    false,
		},
		{
			// A bare kind has a recognised prefix but no workload name. The informer runs this on
			// every watch event, so reading the missing part would panic the collector.
			name:      "no separator after a recognised kind",
			namespace: "prod",
			objName:   "deployment",
			wantOK:    false,
		},
		{
			name:      "empty workload name",
			namespace: "prod",
			objName:   "deployment-",
			want:      workloadKey{Namespace: "prod", Kind: "Deployment", Name: ""},
			wantOK:    true,
		},
		{
			name:      "missing namespace",
			namespace: "",
			objName:   "deployment-checkout",
			wantOK:    false,
		},
		{
			name:      "missing object name",
			namespace: "prod",
			objName:   "",
			wantOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := newInstrumentationConfig(tt.namespace, tt.objName, "app")

			got, ok := workloadKeyFromObject(u)
			require.Equal(t, tt.wantOK, ok)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestHandleInstrumentationConfigPopulatesTheCache(t *testing.T) {
	o, _ := newInformerTestExtension(t)

	o.handleInstrumentationConfig(newInstrumentationConfig("prod", "deploymentconfig-checkout", "app", "sidecar"))

	require.ElementsMatch(t,
		[]string{"prod/DeploymentConfig/checkout/app", "prod/DeploymentConfig/checkout/sidecar"},
		cacheKeys(o.cache))

	cfg, found := o.cache.Get("prod/DeploymentConfig/checkout/app")
	require.True(t, found)
	require.Equal(t, "app", cfg.ContainerName)
	require.NotNil(t, cfg.UrlTemplatization)
	require.Equal(t, []string{"/users/{id}"}, cfg.UrlTemplatization.Templates)

	require.True(t, o.cache.isSourceWorkloadPrefix("prod/DeploymentConfig/checkout/"))
}

// An InstrumentationConfig update is applied as "set everything desired, then delete what is left
// over". A container dropped from the spec must disappear from the cache, otherwise its stale
// config keeps being applied to telemetry for as long as the collector runs.
func TestHandleInstrumentationConfigRemovesContainersDroppedFromTheSpec(t *testing.T) {
	o, _ := newInformerTestExtension(t)
	cb := &recordingCallback{}
	o.cache.addCallback(cb)

	o.handleInstrumentationConfig(newInstrumentationConfig("prod", "deployment-checkout", "app", "sidecar"))
	cb.events = nil

	o.handleInstrumentationConfig(newInstrumentationConfig("prod", "deployment-checkout", "app"))

	require.Equal(t, []string{"prod/Deployment/checkout/app"}, cacheKeys(o.cache))
	_, found := o.cache.Get("prod/Deployment/checkout/sidecar")
	require.False(t, found)

	// The documented ordering: the new state is applied before anything is removed, so a consumer
	// never observes the workload with no configuration at all.
	require.Equal(t, []string{
		"set prod/Deployment/checkout/app",
		"delete prod/Deployment/checkout/sidecar",
	}, cb.events)
}

func TestHandleInstrumentationConfigDoesNotTouchOtherWorkloads(t *testing.T) {
	o, _ := newInformerTestExtension(t)

	o.handleInstrumentationConfig(newInstrumentationConfig("prod", "deployment-checkout", "app"))
	o.handleInstrumentationConfig(newInstrumentationConfig("prod", "deployment-frontend", "app"))
	o.handleInstrumentationConfig(newInstrumentationConfig("staging", "deployment-checkout", "app"))

	// Re-syncing one workload with no containers must leave the other two alone.
	o.handleInstrumentationConfig(newInstrumentationConfig("prod", "deployment-checkout"))

	require.ElementsMatch(t,
		[]string{"prod/Deployment/frontend/app", "staging/Deployment/checkout/app"},
		cacheKeys(o.cache))
}

func TestHandleInstrumentationConfigIgnoresUnusableObjects(t *testing.T) {
	tests := []struct {
		name       string
		obj        interface{}
		wantLogged string
	}{
		{
			name:       "not an unstructured object",
			obj:        "instrumentation-config",
			wantLogged: "informer received non-unstructured object",
		},
		{
			name:       "object name is not a workload",
			obj:        newInstrumentationConfig("prod", "replicaset-checkout", "app"),
			wantLogged: "failed to get workload key from instrumentation config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, logs := newInformerTestExtension(t)

			o.handleInstrumentationConfig(tt.obj)

			require.Empty(t, cacheKeys(o.cache))
			require.Len(t, logs.FilterMessage(tt.wantLogged).All(), 1)
		})
	}
}

// A spec that cannot be read must clear the workload rather than leave the previous configuration
// applied, and it must not be confused with "the workload has no containers".
func TestHandleInstrumentationConfigClearsWorkloadWithoutContainers(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(u *unstructured.Unstructured)
		wantLogged string
	}{
		{
			name: "spec missing",
			mutate: func(u *unstructured.Unstructured) {
				delete(u.Object, "spec")
			},
			wantLogged: "failed to get instrumentation config spec; clearing workload state",
		},
		{
			name: "spec empty",
			mutate: func(u *unstructured.Unstructured) {
				u.Object["spec"] = map[string]interface{}{}
			},
			wantLogged: "failed to get instrumentation config spec; clearing workload state",
		},
		{
			name: "workloadCollectorConfig missing",
			mutate: func(u *unstructured.Unstructured) {
				u.Object["spec"] = map[string]interface{}{"serviceName": "checkout"}
			},
		},
		{
			name: "workloadCollectorConfig empty",
			mutate: func(u *unstructured.Unstructured) {
				u.Object["spec"] = map[string]interface{}{"workloadCollectorConfig": []interface{}{}}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, logs := newInformerTestExtension(t)
			o.handleInstrumentationConfig(newInstrumentationConfig("prod", "deployment-checkout", "app"))
			require.NotEmpty(t, cacheKeys(o.cache))

			u := newInstrumentationConfig("prod", "deployment-checkout", "app")
			tt.mutate(u)
			o.handleInstrumentationConfig(u)

			require.Empty(t, cacheKeys(o.cache))
			if tt.wantLogged != "" {
				require.Len(t, logs.FilterMessage(tt.wantLogged).All(), 1)
			}
		})
	}
}

// A malformed entry must not take the rest of the workload's containers down with it.
func TestParseWorkloadCollectorConfigSkipsInvalidEntries(t *testing.T) {
	o, logs := newInformerTestExtension(t)
	key := workloadKey{Namespace: "prod", Kind: "Deployment", Name: "checkout"}

	entries := o.parseWorkloadCollectorConfig(key, []interface{}{
		"not-a-map",
		map[string]interface{}{"containerName": ""},
		map[string]interface{}{"containerName": int64(7)},
		map[string]interface{}{"containerName": "app"},
	})

	require.Len(t, entries, 1)
	require.Equal(t, "prod/Deployment/checkout/app", entries[0].key)
	require.Equal(t, "app", entries[0].cfg.ContainerName)

	require.Len(t, logs.FilterMessage("failed to get container collector config from workload collector config").All(), 1)
	require.Len(t, logs.FilterMessage("skipping container collector config with empty containerName").All(), 1)
}

// The workload index entry drives IsActiveSource. Leaving it behind on delete makes the collector
// keep treating a removed Source as active for the lifetime of the process.
func TestHandleInstrumentationConfigDeleteClearsTheWorkload(t *testing.T) {
	o, _ := newInformerTestExtension(t)
	cb := &recordingCallback{}
	o.cache.addCallback(cb)

	ic := newInstrumentationConfig("prod", "deployment-checkout", "app", "sidecar")
	o.handleInstrumentationConfig(ic)
	cb.events = nil

	o.handleInstrumentationConfigDelete(ic)

	require.Empty(t, cacheKeys(o.cache))
	require.False(t, o.cache.isSourceWorkloadPrefix("prod/Deployment/checkout/"))
	_, found := o.cache.GetDataStreams("prod/Deployment/checkout/")
	require.False(t, found)
	require.ElementsMatch(t, []string{
		"delete prod/Deployment/checkout/app",
		"delete prod/Deployment/checkout/sidecar",
	}, cb.events)
}

// A watch that falls behind delivers the final delete wrapped in a tombstone. Failing to unwrap it
// leaves the workload in the cache forever.
func TestHandleInstrumentationConfigDeleteUnwrapsTombstone(t *testing.T) {
	o, _ := newInformerTestExtension(t)
	ic := newInstrumentationConfig("prod", "deployment-checkout", "app")
	o.handleInstrumentationConfig(ic)
	require.NotEmpty(t, cacheKeys(o.cache))

	o.handleInstrumentationConfigDelete(k8scache.DeletedFinalStateUnknown{Key: "prod/deployment-checkout", Obj: ic})

	require.Empty(t, cacheKeys(o.cache))
	require.False(t, o.cache.isSourceWorkloadPrefix("prod/Deployment/checkout/"))
}

func TestHandleInstrumentationConfigDeleteIgnoresUnusableObjects(t *testing.T) {
	tests := []struct {
		name string
		obj  interface{}
	}{
		{name: "not an unstructured object", obj: "instrumentation-config"},
		{name: "tombstone wrapping an unusable object", obj: k8scache.DeletedFinalStateUnknown{Key: "prod/x", Obj: "gone"}},
		{name: "object name is not a workload", obj: newInstrumentationConfig("prod", "replicaset-checkout", "app")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, _ := newInformerTestExtension(t)
			o.handleInstrumentationConfig(newInstrumentationConfig("prod", "deployment-checkout", "app"))

			require.NotPanics(t, func() { o.handleInstrumentationConfigDelete(tt.obj) })

			require.Equal(t, []string{"prod/Deployment/checkout/app"}, cacheKeys(o.cache))
		})
	}
}

func TestExtractDataStreamLabels(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		want   []string
	}{
		{
			name:   "no labels",
			labels: nil,
			want:   nil,
		},
		{
			name: "data stream labels",
			labels: map[string]string{
				"odigos.io/data-stream-payments": "true",
				"odigos.io/data-stream-checkout": "true",
			},
			want: []string{"payments", "checkout"},
		},
		{
			name:   "membership is opt-in, only \"true\" counts",
			labels: map[string]string{"odigos.io/data-stream-payments": "false"},
			want:   nil,
		},
		{
			name:   "unrelated label",
			labels: map[string]string{"odigos.io/inject-instrumentation": "true"},
			want:   nil,
		},
		{
			name:   "empty stream name",
			labels: map[string]string{"odigos.io/data-stream-": "true"},
			want:   []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.ElementsMatch(t, tt.want, extractDataStreamLabels(tt.labels))
		})
	}
}

func TestHandleInstrumentationConfigTracksDataStreamMembership(t *testing.T) {
	o, _ := newInformerTestExtension(t)

	ic := newInstrumentationConfig("prod", "deployment-checkout", "app")
	ic.SetLabels(map[string]string{"odigos.io/data-stream-payments": "true"})
	o.handleInstrumentationConfig(ic)

	streams, found := o.cache.GetDataStreams("prod/Deployment/checkout/")
	require.True(t, found)
	require.Equal(t, []string{"payments"}, streams)

	// Removing the label must remove the membership, not keep routing to the old data stream.
	ic.SetLabels(nil)
	o.handleInstrumentationConfig(ic)

	streams, found = o.cache.GetDataStreams("prod/Deployment/checkout/")
	require.True(t, found)
	require.Nil(t, streams)
}

// A workload with no container config is still a Source: it must stay in the index so data stream
// routing keeps working for it.
func TestHandleInstrumentationConfigKeepsDataStreamsForWorkloadWithoutContainers(t *testing.T) {
	o, _ := newInformerTestExtension(t)

	ic := newInstrumentationConfig("prod", "deployment-checkout")
	ic.SetLabels(map[string]string{"odigos.io/data-stream-payments": "true"})
	o.handleInstrumentationConfig(ic)

	require.Empty(t, cacheKeys(o.cache))
	streams, found := o.cache.GetDataStreams("prod/Deployment/checkout/")
	require.True(t, found)
	require.Equal(t, []string{"payments"}, streams)
}
