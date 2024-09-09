package instrumentationrules

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type HttpPayloadCollection struct {

	// Limit payload collection to specific mime types based on the content type header.
	// When not specified, all mime types payloads will be collected.
	// Empty array will make the rule ineffective.
	MimeTypes *[]string `json:"mimeTypes,omitempty"`

	// Maximum length of the payload to collect.
	// If the payload is longer than this value, it will be truncated or dropped, based on the value of `dropPartialPayloads` config option
	MaxPayloadLength *int64 `json:"maxPayloadLength,omitempty"`

	// If the payload is larger than the MaxPayloadLength, this parameter will determine if the payload should be partially collected up to the allowed length, or not collected at all.
	// This is useful if you require some decoding of the payload (like json) and having it partially is not useful.
	DropPartialPayloads *bool `json:"dropPartialPayloads,omitempty"`
}

// Rule for collecting payloads for a DbStatement
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type DbQueryPayloadCollection struct {

	// Maximum length of the payload to collect.
	// If the payload is longer than this value, it will be truncated or dropped, based on the value of `dropPartialPayloads` config option
	MaxPayloadLength *int64 `json:"maxPayloadLength,omitempty"`

	// If the payload is larger than the MaxPayloadLength, this parameter will determine if the payload should be partially collected up to the allowed length, or not collected at all.
	// This is useful if you require some decoding of the payload (like json) and having it partially is not useful.
	DropPartialPayloads *bool `json:"dropPartialPayloads,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type PayloadCollection struct {
	// Collect HTTP request payload data when available.
	// Can be a client (outgoing) request or a server (incoming) request, depending on the instrumentation library
	HttpRequest *HttpPayloadCollection `json:"httpRequest,omitempty"`

	// rule for collecting the response part of an http payload.
	// Can be a client response or a server response, depending on the instrumentation library
	HttpResponse *HttpPayloadCollection `json:"httpResponse,omitempty"`

	// rule for collecting db payloads for the mentioned workload and instrumentation libraries
	DbQuery *DbQueryPayloadCollection `json:"dbQuery,omitempty"`
}
