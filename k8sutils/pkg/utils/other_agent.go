package utils

import "errors"

var ErrOtherAgentRun = errors.New("device not added to any container due to the presence of another agent")
