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
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

// ContainerAgentConfigApplyConfiguration represents a declarative configuration of the ContainerAgentConfig type for use
// with apply.
type ContainerAgentConfigApplyConfiguration struct {
	ContainerName       *string                            `json:"containerName,omitempty"`
	AgentEnabled        *bool                              `json:"agentEnabled,omitempty"`
	AgentEnabledReason  *odigosv1alpha1.AgentEnabledReason `json:"agentEnabledReason,omitempty"`
	AgentEnabledMessage *string                            `json:"agentEnabledMessage,omitempty"`
	OtelDistroName      *string                            `json:"otelDistroName,omitempty"`
	DistroParams        map[string]string                  `json:"distroParams,omitempty"`
}

// ContainerAgentConfigApplyConfiguration constructs a declarative configuration of the ContainerAgentConfig type for use with
// apply.
func ContainerAgentConfig() *ContainerAgentConfigApplyConfiguration {
	return &ContainerAgentConfigApplyConfiguration{}
}

// WithContainerName sets the ContainerName field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ContainerName field is set to the value of the last call.
func (b *ContainerAgentConfigApplyConfiguration) WithContainerName(value string) *ContainerAgentConfigApplyConfiguration {
	b.ContainerName = &value
	return b
}

// WithAgentEnabled sets the AgentEnabled field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the AgentEnabled field is set to the value of the last call.
func (b *ContainerAgentConfigApplyConfiguration) WithAgentEnabled(value bool) *ContainerAgentConfigApplyConfiguration {
	b.AgentEnabled = &value
	return b
}

// WithAgentEnabledReason sets the AgentEnabledReason field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the AgentEnabledReason field is set to the value of the last call.
func (b *ContainerAgentConfigApplyConfiguration) WithAgentEnabledReason(value odigosv1alpha1.AgentEnabledReason) *ContainerAgentConfigApplyConfiguration {
	b.AgentEnabledReason = &value
	return b
}

// WithAgentEnabledMessage sets the AgentEnabledMessage field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the AgentEnabledMessage field is set to the value of the last call.
func (b *ContainerAgentConfigApplyConfiguration) WithAgentEnabledMessage(value string) *ContainerAgentConfigApplyConfiguration {
	b.AgentEnabledMessage = &value
	return b
}

// WithOtelDistroName sets the OtelDistroName field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the OtelDistroName field is set to the value of the last call.
func (b *ContainerAgentConfigApplyConfiguration) WithOtelDistroName(value string) *ContainerAgentConfigApplyConfiguration {
	b.OtelDistroName = &value
	return b
}

// WithDistroParams puts the entries into the DistroParams field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the DistroParams field,
// overwriting an existing map entries in DistroParams field with the same key.
func (b *ContainerAgentConfigApplyConfiguration) WithDistroParams(entries map[string]string) *ContainerAgentConfigApplyConfiguration {
	if b.DistroParams == nil && len(entries) > 0 {
		b.DistroParams = make(map[string]string, len(entries))
	}
	for k, v := range entries {
		b.DistroParams[k] = v
	}
	return b
}
