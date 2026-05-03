package unixfd

import (
	"fmt"
	"strconv"
	"strings"
)

// LogsAttrEventType distinguishes register from unregister attribute events.
type LogsAttrEventType int

const (
	LogsAttrRegister LogsAttrEventType = iota
	LogsAttrUnregister
)

// LogsAttrEvent is a parsed attribute event from the logs attribute stream.
type LogsAttrEvent struct {
	Type  LogsAttrEventType
	PID   uint32
	Attrs string // packed key:val,key:val — only meaningful for Register
}

// EncodeLogsAttrRegister encodes a register event: "R<pid>:<attrs>".
func EncodeLogsAttrRegister(pid uint32, packedAttrs string) string {
	return fmt.Sprintf("R%d:%s", pid, packedAttrs)
}

// EncodeLogsAttrUnregister encodes an unregister event: "U<pid>".
func EncodeLogsAttrUnregister(pid uint32) string {
	return fmt.Sprintf("U%d", pid)
}

// DecodeLogsAttrEvent parses a single line from the attribute stream.
// Returns the parsed event and true on success, or a zero event and false
// if the line is malformed.
func DecodeLogsAttrEvent(line string) (LogsAttrEvent, bool) {
	if len(line) < 2 {
		return LogsAttrEvent{}, false
	}
	switch line[0] {
	case 'R':
		idx := strings.IndexByte(line, ':')
		if idx < 2 {
			return LogsAttrEvent{}, false
		}
		pid, err := strconv.ParseUint(line[1:idx], 10, 32)
		if err != nil {
			return LogsAttrEvent{}, false
		}
		return LogsAttrEvent{
			Type:  LogsAttrRegister,
			PID:   uint32(pid),
			Attrs: line[idx+1:],
		}, true
	case 'U':
		pid, err := strconv.ParseUint(line[1:], 10, 32)
		if err != nil {
			return LogsAttrEvent{}, false
		}
		return LogsAttrEvent{
			Type: LogsAttrUnregister,
			PID:  uint32(pid),
		}, true
	default:
		return LogsAttrEvent{}, false
	}
}
