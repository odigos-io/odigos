package instrumentationrules

type HttpPayloadCollection struct {
	MimeTypes           *[]string `json:"mimeTypes,omitempty"`
	MaxPayloadLength    *int64    `json:"maxPayloadLength,omitempty"`
	DropPartialPayloads *bool     `json:"dropPartialPayloads,omitempty"`
}

type DbQueryPayloadCollection struct {
	MaxPayloadLength    *int64 `json:"maxPayloadLength,omitempty"`
	DropPartialPayloads *bool  `json:"dropPartialPayloads,omitempty"`
}

type MessagingPayloadCollection struct {
	MaxPayloadLength    *int64 `json:"maxPayloadLength,omitempty"`
	DropPartialPayloads *bool  `json:"dropPartialPayloads,omitempty"`
}

type PayloadCollection struct {
	HttpRequest  *HttpPayloadCollection      `json:"httpRequest,omitempty"`
	HttpResponse *HttpPayloadCollection      `json:"httpResponse,omitempty"`
	DbQuery      *DbQueryPayloadCollection   `json:"dbQuery,omitempty"`
	Messaging    *MessagingPayloadCollection `json:"messaging,omitempty"`
}
