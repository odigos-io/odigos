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

// this is the same interface as what controller-runtime cache is using for it's options:
// https://github.com/kubernetes-sigs/controller-runtime/blob/37c380b7405b67e31ca8feaf0e2132b747d940aa/pkg/cache/cache.go#L260
// allowing us to use it as a replacement for the default NewInformer func.
type NewImformerFunc func(originalListerWatcher cache.ListerWatcher, exampleObject runtime.Object, defaultEventHandlerResyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer

// this is a way for the consumers of this package to define the transform functions for the different types of objects.
// if an object is in the map, this package will wrap the informer so the transform function will be applied to the objects
// for each page (500 objects) of lists operations.
type gvkToTransformFunc map[k8sschema.GroupVersionKind]cache.TransformFunc

// we add an annotation after transforming an object to indicate that it has been transformed.
// this is to guarantee that we ignore an object if it has been transformed already.
var objectTransformedAnnotation = "odigos.io/cache-transformed"

func CreateNewInformerWithTransofrmFunc(scheme *runtime.Scheme, cacheByObjectConfig map[client.Object]ctrlcache.ByObject) NewImformerFunc {
	transformFuncs := objectsTransformFromControllerRuntimeCache(scheme, cacheByObjectConfig)
	return func(originalListerWatcher cache.ListerWatcher, exampleObject runtime.Object, defaultEventHandlerResyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {

		gvk := getGvkFromExampleObject(scheme, exampleObject)
		if gvk == nil {
			// cannot determine the GVK(pods, deployments, etc), so we use the original lister watcher
			return cache.NewSharedIndexInformer(originalListerWatcher, exampleObject, defaultEventHandlerResyncPeriod, indexers)
		}

		transformFunc, ok := transformFuncs[*gvk]
		if !ok {
			// for gvk for which no transform function is defined, we use the original lister watcher (noop)
			return cache.NewSharedIndexInformer(originalListerWatcher, exampleObject, defaultEventHandlerResyncPeriod, indexers)
		}

		originalListerWatcherWithContext := cache.ToListerWatcherWithContext(originalListerWatcher)
		listerWatcherContextWrapped := createListWatchWrapperWithTransform(transformFunc, originalListerWatcherWithContext)
		listerWatcherWrapped, ok := listerWatcherContextWrapped.(cache.ListerWatcher)
		if !ok {
			// this should never happen, but just in case
			return cache.NewSharedIndexInformer(originalListerWatcher, exampleObject, defaultEventHandlerResyncPeriod, indexers)
		}

		// return the informer with the wrapped lister watcher
		return cache.NewSharedIndexInformer(listerWatcherWrapped, exampleObject, defaultEventHandlerResyncPeriod, indexers)
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

func createListWatchWrapperWithTransform(transformFunc cache.TransformFunc, originalListerWatcherWithContext cache.ListerWatcherWithContext) cache.ListerWatcherWithContext {
	return &cache.ListWatch{
		ListWithContextFunc: func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
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

func objectsTransformFromControllerRuntimeCache(scheme *runtime.Scheme, cacheByObjectConfig map[client.Object]ctrlcache.ByObject) gvkToTransformFunc {
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
