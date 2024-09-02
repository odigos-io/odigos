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

// InstrumentedApplicationInformer provides access to a shared informer and lister for
// InstrumentedApplications.
type InstrumentedApplicationInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.InstrumentedApplicationLister
}

type instrumentedApplicationInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewInstrumentedApplicationInformer constructs a new informer for InstrumentedApplication type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewInstrumentedApplicationInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredInstrumentedApplicationInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredInstrumentedApplicationInformer constructs a new informer for InstrumentedApplication type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredInstrumentedApplicationInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.OdigosV1alpha1().InstrumentedApplications(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.OdigosV1alpha1().InstrumentedApplications(namespace).Watch(context.TODO(), options)
			},
		},
		&odigosv1alpha1.InstrumentedApplication{},
		resyncPeriod,
		indexers,
	)
}

func (f *instrumentedApplicationInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredInstrumentedApplicationInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *instrumentedApplicationInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&odigosv1alpha1.InstrumentedApplication{}, f.defaultInformer)
}

func (f *instrumentedApplicationInformer) Lister() v1alpha1.InstrumentedApplicationLister {
	return v1alpha1.NewInstrumentedApplicationLister(f.Informer().GetIndexer())
}
