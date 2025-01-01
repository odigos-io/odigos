package utils

import (
	"encoding/json"

	"github.com/go-logr/logr"
)

func MarshalConfig(config map[string]interface{}, logger logr.Logger) []byte {
	data, err := json.Marshal(config)
	if err != nil {
		logger.Error(err, "Failed to marshal processor config")
		return []byte("{}")
	}
	return data
}

func Contains(arr []string, val string) bool {
	for _, item := range arr {
		if item == val {
			return true
		}
	}
	return false
}
