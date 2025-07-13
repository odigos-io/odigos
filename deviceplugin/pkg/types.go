package pkg

import "github.com/odigos-io/odigos/deviceplugin/pkg/instrumentation"

// Options defines inputs for the DevicePlugin
type Options struct {
	DeviceInjectionCallbacks instrumentation.OtelSdksLsf
}
