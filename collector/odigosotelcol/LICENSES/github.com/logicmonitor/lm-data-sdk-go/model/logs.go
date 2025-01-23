package model

type LogInput struct {
	Message    interface{}
	ResourceID map[string]interface{}
	Metadata   map[string]interface{}
	Timestamp  string
}

type LogPayload map[string]interface{}
