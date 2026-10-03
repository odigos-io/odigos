package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

// goffMod is one module of an offsets file, reduced to the two things the
// service reads out of it: the module path and the versions it supports.
type goffMod struct {
	name     string
	versions []string
}

// goffFileJSON renders an offsets file the way the public offsets URL serves it
// — raw JSON, with the module/package/struct/field/offset nesting the parser
// walks. The wire format is spelled out here rather than marshalled from the
// production structs, so a struct-tag change shows up as a test failure.
func goffFileJSON(timestamp string, mods ...goffMod) string {
	parts := make([]string, 0, len(mods))
	for _, mod := range mods {
		quoted := make([]string, 0, len(mod.versions))
		for _, v := range mod.versions {
			quoted = append(quoted, fmt.Sprintf("%q", v))
		}
		parts = append(parts, fmt.Sprintf(
			`{"module":%q,"packages":[{"package":%q,"structs":[{"struct":"Foo","fields":[{"field":"Bar","offsets":[{"offset":8,"versions":[%s]}]}]}]}]}`,
			mod.name, mod.name+"/pkg", strings.Join(quoted, ",")))
	}
	fields := make([]string, 0, 2)
	// An empty timestamp has to be omitted rather than sent as "": time.Time
	// rejects the empty string, which would fail the fixture instead of
	// exercising the zero-timestamp branch.
	if timestamp != "" {
		fields = append(fields, fmt.Sprintf(`"timestamp":%q`, timestamp))
	}
	fields = append(fields, fmt.Sprintf(`"mods":[%s]`, strings.Join(parts, ",")))
	return "{" + strings.Join(fields, ",") + "}"
}

// goffPayload wraps an offsets file the way the ConfigMap stores it: a single
// JSON-encoded string. Written out independently of encodeGoOffsets so the
// round trip is not asserted through the encoder it is meant to check.
func goffPayload(t *testing.T, fileContent string) string {
	t.Helper()
	encoded, err := json.Marshal(fileContent)
	require.NoError(t, err)
	return string(encoded)
}

// goffCache stands in for the informer-backed cache client. Reads in this
// service come from the cache while writes go through the live typed client, so
// the two are deliberately separate stores and a write is not visible to the
// next read until the informer catches up. Serving the resourceVersion from a
// script lets a test decide after how many polls that happens.
type goffCache struct {
	client.Client

	mu   sync.Mutex
	gets int

	// configMap is what Get returns; nil makes the cache report NotFound.
	configMap *corev1.ConfigMap
	// getErr, when set, is returned instead of reading configMap.
	getErr error
	// resourceVersions is consumed one entry per Get, the last entry repeating.
	resourceVersions []string
}

func (c *goffCache) Get(_ context.Context, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gets++

	if c.getErr != nil {
		return c.getErr
	}
	if c.configMap == nil {
		return apierrors.NewNotFound(corev1.Resource("configmaps"), key.Name)
	}
	out, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return fmt.Errorf("goffCache: unexpected object type %T", obj)
	}
	c.configMap.DeepCopyInto(out)
	if len(c.resourceVersions) > 0 {
		out.ResourceVersion = c.resourceVersions[min(c.gets-1, len(c.resourceVersions)-1)]
	}
	return nil
}

func (c *goffCache) getCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.gets
}

// goffCurrentCache is a cache that already agrees with whatever the fake typed
// client reports for a write (it reports no resourceVersion at all), so the
// post-write wait resolves on its first poll and write tests stay fast.
func goffCurrentCache() *goffCache {
	return &goffCache{configMap: &corev1.ConfigMap{}, resourceVersions: []string{""}}
}

// goffUseClients points the service at a fake cluster and restores the previous
// globals afterwards.
func goffUseClients(t *testing.T, cache client.Client, liveObjects ...runtime.Object) *k8sfake.Clientset {
	t.Helper()

	clientset := k8sfake.NewSimpleClientset(liveObjects...)
	previousCache := kube.CacheClient
	previousDefault := kube.DefaultClient
	kube.CacheClient = cache
	kube.SetDefaultClient(&kube.Client{Interface: clientset})
	t.Cleanup(func() {
		kube.CacheClient = previousCache
		kube.SetDefaultClient(previousDefault)
	})
	return clientset
}

func goffOffsetsConfigMap(data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.GoOffsetsConfigMap,
			Namespace: env.GetCurrentNamespace(),
		},
		Data: data,
	}
}

// goffDeploymentConfigMap is the object GetTier reads the cluster's tier from.
// An empty tier leaves the key out entirely.
func goffDeploymentConfigMap(tier string) *corev1.ConfigMap {
	data := map[string]string{}
	if tier != "" {
		data[k8sconsts.OdigosDeploymentConfigMapTierKey] = tier
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosDeploymentConfigMapName,
			Namespace: env.GetCurrentNamespace(),
		},
		Data: data,
	}
}

// goffWriteActions lists every action that would have changed cluster state, so
// a refused mutation can be shown to have written nothing at all rather than
// merely to have returned an error.
func goffWriteActions(clientset *k8sfake.Clientset) []string {
	var writes []string
	for _, action := range clientset.Actions() {
		switch action.GetVerb() {
		case "get", "list", "watch":
			continue
		}
		writes = append(writes, action.GetVerb()+" "+action.GetResource().Resource)
	}
	return writes
}

func goffStoredOffsets(t *testing.T, clientset *k8sfake.Clientset) *corev1.ConfigMap {
	t.Helper()
	cm, err := clientset.CoreV1().ConfigMaps(env.GetCurrentNamespace()).
		Get(context.Background(), k8sconsts.GoOffsetsConfigMap, metav1.GetOptions{})
	require.NoError(t, err)
	return cm
}

// A missing ConfigMap is a cluster state the UI explains; a ConfigMap that is
// there but does not carry the offsets key is malformed and must stay an error,
// because recreating it would discard whatever else it holds. The two read as
// the same empty table if they are ever collapsed.
func TestGetGoOffsets_configMapStates(t *testing.T) {
	validFile := goffFileJSON("2026-07-29T00:16:13.51429777Z", goffMod{name: "example.com/mod", versions: []string{"1.0.0"}})

	tests := []struct {
		name          string
		data          map[string]string
		absent        bool
		wantErr       string
		wantInstalled bool
		wantMods      int
	}{
		{
			name:          "configmap absent",
			absent:        true,
			wantInstalled: false,
		},
		{
			name:          "offsets present",
			data:          map[string]string{k8sconsts.GoOffsetsFileName: goffPayload(t, validFile)},
			wantInstalled: true,
			wantMods:      1,
		},
		{
			name:          "key present but empty",
			data:          map[string]string{k8sconsts.GoOffsetsFileName: ""},
			wantInstalled: true,
		},
		{
			name:          "key present but only whitespace",
			data:          map[string]string{k8sconsts.GoOffsetsFileName: "   \n  "},
			wantInstalled: true,
		},
		{
			name:          "payload is only a signature trailer",
			data:          map[string]string{k8sconsts.GoOffsetsFileName: goffPayload(t, "---SIGNATURE---deadbeef")},
			wantInstalled: true,
		},
		{
			// A confusable neighbour must not be mistaken for the real key.
			name:    "only a lookalike key",
			data:    map[string]string{k8sconsts.GoOffsetsFileName + ".bak": goffPayload(t, validFile)},
			wantErr: fmt.Sprintf("key %q not found in ConfigMap", k8sconsts.GoOffsetsFileName),
		},
		{
			name:    "no keys at all",
			data:    map[string]string{},
			wantErr: fmt.Sprintf("key %q not found in ConfigMap", k8sconsts.GoOffsetsFileName),
		},
		{
			// The ConfigMap stores a JSON-encoded string; the bare file is what
			// the offsets URL serves and is not valid here.
			name:    "payload stored unwrapped",
			data:    map[string]string{k8sconsts.GoOffsetsFileName: validFile},
			wantErr: "invalid go offsets JSON",
		},
		{
			name:    "payload wraps garbage",
			data:    map[string]string{k8sconsts.GoOffsetsFileName: goffPayload(t, "{not json")},
			wantErr: "invalid go offsets JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cacheObjects []client.Object
			if !tt.absent {
				cacheObjects = append(cacheObjects, goffOffsetsConfigMap(tt.data))
			}
			goffUseClients(t, newFakeClient(newScheme(), cacheObjects))

			offsets, err := GetGoOffsets(context.Background())
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, offsets)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, offsets)
			assert.Equal(t, tt.wantInstalled, offsets.Installed)
			// mods is a non-null GraphQL list: nil would marshal as null and
			// break the query rather than render an empty table.
			assert.NotNil(t, offsets.Mods)
			assert.Len(t, offsets.Mods, tt.wantMods)
		})
	}
}

// Only a NotFound means "not installed". Any other read failure — an RBAC denial
// is the usual one — has to surface as an error, because reporting it as absence
// invites the operator to recreate a ConfigMap that is already there.
func TestGetGoOffsets_readFailureIsNotMistakenForAbsence(t *testing.T) {
	denied := apierrors.NewForbidden(corev1.Resource("configmaps"), k8sconsts.GoOffsetsConfigMap, errors.New("no permission"))
	goffUseClients(t, &goffCache{getErr: denied})

	offsets, err := GetGoOffsets(context.Background())
	require.Error(t, err)
	assert.Nil(t, offsets)
	// One ordered phrase: namespace then name, so a swapped pair of format
	// arguments is visible.
	assert.ErrorContains(t, err, fmt.Sprintf("failed to get ConfigMap %s/%s",
		env.GetCurrentNamespace(), k8sconsts.GoOffsetsConfigMap))
	assert.True(t, apierrors.IsForbidden(err), "the original API error has to stay unwrappable")
}

// The read error has to name the namespace and object it failed on, in that
// order, so an operator can tell which cluster object to look at.
func TestGetGoOffsets_readFailureNamesTheConfigMap(t *testing.T) {
	cm := goffOffsetsConfigMap(map[string]string{})
	goffUseClients(t, newFakeClient(newScheme(), []client.Object{cm}))

	_, err := GetGoOffsets(context.Background())
	require.Error(t, err)
	assert.ErrorContains(t, err, fmt.Sprintf("not found in ConfigMap %s/%s",
		env.GetCurrentNamespace(), k8sconsts.GoOffsetsConfigMap))
}

// The version range is normalised by the version library while the per-minor
// lists keep the file's raw strings, and both orderings are numeric: a lexical
// sort puts 1.10 below 1.2 and reports the wrong supported range.
func TestGetGoOffsets_rangeIsNormalisedAndOrderedNumerically(t *testing.T) {
	file := goffFileJSON("2026-07-29T00:16:13.51429777Z", goffMod{
		name:     "example.com/mod",
		versions: []string{"1.2.0", "1.10.0", "1.0", "1.10.3"},
	})
	goffUseClients(t, newFakeClient(newScheme(), []client.Object{
		goffOffsetsConfigMap(map[string]string{k8sconsts.GoOffsetsFileName: goffPayload(t, file)}),
	}))

	offsets, err := GetGoOffsets(context.Background())
	require.NoError(t, err)
	require.Len(t, offsets.Mods, 1)

	mod := offsets.Mods[0]
	// "1.0" is normalised to three segments for the range, ...
	assert.Equal(t, "1.0.0", mod.MinVersion)
	assert.Equal(t, "1.10.3", mod.MaxVersion)

	// ... but the listed versions stay exactly as the file spelled them.
	got := make(map[string][]string, len(mod.MinorVersions))
	order := make([]string, 0, len(mod.MinorVersions))
	for _, minor := range mod.MinorVersions {
		order = append(order, minor.MinorVersion)
		got[minor.MinorVersion] = minor.Versions
	}
	assert.Equal(t, []string{"1.10", "1.2", "1.0"}, order)
	assert.Equal(t, []string{"1.10.0", "1.10.3"}, got["1.10"])
	assert.Equal(t, []string{"1.0"}, got["1.0"])

	assert.Equal(t, "2026-07-29T00:16:13.51429777Z", offsets.Timestamp)
}

// Replacing the offsets is a pro-tier feature. The refusal has to leave the
// cluster untouched, and the permitted tiers have to actually get through, or
// the gate is only half tested.
func TestUpdateGoOffsets_tierGate(t *testing.T) {
	file := goffFileJSON("2026-07-30T00:00:00Z", goffMod{name: "example.com/mod", versions: []string{"1.0.0"}})

	tests := []struct {
		name        string
		tier        string
		noConfigMap bool
		wantRefused bool
	}{
		{name: "community is refused", tier: string(model.TierCommunity), wantRefused: true},
		{
			// GetTier falls back to community when it cannot read the tier, so a
			// cluster with no deployment ConfigMap fails closed.
			name:        "unreadable tier fails closed",
			noConfigMap: true,
			wantRefused: true,
		},
		{name: "onprem is allowed", tier: string(model.TierOnprem)},
		{name: "cloud is allowed", tier: string(model.TierCloud)},
		{
			// Current behaviour: a deployment ConfigMap that is present but
			// carries no tier key reads as an unknown tier, which is not the
			// community literal and so is not refused.
			name: "unknown tier is not refused",
		},
	}

	refused, allowed := 0, 0
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var liveObjects []runtime.Object
			if !tt.noConfigMap {
				liveObjects = append(liveObjects, goffDeploymentConfigMap(tt.tier))
			}
			clientset := goffUseClients(t, goffCurrentCache(), liveObjects...)

			err := UpdateGoOffsets(context.Background(), file)
			if tt.wantRefused {
				refused++
				require.Error(t, err)
				assert.ErrorContains(t, err, "only available in Odigos pro tier")
				// A refusal that still wrote would hand community clusters the
				// feature anyway.
				assert.Empty(t, goffWriteActions(clientset))
				return
			}

			allowed++
			require.NoError(t, err)
			stored := goffStoredOffsets(t, clientset)
			assert.Equal(t, goffPayload(t, file), stored.Data[k8sconsts.GoOffsetsFileName])
		})
	}

	// Guards against a table where every row happens to land on one branch.
	assert.Equal(t, 2, refused)
	assert.Equal(t, 3, allowed)
}

// An update writes one key of an existing ConfigMap. Everything else in it —
// other data keys, and the ownership metadata Helm needs to adopt the object —
// has to survive, because the object is handed back to Update wholesale.
func TestUpdateGoOffsets_updatePreservesTheRestOfTheConfigMap(t *testing.T) {
	oldFile := goffFileJSON("2026-07-01T00:00:00Z", goffMod{name: "example.com/mod", versions: []string{"1.0.0"}})
	newFile := goffFileJSON("2026-07-30T00:00:00Z", goffMod{name: "example.com/mod", versions: []string{"1.0.0", "1.1.0"}})

	existing := goffOffsetsConfigMap(map[string]string{
		k8sconsts.GoOffsetsFileName: goffPayload(t, oldFile),
		"unrelated.json":            "keep me",
	})
	existing.Labels = map[string]string{
		helmManagedByLabel:             "Helm",
		k8sconsts.OdigosSystemLabelKey: "true",
		"app.kubernetes.io/part-of":    "odigos",
	}
	existing.Annotations = map[string]string{
		helmReleaseNameAnnotation: "odigos",
		helmReleaseNsAnnotation:   "odigos-system",
		"unrelated/annotation":    "keep me too",
	}

	clientset := goffUseClients(t, goffCurrentCache(),
		goffDeploymentConfigMap(string(model.TierOnprem)), existing.DeepCopy())

	require.NoError(t, UpdateGoOffsets(context.Background(), newFile))

	stored := goffStoredOffsets(t, clientset)
	// The write has to have changed something, or the assertions below would
	// also pass against an update that never happened.
	assert.NotEqual(t, goffPayload(t, oldFile), stored.Data[k8sconsts.GoOffsetsFileName])
	assert.Equal(t, goffPayload(t, newFile), stored.Data[k8sconsts.GoOffsetsFileName])
	assert.Equal(t, "keep me", stored.Data["unrelated.json"])
	assert.Equal(t, existing.Labels, stored.Labels)
	assert.Equal(t, existing.Annotations, stored.Annotations)
}

func TestUpdateGoOffsets_updateWithNoDataMap(t *testing.T) {
	file := goffFileJSON("2026-07-30T00:00:00Z", goffMod{name: "example.com/mod", versions: []string{"1.0.0"}})

	existing := goffOffsetsConfigMap(nil)
	require.Nil(t, existing.Data)
	clientset := goffUseClients(t, goffCurrentCache(),
		goffDeploymentConfigMap(string(model.TierOnprem)), existing)

	require.NoError(t, UpdateGoOffsets(context.Background(), file))
	assert.Equal(t, goffPayload(t, file), goffStoredOffsets(t, clientset).Data[k8sconsts.GoOffsetsFileName])
}

// Recreating the ConfigMap is the recovery path after it was deleted, and Helm
// refuses to adopt a resource it did not create: without the release's
// ownership metadata copied off a sibling the chart owns, the next
// `helm upgrade` fails on this object. Only those three keys may be copied —
// inheriting the sibling's other metadata would mislabel the object.
func TestUpdateGoOffsets_recreateCopiesHelmOwnership(t *testing.T) {
	file := goffFileJSON("2026-07-30T00:00:00Z", goffMod{name: "example.com/mod", versions: []string{"1.0.0"}})

	systemOnly := map[string]string{k8sconsts.OdigosSystemLabelKey: "true"}

	tests := []struct {
		name            string
		siblingLabels   map[string]string
		siblingAnnos    map[string]string
		noSibling       bool
		wantLabels      map[string]string
		wantAnnotations map[string]string
	}{
		{
			name:            "no sibling to copy from",
			noSibling:       true,
			wantLabels:      systemOnly,
			wantAnnotations: map[string]string{},
		},
		{
			name:          "full helm ownership",
			siblingLabels: map[string]string{helmManagedByLabel: "Helm", "app.kubernetes.io/part-of": "odigos"},
			siblingAnnos: map[string]string{
				helmReleaseNameAnnotation: "odigos",
				helmReleaseNsAnnotation:   "odigos-system",
				"unrelated/annotation":    "ignored",
			},
			wantLabels: map[string]string{
				k8sconsts.OdigosSystemLabelKey: "true",
				helmManagedByLabel:             "Helm",
			},
			wantAnnotations: map[string]string{
				helmReleaseNameAnnotation: "odigos",
				helmReleaseNsAnnotation:   "odigos-system",
			},
		},
		{
			// A CLI install owns the sibling without Helm metadata.
			name:            "sibling has no ownership metadata",
			siblingLabels:   map[string]string{"app.kubernetes.io/part-of": "odigos"},
			siblingAnnos:    map[string]string{"unrelated/annotation": "ignored"},
			wantLabels:      systemOnly,
			wantAnnotations: map[string]string{},
		},
		{
			name:            "ownership keys present but empty",
			siblingLabels:   map[string]string{helmManagedByLabel: ""},
			siblingAnnos:    map[string]string{helmReleaseNameAnnotation: "", helmReleaseNsAnnotation: ""},
			wantLabels:      systemOnly,
			wantAnnotations: map[string]string{},
		},
		{
			// Neighbouring keys that are not the ones Helm looks at.
			name:          "lookalike ownership keys",
			siblingLabels: map[string]string{"odigos.io/managed-by": "Helm", "managed-by": "Helm"},
			siblingAnnos: map[string]string{
				"helm.sh/release-name":         "odigos",
				"meta.helm.sh/release":         "odigos",
				"meta.helm.sh/release-name-ns": "odigos-system",
			},
			wantLabels:      systemOnly,
			wantAnnotations: map[string]string{},
		},
		{
			// Each annotation is copied on its own, so dropping one is visible.
			name:          "only the release name is set",
			siblingLabels: map[string]string{helmManagedByLabel: "Helm"},
			siblingAnnos:  map[string]string{helmReleaseNameAnnotation: "odigos"},
			wantLabels: map[string]string{
				k8sconsts.OdigosSystemLabelKey: "true",
				helmManagedByLabel:             "Helm",
			},
			wantAnnotations: map[string]string{helmReleaseNameAnnotation: "odigos"},
		},
		{
			name:          "only the release namespace is set",
			siblingLabels: map[string]string{helmManagedByLabel: "Helm"},
			siblingAnnos:  map[string]string{helmReleaseNsAnnotation: "odigos-system"},
			wantLabels: map[string]string{
				k8sconsts.OdigosSystemLabelKey: "true",
				helmManagedByLabel:             "Helm",
			},
			wantAnnotations: map[string]string{helmReleaseNsAnnotation: "odigos-system"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			liveObjects := []runtime.Object{goffDeploymentConfigMap(string(model.TierOnprem))}
			if !tt.noSibling {
				liveObjects = append(liveObjects, &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:        consts.OdigosConfigurationName,
						Namespace:   env.GetCurrentNamespace(),
						Labels:      tt.siblingLabels,
						Annotations: tt.siblingAnnos,
					},
				})
			}
			clientset := goffUseClients(t, goffCurrentCache(), liveObjects...)

			require.NoError(t, UpdateGoOffsets(context.Background(), file))

			stored := goffStoredOffsets(t, clientset)
			assert.Equal(t, tt.wantLabels, stored.Labels)
			assert.Equal(t, tt.wantAnnotations, stored.Annotations)
			// A recreated ConfigMap that odiglet cannot mount is no recovery.
			assert.Equal(t, goffPayload(t, file), stored.Data[k8sconsts.GoOffsetsFileName])
		})
	}
}

// The recreate branch is reached on NotFound alone. A read that failed for any
// other reason must not fall through to it: creating over a ConfigMap that is
// already there fails, and the mutation would report that instead of the real
// permission problem.
func TestUpdateGoOffsets_unreadableConfigMapDoesNotFallThroughToRecreate(t *testing.T) {
	file := goffFileJSON("2026-07-30T00:00:00Z", goffMod{name: "example.com/mod", versions: []string{"1.0.0"}})

	clientset := goffUseClients(t, goffCurrentCache(), goffDeploymentConfigMap(string(model.TierOnprem)))
	clientset.PrependReactor("get", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		if action.(k8stesting.GetAction).GetName() != k8sconsts.GoOffsetsConfigMap {
			return false, nil, nil
		}
		return true, nil, apierrors.NewForbidden(corev1.Resource("configmaps"), k8sconsts.GoOffsetsConfigMap, errors.New("no permission"))
	})

	err := UpdateGoOffsets(context.Background(), file)
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to get ConfigMap")
	assert.Empty(t, goffWriteActions(clientset))
}

// A rejected write has to name the object it was for; silently succeeding would
// leave the UI showing offsets the cluster never accepted.
func TestUpdateGoOffsets_wrapsWriteFailures(t *testing.T) {
	file := goffFileJSON("2026-07-30T00:00:00Z", goffMod{name: "example.com/mod", versions: []string{"1.0.0"}})
	target := fmt.Sprintf("%s/%s", env.GetCurrentNamespace(), k8sconsts.GoOffsetsConfigMap)

	tests := []struct {
		name     string
		verb     string
		existing bool
		wantErr  string
	}{
		{name: "update is rejected", verb: "update", existing: true, wantErr: "failed to update ConfigMap " + target},
		{name: "create is rejected", verb: "create", wantErr: "failed to create ConfigMap " + target},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			liveObjects := []runtime.Object{goffDeploymentConfigMap(string(model.TierOnprem))}
			if tt.existing {
				liveObjects = append(liveObjects, goffOffsetsConfigMap(map[string]string{}))
			}
			clientset := goffUseClients(t, goffCurrentCache(), liveObjects...)
			clientset.PrependReactor(tt.verb, "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, apierrors.NewForbidden(corev1.Resource("configmaps"), k8sconsts.GoOffsetsConfigMap, errors.New("no permission"))
			})

			err := UpdateGoOffsets(context.Background(), file)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// An update replaces the ConfigMap wholesale, so checking an empty candidate —
// a fetch the browser truncated, say — has to warn that it would drop
// everything rather than report nothing to do.
func TestCheckGoOffsetsUpdates_emptyCandidateReportsEverythingRemoved(t *testing.T) {
	installed := goffFileJSON("2026-07-01T00:00:00Z", goffMod{name: "example.com/mod", versions: []string{"1.0.0", "1.1.0"}})
	goffUseClients(t, newFakeClient(newScheme(), []client.Object{
		goffOffsetsConfigMap(map[string]string{k8sconsts.GoOffsetsFileName: goffPayload(t, installed)}),
	}))

	check, err := CheckGoOffsetsUpdates(context.Background(), "   ")
	require.NoError(t, err)
	assert.True(t, check.HasUpdates)
	assert.Equal(t, "", check.ProposedTimestamp)
	require.Len(t, check.Mods, 1)
	assert.True(t, check.Mods[0].IsRemoved)
	assert.False(t, check.Mods[0].IsNew)
	for _, minor := range check.Mods[0].MinorVersions {
		assert.True(t, minor.IsRemoved, "minor %s", minor.MinorVersion)
		for _, ver := range minor.Versions {
			assert.True(t, ver.IsRemoved, "version %s", ver.Version)
		}
	}
}

// Whatever the browser fetched has to come back out of the ConfigMap unchanged,
// signature trailer and all: the writer JSON-encodes the file into one key and
// the reader has to undo exactly that before parsing.
func TestUpdateGoOffsets_writtenPayloadReadsBackThroughGetGoOffsets(t *testing.T) {
	file := goffFileJSON("2026-07-29T00:16:13.51429777Z",
		goffMod{name: "example.com/mod", versions: []string{"1.0.0", "1.1.0"}},
	) + "\n---SIGNATURE---\nZGVhZGJlZWY=\n"

	clientset := goffUseClients(t, goffCurrentCache(),
		goffDeploymentConfigMap(string(model.TierOnprem)))
	require.NoError(t, UpdateGoOffsets(context.Background(), file))
	written := goffStoredOffsets(t, clientset).Data[k8sconsts.GoOffsetsFileName]

	// Hand the informer cache exactly what the live client stored.
	goffUseClients(t, newFakeClient(newScheme(), []client.Object{
		goffOffsetsConfigMap(map[string]string{k8sconsts.GoOffsetsFileName: written}),
	}))

	offsets, err := GetGoOffsets(context.Background())
	require.NoError(t, err)
	assert.True(t, offsets.Installed)
	assert.Equal(t, "2026-07-29T00:16:13.51429777Z", offsets.Timestamp)
	require.Len(t, offsets.Mods, 1)
	assert.Equal(t, "example.com/mod", offsets.Mods[0].Module)
	assert.Equal(t, "1.0.0", offsets.Mods[0].MinVersion)
	assert.Equal(t, "1.1.0", offsets.Mods[0].MaxVersion)
}

// A deleted ConfigMap leaves nothing installed, so the check has to report the
// whole candidate as new rather than fail — that report is what tells the
// operator an update will put the manifest back.
func TestCheckGoOffsetsUpdates_withNoConfigMapEverythingIsNew(t *testing.T) {
	file := goffFileJSON("2026-07-30T00:00:00Z", goffMod{name: "example.com/mod", versions: []string{"1.0.0"}})
	goffUseClients(t, newFakeClient(newScheme(), nil))

	check, err := CheckGoOffsetsUpdates(context.Background(), file)
	require.NoError(t, err)
	assert.True(t, check.HasUpdates)
	assert.Equal(t, "", check.CurrentTimestamp)
	assert.Equal(t, "2026-07-30T00:00:00Z", check.ProposedTimestamp)
	require.Len(t, check.Mods, 1)
	assert.True(t, check.Mods[0].IsNew)
	assert.False(t, check.Mods[0].IsRemoved)
}

// The two timestamps are the same type and sit next to each other, so a swap
// compiles: the installed and candidate fixtures carry different values.
func TestCheckGoOffsetsUpdates_reportsBothTimestampsFromTheirOwnSide(t *testing.T) {
	installed := goffFileJSON("2026-07-01T00:00:00Z", goffMod{name: "example.com/mod", versions: []string{"1.0.0"}})
	candidate := goffFileJSON("2026-07-30T00:00:00Z", goffMod{name: "example.com/mod", versions: []string{"1.0.0", "1.1.0"}})

	goffUseClients(t, newFakeClient(newScheme(), []client.Object{
		goffOffsetsConfigMap(map[string]string{k8sconsts.GoOffsetsFileName: goffPayload(t, installed)}),
	}))

	check, err := CheckGoOffsetsUpdates(context.Background(), candidate)
	require.NoError(t, err)
	assert.Equal(t, "2026-07-01T00:00:00Z", check.CurrentTimestamp)
	assert.Equal(t, "2026-07-30T00:00:00Z", check.ProposedTimestamp)
	assert.True(t, check.HasUpdates)
}

// Neither side of the comparison may be silently treated as empty: a malformed
// installed ConfigMap, or a candidate the browser truncated, would otherwise
// report every module as new or as removed.
func TestCheckGoOffsetsUpdates_bothSidesRejectMalformedInput(t *testing.T) {
	valid := goffFileJSON("2026-07-30T00:00:00Z", goffMod{name: "example.com/mod", versions: []string{"1.0.0"}})

	t.Run("installed side", func(t *testing.T) {
		goffUseClients(t, newFakeClient(newScheme(), []client.Object{
			goffOffsetsConfigMap(map[string]string{k8sconsts.GoOffsetsFileName: "{not json"}),
		}))
		_, err := CheckGoOffsetsUpdates(context.Background(), valid)
		assert.ErrorContains(t, err, "invalid go offsets JSON")
	})

	t.Run("candidate side", func(t *testing.T) {
		goffUseClients(t, newFakeClient(newScheme(), []client.Object{
			goffOffsetsConfigMap(map[string]string{k8sconsts.GoOffsetsFileName: goffPayload(t, valid)}),
		}))
		_, err := CheckGoOffsetsUpdates(context.Background(), "{not json")
		assert.ErrorContains(t, err, "invalid go offsets JSON")
	})

	t.Run("missing key on the installed side", func(t *testing.T) {
		goffUseClients(t, newFakeClient(newScheme(), []client.Object{goffOffsetsConfigMap(map[string]string{})}))
		_, err := CheckGoOffsetsUpdates(context.Background(), valid)
		assert.ErrorContains(t, err, fmt.Sprintf("key %q not found", k8sconsts.GoOffsetsFileName))
	})
}

// Reads are served from the informer cache and writes go through the live
// client, so returning before the cache has caught up hands the UI back the
// state it just replaced.
func TestAwaitGoOffsetsCache_pollsUntilTheCacheCatchesUp(t *testing.T) {
	cache := &goffCache{
		configMap:        &corev1.ConfigMap{},
		resourceVersions: []string{"41", "41", "42"},
	}
	goffUseClients(t, cache)

	awaitGoOffsetsCache(context.Background(), env.GetCurrentNamespace(), "42")
	assert.Equal(t, 3, cache.getCount())
}

func TestAwaitGoOffsetsCache_returnsOnTheFirstPollWhenAlreadyCurrent(t *testing.T) {
	cache := &goffCache{configMap: &corev1.ConfigMap{}, resourceVersions: []string{"42"}}
	goffUseClients(t, cache)

	awaitGoOffsetsCache(context.Background(), env.GetCurrentNamespace(), "42")
	assert.Equal(t, 1, cache.getCount())
}

func TestAwaitGoOffsetsCache_stopsOnACancelledContext(t *testing.T) {
	cache := &goffCache{configMap: &corev1.ConfigMap{}, resourceVersions: []string{"41"}}
	goffUseClients(t, cache)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	awaitGoOffsetsCache(ctx, env.GetCurrentNamespace(), "42")
	assert.Equal(t, 1, cache.getCount())
}

// The write already succeeded, so a cache that never catches up must be given
// up on rather than block the mutation forever.
func TestAwaitGoOffsetsCache_givesUpWhenTheCacheNeverCatchesUp(t *testing.T) {
	cache := &goffCache{configMap: &corev1.ConfigMap{}, resourceVersions: []string{"41"}}
	goffUseClients(t, cache)

	awaitGoOffsetsCache(context.Background(), env.GetCurrentNamespace(), "42")
	// It returned, and it kept polling for a while rather than bailing out on
	// the first miss.
	assert.Greater(t, cache.getCount(), 5)
}

// The wait has to follow the resourceVersion the API server assigned to the
// write, not the one the request was built from.
func TestUpdateGoOffsets_waitsForTheResourceVersionTheWriteReturned(t *testing.T) {
	file := goffFileJSON("2026-07-30T00:00:00Z", goffMod{name: "example.com/mod", versions: []string{"1.0.0"}})

	existing := goffOffsetsConfigMap(map[string]string{k8sconsts.GoOffsetsFileName: ""})
	existing.ResourceVersion = "41"
	cache := &goffCache{
		configMap:        &corev1.ConfigMap{},
		resourceVersions: []string{"41", "41", "42"},
	}
	clientset := goffUseClients(t, cache, goffDeploymentConfigMap(string(model.TierOnprem)), existing)
	// Stand in for the API server bumping the resourceVersion on a write.
	clientset.PrependReactor("update", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		cm := action.(k8stesting.UpdateAction).GetObject().(*corev1.ConfigMap).DeepCopy()
		cm.ResourceVersion = "42"
		return true, cm, nil
	})

	require.NoError(t, UpdateGoOffsets(context.Background(), file))
	assert.Equal(t, 3, cache.getCount())
}
