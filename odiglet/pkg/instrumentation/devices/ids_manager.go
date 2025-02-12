package devices

import (
	"github.com/google/uuid"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

type DeviceManager interface {
	Init(initialDevices int64)
	GetDevices() []*v1beta1.Device
}

type IDManager struct {
	devices []string
}

func NewIDManager(initialSize int64) *IDManager {
	m := &IDManager{
		devices: make([]string, initialSize),
	}

	m.Init(initialSize)
	return m
}

func (m *IDManager) Init(initialDevices int64) {
	for i := int64(0); i < initialDevices; i++ {
		m.devices[i] = uuid.New().String()
	}
}

func (m *IDManager) GetDevices() []*v1beta1.Device {
	var devicesList []*v1beta1.Device
	for _, deviceId := range m.devices {
		devicesList = append(devicesList, &v1beta1.Device{
			ID:     deviceId,
			Health: v1beta1.Healthy,
		})
	}

	return devicesList
}
