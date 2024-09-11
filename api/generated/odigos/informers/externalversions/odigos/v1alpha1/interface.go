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
	internalinterfaces "github.com/odigos-io/odigos/api/generated/odigos/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// CollectorsGroups returns a CollectorsGroupInformer.
	CollectorsGroups() CollectorsGroupInformer
	// Destinations returns a DestinationInformer.
	Destinations() DestinationInformer
	// InstrumentationConfigs returns a InstrumentationConfigInformer.
	InstrumentationConfigs() InstrumentationConfigInformer
	// InstrumentationInstances returns a InstrumentationInstanceInformer.
	InstrumentationInstances() InstrumentationInstanceInformer
	// InstrumentationRules returns a InstrumentationRuleInformer.
	InstrumentationRules() InstrumentationRuleInformer
	// InstrumentedApplications returns a InstrumentedApplicationInformer.
	InstrumentedApplications() InstrumentedApplicationInformer
	// OdigosConfigurations returns a OdigosConfigurationInformer.
	OdigosConfigurations() OdigosConfigurationInformer
	// Processors returns a ProcessorInformer.
	Processors() ProcessorInformer
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

// CollectorsGroups returns a CollectorsGroupInformer.
func (v *version) CollectorsGroups() CollectorsGroupInformer {
	return &collectorsGroupInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Destinations returns a DestinationInformer.
func (v *version) Destinations() DestinationInformer {
	return &destinationInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// InstrumentationConfigs returns a InstrumentationConfigInformer.
func (v *version) InstrumentationConfigs() InstrumentationConfigInformer {
	return &instrumentationConfigInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// InstrumentationInstances returns a InstrumentationInstanceInformer.
func (v *version) InstrumentationInstances() InstrumentationInstanceInformer {
	return &instrumentationInstanceInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// InstrumentationRules returns a InstrumentationRuleInformer.
func (v *version) InstrumentationRules() InstrumentationRuleInformer {
	return &instrumentationRuleInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// InstrumentedApplications returns a InstrumentedApplicationInformer.
func (v *version) InstrumentedApplications() InstrumentedApplicationInformer {
	return &instrumentedApplicationInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// OdigosConfigurations returns a OdigosConfigurationInformer.
func (v *version) OdigosConfigurations() OdigosConfigurationInformer {
	return &odigosConfigurationInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Processors returns a ProcessorInformer.
func (v *version) Processors() ProcessorInformer {
	return &processorInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
