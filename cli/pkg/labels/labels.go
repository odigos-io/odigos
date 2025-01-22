package labels

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
)

var OdigosSystem = map[string]string{
	k8sconsts.OdigosSystemLabelKey: k8sconsts.OdigosSystemLabelValue,
}
