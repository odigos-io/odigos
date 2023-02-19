package devices

import (
	"github.com/google/uuid"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

type DeviceManager interface {
	Init(initialDevices int)
	Evacuate(deviceId string)
	Channel() chan []*v1beta1.Device
}

type IDManager struct {
	devices     map[string]struct{}
	devicesChan chan []*v1beta1.Device
}

func NewIDManager() *IDManager {
	m := &IDManager{}
	m.Init(getInitialDeviceAmount())
	return m
}

func (m *IDManager) Init(initialDevices int) {
	m.devices = make(map[string]struct{})
	m.devicesChan = make(chan []*v1beta1.Device, initialDevices)

	for i := 0; i < initialDevices; i++ {
		m.devices[uuid.New().String()] = struct{}{}
	}

	m.updateDevices()
}

func (m *IDManager) Evacuate(deviceId string) {
	delete(m.devices, deviceId)
	m.devices[uuid.New().String()] = struct{}{}
	m.updateDevices()
}

func (m *IDManager) Channel() chan []*v1beta1.Device {
	return m.devicesChan
}

func (m *IDManager) updateDevices() {
	var devicesList []*v1beta1.Device
	for deviceId := range m.devices {
		devicesList = append(devicesList, &v1beta1.Device{
			ID:     deviceId,
			Health: v1beta1.Healthy,
		})
	}

	m.devicesChan <- devicesList
}

func getInitialDeviceAmount() int {
	return 10
}
