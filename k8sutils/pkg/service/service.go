package service

import (
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

func SameNodeOTLPDataCollectionEndpoint() string {
	return fmt.Sprintf("%s.%s:%d", k8sconsts.OdigosNodeCollectorSameNodeServiceName, env.GetCurrentNamespace(), consts.OTLPPort)
}

func SameNodeOTLPHttpDataCollectionEndpoint() string {
	return fmt.Sprintf("http://%s.%s:%d", k8sconsts.OdigosNodeCollectorSameNodeServiceName, env.GetCurrentNamespace(), consts.OTLPHttpPort)
}