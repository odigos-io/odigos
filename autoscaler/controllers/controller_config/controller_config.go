package controllerconfig

import "k8s.io/apimachinery/pkg/util/version"

type ControllerConfig struct {
	K8sVersion *version.Version
}
