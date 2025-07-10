package shared

import (
	"context"

	"github.com/odigos-io/odigos/instrumentation"
	proto "github.com/odigos-io/odigos/instrumentation/plugin/proto/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

type GRPCServer struct {
	proto.UnimplementedOdigletPluginV1Server
	Impl PluginV1
}

var _ proto.OdigletPluginV1Server = (*GRPCServer)(nil)

func (s *GRPCServer) Attach(ctx context.Context, in *proto.AttachRequest) (*emptypb.Empty, error) {
	var (
		pid         int
		serviceName string
	)

	if in.ProcessId != nil {
		pid = int(*in.ProcessId)
	}

	if in.ServiceName != nil {
		serviceName = *in.ServiceName
	}

	settings := instrumentation.Settings{
		ServiceName:        serviceName,
		ResourceAttributes: ToAttributesSlice(in.ResourceAttributes),
		// TODO: handle InitialConfig
	}

	// call the actual implementation
	err := s.Impl.Attach(ctx, pid, settings)
	return &emptypb.Empty{}, err
}

// TODO: Implement ApplyConfig

func (s *GRPCServer) Detach(ctx context.Context, in *proto.DetachRequest) (*emptypb.Empty, error) {
	var pid int
	if in.ProcessId != nil {
		pid = int(*in.ProcessId)
	}

	err := s.Impl.Detach(ctx, pid)
	return &emptypb.Empty{}, err
}
