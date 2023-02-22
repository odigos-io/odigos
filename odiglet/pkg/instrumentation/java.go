package instrumentation

import (
	"context"
	"fmt"
	"github.com/keyval-dev/odigos/odiglet/pkg/env"
	"github.com/keyval-dev/odigos/odiglet/pkg/instrumentation/consts"
	"github.com/keyval-dev/odigos/odiglet/pkg/instrumentation/devices"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	"github.com/kubevirt/device-plugin-manager/pkg/dpm"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	otelResourceAttributesEnvVar = "OTEL_RESOURCE_ATTRIBUTES"
	otelResourceAttrPatteern     = "service.name=%s,odigos.device=java"
	javaToolOptionsEnvVar        = "JAVA_TOOL_OPTIONS"
	javaOptsEnvVar               = "JAVA_OPTS"
	javaToolOptionsPattern       = "-javaagent:/odigos/javaagent.jar " +
		"-Dotel.traces.sampler=always_on -Dotel.exporter.otlp.endpoint=http://%s:%d"
)

type plugin struct {
	idsManager devices.DeviceManager
	stopCh     chan struct{}
}

func NewJavaPlugin(idManager devices.DeviceManager) dpm.PluginInterface {
	return &plugin{
		idsManager: idManager,
		stopCh:     make(chan struct{}),
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
	devicesList := p.idsManager.GetDevices()
	log.Logger.V(0).Info("ListAndWatch", "devices", devicesList)
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
		javaOpts := fmt.Sprintf(javaToolOptionsPattern, env.Current.NodeIP, consts.OTLPPort)
		res.ContainerResponses = append(res.ContainerResponses, &v1beta1.ContainerAllocateResponse{
			Envs: map[string]string{
				otelResourceAttributesEnvVar: fmt.Sprintf(otelResourceAttrPatteern, deviceId),
				javaToolOptionsEnvVar:        javaOpts,
				javaOptsEnvVar:               javaOpts,
			},
			Mounts: []*v1beta1.Mount{
				{
					ContainerPath: "/odigos",
					HostPath:      "/odigos",
					ReadOnly:      true,
				},
			},
		})
	}

	return res, nil
}

func (p *plugin) PreStartContainer(ctx context.Context, request *v1beta1.PreStartContainerRequest) (*v1beta1.PreStartContainerResponse, error) {
	return &v1beta1.PreStartContainerResponse{}, nil
}
