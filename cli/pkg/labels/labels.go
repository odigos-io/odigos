package labels

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
)

var OdigosSystem = map[string]string{
	k8sconsts.OdigosSystemLabelKey: k8sconsts.OdigosSystemLabelValue,
}

var OdigosCopiedImagePullSecret = map[string]string{
	k8sconsts.OdigosCopiedImagePullSecretLabel: k8sconsts.OdigosSystemLabelValue,
}
