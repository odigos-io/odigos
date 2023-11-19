package instrumentation_ebpf

import (
	"github.com/keyval-dev/odigos/common/consts"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func hasEbpfInstrumentationAnnotation(obj client.Object) bool {
	if obj == nil {
		return false
	}

	annotations := obj.GetAnnotations()
	if annotations == nil {
		return false
	}

	_, exists := annotations[consts.EbpfInstrumentationAnnotation]
	return exists
}
