package common

// Note: this configuration is currently only relevant for k8s,
// but is used in odigosConfiguration which is declared in the common package.
// We should revisit this decision later on and consider if the config should be k8s specific,
// then move it to the api module.

// +kubebuilder:validation:Enum=k8s-virtual-device;k8s-host-path
type MountMethod string

const (
	K8sVirtualDeviceMountMethod MountMethod = "k8s-virtual-device"
	K8sHostPathMountMethod      MountMethod = "k8s-host-path"
)
