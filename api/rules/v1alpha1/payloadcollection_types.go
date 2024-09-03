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

package v1alpha1

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HttpPayloadCollectionRule struct {

	// which mime types to allow for collection, as specified in the http header
	// if any item in the list matches, the payload will be considered for collection
	// if empty list - non of the mime types will be collected
	// if nil - all mime types will be collected
	AllowedMimeType *[]string `json:"allowedMimeTypePatterns,omitempty"`

	// the maximum length of the payload to collect
	// This value relates to the actual payload length in the attribute, which might be different than the length in bytes due to encoding.
	// if the content length is below or equal to this value, the payload will be collected
	// if the content length is above this value, the decision to collect will be based on the dropPartialPayloads parameter
	MaxPayloadLength *int64 `json:"maxPayloadLength,omitempty"`

	// If the payload is larger than the MaxPayloadLength, this parameter will determine if the payload should be partially collected up to the allowed length, or not collected at all.
	// This is useful if you require some decoding of the payload (like json) and having it partially is not useful.
	DropPartialPayloads *bool `json:"dropPartialPayloads,omitempty"`
}

// Rule for collecting payloads for a DbStatement
type DbStatementPayloadCollectionRule struct {

	// the maximum length of the payload to collect
	// This value relates to the actual payload length in the attribute, which might be different than the length in bytes due to encoding.
	// if the content length is below or equal to this value, the payload will be collected
	// if the content length is above this value, the decision to collect will be based on the dropPartialPayloads parameter
	MaxPayloadLength *int64 `json:"maxPayloadLength,omitempty"`

	// If the payload is larger than the maxContentLength, this parameter will determine if the payload should be partially collected up to the allowed length, or not collected at all.
	// This is useful if the db statement is only useful when complete.
	DropPartialPayloads *bool `json:"dropPartialPayloads,omitempty"`
}

type InstrumentationLibraryId struct {
	// The name of the instrumentation library
	Name string `json:"name"`

	// The language in which this library will collect data
	Language common.ProgrammingLanguage `json:"language"`

	// SpanKind is only supported by Golang and will be ignored for any other SDK language.
	// In Go, SpanKind is used because the same instrumentation library can be utilized for different span kinds (e.g., client/server).
	SpanKind common.SpanKind `json:"spanKind,omitempty"`
}

type PayloadCollectionSpec struct {

	// free text to give a human readable name to the rule if desired
	RuleName string `json:"ruleName,omitempty"`

	// Place to document the purpose of the rule if desired
	Notes string `json:"notes,omitempty"`

	// A flag for users allowing to temporarily disable the rule, but keep it around for future use
	Disabled bool `json:"disabled,omitempty"`

	// To which workloads should this rule apply
	// Empty list will make this rule ineffective for all workloads
	// nil will make this rule apply to all workloads
	Workloads *[]workload.PodWorkload `json:"workloads,omitempty"`

	// For fine grained control, the user can specify the instrumentation library to use.
	// One can specify same rule for multiple languages and libraries at the same time.
	// If nil, all instrumentation libraries will be used.
	// If empty, no instrumentation libraries will be used.
	InstrumentationLibraries *[]InstrumentationLibraryId `json:"instrumentationLibraries,omitempty"`

	// rule for collecting the request part of an http payload.
	// request can be a client request (incoming ) or a server request, depending on the instrumentation library
	HttpRequestPayloadCollectionRule *HttpPayloadCollectionRule `json:"httpRequestPayloadCollectionRule,omitempty"`

	// rule for collecting the response part of an http payload.
	// response can be a client response or a server response, depending on the instrumentation library
	HttpResponsePayloadCollectionRule *HttpPayloadCollectionRule `json:"httpResponsePayloadCollectionRule,omitempty"`

	// rule for collecting db payloads for the mentioned workload and instrumentation libraries
	DbStatementPayloadCollectionRule *DbStatementPayloadCollectionRule `json:"dbStatementPayloadCollectionRule,omitempty"`
}

type PayloadCollectionStatus struct {
	// Represents the observations of a payloadcollection's current state.
	// Known .status.conditions.type are: "Available", "Progressing"
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=payloadcollection,scope=Namespaced
//+kubebuilder:metadata:labels=metadata.labels.odigos.io/config=1
//+kubebuilder:metadata:labels=metadata.labels.odigos.io/system-object=true

type PayloadCollection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PayloadCollectionSpec   `json:"spec,omitempty"`
	Status PayloadCollectionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type PayloadCollectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PayloadCollection `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PayloadCollection{}, &PayloadCollectionList{})
}
