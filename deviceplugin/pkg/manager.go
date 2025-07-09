package pkg

// DevicePlugin is the struct representing your plugin runner
type DevicePlugin struct {
	opts Options
}

// New creates a new DevicePlugin instance
func New(opts Options) *DevicePlugin {
	return &DevicePlugin{opts: opts}
}

// Run starts the device plugin manager
func (d *DevicePlugin) Run() error {
	return runDeviceManager(d.opts)
}
