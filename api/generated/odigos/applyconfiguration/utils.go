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
// Code generated by applyconfiguration-gen. DO NOT EDIT.

package applyconfiguration

import (
	internal "github.com/odigos-io/odigos/api/generated/odigos/applyconfiguration/internal"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/generated/odigos/applyconfiguration/odigos/v1alpha1"
	v1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	testing "k8s.io/client-go/testing"
)

// ForKind returns an apply configuration type for the given GroupVersionKind, or nil if no
// apply configuration type exists for the given GroupVersionKind.
func ForKind(kind schema.GroupVersionKind) interface{} {
	switch kind {
	// Group=odigos.io, Version=v1alpha1
	case v1alpha1.SchemeGroupVersion.WithKind("Attribute"):
		return &odigosv1alpha1.AttributeApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("AttributeCondition"):
		return &odigosv1alpha1.AttributeConditionApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("AttributesAndSamplerRule"):
		return &odigosv1alpha1.AttributesAndSamplerRuleApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("CollectorsGroup"):
		return &odigosv1alpha1.CollectorsGroupApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("CollectorsGroupResourcesSettings"):
		return &odigosv1alpha1.CollectorsGroupResourcesSettingsApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("CollectorsGroupSpec"):
		return &odigosv1alpha1.CollectorsGroupSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("CollectorsGroupStatus"):
		return &odigosv1alpha1.CollectorsGroupStatusApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ConfigOption"):
		return &odigosv1alpha1.ConfigOptionApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ContainerAgentConfig"):
		return &odigosv1alpha1.ContainerAgentConfigApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("Destination"):
		return &odigosv1alpha1.DestinationApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("DestinationSpec"):
		return &odigosv1alpha1.DestinationSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("DestinationStatus"):
		return &odigosv1alpha1.DestinationStatusApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("EnvVar"):
		return &odigosv1alpha1.EnvVarApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("HeadSamplingConfig"):
		return &odigosv1alpha1.HeadSamplingConfigApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("InstrumentationConfig"):
		return &odigosv1alpha1.InstrumentationConfigApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("InstrumentationConfigSpec"):
		return &odigosv1alpha1.InstrumentationConfigSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("InstrumentationConfigStatus"):
		return &odigosv1alpha1.InstrumentationConfigStatusApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("InstrumentationInstance"):
		return &odigosv1alpha1.InstrumentationInstanceApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("InstrumentationInstanceSpec"):
		return &odigosv1alpha1.InstrumentationInstanceSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("InstrumentationInstanceStatus"):
		return &odigosv1alpha1.InstrumentationInstanceStatusApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("InstrumentationLibraryConfig"):
		return &odigosv1alpha1.InstrumentationLibraryConfigApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("InstrumentationLibraryConfigTraces"):
		return &odigosv1alpha1.InstrumentationLibraryConfigTracesApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("InstrumentationLibraryGlobalId"):
		return &odigosv1alpha1.InstrumentationLibraryGlobalIdApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("InstrumentationLibraryId"):
		return &odigosv1alpha1.InstrumentationLibraryIdApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("InstrumentationLibraryOptions"):
		return &odigosv1alpha1.InstrumentationLibraryOptionsApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("InstrumentationLibraryStatus"):
		return &odigosv1alpha1.InstrumentationLibraryStatusApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("InstrumentationRule"):
		return &odigosv1alpha1.InstrumentationRuleApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("InstrumentationRuleSpec"):
		return &odigosv1alpha1.InstrumentationRuleSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("InstrumentationRuleStatus"):
		return &odigosv1alpha1.InstrumentationRuleStatusApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("InstrumentedApplication"):
		return &odigosv1alpha1.InstrumentedApplicationApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("InstrumentedApplicationSpec"):
		return &odigosv1alpha1.InstrumentedApplicationSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("InstrumentedApplicationStatus"):
		return &odigosv1alpha1.InstrumentedApplicationStatusApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("OptionByContainer"):
		return &odigosv1alpha1.OptionByContainerApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("OtherAgent"):
		return &odigosv1alpha1.OtherAgentApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("Processor"):
		return &odigosv1alpha1.ProcessorApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ProcessorSpec"):
		return &odigosv1alpha1.ProcessorSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("RuntimeDetailsByContainer"):
		return &odigosv1alpha1.RuntimeDetailsByContainerApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("SdkConfig"):
		return &odigosv1alpha1.SdkConfigApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("Source"):
		return &odigosv1alpha1.SourceApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("SourceSelector"):
		return &odigosv1alpha1.SourceSelectorApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("SourceSpec"):
		return &odigosv1alpha1.SourceSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("SourceStatus"):
		return &odigosv1alpha1.SourceStatusApplyConfiguration{}

	}
	return nil
}

func NewTypeConverter(scheme *runtime.Scheme) *testing.TypeConverter {
	return &testing.TypeConverter{Scheme: scheme, TypeResolver: internal.Parser()}
}
