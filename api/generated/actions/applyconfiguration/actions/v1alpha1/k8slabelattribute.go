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

package v1alpha1

// K8sLabelAttributeApplyConfiguration represents a declarative configuration of the K8sLabelAttribute type for use
// with apply.
type K8sLabelAttributeApplyConfiguration struct {
	LabelKey     *string `json:"labelKey,omitempty"`
	AttributeKey *string `json:"attributeKey,omitempty"`
}

// K8sLabelAttributeApplyConfiguration constructs a declarative configuration of the K8sLabelAttribute type for use with
// apply.
func K8sLabelAttribute() *K8sLabelAttributeApplyConfiguration {
	return &K8sLabelAttributeApplyConfiguration{}
}

// WithLabelKey sets the LabelKey field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the LabelKey field is set to the value of the last call.
func (b *K8sLabelAttributeApplyConfiguration) WithLabelKey(value string) *K8sLabelAttributeApplyConfiguration {
	b.LabelKey = &value
	return b
}

// WithAttributeKey sets the AttributeKey field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the AttributeKey field is set to the value of the last call.
func (b *K8sLabelAttributeApplyConfiguration) WithAttributeKey(value string) *K8sLabelAttributeApplyConfiguration {
	b.AttributeKey = &value
	return b
}
