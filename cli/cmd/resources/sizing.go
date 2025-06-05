package resources

import (
	"fmt"

	"github.com/odigos-io/odigos/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Default size_s resource values
var defaultResources = struct {
	RequestMemoryMiB int
	LimitMemoryMiB   int
	RequestCPUm      int
	LimitCPUm        int
}{
	RequestMemoryMiB: 64,
	LimitMemoryMiB:   512,
	RequestCPUm:      10,
	LimitCPUm:        500,
}

// GetDefaultResourceRequirements returns the size_s resource requirements.
// This is used as a fallback when a component's configuration is nil.
func GetDefaultResourceRequirements() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse(fmt.Sprintf("%dm", defaultResources.LimitCPUm)),
			"memory": resource.MustParse(fmt.Sprintf("%dMi", defaultResources.LimitMemoryMiB)),
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse(fmt.Sprintf("%dm", defaultResources.RequestCPUm)),
			"memory": resource.MustParse(fmt.Sprintf("%dMi", defaultResources.RequestMemoryMiB)),
		},
	}
}

// GetResourceRequirementsWithDefaults converts a ResourceConfig to ResourceRequirements,
// using size_s values for any fields that are 0.
func GetResourceRequirementsWithDefaults(rc common.ResourceConfig) corev1.ResourceRequirements {
	limitCPU := rc.LimitCPUm
	if limitCPU == 0 {
		limitCPU = defaultResources.LimitCPUm
	}

	limitMem := rc.LimitMemoryMiB
	if limitMem == 0 {
		limitMem = defaultResources.LimitMemoryMiB
	}

	reqCPU := rc.RequestCPUm
	if reqCPU == 0 {
		reqCPU = defaultResources.RequestCPUm
	}

	reqMem := rc.RequestMemoryMiB
	if reqMem == 0 {
		reqMem = defaultResources.RequestMemoryMiB
	}

	return corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse(fmt.Sprintf("%dm", limitCPU)),
			"memory": resource.MustParse(fmt.Sprintf("%dMi", limitMem)),
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse(fmt.Sprintf("%dm", reqCPU)),
			"memory": resource.MustParse(fmt.Sprintf("%dMi", reqMem)),
		},
	}
}
