package watchers

import (
	"github.com/odigos-io/odigos/frontend/endpoints/sse"
)

func genericErrorMessage(event sse.MessageEvent, crd string, data string) {
	sse.SendMessageToClient(sse.SSEMessage{
		Event: event,
		Type: sse.MessageTypeError,
		Target: "",
		Data: "Something went wrong: " + data,
		CRDType: crd,
	})
}