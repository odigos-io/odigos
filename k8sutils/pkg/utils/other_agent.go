package utils

import "errors"

var OtherAgentRunError = errors.New("device not added to any container due to the presence of another agent")
