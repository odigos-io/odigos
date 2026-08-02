package instrumentation

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/odigos-io/odigos-device-plugin/pkg/dpm"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

type plugin struct {
	v1beta1.UnimplementedDevicePluginServer
	devices []*v1beta1.Device
	stopCh  chan struct{}
}

func NewPlugin(initialSize int64) dpm.PluginInterface {
	devicesList := make([]*v1beta1.Device, initialSize)
	for i := int64(0); i < initialSize; i++ {
		devicesList[i] = &v1beta1.Device{
			ID:     uuid.New().String(),
			Health: v1beta1.Healthy,
		}
	}

	return &plugin{
		devices: devicesList,
		stopCh:  make(chan struct{}),
	}
}

func (p *plugin) GetDevicePluginOptions(ctx context.Context, empty *v1beta1.Empty) (*v1beta1.DevicePluginOptions, error) {
	return &v1beta1.DevicePluginOptions{
		PreStartRequired:                false,
		GetPreferredAllocationAvailable: false,
	}, nil
}

func (p *plugin) ListAndWatch(empty *v1beta1.Empty, server v1beta1.DevicePlugin_ListAndWatchServer) error {
	logger := commonlogger.LoggerCompat().With("subsystem", "deviceplugin")
	logger.Debug("ListAndWatch", "devices", p.devices)
	err := server.Send(&v1beta1.ListAndWatchResponse{
		Devices: p.devices,
	})

	if err != nil {
		logger.Error("Failed to send ListAndWatchResponse", "err", err)
	}

	<-p.stopCh
	server.Send(&v1beta1.ListAndWatchResponse{
		Devices: []*v1beta1.Device{},
	})
	return nil
}

func (p *plugin) Stop() error {
	logger := commonlogger.LoggerCompat().With("subsystem", "deviceplugin")
	logger.Info("Stopping Odigos Device Plugin ...")
	p.stopCh <- struct{}{}
	return nil
}

func (p *plugin) GetPreferredAllocation(ctx context.Context, request *v1beta1.PreferredAllocationRequest) (*v1beta1.PreferredAllocationResponse, error) {
	return &v1beta1.PreferredAllocationResponse{}, nil
}

func (p *plugin) Allocate(ctx context.Context, request *v1beta1.AllocateRequest) (*v1beta1.AllocateResponse, error) {
	res := &v1beta1.AllocateResponse{}

	logger := commonlogger.LoggerCompat().With("subsystem", "deviceplugin")
	for _, req := range request.ContainerRequests {
		if len(req.DevicesIds) != 1 {
			// instrumentation device amount must equal to 1.
			// this is what being set on the container resource request:
			// resources:
			//   instrumentation.odigos.io/generic: 1
			// so it must equal to 1 if coming from odigos.
			// larger amount can exhaust the limited device we allocated.
			logger.Error("got instrumentation device not equal to 1", "devices", req.DevicesIds)
			return nil, fmt.Errorf("got instrumentation device not equal to 1, devices: %v", req.DevicesIds)
		}

		containerAllocateResponse := &v1beta1.ContainerAllocateResponse{
			Mounts: []*v1beta1.Mount{
				{
					ContainerPath: OdigosAgentsDirectory,
					HostPath:      OdigosAgentsDirectory,
					ReadOnly:      true,
				},
			},
		}
		res.ContainerResponses = append(res.ContainerResponses, containerAllocateResponse)
	}

	return res, nil
}

func (p *plugin) PreStartContainer(ctx context.Context, request *v1beta1.PreStartContainerRequest) (*v1beta1.PreStartContainerResponse, error) {
	return &v1beta1.PreStartContainerResponse{}, nil
}
