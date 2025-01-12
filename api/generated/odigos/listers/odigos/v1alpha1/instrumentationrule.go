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
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	labels "k8s.io/apimachinery/pkg/labels"
	listers "k8s.io/client-go/listers"
	cache "k8s.io/client-go/tools/cache"
)

// InstrumentationRuleLister helps list InstrumentationRules.
// All objects returned here must be treated as read-only.
type InstrumentationRuleLister interface {
	// List lists all InstrumentationRules in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*odigosv1alpha1.InstrumentationRule, err error)
	// InstrumentationRules returns an object that can list and get InstrumentationRules.
	InstrumentationRules(namespace string) InstrumentationRuleNamespaceLister
	InstrumentationRuleListerExpansion
}

// instrumentationRuleLister implements the InstrumentationRuleLister interface.
type instrumentationRuleLister struct {
	listers.ResourceIndexer[*odigosv1alpha1.InstrumentationRule]
}

// NewInstrumentationRuleLister returns a new InstrumentationRuleLister.
func NewInstrumentationRuleLister(indexer cache.Indexer) InstrumentationRuleLister {
	return &instrumentationRuleLister{listers.New[*odigosv1alpha1.InstrumentationRule](indexer, odigosv1alpha1.Resource("instrumentationrule"))}
}

// InstrumentationRules returns an object that can list and get InstrumentationRules.
func (s *instrumentationRuleLister) InstrumentationRules(namespace string) InstrumentationRuleNamespaceLister {
	return instrumentationRuleNamespaceLister{listers.NewNamespaced[*odigosv1alpha1.InstrumentationRule](s.ResourceIndexer, namespace)}
}

// InstrumentationRuleNamespaceLister helps list and get InstrumentationRules.
// All objects returned here must be treated as read-only.
type InstrumentationRuleNamespaceLister interface {
	// List lists all InstrumentationRules in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*odigosv1alpha1.InstrumentationRule, err error)
	// Get retrieves the InstrumentationRule from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*odigosv1alpha1.InstrumentationRule, error)
	InstrumentationRuleNamespaceListerExpansion
}

// instrumentationRuleNamespaceLister implements the InstrumentationRuleNamespaceLister
// interface.
type instrumentationRuleNamespaceLister struct {
	listers.ResourceIndexer[*odigosv1alpha1.InstrumentationRule]
}
