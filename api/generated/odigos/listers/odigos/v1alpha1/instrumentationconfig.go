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
// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// InstrumentationConfigLister helps list InstrumentationConfigs.
// All objects returned here must be treated as read-only.
type InstrumentationConfigLister interface {
	// List lists all InstrumentationConfigs in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.InstrumentationConfig, err error)
	// InstrumentationConfigs returns an object that can list and get InstrumentationConfigs.
	InstrumentationConfigs(namespace string) InstrumentationConfigNamespaceLister
	InstrumentationConfigListerExpansion
}

// instrumentationConfigLister implements the InstrumentationConfigLister interface.
type instrumentationConfigLister struct {
	indexer cache.Indexer
}

// NewInstrumentationConfigLister returns a new InstrumentationConfigLister.
func NewInstrumentationConfigLister(indexer cache.Indexer) InstrumentationConfigLister {
	return &instrumentationConfigLister{indexer: indexer}
}

// List lists all InstrumentationConfigs in the indexer.
func (s *instrumentationConfigLister) List(selector labels.Selector) (ret []*v1alpha1.InstrumentationConfig, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.InstrumentationConfig))
	})
	return ret, err
}

// InstrumentationConfigs returns an object that can list and get InstrumentationConfigs.
func (s *instrumentationConfigLister) InstrumentationConfigs(namespace string) InstrumentationConfigNamespaceLister {
	return instrumentationConfigNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// InstrumentationConfigNamespaceLister helps list and get InstrumentationConfigs.
// All objects returned here must be treated as read-only.
type InstrumentationConfigNamespaceLister interface {
	// List lists all InstrumentationConfigs in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.InstrumentationConfig, err error)
	// Get retrieves the InstrumentationConfig from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.InstrumentationConfig, error)
	InstrumentationConfigNamespaceListerExpansion
}

// instrumentationConfigNamespaceLister implements the InstrumentationConfigNamespaceLister
// interface.
type instrumentationConfigNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all InstrumentationConfigs in the indexer for a given namespace.
func (s instrumentationConfigNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.InstrumentationConfig, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.InstrumentationConfig))
	})
	return ret, err
}

// Get retrieves the InstrumentationConfig from the indexer for a given namespace and name.
func (s instrumentationConfigNamespaceLister) Get(name string) (*v1alpha1.InstrumentationConfig, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("instrumentationconfig"), name)
	}
	return obj.(*v1alpha1.InstrumentationConfig), nil
}
