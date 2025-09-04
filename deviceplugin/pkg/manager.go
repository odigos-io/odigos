package pkg

// DevicePlugin is the struct representing your plugin runner
type DevicePlugin struct {
}

// New creates a new DevicePlugin instance
func New() *DevicePlugin {
	return &DevicePlugin{}
}

// Run starts the device plugin manager
func (d *DevicePlugin) Run() error {
	return runDeviceManager()
}
