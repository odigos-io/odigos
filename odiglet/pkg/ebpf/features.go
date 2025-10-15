package ebpf

import (
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/features"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
)

// Check if the current kernel supports the required features
func IsRingBufferSupported() bool {
	ringEn := false
	if features.HaveMapType(ebpf.RingBuf) == nil {
		ringEn = true
		log.Logger.V(0).Info("Kernel supports ring buffer")
	}
	return ringEn
}
