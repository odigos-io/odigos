package sampling

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
)

func IsServiceInRuleScope(services []odigosv1.SourcesScope, pw k8sconsts.PodWorkload, containerName string, containerLanguage common.ProgrammingLanguage) bool {
	if len(services) == 0 {
		// empty list means all services are matched
		return true
	}
	for _, service := range services {
		// check any service field if set
		if service.WorkloadName != "" && service.WorkloadName != pw.Name {
			continue
		}
		if service.WorkloadKind != "" && service.WorkloadKind != pw.Kind {
			continue
		}
		if service.WorkloadNamespace != "" && service.WorkloadNamespace != pw.Namespace {
			continue
		}
		if service.ContainerName != "" && service.ContainerName != containerName {
			continue
		}
		if service.WorkloadLanguage != "" && service.WorkloadLanguage != containerLanguage {
			continue
		}
		// if all fields matched, this source container matches the services filters in the rule and should be taken
		return true
	}
	return false
}
