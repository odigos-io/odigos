package traces

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/k8sutils/pkg/scope"
)

// CalculateInferDbAttributesConfig returns config when any matching InferDbAttributes Action applies.
// Returns nil when no matching action is found.
func CalculateInferDbAttributesConfig(agentLevelActions *[]odigosv1.Action, language common.ProgrammingLanguage, pw k8sconsts.PodWorkload) *actions.InferDbAttributesConfig {
	for _, action := range *agentLevelActions {
		if action.Spec.InferDbAttributes == nil {
			continue
		}
		if scope.SourceScopeMatchesContainer(action.Spec.InferDbAttributes.Scopes, pw, language) {
			return &actions.InferDbAttributesConfig{}
		}
	}
	return nil
}
