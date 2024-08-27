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

// WorkloadInstrumentationConfigApplyConfiguration represents a declarative configuration of the WorkloadInstrumentationConfig type for use
// with apply.
type WorkloadInstrumentationConfigApplyConfiguration struct {
	OptionKey                *string                                    `json:"optionKey,omitempty"`
	SpanKind                 *common.SpanKind                           `json:"spanKind,omitempty"`
	OptionValueBoolean       *bool                                      `json:"optionValueBoolean,omitempty"`
	InstrumentationLibraries []InstrumentationLibraryApplyConfiguration `json:"instrumentationLibraries,omitempty"`
}

// WorkloadInstrumentationConfigApplyConfiguration constructs a declarative configuration of the WorkloadInstrumentationConfig type for use with
// apply.
func WorkloadInstrumentationConfig() *WorkloadInstrumentationConfigApplyConfiguration {
	return &WorkloadInstrumentationConfigApplyConfiguration{}
}

// WithOptionKey sets the OptionKey field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the OptionKey field is set to the value of the last call.
func (b *WorkloadInstrumentationConfigApplyConfiguration) WithOptionKey(value string) *WorkloadInstrumentationConfigApplyConfiguration {
	b.OptionKey = &value
	return b
}

// WithSpanKind sets the SpanKind field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the SpanKind field is set to the value of the last call.
func (b *WorkloadInstrumentationConfigApplyConfiguration) WithSpanKind(value common.SpanKind) *WorkloadInstrumentationConfigApplyConfiguration {
	b.SpanKind = &value
	return b
}

// WithOptionValueBoolean sets the OptionValueBoolean field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the OptionValueBoolean field is set to the value of the last call.
func (b *WorkloadInstrumentationConfigApplyConfiguration) WithOptionValueBoolean(value bool) *WorkloadInstrumentationConfigApplyConfiguration {
	b.OptionValueBoolean = &value
	return b
}

// WithInstrumentationLibraries adds the given value to the InstrumentationLibraries field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the InstrumentationLibraries field.
func (b *WorkloadInstrumentationConfigApplyConfiguration) WithInstrumentationLibraries(values ...*InstrumentationLibraryApplyConfiguration) *WorkloadInstrumentationConfigApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithInstrumentationLibraries")
		}
		b.InstrumentationLibraries = append(b.InstrumentationLibraries, *values[i])
	}
	return b
}
