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

// CollectorsGroupStatusApplyConfiguration represents an declarative configuration of the CollectorsGroupStatus type for use
// with apply.
type CollectorsGroupStatusApplyConfiguration struct {
	Ready           *bool                        `json:"ready,omitempty"`
	ReceiverSignals []common.ObservabilitySignal `json:"receiverSignals,omitempty"`
}

// CollectorsGroupStatusApplyConfiguration constructs an declarative configuration of the CollectorsGroupStatus type for use with
// apply.
func CollectorsGroupStatus() *CollectorsGroupStatusApplyConfiguration {
	return &CollectorsGroupStatusApplyConfiguration{}
}

// WithReady sets the Ready field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Ready field is set to the value of the last call.
func (b *CollectorsGroupStatusApplyConfiguration) WithReady(value bool) *CollectorsGroupStatusApplyConfiguration {
	b.Ready = &value
	return b
}

// WithReceiverSignals adds the given value to the ReceiverSignals field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the ReceiverSignals field.
func (b *CollectorsGroupStatusApplyConfiguration) WithReceiverSignals(values ...common.ObservabilitySignal) *CollectorsGroupStatusApplyConfiguration {
	for i := range values {
		b.ReceiverSignals = append(b.ReceiverSignals, values[i])
	}
	return b
}
