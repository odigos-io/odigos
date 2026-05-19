package instrumentationrules

import "github.com/odigos-io/odigos/common/consts"

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type HttpPayloadCollection struct {

	// Limit payload collection to specific mime types based on the content type header.
	// When not specified, all mime types payloads will be collected.
	// Empty array will make the rule ineffective.
	MimeTypes *[]string `json:"mimeTypes,omitempty" yaml:"mimeTypes,omitempty"`

	// Maximum length of the payload to collect.
	// If the payload is longer than this value, it will be truncated or dropped, based on the value of `dropPartialPayloads` config option
	MaxPayloadLength *int64 `json:"maxPayloadLength,omitempty" yaml:"maxPayloadLength,omitempty"`

	// If payload exceeds MaxPayloadLength, controls whether to truncate it or drop it entirely.
	DropPartialPayloads *bool `json:"dropPartialPayloads,omitempty" yaml:"dropPartialPayloads,omitempty"`
}

// Rule for collecting payloads for a Db Query Text
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type DbQueryPayloadCollection struct {

	// Maximum length of the payload to collect.
	// If the payload is longer than this value, it will be truncated or dropped, based on the value of `dropPartialPayloads` config option
	MaxPayloadLength *int64 `json:"maxPayloadLength,omitempty" yaml:"maxPayloadLength,omitempty"`

	// If payload exceeds MaxPayloadLength, controls whether to truncate it or drop it entirely.
	DropPartialPayloads *bool `json:"dropPartialPayloads,omitempty" yaml:"dropPartialPayloads,omitempty"`

	// The sanitization policy to use for collecting the DB query payloads.
	// If not specified, the default sanitization policy will be used:
	// (collect sanitized payloads if possible, and fall back to full if sanitization isn't supported).
	SanitizationPolicy *consts.DbQuerySanitizationPolicy `json:"sanitizationPolicy,omitempty" yaml:"sanitizationPolicy,omitempty"`
}

// Rule for collecting messaging related payloads
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type MessagingPayloadCollection struct {

	// Maximum length of the payload to collect.
	// If the payload is longer than this value, it will be truncated or dropped, based on the value of `dropPartialPayloads` config option
	MaxPayloadLength *int64 `json:"maxPayloadLength,omitempty" yaml:"maxPayloadLength,omitempty"`

	// If payload exceeds MaxPayloadLength, controls whether to truncate it or drop it entirely.
	DropPartialPayloads *bool `json:"dropPartialPayloads,omitempty" yaml:"dropPartialPayloads,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type PayloadCollection struct {
	// Collect HTTP request payload data when available.
	// Can be a client (outgoing) request or a server (incoming) request, depending on the instrumentation library
	HttpRequest *HttpPayloadCollection `json:"httpRequest,omitempty" yaml:"httpRequest,omitempty"`

	// rule for collecting the response part of an http payload.
	// Can be a client response or a server response, depending on the instrumentation library
	HttpResponse *HttpPayloadCollection `json:"httpResponse,omitempty" yaml:"httpResponse,omitempty"`

	// rule for collecting db payloads for the mentioned workload and instrumentation libraries
	DbQuery *DbQueryPayloadCollection `json:"dbQuery,omitempty" yaml:"dbQuery,omitempty"`

	// rule for collecting messaging payloads for the mentioned workload and instrumentation libraries
	Messaging *MessagingPayloadCollection `json:"messaging,omitempty" yaml:"messaging,omitempty"`
}
