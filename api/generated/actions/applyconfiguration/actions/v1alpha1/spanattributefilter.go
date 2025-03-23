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

// SpanAttributeFilterApplyConfiguration represents a declarative configuration of the SpanAttributeFilter type for use
// with apply.
type SpanAttributeFilterApplyConfiguration struct {
	AttributeKey          *string  `json:"attribute_key,omitempty"`
	Condition             *string  `json:"condition,omitempty"`
	ExpectedValue         *string  `json:"expected_value,omitempty"`
	FallbackSamplingRatio *float64 `json:"fallback_sampling_ratio,omitempty"`
}

// SpanAttributeFilterApplyConfiguration constructs a declarative configuration of the SpanAttributeFilter type for use with
// apply.
func SpanAttributeFilter() *SpanAttributeFilterApplyConfiguration {
	return &SpanAttributeFilterApplyConfiguration{}
}

// WithAttributeKey sets the AttributeKey field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the AttributeKey field is set to the value of the last call.
func (b *SpanAttributeFilterApplyConfiguration) WithAttributeKey(value string) *SpanAttributeFilterApplyConfiguration {
	b.AttributeKey = &value
	return b
}

// WithCondition sets the Condition field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Condition field is set to the value of the last call.
func (b *SpanAttributeFilterApplyConfiguration) WithCondition(value string) *SpanAttributeFilterApplyConfiguration {
	b.Condition = &value
	return b
}

// WithExpectedValue sets the ExpectedValue field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ExpectedValue field is set to the value of the last call.
func (b *SpanAttributeFilterApplyConfiguration) WithExpectedValue(value string) *SpanAttributeFilterApplyConfiguration {
	b.ExpectedValue = &value
	return b
}

// WithFallbackSamplingRatio sets the FallbackSamplingRatio field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the FallbackSamplingRatio field is set to the value of the last call.
func (b *SpanAttributeFilterApplyConfiguration) WithFallbackSamplingRatio(value float64) *SpanAttributeFilterApplyConfiguration {
	b.FallbackSamplingRatio = &value
	return b
}
