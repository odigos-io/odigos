package custom

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func ShouldApplyCustomDataCollection(dests *odigosv1.DestinationList) bool {
	for _, dst := range dests.Items {
		if destRequiresCustomDataCollection(dst) {
			return true
		}
	}
	return false
}

func destRequiresCustomDataCollection(dest odigosv1.Destination) bool {
	if DestRequiresCustom(dest.Spec.Type) {
		for _, s := range dest.Spec.Signals {
			if s != common.TracesObservabilitySignal {
				return true
			}
		}
	}

	return false
}

func DestRequiresCustom(destType common.DestinationType) bool {
	switch destType {
	case common.HoneycombDestinationType:
		return true
	default:
		return false
	}
}

func AddCustomConfigMap(dests *odigosv1.DestinationList, cm *corev1.ConfigMap) {
	for _, dst := range dests.Items {
		if dst.Spec.Type == common.HoneycombDestinationType {
			addHoneycombConfig(cm, dst)
			return
		}
	}
}

func ApplyCustomChangesToDaemonSet(ds *v1.DaemonSet, dests *odigosv1.DestinationList) {
	secretName := ""
	for _, dst := range dests.Items {
		if dst.Spec.Type == common.HoneycombDestinationType {
			secretName = dst.Spec.SecretRef.Name
			break
		}
	}
	addHoneycombToDaemonSet(ds, secretName)
}
