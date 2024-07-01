package client

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubernetesClientFromCacheWithAPIFallback struct {
	Cache     client.Client
	APIServer client.Reader
}

func NewKubernetesClientFromCacheWithAPIFallback(cache client.Client, apiServer client.Reader) *KubernetesClientFromCacheWithAPIFallback {
	return &KubernetesClientFromCacheWithAPIFallback{
		Cache:     cache,
		APIServer: apiServer,
	}
}

func (k *KubernetesClientFromCacheWithAPIFallback) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	err := k.Cache.Get(ctx, key, obj, opts...)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}

		err = k.APIServer.Get(ctx, key, obj, opts...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k *KubernetesClientFromCacheWithAPIFallback) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	err := k.Cache.List(ctx, list, opts...)
	if err != nil || meta.LenList(list) == 0 {
		if err != nil && client.IgnoreNotFound(err) != nil {
			return err
		}

		err = k.APIServer.List(ctx, list, opts...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k *KubernetesClientFromCacheWithAPIFallback) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	return k.Cache.Create(ctx, obj, opts...)
}

func (k *KubernetesClientFromCacheWithAPIFallback) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return k.Cache.Delete(ctx, obj, opts...)
}

func (k *KubernetesClientFromCacheWithAPIFallback) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return k.Cache.Update(ctx, obj, opts...)
}

func (k *KubernetesClientFromCacheWithAPIFallback) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return k.Cache.Patch(ctx, obj, patch, opts...)
}

func (k *KubernetesClientFromCacheWithAPIFallback) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return k.Cache.DeleteAllOf(ctx, obj, opts...)
}

func (k *KubernetesClientFromCacheWithAPIFallback) Status() client.SubResourceWriter {
	return k.Cache.Status()
}

func (k *KubernetesClientFromCacheWithAPIFallback) SubResource(subResource string) client.SubResourceClient {
	return k.Cache.SubResource(subResource)
}

func (k *KubernetesClientFromCacheWithAPIFallback) Scheme() *runtime.Scheme {
	return k.Cache.Scheme()
}

func (k *KubernetesClientFromCacheWithAPIFallback) RESTMapper() meta.RESTMapper {
	return k.Cache.RESTMapper()
}

func (k *KubernetesClientFromCacheWithAPIFallback) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	return k.Cache.GroupVersionKindFor(obj)
}

func (k *KubernetesClientFromCacheWithAPIFallback) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	return k.Cache.IsObjectNamespaced(obj)
}
