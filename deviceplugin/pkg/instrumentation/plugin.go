package instrumentation

import (
	"context"

	"github.com/odigos-io/odigos-device-plugin/pkg/dpm"
	"github.com/odigos-io/odigos/procdiscovery/pkg/libc"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/deviceplugin/pkg/instrumentation/devices"
	"github.com/odigos-io/odigos/deviceplugin/pkg/log"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

type LangSpecificFunc func(deviceId string) *v1beta1.ContainerAllocateResponse

type plugin struct {
	idsManager       devices.DeviceManager
	stopCh           chan struct{}
	LangSpecificFunc LangSpecificFunc
}

func NewPlugin(initialSize int64, lsf LangSpecificFunc) dpm.PluginInterface {
	idManager := devices.NewIDManager(initialSize)

	return &plugin{
		idsManager:       idManager,
		stopCh:           make(chan struct{}),
		LangSpecificFunc: lsf,
	}
}

func NewMuslPlugin(lang common.ProgrammingLanguage, maxPods int64, lsf LangSpecificFunc) dpm.PluginInterface {
	wrappedLsf := func(deviceId string) *v1beta1.ContainerAllocateResponse {
		res := lsf(deviceId)
		libc.ModifyEnvVarsForMusl(lang, res.Envs)
		return res
	}

	return NewPlugin(maxPods, wrappedLsf)
}

func (p *plugin) GetDevicePluginOptions(ctx context.Context, empty *v1beta1.Empty) (*v1beta1.DevicePluginOptions, error) {
	return &v1beta1.DevicePluginOptions{
		PreStartRequired:                false,
		GetPreferredAllocationAvailable: false,
	}, nil
}

func (p *plugin) ListAndWatch(empty *v1beta1.Empty, server v1beta1.DevicePlugin_ListAndWatchServer) error {
	devicesList := p.idsManager.GetDevices()
	log.Logger.V(3).Info("ListAndWatch", "devices", devicesList)
	err := server.Send(&v1beta1.ListAndWatchResponse{
		Devices: devicesList,
	})

	if err != nil {
		log.Logger.Error(err, "Failed to send ListAndWatchResponse")
	}

	<-p.stopCh
	server.Send(&v1beta1.ListAndWatchResponse{
		Devices: []*v1beta1.Device{},
	})
	return nil
}

func (p *plugin) Stop() error {
	log.Logger.V(0).Info("Stopping Odigos Device Plugin ...")
	p.stopCh <- struct{}{}
	return nil
}

func (p *plugin) GetPreferredAllocation(ctx context.Context, request *v1beta1.PreferredAllocationRequest) (*v1beta1.PreferredAllocationResponse, error) {
	return &v1beta1.PreferredAllocationResponse{}, nil
}

func (p *plugin) Allocate(ctx context.Context, request *v1beta1.AllocateRequest) (*v1beta1.AllocateResponse, error) {
	res := &v1beta1.AllocateResponse{}

	for _, req := range request.ContainerRequests {
		if len(req.DevicesIDs) != 1 {
			log.Logger.V(0).Info("got  instrumentation device not equal to 1, skipping", "devices", req.DevicesIDs)
			continue
		}

		deviceId := req.DevicesIDs[0]
		res.ContainerResponses = append(res.ContainerResponses, p.LangSpecificFunc(deviceId))
	}

	return res, nil
}

func (p *plugin) PreStartContainer(ctx context.Context, request *v1beta1.PreStartContainerRequest) (*v1beta1.PreStartContainerResponse, error) {
	return &v1beta1.PreStartContainerResponse{}, nil
}
