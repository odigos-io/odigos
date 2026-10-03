package cache

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	clientfeatures "k8s.io/client-go/features"
	clientfeaturestesting "k8s.io/client-go/features/testing"
	"k8s.io/client-go/tools/cache"
	ctrlcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// the GVKs are written out here instead of reusing the package variables, since the whole point of
// the comparison is that these are the api server kinds the wrapping applies to.
var (
	podGvk        = k8sschema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}
	deploymentGvk = k8sschema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}
	daemonSetGvk  = k8sschema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "DaemonSet"}
)

// the marker is written onto cached objects by one component and read back by another, and it has
// to keep its odigos.io prefix to survive annotation stripping.
const transformedMarkKey = "odigos.io/cache-transformed"

func transformScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, v1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	return scheme
}

// recordingListerWatcher is a cache.ListerWatcher (and deliberately not a
// cache.ListerWatcherWithContext) that records the options every call was made with.
type recordingListerWatcher struct {
	mu           sync.Mutex
	listOptions  []metav1.ListOptions
	watchOptions []metav1.ListOptions

	listResult func() (runtime.Object, error)
	watcher    watch.Interface
	watchErr   error
}

func (r *recordingListerWatcher) List(options metav1.ListOptions) (runtime.Object, error) {
	r.mu.Lock()
	r.listOptions = append(r.listOptions, options)
	r.mu.Unlock()

	if r.listResult == nil {
		return &v1.PodList{}, nil
	}
	return r.listResult()
}

func (r *recordingListerWatcher) Watch(options metav1.ListOptions) (watch.Interface, error) {
	r.mu.Lock()
	r.watchOptions = append(r.watchOptions, options)
	r.mu.Unlock()

	if r.watchErr != nil {
		return nil, r.watchErr
	}
	if r.watcher == nil {
		return watch.NewFake(), nil
	}
	return r.watcher, nil
}

func (r *recordingListerWatcher) recordedListOptions() []metav1.ListOptions {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]metav1.ListOptions{}, r.listOptions...)
}

func (r *recordingListerWatcher) firstListOptions(t *testing.T) metav1.ListOptions {
	t.Helper()
	recorded := r.recordedListOptions()
	require.NotEmpty(t, recorded, "the lister watcher was never listed")
	return recorded[0]
}

func podListOf(names ...string) *v1.PodList {
	list := &v1.PodList{ListMeta: metav1.ListMeta{ResourceVersion: "142"}}
	for _, name := range names {
		list.Items = append(list.Items, v1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		})
	}
	return list
}

// markingTransform records the objects it was handed and labels each one, so that a test can tell
// which transform function ran on which object.
func markingTransform(mark string, seen *[]string) cache.TransformFunc {
	return func(obj any) (any, error) {
		accessor, ok := obj.(metav1.Object)
		if !ok {
			return nil, errors.New("not a kubernetes object")
		}
		if seen != nil {
			*seen = append(*seen, accessor.GetName())
		}
		labels := accessor.GetLabels()
		if labels == nil {
			labels = map[string]string{}
		}
		labels["transformed-by"] = mark
		accessor.SetLabels(labels)
		return obj, nil
	}
}

func wrappedPodListerWatcher(original cache.ListerWatcher, transform cache.TransformFunc) cache.ListerWatcher {
	return getListerWatcherForGvk(podGvk, original, gvkToTransformFunc{podGvk: transform})
}

func TestIsHighMemoryGvk(t *testing.T) {
	for _, tc := range []struct {
		name string
		gvk  k8sschema.GroupVersionKind
		want bool
	}{
		{name: "pod", gvk: podGvk, want: true},
		{name: "deployment", gvk: deploymentGvk, want: true},
		{name: "daemonset", gvk: daemonSetGvk},
		{name: "statefulset", gvk: k8sschema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"}},
		{name: "configmap", gvk: k8sschema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"}},
		{
			name: "instrumentationconfig",
			gvk:  k8sschema.GroupVersionKind{Group: "odigos.io", Version: "v1alpha1", Kind: "InstrumentationConfig"},
		},
		{name: "pod in the wrong group", gvk: k8sschema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Pod"}},
		{name: "deployment in the wrong group", gvk: k8sschema.GroupVersionKind{Group: "", Version: "v1", Kind: "Deployment"}},
		{
			name: "deployment in the wrong version",
			gvk:  k8sschema.GroupVersionKind{Group: "apps", Version: "v1beta1", Kind: "Deployment"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isHighMemoryGvk(tc.gvk))
		})
	}
}

func TestGetGvkFromExampleObject(t *testing.T) {
	scheme := transformScheme(t)

	t.Run("pod", func(t *testing.T) {
		assert.Equal(t, &podGvk, getGvkFromExampleObject(scheme, &v1.Pod{}))
	})

	t.Run("deployment", func(t *testing.T) {
		assert.Equal(t, &deploymentGvk, getGvkFromExampleObject(scheme, &appsv1.Deployment{}))
	})

	t.Run("nil scheme", func(t *testing.T) {
		assert.Nil(t, getGvkFromExampleObject(nil, &v1.Pod{}))
	})

	t.Run("type the scheme does not know", func(t *testing.T) {
		emptyScheme := runtime.NewScheme()
		assert.Nil(t, getGvkFromExampleObject(emptyScheme, &v1.Pod{}))
	})
}

func TestObjectsTransformFromControllerRuntimeCacheKeepsOnlyObjectsWithATransform(t *testing.T) {
	scheme := transformScheme(t)

	transformFuncs := objectsTransformFromControllerRuntimeCache(scheme, map[client.Object]ctrlcache.ByObject{
		&v1.Pod{}:              {Transform: markingTransform("pod", nil)},
		&appsv1.Deployment{}:   {Transform: markingTransform("deployment", nil)},
		&appsv1.DaemonSet{}:    {},
		&v1.ConfigMap{}:        {Transform: markingTransform("configmap", nil)},
		&v1.PersistentVolume{}: {Transform: nil},
	})

	assert.Len(t, transformFuncs, 3)
	assert.Contains(t, transformFuncs, podGvk)
	assert.Contains(t, transformFuncs, deploymentGvk)
	assert.NotContains(t, transformFuncs, daemonSetGvk, "an object without a transform must not be registered")

	// the map has to associate each object with its own transform, not just with some transform.
	pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "checkout"}}
	_, err := transformFuncs[podGvk](pod)
	require.NoError(t, err)
	assert.Equal(t, "pod", pod.Labels["transformed-by"])

	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "checkout"}}
	_, err = transformFuncs[deploymentGvk](deployment)
	require.NoError(t, err)
	assert.Equal(t, "deployment", deployment.Labels["transformed-by"])
}

func TestObjectsTransformFromControllerRuntimeCacheSkipsObjectsMissingFromTheScheme(t *testing.T) {
	schemeWithoutApps := runtime.NewScheme()
	require.NoError(t, v1.AddToScheme(schemeWithoutApps))

	transformFuncs := objectsTransformFromControllerRuntimeCache(schemeWithoutApps, map[client.Object]ctrlcache.ByObject{
		&v1.Pod{}:            {Transform: markingTransform("pod", nil)},
		&appsv1.Deployment{}: {Transform: markingTransform("deployment", nil)},
	})

	assert.Len(t, transformFuncs, 1)
	assert.Contains(t, transformFuncs, podGvk)
}

func TestGetListerWatcherForGvkOnlyWrapsHighMemoryKindsThatHaveATransform(t *testing.T) {
	transform := markingTransform("pod", nil)

	t.Run("kind that is not high memory is left alone", func(t *testing.T) {
		original := &recordingListerWatcher{}

		got := getListerWatcherForGvk(daemonSetGvk, original, gvkToTransformFunc{daemonSetGvk: transform})

		assert.Same(t, original, got, "only pods and deployments should be wrapped")
	})

	t.Run("high memory kind without a transform is left alone", func(t *testing.T) {
		original := &recordingListerWatcher{}

		got := getListerWatcherForGvk(podGvk, original, gvkToTransformFunc{deploymentGvk: transform})

		assert.Same(t, original, got, "there is nothing to apply per page")
	})

	for name, gvk := range map[string]k8sschema.GroupVersionKind{"pod": podGvk, "deployment": deploymentGvk} {
		t.Run(name+" with a transform is wrapped", func(t *testing.T) {
			original := &recordingListerWatcher{}

			got := getListerWatcherForGvk(gvk, original, gvkToTransformFunc{gvk: transform})

			assert.NotSame(t, original, got)
		})
	}
}

// the whole reason this package exists: a list with resourceVersion "0" is served from the api
// server watch cache, which cannot paginate and therefore returns every object at once.
func TestTheWrappedListAsksForAFullyPaginatedList(t *testing.T) {
	for _, tc := range []struct {
		name            string
		resourceVersion string
		want            string
	}{
		{name: "the client-go default is rewritten", resourceVersion: "0", want: ""},
		{name: "an empty resource version stays empty", resourceVersion: "", want: ""},
		{name: "a real resource version is passed through", resourceVersion: "142", want: "142"},
		{name: "a resource version that merely starts with zero is passed through", resourceVersion: "0142", want: "0142"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := &recordingListerWatcher{listResult: func() (runtime.Object, error) {
				return podListOf("checkout"), nil
			}}
			wrapped := wrappedPodListerWatcher(original, markingTransform("pod", nil))

			_, err := wrapped.List(metav1.ListOptions{
				ResourceVersion: tc.resourceVersion,
				Limit:           500,
				Continue:        "next-page-token",
				FieldSelector:   "spec.nodeName=node-1",
			})

			require.NoError(t, err)
			forwarded := original.firstListOptions(t)
			assert.Equal(t, tc.want, forwarded.ResourceVersion)
			assert.Equal(t, int64(500), forwarded.Limit, "paging must be left in place")
			assert.Equal(t, "next-page-token", forwarded.Continue)
			assert.Equal(t, "spec.nodeName=node-1", forwarded.FieldSelector)
		})
	}
}

func TestTheWrappedListTransformsEveryObjectInThePage(t *testing.T) {
	var seen []string
	original := &recordingListerWatcher{listResult: func() (runtime.Object, error) {
		return podListOf("checkout", "frontend", "inventory"), nil
	}}
	wrapped := wrappedPodListerWatcher(original, markingTransform("pod", &seen))

	listed, err := wrapped.List(metav1.ListOptions{ResourceVersion: "0"})

	require.NoError(t, err)
	assert.Equal(t, []string{"checkout", "frontend", "inventory"}, seen, "every object in the page is transformed")

	podList, ok := listed.(*v1.PodList)
	require.True(t, ok)
	require.Len(t, podList.Items, 3, "the transformed page must keep every object")
	assert.Equal(t, []string{"checkout", "frontend", "inventory"},
		[]string{podList.Items[0].Name, podList.Items[1].Name, podList.Items[2].Name})
	for _, pod := range podList.Items {
		assert.Equal(t, "pod", pod.Labels["transformed-by"],
			"the list handed back to the informer must contain the transformed objects")
	}
	assert.Equal(t, "142", podList.ResourceVersion, "the list metadata the informer watches from is preserved")
}

func TestTheWrappedListToleratesAnEmptyPage(t *testing.T) {
	original := &recordingListerWatcher{listResult: func() (runtime.Object, error) {
		return podListOf(), nil
	}}
	wrapped := wrappedPodListerWatcher(original, markingTransform("pod", nil))

	listed, err := wrapped.List(metav1.ListOptions{ResourceVersion: "0"})

	require.NoError(t, err)
	podList, ok := listed.(*v1.PodList)
	require.True(t, ok)
	assert.Empty(t, podList.Items)
}

func TestTheWrappedListSurfacesFailures(t *testing.T) {
	t.Run("the underlying list failed", func(t *testing.T) {
		original := &recordingListerWatcher{listResult: func() (runtime.Object, error) {
			return nil, errors.New("etcd is having a bad day")
		}}
		wrapped := wrappedPodListerWatcher(original, markingTransform("pod", nil))

		listed, err := wrapped.List(metav1.ListOptions{ResourceVersion: "0"})

		require.ErrorContains(t, err, "etcd is having a bad day")
		assert.Nil(t, listed)
	})

	t.Run("the response is not a list", func(t *testing.T) {
		original := &recordingListerWatcher{listResult: func() (runtime.Object, error) {
			return &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "checkout"}}, nil
		}}
		wrapped := wrappedPodListerWatcher(original, markingTransform("pod", nil))

		listed, err := wrapped.List(metav1.ListOptions{ResourceVersion: "0"})

		require.Error(t, err)
		assert.Nil(t, listed)
	})

	t.Run("a transform returned the wrong type", func(t *testing.T) {
		original := &recordingListerWatcher{listResult: func() (runtime.Object, error) {
			return podListOf("checkout"), nil
		}}
		swapTheType := func(obj any) (any, error) {
			return &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "checkout"}}, nil
		}
		wrapped := wrappedPodListerWatcher(original, swapTheType)

		listed, err := wrapped.List(metav1.ListOptions{ResourceVersion: "0"})

		require.Error(t, err, "a transform that changes the object type must not corrupt the cache")
		assert.Nil(t, listed)
	})

	t.Run("a transform failed", func(t *testing.T) {
		original := &recordingListerWatcher{listResult: func() (runtime.Object, error) {
			return podListOf("checkout", "frontend"), nil
		}}
		// the second object is the one that fails, so a loop that only transforms the first object
		// (or that swallows the error) is visible here.
		failOnFrontend := func(obj any) (any, error) {
			if obj.(metav1.Object).GetName() == "frontend" {
				return nil, errors.New("cannot strip frontend")
			}
			return obj, nil
		}
		wrapped := wrappedPodListerWatcher(original, failOnFrontend)

		listed, err := wrapped.List(metav1.ListOptions{ResourceVersion: "0"})

		require.ErrorContains(t, err, "cannot strip frontend")
		assert.Nil(t, listed, "a partially transformed page must never reach the cache")
	})
}

// the watchlist api streams the initial list and client-go transforms each object as it arrives,
// so the watch call must be forwarded exactly as client-go built it.
func TestTheWrappedWatchIsForwardedUnmodified(t *testing.T) {
	underlyingWatcher := watch.NewFake()
	defer underlyingWatcher.Stop()
	original := &recordingListerWatcher{watcher: underlyingWatcher}
	wrapped := wrappedPodListerWatcher(original, markingTransform("pod", nil))

	options := metav1.ListOptions{
		ResourceVersion:      "0",
		Watch:                true,
		AllowWatchBookmarks:  true,
		SendInitialEvents:    boolPtr(true),
		ResourceVersionMatch: metav1.ResourceVersionMatchNotOlderThan,
	}
	got, err := wrapped.Watch(options)

	require.NoError(t, err)
	assert.Same(t, underlyingWatcher, got)
	require.Len(t, original.watchOptions, 1)
	assert.Equal(t, options, original.watchOptions[0], "the watch options must not be rewritten")
	assert.Empty(t, original.recordedListOptions(), "a watch must not trigger a list")
}

func TestTheWrappedWatchSurfacesFailures(t *testing.T) {
	original := &recordingListerWatcher{watchErr: errors.New("watch rejected")}
	wrapped := wrappedPodListerWatcher(original, markingTransform("pod", nil))

	got, err := wrapped.Watch(metav1.ListOptions{})

	require.ErrorContains(t, err, "watch rejected")
	assert.Nil(t, got)
}

func TestCreateNewInformerWithTransformFuncReturnsAnInformerForAnObjectMissingFromTheScheme(t *testing.T) {
	emptyScheme := runtime.NewScheme()
	newInformer := CreateNewInformerWithTransformFunc(emptyScheme, map[client.Object]ctrlcache.ByObject{})

	informer := newInformer(&recordingListerWatcher{}, &v1.Pod{}, 0, cache.Indexers{})

	assert.NotNil(t, informer)
}

// end to end proof that an informer built by this package lists through the wrapper: the informer
// asks client-go's default resourceVersion "0" and the api server must still see a paginated list.
func TestAnInformerForPodsListsThroughTheTransformingWrapper(t *testing.T) {
	clientfeaturestesting.SetFeatureDuringTest(t, clientfeatures.WatchListClient, false)

	var seen []string
	original := &recordingListerWatcher{listResult: func() (runtime.Object, error) {
		return podListOf("checkout"), nil
	}}
	newInformer := CreateNewInformerWithTransformFunc(transformScheme(t), map[client.Object]ctrlcache.ByObject{
		&v1.Pod{}: {Transform: markingTransform("pod", &seen)},
	})

	informer := newInformer(original, &v1.Pod{}, 0, cache.Indexers{})
	stored := runInformerUntilSynced(t, informer)

	assert.Equal(t, "", original.firstListOptions(t).ResourceVersion,
		"the informer must not be able to reach the api server watch cache for pods")
	assert.Equal(t, []string{"checkout"}, seen)
	require.Len(t, stored, 1)
	pod, ok := stored[0].(*v1.Pod)
	require.True(t, ok)
	assert.Equal(t, "pod", pod.Labels["transformed-by"], "the cached object is the transformed one")
}

func TestAnInformerForAKindThatIsNotHighMemoryKeepsTheDefaultListBehaviour(t *testing.T) {
	clientfeaturestesting.SetFeatureDuringTest(t, clientfeatures.WatchListClient, false)

	original := &recordingListerWatcher{listResult: func() (runtime.Object, error) {
		return &v1.ConfigMapList{ListMeta: metav1.ListMeta{ResourceVersion: "7"}}, nil
	}}
	newInformer := CreateNewInformerWithTransformFunc(transformScheme(t), map[client.Object]ctrlcache.ByObject{
		&v1.ConfigMap{}: {Transform: markingTransform("configmap", nil)},
	})

	informer := newInformer(original, &v1.ConfigMap{}, 0, cache.Indexers{})
	runInformerUntilSynced(t, informer)

	assert.Equal(t, "0", original.firstListOptions(t).ResourceVersion,
		"small objects keep the cheap watch cache list")
}

func runInformerUntilSynced(t *testing.T, informer cache.SharedIndexInformer) []any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		informer.RunWithContext(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})

	require.True(t, cache.WaitForCacheSync(ctx.Done(), informer.HasSynced), "the informer never synced")
	return informer.GetStore().List()
}

func TestMarkObjectAsTransformedIsVisibleToIsObjectTransformed(t *testing.T) {
	pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "checkout"}}

	require.False(t, IsObjectTransformed(pod), "an object with no annotations was never transformed")

	MarkObjectAsTransformed(pod)

	assert.True(t, IsObjectTransformed(pod))
	assert.Contains(t, pod.Annotations, transformedMarkKey)
}

func TestIsObjectTransformedIgnoresUnrelatedAnnotations(t *testing.T) {
	pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:        "checkout",
		Annotations: map[string]string{"kubernetes.io/psp": "restricted"},
	}}

	assert.False(t, IsObjectTransformed(pod))
}

func TestMarkObjectAsTransformedKeepsTheExistingAnnotations(t *testing.T) {
	pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:        "checkout",
		Annotations: map[string]string{"kubernetes.io/psp": "restricted"},
	}}

	MarkObjectAsTransformed(pod)
	MarkObjectAsTransformed(pod)

	assert.Equal(t, "restricted", pod.Annotations["kubernetes.io/psp"])
	assert.True(t, IsObjectTransformed(pod))
	assert.Len(t, pod.Annotations, 2, "marking twice must not add a second marker")
}

// a transform runs more than once for the same object now that lists are transformed per page,
// and the guard against that is an annotation. StripAnnotations drops everything that is not a
// kubernetes.io or odigos.io key, so the marker only survives because of its odigos.io prefix.
func TestTheTransformedMarkerSurvivesAnnotationStripping(t *testing.T) {
	pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:        "checkout",
		Annotations: map[string]string{"helm.sh/release": "checkout"},
	}}

	MarkObjectAsTransformed(pod)
	StripPod(pod)

	assert.True(t, IsObjectTransformed(pod), "a stripped object would otherwise be transformed again")
	assert.NotContains(t, pod.Annotations, "helm.sh/release")
}
