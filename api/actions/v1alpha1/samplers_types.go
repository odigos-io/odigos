/*
Copyright 2025.

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

package v1alpha1

import "github.com/odigos-io/odigos/api/k8sconsts"

const (
	ActionNameSamplers             = "Samplers"
	ActionNameSpanAttributeSampler = "SpanAttributeSampler"
	ActionNameLatencySampler       = "LatencySampler"
	ActionNameErrorSampler         = "ErrorSampler"
	ActionNameServiceNameSampler   = "ServiceNameSampler"
	ActionNameProbabilisticSampler = "ProbabilisticSampler"
	ActionNameIgnoreHealthChecks   = "IgnoreHealthChecks"
)

type SamplersConfig struct {
	DefaultSamplerConfig `json:",inline"`

	// ErrorSamplerSpec defines the desired state of ErrorSampler
	ErrorSampler *ErrorSamplerConfig `json:"errorSampler,omitempty"`

	// LatencySamplerSpec defines the desired state of LatencySampler
	LatencySampler *LatencySamplerConfig `json:"latencySampler,omitempty"`

	// ServiceNameSamplerSpec defines the desired state of ServiceNameSampler
	ServiceNameSampler *ServiceNameSamplerConfig `json:"serviceNameSampler,omitempty"`

	// SpanAttributeSamplerSpec defines the desired state of SpanAttributeSampler
	SpanAttributeSampler *SpanAttributeSamplerConfig `json:"spanAttributeSampler,omitempty"`

	// ProbabilisticSamplerSpec defines the desired state of ProbabilisticSampler
	ProbabilisticSampler *ProbabilisticSamplerConfig `json:"probabilisticSampler,omitempty"`

	// automatically sample out health checks by a set fraction.
	// health checks are discovered based on the livenessProbe and readinessProbe in the workload spec.
	// if set, health checks are sampled regardless of any other configuration (head sampling at agent level, before traces are pushed into the collector pipeline)
	// @deprecated: use odigos config to enable/disable this behavior instead.
	IgnoreHealthChecks *IgnoreHealthChecksConfig `json:"ignoreHealthChecks,omitempty"`
}

// DefaultSamplerConfig is a base config for all samplers.
type DefaultSamplerConfig struct{}

func (DefaultSamplerConfig) ProcessorType() string {
	return "odigossampling"
}

func (DefaultSamplerConfig) OrderHint() int {
	return -24
}

func (DefaultSamplerConfig) CollectorRoles() []k8sconsts.CollectorRole {
	return []k8sconsts.CollectorRole{
		k8sconsts.CollectorsRoleClusterGateway,
	}
}

// ErrorSamplerConfig defines the configuration for the ErrorSampler action.
type ErrorSamplerConfig struct {
	// FallbackSamplingRatio determines the percentage (0–100) of non-error traces
	// that should be sampled. Error traces are always sampled.
	FallbackSamplingRatio float64 `json:"fallback_sampling_ratio"`
}

// LatencySamplerConfig defines the configuration for the LatencySampler action.
type LatencySamplerConfig struct {
	// EndpointsFilters defines the list of route-based latency sampling filters.
	// Each filter targets a specific service and HTTP route with a latency threshold.
	EndpointsFilters []HttpRouteFilter `json:"endpoints_filters"`
}

// ServiceNameSamplerConfig defines the configuration for the ServiceNameSampler action.
type ServiceNameSamplerConfig struct {
	// ServicesNameFilters defines rules for sampling traces based on the presence
	// of specific service names. If a trace contains a span from one of the listed
	// services, the associated sampling ratio is applied.
	ServicesNameFilters []ServiceNameFilter `json:"services_name_filters"`
}

// SpanAttributeSamplerConfig defines the configuration for the SpanAttributeSampler action.
type SpanAttributeSamplerConfig struct {
	// AttributeFilters defines a list of criteria to decide how spans should be
	// sampled based on their attributes. At least one filter is required.
	AttributeFilters []SpanAttributeFilter `json:"attribute_filters"`
}

// ProbabilisticSamplerConfig defines the configuration for the ProbabilisticSampler action.
type ProbabilisticSamplerConfig struct {
	// SamplingPercentage determines the percentage (0–100) of traces to sample.
	SamplingPercentage string `json:"sampling_percentage"`
}

type IgnoreHealthChecksConfig struct {
	// FractionToRecord determines the percentage (0–100) of health checks traces to record.
	// should be in range [0, 1]
	// 0 (default) means no health checks traces will be recorded
	// 1 means all health checks traces will be recorded
	// +kubebuilder:default:=0
	// +kubebuilder:validation:Minimum:=0
	// +kubebuilder:validation:Maximum:=1
	// @deprecated: use odigos config to enable/disable this behavior instead.
	FractionToRecord float64 `json:"fractionToRecord,omitempty"`
}

func (ProbabilisticSamplerConfig) ProcessorType() string {
	return "probabilistic_sampler"
}

func (ProbabilisticSamplerConfig) OrderHint() int {
	return 1
}

func (ProbabilisticSamplerConfig) CollectorRoles() []k8sconsts.CollectorRole {
	return []k8sconsts.CollectorRole{
		k8sconsts.CollectorsRoleNodeCollector,
	}
}
