package shared

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/odigos-io/odigos/instrumentation"
	proto "github.com/odigos-io/odigos/instrumentation/plugin/proto/v1"
)

// Handshake is a common handshake that is shared by plugin and host.
var Handshake = plugin.HandshakeConfig{
	// This isn't required when using VersionedPlugins
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN",
	MagicCookieValue: "hello",
}

const InstrumentationPluginName = "odigos.io.instrumentation.plugin.v1"

var PluginMap = map[string]plugin.Plugin{
	InstrumentationPluginName: &InstrumentationPlugin{},
}

type Instrumentation interface {
	Start(ctx context.Context, pid int, settings instrumentation.Settings) error
	ApplyConfig(ctx context.Context, pid int, config instrumentation.Config) error
	Close(ctx context.Context, pid int) error
}

type InstrumentationPlugin struct {
	plugin.GRPCPlugin
	plugin.NetRPCUnsupportedPlugin

	Impl Instrumentation
}

func (p *InstrumentationPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterInstrumentationServer(s, &GRPCServer{Impl: p.Impl})
	return nil
}

func (p *InstrumentationPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{client: proto.NewInstrumentationClient(c)}, nil
}