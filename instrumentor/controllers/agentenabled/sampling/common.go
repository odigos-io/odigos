package sampling

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	commonapi "github.com/odigos-io/odigos/common/api"
	"github.com/odigos-io/odigos/common"
)

// IsServiceInRuleScope returns true if the list is empty (match all) or any scope matches the given workload/container/language.
func IsServiceInRuleScope(services []commonapi.SourcesScope, pw k8sconsts.PodWorkload, containerName string, containerLanguage common.ProgrammingLanguage) bool {
	ref := commonapi.WorkloadRef{Name: pw.Name, Namespace: pw.Namespace, Kind: string(pw.Kind)}
	return commonapi.AnySourceScopeMatchesContainer(services, ref, containerName, containerLanguage)
}
