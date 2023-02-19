package instrumentation

import (
	"context"
	"fmt"
	"github.com/keyval-dev/odigos/odiglet/pkg/instrumentation/devices"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	"github.com/kubevirt/device-plugin-manager/pkg/dpm"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

type plugin struct {
	idsManager *devices.IDManager
}

func NewInstrumentationPlugin() dpm.PluginInterface {
	return &plugin{
		idsManager: devices.NewIDManager(),
	}
}

func (p *plugin) GetDevicePluginOptions(ctx context.Context, empty *v1beta1.Empty) (*v1beta1.DevicePluginOptions, error) {
	fmt.Println("GetDevicePluginOptions")
	return &v1beta1.DevicePluginOptions{
		PreStartRequired:                false,
		GetPreferredAllocationAvailable: false,
	}, nil
}

func (p *plugin) ListAndWatch(empty *v1beta1.Empty, server v1beta1.DevicePlugin_ListAndWatchServer) error {
	for newDevices := range p.idsManager.Channel() {
		log.Logger.V(0).Info("ListAndWatch", "newDevices", newDevices)
		err := server.Send(&v1beta1.ListAndWatchResponse{
			Devices: newDevices,
		})

		if err != nil {
			log.Logger.Error(err, "Failed to send ListAndWatchResponse")
		}
	}

	return nil
}

func (p *plugin) Stop() error {
	log.Logger.V(0).Info("Stopping Odigos Device Plugin ...")
	return nil
}

func (p *plugin) GetPreferredAllocation(ctx context.Context, request *v1beta1.PreferredAllocationRequest) (*v1beta1.PreferredAllocationResponse, error) {
	log.Logger.V(0).Info("GetPreferredAllocation request: %v", request)
	return &v1beta1.PreferredAllocationResponse{}, nil
}

func (p *plugin) Allocate(ctx context.Context, request *v1beta1.AllocateRequest) (*v1beta1.AllocateResponse, error) {
	log.Logger.V(0).Info("Allocate request: %v", request)

	err := getPodResources()
	if err != nil {
		return nil, err
	}

	res := &v1beta1.AllocateResponse{}

	for range request.ContainerRequests {
		res.ContainerResponses = append(res.ContainerResponses, &v1beta1.ContainerAllocateResponse{
			Envs: map[string]string{
				"ODIGOS_INSTRUMENTATION": "true",
				"EDEN_TEST":              "${HOSTNAME}",
			},
			Mounts: []*v1beta1.Mount{
				{
					ContainerPath: "/odigos/EDEN_MOUNT",
					HostPath:      "/odigos/EDEN_MOUNT",
				},
			},
		})
	}

	return res, nil
}

func (p *plugin) PreStartContainer(ctx context.Context, request *v1beta1.PreStartContainerRequest) (*v1beta1.PreStartContainerResponse, error) {
	log.Logger.V(0).Info("PreStartContainer request: %v", request)
	return &v1beta1.PreStartContainerResponse{}, nil
}
