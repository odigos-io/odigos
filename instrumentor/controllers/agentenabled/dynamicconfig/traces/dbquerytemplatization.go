package traces

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/k8sutils/pkg/scope"
)

// CalculateDbQueryTemplatizationConfig merges matching DbQueryTemplatization Actions for a container.
// TemplatizeLiterals is OR'd across matching actions. Returns nil when no matching action enables it.
func CalculateDbQueryTemplatizationConfig(agentLevelActions *[]odigosv1.Action, language common.ProgrammingLanguage, pw k8sconsts.PodWorkload) *actions.DbQueryTemplatizationConfig {
	templatizeLiterals := false

	for _, action := range *agentLevelActions {
		if action.Spec.DbQueryTemplatization == nil {
			continue
		}
		if scope.SourceScopeMatchesContainer(action.Spec.DbQueryTemplatization.Scopes, pw, language) {
			templatizeLiterals = templatizeLiterals || action.Spec.DbQueryTemplatization.TemplatizeLiterals
		}
	}

	if !templatizeLiterals {
		return nil
	}

	return &actions.DbQueryTemplatizationConfig{
		TemplatizeLiterals: true,
	}
}
