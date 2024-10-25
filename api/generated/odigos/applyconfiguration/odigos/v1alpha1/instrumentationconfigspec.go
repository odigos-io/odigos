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

// InstrumentationConfigSpecApplyConfiguration represents a declarative configuration of the InstrumentationConfigSpec type for use
// with apply.
type InstrumentationConfigSpecApplyConfiguration struct {
	RuntimeDetailsInvalidated *bool                                             `json:"runtimeDetailsInvalidated,omitempty"`
	Config                    []WorkloadInstrumentationConfigApplyConfiguration `json:"config,omitempty"`
	SdkConfigs                []SdkConfigApplyConfiguration                     `json:"sdkConfigs,omitempty"`
}

// InstrumentationConfigSpecApplyConfiguration constructs a declarative configuration of the InstrumentationConfigSpec type for use with
// apply.
func InstrumentationConfigSpec() *InstrumentationConfigSpecApplyConfiguration {
	return &InstrumentationConfigSpecApplyConfiguration{}
}

// WithRuntimeDetailsInvalidated sets the RuntimeDetailsInvalidated field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the RuntimeDetailsInvalidated field is set to the value of the last call.
func (b *InstrumentationConfigSpecApplyConfiguration) WithRuntimeDetailsInvalidated(value bool) *InstrumentationConfigSpecApplyConfiguration {
	b.RuntimeDetailsInvalidated = &value
	return b
}

// WithConfig adds the given value to the Config field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Config field.
func (b *InstrumentationConfigSpecApplyConfiguration) WithConfig(values ...*WorkloadInstrumentationConfigApplyConfiguration) *InstrumentationConfigSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithConfig")
		}
		b.Config = append(b.Config, *values[i])
	}
	return b
}

// WithSdkConfigs adds the given value to the SdkConfigs field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the SdkConfigs field.
func (b *InstrumentationConfigSpecApplyConfiguration) WithSdkConfigs(values ...*SdkConfigApplyConfiguration) *InstrumentationConfigSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithSdkConfigs")
		}
		b.SdkConfigs = append(b.SdkConfigs, *values[i])
	}
	return b
}