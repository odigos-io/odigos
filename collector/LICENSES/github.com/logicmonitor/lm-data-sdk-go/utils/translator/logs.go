package translator

import (
	"github.com/logicmonitor/lm-data-sdk-go/model"
)

func ConvertToLMLogInput(logMessage interface{}, timestamp string, resourceIDMap, metadata map[string]interface{}) model.LogInput {
	return model.LogInput{
		Message:    logMessage,
		ResourceID: resourceIDMap,
		Metadata:   metadata,
		Timestamp:  timestamp,
	}
}
