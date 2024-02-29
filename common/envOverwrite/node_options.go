package envOverwrite

import (
	"fmt"

	"github.com/keyval-dev/odigos/common"
)

type nodeOptions struct{}

func (n *nodeOptions) EnvName() string {
	return "NODE_OPTIONS"
}

func (n *nodeOptions) ValueFor(sdkType common.OtelSdkType) string {
	if sdkType == common.NativeOtelSdkType {
		return fmt.Sprintf("--require %s", "/var/odigos/nodejs/autoinstrumentation.js")
	}

	return fmt.Sprintf("--require %s", "/var/odigos/nodejs-ebpf/dist/loader.js")
}
