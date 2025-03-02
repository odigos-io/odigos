package controllerconfig

import "k8s.io/apimachinery/pkg/util/version"

type ControllerConfig struct {
	// TODO: this should be removed once the hpa logic uses the feature package for its checks
	K8sVersion     *version.Version
	CollectorImage string
}
