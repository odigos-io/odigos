package unixfd

import (
	"fmt"
	"strconv"
	"strings"
)

// AttrEventType distinguishes register from unregister attribute events.
// Shared between the logs FD stream and the profiles attribute stream — wire format identical.
type AttrEventType int

const (
	AttrEventRegister AttrEventType = iota
	AttrEventUnregister
)

// AttrEvent is a parsed register / unregister event from an attribute stream.
type AttrEvent struct {
	Type  AttrEventType
	PID   uint32
	Attrs string // packed key:val,key:val — only meaningful for Register
}

// EncodeAttrRegister encodes a register event as "R<pid>:<attrs>".
func EncodeAttrRegister(pid uint32, packedAttrs string) string {
	return fmt.Sprintf("R%d:%s", pid, packedAttrs)
}

// EncodeAttrUnregister encodes an unregister event as "U<pid>".
func EncodeAttrUnregister(pid uint32) string {
	return fmt.Sprintf("U%d", pid)
}

// DecodeAttrEvent parses a single line from the attribute stream.
// Returns the parsed event and true on success; zero event + false on malformed input.
func DecodeAttrEvent(line string) (AttrEvent, bool) {
	if len(line) < 2 {
		return AttrEvent{}, false
	}
	switch line[0] {
	case 'R':
		idx := strings.IndexByte(line, ':')
		if idx < 2 {
			return AttrEvent{}, false
		}
		pid, err := strconv.ParseUint(line[1:idx], 10, 32)
		if err != nil {
			return AttrEvent{}, false
		}
		return AttrEvent{
			Type:  AttrEventRegister,
			PID:   uint32(pid),
			Attrs: line[idx+1:],
		}, true
	case 'U':
		pid, err := strconv.ParseUint(line[1:], 10, 32)
		if err != nil {
			return AttrEvent{}, false
		}
		return AttrEvent{
			Type: AttrEventUnregister,
			PID:  uint32(pid),
		}, true
	default:
		return AttrEvent{}, false
	}
}
