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

import (
	common "github.com/odigos-io/odigos/common"
)

// SpanAttributeSamplerSpecApplyConfiguration represents a declarative configuration of the SpanAttributeSamplerSpec type for use
// with apply.
type SpanAttributeSamplerSpecApplyConfiguration struct {
	ActionName        *string                                 `json:"actionName,omitempty"`
	Notes             *string                                 `json:"notes,omitempty"`
	Disabled          *bool                                   `json:"disabled,omitempty"`
	Signals           []common.ObservabilitySignal            `json:"signals,omitempty"`
	AttributesFilters []SpanAttributeFilterApplyConfiguration `json:"attributes_filters,omitempty"`
}

// SpanAttributeSamplerSpecApplyConfiguration constructs a declarative configuration of the SpanAttributeSamplerSpec type for use with
// apply.
func SpanAttributeSamplerSpec() *SpanAttributeSamplerSpecApplyConfiguration {
	return &SpanAttributeSamplerSpecApplyConfiguration{}
}

// WithActionName sets the ActionName field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ActionName field is set to the value of the last call.
func (b *SpanAttributeSamplerSpecApplyConfiguration) WithActionName(value string) *SpanAttributeSamplerSpecApplyConfiguration {
	b.ActionName = &value
	return b
}

// WithNotes sets the Notes field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Notes field is set to the value of the last call.
func (b *SpanAttributeSamplerSpecApplyConfiguration) WithNotes(value string) *SpanAttributeSamplerSpecApplyConfiguration {
	b.Notes = &value
	return b
}

// WithDisabled sets the Disabled field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Disabled field is set to the value of the last call.
func (b *SpanAttributeSamplerSpecApplyConfiguration) WithDisabled(value bool) *SpanAttributeSamplerSpecApplyConfiguration {
	b.Disabled = &value
	return b
}

// WithSignals adds the given value to the Signals field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Signals field.
func (b *SpanAttributeSamplerSpecApplyConfiguration) WithSignals(values ...common.ObservabilitySignal) *SpanAttributeSamplerSpecApplyConfiguration {
	for i := range values {
		b.Signals = append(b.Signals, values[i])
	}
	return b
}

// WithAttributesFilters adds the given value to the AttributesFilters field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the AttributesFilters field.
func (b *SpanAttributeSamplerSpecApplyConfiguration) WithAttributesFilters(values ...*SpanAttributeFilterApplyConfiguration) *SpanAttributeSamplerSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithAttributesFilters")
		}
		b.AttributesFilters = append(b.AttributesFilters, *values[i])
	}
	return b
}
