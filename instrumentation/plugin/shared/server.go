package shared

import (
	"context"

	"github.com/odigos-io/odigos/instrumentation"
	proto "github.com/odigos-io/odigos/instrumentation/plugin/proto/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

type GRPCServer struct {
	proto.UnimplementedInstrumentationServer
	Impl Instrumentation
}

var _ proto.InstrumentationServer = (*GRPCServer)(nil)

func (s *GRPCServer) Start(ctx context.Context, in *proto.InstrumentationRequest) (*emptypb.Empty, error) {
	var (
		pid     int
		serviceName string
	)

	if in.ProcessId != nil {
		pid = int(*in.ProcessId)
	}

	if in.ServiceName != nil {
		serviceName = *in.ServiceName
	}

	settings := instrumentation.Settings{
		ServiceName: serviceName,
		ResourceAttributes: ToAttributesSlice(in.ResourceAttributes),
		// TODO: handle InitialConfig
	}

	// call the actual implementation
	err := s.Impl.Start(ctx, pid, settings)
	return &emptypb.Empty{}, err
}

// TODO: Implement ApplyConfig

func (s *GRPCServer) Close(ctx context.Context, in *proto.InstrumentationCloseRequest) (*emptypb.Empty, error) {
	var pid int
	if in.ProcessId != nil {
		pid = int(*in.ProcessId)
	}

	err := s.Impl.Close(ctx, pid)
	return &emptypb.Empty{}, err
}

