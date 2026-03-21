package cache

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	ctrlcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// ~~~~~~~~~!!! Important info regarding memory spikes during cache initialization !!!~~~~~~~~~~
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
//
// we use the client-go SharedIndexInformer to fill the cache.
// the cache is managed by the client-go and we populate it and react to changes via informers.
// the informers sync the cache generically for all objects by using a ListerWatcher interface.
// paging is used by default with limit of 500 objects per list operation.
// the informer also sets resourceVersion to 0 which performs a fast and efficient list operation,
// served from the api server watch cache instead of expensive etcd requests.
// however, there is a caveat: the api server watch cache cannot support paging list requests,
// thus it returns the full list of objects in a single response.
// see: https://github.com/kubernetes/kubernetes/issues/118394
// so setting resourceVersion to 0 means efficient list but no paging, which overloads our component
// with a huge initial list of objects that have to all be stored in memory at once.
// this can easily exhaust the GOMEMLIMIT in large clusters and cause OOM to the process.
// see also where client-go sets resourceVersion to 0 by default:
// https://github.com/kubernetes/client-go/blob/01310540169fb3613931c57443e1e4155684d8ac/tools/cache/reflector.go#L1127
//
// to bypass this issue for high memory objects (pods and deployments), we override the list
// operation to set resourceVersion to "" and perform a full list operation.
// this is a trade-off between memory and performance, as the full list operation is more resource intensive for the api server and etcd,
// but allows us to fetch, decode, and transform each page individually, without having to store the full raw list in memory at once.
// this is done by wrapping the original lister watcher with a custom list watch wrapper that applies the transform function to each page of objects.
// we minimize this to only pods and deployments for now, as they are the ones with large number of objects,
// and major memory spikes cause issues in the cache initialization process.
//
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
//

// this is the same interface as what controller-runtime cache is using for its options:
// https://github.com/kubernetes-sigs/controller-runtime/blob/37c380b7405b67e31ca8feaf0e2132b747d940aa/pkg/cache/cache.go#L260
// allowing us to use it as a replacement for the default NewInformer func.
type NewInformerFunc func(
	originalListerWatcher cache.ListerWatcher,
	exampleObject runtime.Object,
	defaultEventHandlerResyncPeriod time.Duration,
	indexers cache.Indexers,
) cache.SharedIndexInformer

// this is a way for the consumers of this package to define the transform functions for the different types of objects.
// if an object is in the map, this package will wrap the informer so the transform function will be applied to the objects
// for each page (500 objects) of list operations.
type gvkToTransformFunc map[k8sschema.GroupVersionKind]cache.TransformFunc

// we add an annotation after transforming an object to indicate that it has been transformed.
// this is to guarantee that we ignore an object if it has been transformed already.
var objectTransformedAnnotation = "odigos.io/cache-transformed"

// GVKs for kinds that use per-page list transform + empty resourceVersion (see package comment).
var (
	gvkPod        = k8sschema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}
	gvkDeployment = k8sschema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}
)

// Controller-Runtime framework uses the k8s client-go project for the objects cache.
// Client-go uses informers to fill and maintain the cache.
// for large objects, we use Transform to strip irrelevant fields and keep in memory just what we need.
// however, during initialization, the informer will run a single list operation to fill the cache,
// and then watch for changes. the initial list pulls into memory all the objects of a specific kind at once.
// unfortunately, the transform function is called only after the full list is fetched,
// which can cause high spikes in memory at startup and OOM.
// This function receives the map of objects to transform, and wraps the informer so the transform happens
// per list page (500 objects) and not on the full list, thus reducing the memory we have to keep at single time.
//
// the result of this function can be used as a parameter to the cache.Options.NewInformer field.
// example:
//
//	cacheOptions := cache.Options{
//		Scheme:                      scheme,
//		ByObject:                    cacheByObjectConfig,
//		NewInformer:                 CreateNewInformerWithTransformFunc(scheme, cacheByObjectConfig),
//	}
func CreateNewInformerWithTransformFunc(scheme *runtime.Scheme, cacheByObjectConfig map[client.Object]ctrlcache.ByObject) NewInformerFunc {
	transformFuncs := objectsTransformFromControllerRuntimeCache(scheme, cacheByObjectConfig)
	return func(
		originalListerWatcher cache.ListerWatcher,
		exampleObject runtime.Object,
		defaultEventHandlerResyncPeriod time.Duration,
		indexers cache.Indexers,
	) cache.SharedIndexInformer {
		gvk := getGvkFromExampleObject(scheme, exampleObject)
		if gvk == nil {
			// cannot determine the GVK (pods, deployments, etc.), so we use the original lister watcher
			return cache.NewSharedIndexInformer(originalListerWatcher, exampleObject, defaultEventHandlerResyncPeriod, indexers)
		}

		listerWatcher := getListerWatcherForGvk(*gvk, originalListerWatcher, transformFuncs)

		return cache.NewSharedIndexInformer(listerWatcher, exampleObject, defaultEventHandlerResyncPeriod, indexers)
	}
}

// helper function to check if an object has been transformed.
// useful to skip transforming an object if it has been transformed already.
// use in your transform function (which now can be called more than once for the same object)
func IsObjectTransformed(obj metav1.Object) bool {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		return false
	}
	_, found := annotations[objectTransformedAnnotation]
	return found
}

// helper function to mark an object as transformed.
// user can then check if the object has been transformed by calling IsObjectTransformed.
// useful to skip transforming an object if it has been transformed already.
// use in your transform function after transforming the object (which now can be called more than once for the same object)
func MarkObjectAsTransformed(obj metav1.Object) {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[objectTransformedAnnotation] = "true"
	obj.SetAnnotations(annotations)
}

// given a GVK, return the appropriate lister watcher with the transform function applied if needed.
// we use this to wrap the original lister watcher with the transform function if needed,
// and also make sure list requests for api server are paginated to avoid memory spikes.
func getListerWatcherForGvk(gvk k8sschema.GroupVersionKind,
	originalListerWatcher cache.ListerWatcher,
	transformFuncs gvkToTransformFunc) cache.ListerWatcher {
	// only handle pods and deployments for now, as they are the ones with large number of objects
	// and major memory spikes cause issues in the cache initialization process.
	switch gvk {
	case gvkPod, gvkDeployment:
		// continue below
	default:
		return originalListerWatcher
	}

	transformFunc, ok := transformFuncs[gvk]
	if !ok {
		// for gvk for which no transform function is defined, we use the original lister watcher (noop)
		return originalListerWatcher
	}

	originalListerWatcherWithContext := cache.ToListerWatcherWithContext(originalListerWatcher)
	listerWatcherContextWrapped := createListWatchWrapperWithTransform(transformFunc, originalListerWatcherWithContext)
	listerWatcherWrapped, ok := listerWatcherContextWrapped.(cache.ListerWatcher)
	if !ok {
		// this should never happen, but just in case
		return originalListerWatcher
	}

	return listerWatcherWrapped
}

func createListWatchWrapperWithTransform(
	transformFunc cache.TransformFunc,
	originalListerWatcherWithContext cache.ListerWatcherWithContext,
) cache.ListerWatcherWithContext {
	return &cache.ListWatch{
		ListWithContextFunc: func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			if options.ResourceVersion == "0" {
				options.ResourceVersion = ""
			}
			list, err := originalListerWatcherWithContext.ListWithContext(ctx, options)
			if err != nil {
				return nil, err
			}
			items, err := meta.ExtractList(list)
			if err != nil {
				return nil, err
			}
			transformed := make([]runtime.Object, 0, len(items))
			for _, item := range items {
				t, err := transformFunc(item)
				if err != nil {
					return nil, err
				}
				transformed = append(transformed, t.(runtime.Object))
			}
			if err := meta.SetList(list, transformed); err != nil {
				return nil, err
			}
			return list, nil
		},
		WatchFuncWithContext: func(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
			return originalListerWatcherWithContext.WatchWithContext(ctx, options)
		},
	}
}

func getGvkFromExampleObject(scheme *runtime.Scheme, exampleObject runtime.Object) *k8sschema.GroupVersionKind {
	var gvk *k8sschema.GroupVersionKind
	if scheme != nil {
		if gvks, _, _ := scheme.ObjectKinds(exampleObject); len(gvks) > 0 {
			gvk = &gvks[0]
		}
	}
	return gvk
}

func objectsTransformFromControllerRuntimeCache(
	scheme *runtime.Scheme,
	cacheByObjectConfig map[client.Object]ctrlcache.ByObject,
) gvkToTransformFunc {
	transformFuncs := gvkToTransformFunc{}
	for object, byObject := range cacheByObjectConfig {
		// if there is no transform function, it's not relevant for us
		transformFunc := byObject.Transform
		if transformFunc == nil {
			continue
		}

		gvk := getGvkFromExampleObject(scheme, object)
		if gvk == nil {
			continue
		}
		transformFuncs[*gvk] = transformFunc
	}
	return transformFuncs
}
