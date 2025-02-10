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
	internalinterfaces "github.com/odigos-io/odigos/api/generated/actions/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// AddClusterInfos returns a AddClusterInfoInformer.
	AddClusterInfos() AddClusterInfoInformer
	// DeleteAttributes returns a DeleteAttributeInformer.
	DeleteAttributes() DeleteAttributeInformer
	// ErrorSamplers returns a ErrorSamplerInformer.
	ErrorSamplers() ErrorSamplerInformer
	// K8sAttributeses returns a K8sAttributesInformer.
	K8sAttributeses() K8sAttributesInformer
	// LatencySamplers returns a LatencySamplerInformer.
	LatencySamplers() LatencySamplerInformer
	// PiiMaskings returns a PiiMaskingInformer.
	PiiMaskings() PiiMaskingInformer
	// ProbabilisticSamplers returns a ProbabilisticSamplerInformer.
	ProbabilisticSamplers() ProbabilisticSamplerInformer
	// RenameAttributes returns a RenameAttributeInformer.
	RenameAttributes() RenameAttributeInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// AddClusterInfos returns a AddClusterInfoInformer.
func (v *version) AddClusterInfos() AddClusterInfoInformer {
	return &addClusterInfoInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// DeleteAttributes returns a DeleteAttributeInformer.
func (v *version) DeleteAttributes() DeleteAttributeInformer {
	return &deleteAttributeInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// ErrorSamplers returns a ErrorSamplerInformer.
func (v *version) ErrorSamplers() ErrorSamplerInformer {
	return &errorSamplerInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// K8sAttributeses returns a K8sAttributesInformer.
func (v *version) K8sAttributeses() K8sAttributesInformer {
	return &k8sAttributesInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// LatencySamplers returns a LatencySamplerInformer.
func (v *version) LatencySamplers() LatencySamplerInformer {
	return &latencySamplerInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// PiiMaskings returns a PiiMaskingInformer.
func (v *version) PiiMaskings() PiiMaskingInformer {
	return &piiMaskingInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// ProbabilisticSamplers returns a ProbabilisticSamplerInformer.
func (v *version) ProbabilisticSamplers() ProbabilisticSamplerInformer {
	return &probabilisticSamplerInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// RenameAttributes returns a RenameAttributeInformer.
func (v *version) RenameAttributes() RenameAttributeInformer {
	return &renameAttributeInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
