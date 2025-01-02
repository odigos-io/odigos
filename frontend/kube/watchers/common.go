package watchers

import (
	"github.com/odigos-io/odigos/frontend/endpoints/sse"
)

func genericErrorMessage(event sse.MessageEvent, crd string, data string) {
	sse.SendMessageToClient(sse.SSEMessage{
		Type:    sse.MessageTypeError,
		Event:   event,
		Data:    "Something went wrong: " + data,
		CRDType: crd,
		Target:  "",
	})
}
