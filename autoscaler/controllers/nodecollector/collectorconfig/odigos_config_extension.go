package collectorconfig

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/consts"
)

// NodeHasURLTemplateProcessor reports whether processors (node-collector role only) include URL templatization.
// Merge NodeOdigosExtDomain when true; gateway still has its own extension from clustercollector.
func NodeHasURLTemplateProcessor(processors []*odigosv1.Processor) bool {
	for _, p := range processors {
		if p != nil && p.Spec.Type == consts.OdigosURLTemplateProcessorType {
			return true
		}
	}
	return false
}

// NodeOdigosExtDomain returns the config map domain for odigos_config_k8s (extensions + service.extensions).
func NodeOdigosExtDomain() config.Config {
	return config.Config{
		Extensions: config.GenericMap{
			k8sconsts.OdigosConfigK8sExtensionType: config.GenericMap{},
		},
		Service: config.Service{
			Extensions: []string{k8sconsts.OdigosConfigK8sExtensionType},
		},
	}
}
