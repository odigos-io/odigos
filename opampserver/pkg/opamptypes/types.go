package opamptypes

import (
	"context"

	commonopamp "github.com/odigos-io/odigos/common/opamp"
	"github.com/odigos-io/odigos/opampserver/protobufs"
)

// ProcessStatus is the outcome of handling one AgentToServer message.
type ProcessStatus int

const (
	ProcessOK ProcessStatus = iota
	ProcessBadRequest
	ProcessError
)

// Processor handles OpAMP messages for any transport.
type Processor interface {
	Process(ctx context.Context, agentToServer *protobufs.AgentToServer, transport commonopamp.OpAmpTransport) (*protobufs.ServerToAgent, ProcessStatus)
}
