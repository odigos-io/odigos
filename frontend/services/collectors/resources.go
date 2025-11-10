package collectors

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/odigos-io/odigos/frontend/graph/model"
)

func buildResourceAmounts(list corev1.ResourceList) *model.ResourceAmounts {
	var cpuStr, memStr *string
	if cpu, ok := list[corev1.ResourceCPU]; ok {
		s := cpu.String()
		cpuStr = &s
	}
	if memory, ok := list[corev1.ResourceMemory]; ok {
		s := memory.String()
		memStr = &s
	}

	return &model.ResourceAmounts{CPU: cpuStr, Memory: memStr}
}
