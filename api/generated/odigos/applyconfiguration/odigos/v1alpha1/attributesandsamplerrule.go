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

// AttributesAndSamplerRuleApplyConfiguration represents an declarative configuration of the AttributesAndSamplerRule type for use
// with apply.
type AttributesAndSamplerRuleApplyConfiguration struct {
	AttributeConditions []AttributeConditionApplyConfiguration `json:"attributeConditions,omitempty"`
	Fraction            *float64                               `json:"fraction,omitempty"`
}

// AttributesAndSamplerRuleApplyConfiguration constructs an declarative configuration of the AttributesAndSamplerRule type for use with
// apply.
func AttributesAndSamplerRule() *AttributesAndSamplerRuleApplyConfiguration {
	return &AttributesAndSamplerRuleApplyConfiguration{}
}

// WithAttributeConditions adds the given value to the AttributeConditions field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the AttributeConditions field.
func (b *AttributesAndSamplerRuleApplyConfiguration) WithAttributeConditions(values ...*AttributeConditionApplyConfiguration) *AttributesAndSamplerRuleApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithAttributeConditions")
		}
		b.AttributeConditions = append(b.AttributeConditions, *values[i])
	}
	return b
}

// WithFraction sets the Fraction field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Fraction field is set to the value of the last call.
func (b *AttributesAndSamplerRuleApplyConfiguration) WithFraction(value float64) *AttributesAndSamplerRuleApplyConfiguration {
	b.Fraction = &value
	return b
}
