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

	versioned "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned"
	internalinterfaces "github.com/odigos-io/odigos/api/generated/odigos/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/odigos-io/odigos/api/generated/odigos/listers/odigos/v1alpha1"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// InstrumentationRuleInformer provides access to a shared informer and lister for
// InstrumentationRules.
type InstrumentationRuleInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.InstrumentationRuleLister
}

type instrumentationRuleInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewInstrumentationRuleInformer constructs a new informer for InstrumentationRule type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewInstrumentationRuleInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredInstrumentationRuleInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredInstrumentationRuleInformer constructs a new informer for InstrumentationRule type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredInstrumentationRuleInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.OdigosV1alpha1().InstrumentationRules(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.OdigosV1alpha1().InstrumentationRules(namespace).Watch(context.TODO(), options)
			},
		},
		&odigosv1alpha1.InstrumentationRule{},
		resyncPeriod,
		indexers,
	)
}

func (f *instrumentationRuleInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredInstrumentationRuleInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *instrumentationRuleInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&odigosv1alpha1.InstrumentationRule{}, f.defaultInformer)
}

func (f *instrumentationRuleInformer) Lister() v1alpha1.InstrumentationRuleLister {
	return v1alpha1.NewInstrumentationRuleLister(f.Informer().GetIndexer())
}
