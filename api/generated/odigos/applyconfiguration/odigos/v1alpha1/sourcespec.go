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
	k8sconsts "github.com/odigos-io/odigos/api/k8sconsts"
)

// SourceSpecApplyConfiguration represents a declarative configuration of the SourceSpec type for use
// with apply.
type SourceSpecApplyConfiguration struct {
	Workload               *k8sconsts.PodWorkload `json:"workload,omitempty"`
	DisableInstrumentation *bool                  `json:"disableInstrumentation,omitempty"`
	OtelServiceName        *string                `json:"otelServiceName,omitempty"`
}

// SourceSpecApplyConfiguration constructs a declarative configuration of the SourceSpec type for use with
// apply.
func SourceSpec() *SourceSpecApplyConfiguration {
	return &SourceSpecApplyConfiguration{}
}

// WithWorkload sets the Workload field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Workload field is set to the value of the last call.
func (b *SourceSpecApplyConfiguration) WithWorkload(value k8sconsts.PodWorkload) *SourceSpecApplyConfiguration {
	b.Workload = &value
	return b
}

// WithDisableInstrumentation sets the DisableInstrumentation field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the DisableInstrumentation field is set to the value of the last call.
func (b *SourceSpecApplyConfiguration) WithDisableInstrumentation(value bool) *SourceSpecApplyConfiguration {
	b.DisableInstrumentation = &value
	return b
}

// WithOtelServiceName sets the OtelServiceName field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the OtelServiceName field is set to the value of the last call.
func (b *SourceSpecApplyConfiguration) WithOtelServiceName(value string) *SourceSpecApplyConfiguration {
	b.OtelServiceName = &value
	return b
}
