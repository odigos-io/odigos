/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	versioned "github.com/odigos-io/odigos/api/generated/rules/clientset/versioned"
	internalinterfaces "github.com/odigos-io/odigos/api/generated/rules/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/odigos-io/odigos/api/generated/rules/listers/odigos/v1alpha1"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// InstrumentationInstanceInformer provides access to a shared informer and lister for
// InstrumentationInstances.
type InstrumentationInstanceInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.InstrumentationInstanceLister
}

type instrumentationInstanceInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewInstrumentationInstanceInformer constructs a new informer for InstrumentationInstance type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewInstrumentationInstanceInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredInstrumentationInstanceInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredInstrumentationInstanceInformer constructs a new informer for InstrumentationInstance type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredInstrumentationInstanceInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.OdigosV1alpha1().InstrumentationInstances(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.OdigosV1alpha1().InstrumentationInstances(namespace).Watch(context.TODO(), options)
			},
		},
		&odigosv1alpha1.InstrumentationInstance{},
		resyncPeriod,
		indexers,
	)
}

func (f *instrumentationInstanceInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredInstrumentationInstanceInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *instrumentationInstanceInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&odigosv1alpha1.InstrumentationInstance{}, f.defaultInformer)
}

func (f *instrumentationInstanceInformer) Lister() v1alpha1.InstrumentationInstanceLister {
	return v1alpha1.NewInstrumentationInstanceLister(f.Informer().GetIndexer())
}
