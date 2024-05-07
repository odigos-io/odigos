package testutil

import (
	"github.com/odigos-io/odigos/common/consts"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func SetOdigosInstrumentationEnabled[W client.Object](obj W) W {
	copy := obj.DeepCopyObject().(W)
	copy.SetLabels(map[string]string{consts.OdigosInstrumentationLabel: consts.InstrumentationEnabled})
	return copy
}

func SetOdigosInstrumentationDisabled[W client.Object](obj W) W {
	copy := obj.DeepCopyObject().(W)
	copy.SetLabels(map[string]string{consts.OdigosInstrumentationLabel: consts.InstrumentationDisabled})
	return copy
}

func DeleteOdigosInstrumentationLabel[W client.Object](obj W) W {
	copy := obj.DeepCopyObject().(W)
	delete(copy.GetLabels(), consts.OdigosInstrumentationLabel)
	return copy
}

func SetReportedNameAnnotation[W client.Object](obj W, reportedName string) W {
	copy := obj.DeepCopyObject().(W)
	copy.SetAnnotations(map[string]string{consts.OdigosReportedNameAnnotation: reportedName})
	return copy
}
