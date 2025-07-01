package shared

import (
	"context"
	"errors"
	"math"

	"github.com/odigos-io/odigos/instrumentation"
	proto "github.com/odigos-io/odigos/instrumentation/plugin/proto/v1"
)

type GRPCClient struct { 
	client proto.InstrumentationClient
}

var _ Instrumentation = (*GRPCClient)(nil)

func (c *GRPCClient) Start(ctx context.Context, pid int, settings instrumentation.Settings) error {
	in := &proto.InstrumentationRequest{}
	if pid != 0 {
		if pid > math.MaxInt32 {
			return errors.New("pid is too large")
		}
		pid := int32(pid)
		in.ProcessId = &pid
	}

	if settings.ServiceName != "" {
		in.ServiceName = &settings.ServiceName
	}

	in.ResourceAttributes = KeyValues(settings.ResourceAttributes)

	// do the RPC call
	_, err := c.client.Start(ctx, in)
	if err != nil {
		return err
	}

	return nil
}

func (c *GRPCClient) ApplyConfig(ctx context.Context, pid int, config instrumentation.Config) error {
	return errors.New("not implemented")
}

func (c *GRPCClient) Close(ctx context.Context, pid int) error {
	in := &proto.InstrumentationCloseRequest{}
	if pid != 0 {
		if pid > math.MaxInt32 {
			return errors.New("pid is too large")
		}
		pid := int32(pid)
		in.ProcessId = &pid
	}

	_, err := c.client.Close(ctx, in)
	if err != nil {
		return err
	}

	return nil
}
