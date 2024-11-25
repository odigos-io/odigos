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

// HttpRouteFilterApplyConfiguration represents a declarative configuration of the HttpRouteFilter type for use
// with apply.
type HttpRouteFilterApplyConfiguration struct {
	HttpRoute               *string  `json:"http_route,omitempty"`
	ServiceName             *string  `json:"service_name,omitempty"`
	MinimumLatencyThreshold *int     `json:"minimum_latency_threshold,omitempty"`
	FallbackSamplingRatio   *float64 `json:"fallback_sampling_ratio,omitempty"`
}

// HttpRouteFilterApplyConfiguration constructs a declarative configuration of the HttpRouteFilter type for use with
// apply.
func HttpRouteFilter() *HttpRouteFilterApplyConfiguration {
	return &HttpRouteFilterApplyConfiguration{}
}

// WithHttpRoute sets the HttpRoute field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the HttpRoute field is set to the value of the last call.
func (b *HttpRouteFilterApplyConfiguration) WithHttpRoute(value string) *HttpRouteFilterApplyConfiguration {
	b.HttpRoute = &value
	return b
}

// WithServiceName sets the ServiceName field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ServiceName field is set to the value of the last call.
func (b *HttpRouteFilterApplyConfiguration) WithServiceName(value string) *HttpRouteFilterApplyConfiguration {
	b.ServiceName = &value
	return b
}

// WithMinimumLatencyThreshold sets the MinimumLatencyThreshold field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the MinimumLatencyThreshold field is set to the value of the last call.
func (b *HttpRouteFilterApplyConfiguration) WithMinimumLatencyThreshold(value int) *HttpRouteFilterApplyConfiguration {
	b.MinimumLatencyThreshold = &value
	return b
}

// WithFallbackSamplingRatio sets the FallbackSamplingRatio field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the FallbackSamplingRatio field is set to the value of the last call.
func (b *HttpRouteFilterApplyConfiguration) WithFallbackSamplingRatio(value float64) *HttpRouteFilterApplyConfiguration {
	b.FallbackSamplingRatio = &value
	return b
}