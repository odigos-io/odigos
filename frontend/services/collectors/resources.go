package services

import (
    corev1 "k8s.io/api/core/v1"

    "github.com/odigos-io/odigos/frontend/graph/model"
)

func buildResourceAmounts(list corev1.ResourceList) *model.ResourceAmounts {
    var have bool
    var cpuM, memMi int
    if q, ok := list[corev1.ResourceCPU]; ok {
        cpuM = int(q.MilliValue())
        have = true
    }
    if q, ok := list[corev1.ResourceMemory]; ok {
        memMi = int(q.Value() / (1024 * 1024))
        have = true
    }
    if !have {
        return nil
    }
    return &model.ResourceAmounts{CPUM: cpuM, MemoryMiB: memMi}
}


