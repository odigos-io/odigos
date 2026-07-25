package predicates

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

func TestIsRuleRelevantForAgentInjection_AllowConcurrentAgents(t *testing.T) {
	allow := true
	if !isRuleRelevantForAgentInjection(&odigosv1.InstrumentationRuleSpec{
		AllowConcurrentAgents: &allow,
	}) {
		t.Fatal("AllowConcurrentAgents-only rule must be relevant for agent injection")
	}

	if isRuleRelevantForAgentInjection(&odigosv1.InstrumentationRuleSpec{}) {
		t.Fatal("empty rule must not be relevant for agent injection")
	}
}
