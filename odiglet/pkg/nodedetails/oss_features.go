package nodedetails

import (
	"context"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

// KernelVersionFeature extracts the kernel version from the Kubernetes Node object.
type KernelVersionFeature struct{}

func (k *KernelVersionFeature) Name() string {
	return "KernelVersion"
}

func (k *KernelVersionFeature) Check(ctx context.Context, node *v1.Node, spec *v1alpha1.NodeDetailsSpec) error {
	spec.KernelVersion = node.Status.NodeInfo.KernelVersion
	return nil
}

// CPUCapacityFeature extracts the CPU capacity from the Kubernetes Node object.
type CPUCapacityFeature struct{}

func (c *CPUCapacityFeature) Name() string {
	return "CPUCapacity"
}

func (c *CPUCapacityFeature) Check(ctx context.Context, node *v1.Node, spec *v1alpha1.NodeDetailsSpec) error {
	if cpu, ok := node.Status.Capacity[v1.ResourceCPU]; ok {
		spec.CPUCapacity = int(cpu.Value()) // in cores
	}
	return nil
}

// MemoryCapacityFeature extracts the memory capacity from the Kubernetes Node object.
type MemoryCapacityFeature struct{}

func (m *MemoryCapacityFeature) Name() string {
	return "MemoryCapacity"
}

func (m *MemoryCapacityFeature) Check(ctx context.Context, node *v1.Node, spec *v1alpha1.NodeDetailsSpec) error {
	if memory, ok := node.Status.Capacity[v1.ResourceMemory]; ok {
		// Convert from bytes to megabytes (MB)
		spec.MemoryCapacity = int(memory.Value() / (1024 * 1024))
	}
	return nil
}
