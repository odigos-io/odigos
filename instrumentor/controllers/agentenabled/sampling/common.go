package sampling

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/scope"
)

// IsServiceInRuleScope returns true if the list is empty (match all) or any scope matches the given workload/container/language.
func IsServiceInRuleScope(services []scope.SourcesScope, pw k8sconsts.PodWorkload, containerName string, containerLanguage common.ProgrammingLanguage) bool {
	return scope.AnySourceScopeMatchesContainer(services, pw, containerName, containerLanguage)
}
