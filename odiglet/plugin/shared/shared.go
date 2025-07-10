package shared

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/odigos-io/odigos/instrumentation"
	proto "github.com/odigos-io/odigos/odiglet/plugin/proto/v1"
)

// Handshake is a common handshake that is shared by plugin and host.
var Handshake = plugin.HandshakeConfig{
	// This isn't required when using VersionedPlugins
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN",
	MagicCookieValue: "hello",
}

const OdigletPluginName = "odigos.io.odiglet.plugin.v1"

var PluginMap = map[string]plugin.Plugin{
	OdigletPluginName: &OdigletPlugin{},
}

type PluginV1 interface {
	Attach(ctx context.Context, pid int, settings instrumentation.Settings) error
	ApplyConfig(ctx context.Context, pid int, config instrumentation.Config) error
	Detach(ctx context.Context, pid int) error
}

type OdigletPlugin struct {
	plugin.GRPCPlugin
	plugin.NetRPCUnsupportedPlugin

	Impl PluginV1
}

func (p *OdigletPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterOdigletPluginV1Server(s, &GRPCServer{Impl: p.Impl})
	return nil
}

func (p *OdigletPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{client: proto.NewOdigletPluginV1Client(c)}, nil
}
