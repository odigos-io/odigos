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
	actionsv1alpha1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	labels "k8s.io/apimachinery/pkg/labels"
	listers "k8s.io/client-go/listers"
	cache "k8s.io/client-go/tools/cache"
)

// AddClusterInfoLister helps list AddClusterInfos.
// All objects returned here must be treated as read-only.
type AddClusterInfoLister interface {
	// List lists all AddClusterInfos in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*actionsv1alpha1.AddClusterInfo, err error)
	// AddClusterInfos returns an object that can list and get AddClusterInfos.
	AddClusterInfos(namespace string) AddClusterInfoNamespaceLister
	AddClusterInfoListerExpansion
}

// addClusterInfoLister implements the AddClusterInfoLister interface.
type addClusterInfoLister struct {
	listers.ResourceIndexer[*actionsv1alpha1.AddClusterInfo]
}

// NewAddClusterInfoLister returns a new AddClusterInfoLister.
func NewAddClusterInfoLister(indexer cache.Indexer) AddClusterInfoLister {
	return &addClusterInfoLister{listers.New[*actionsv1alpha1.AddClusterInfo](indexer, actionsv1alpha1.Resource("addclusterinfo"))}
}

// AddClusterInfos returns an object that can list and get AddClusterInfos.
func (s *addClusterInfoLister) AddClusterInfos(namespace string) AddClusterInfoNamespaceLister {
	return addClusterInfoNamespaceLister{listers.NewNamespaced[*actionsv1alpha1.AddClusterInfo](s.ResourceIndexer, namespace)}
}

// AddClusterInfoNamespaceLister helps list and get AddClusterInfos.
// All objects returned here must be treated as read-only.
type AddClusterInfoNamespaceLister interface {
	// List lists all AddClusterInfos in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*actionsv1alpha1.AddClusterInfo, err error)
	// Get retrieves the AddClusterInfo from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*actionsv1alpha1.AddClusterInfo, error)
	AddClusterInfoNamespaceListerExpansion
}

// addClusterInfoNamespaceLister implements the AddClusterInfoNamespaceLister
// interface.
type addClusterInfoNamespaceLister struct {
	listers.ResourceIndexer[*actionsv1alpha1.AddClusterInfo]
}
