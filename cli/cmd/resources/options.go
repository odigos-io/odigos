package resources

import (
	"github.com/odigos-io/odigos/cli/pkg/autodetect"
	"github.com/odigos-io/odigos/common"
)

type OdigletDaemonSetOptions struct {
	Namespace string
	Version   string

	// Images
	ImagePrefix    string
	OdigletImage   string
	CollectorImage string

	// Deployment/runtime
	Tier                             common.OdigosTier
	OpenShiftEnabled                 bool
	ClusterDetails                   *autodetect.ClusterDetails
	CustomRuntimeSockPath            string
	NodeSelector                     map[string]string
	HealthProbeBindPort              int
	MountMethod                      *common.MountMethod
	CustomContainerRuntimeSocketPath string
	NodeCollectorSizing              common.CollectorNodeConfiguration

	SignalsEnabled map[common.ObservabilitySignal]bool // nil => default all ON
}
