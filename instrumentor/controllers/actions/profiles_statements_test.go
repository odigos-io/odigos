package actions

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// profilesCapableContexts are the only OTTL contexts the transformprocessor accepts for the
// profiles signal in the pinned collector build; emitting any other context fails collector startup.
var profilesCapableContexts = []string{"resource", "scope", "profile"}

func contexts(stmts []OttlStatementConfig) []string {
	out := make([]string, len(stmts))
	for i, s := range stmts {
		out[i] = s.Context
	}
	return out
}

func TestRenameAttributeConfig_Profiles(t *testing.T) {
	cfg, err := renameAttributeConfig(
		map[string]string{"old.attr": "new.attr"},
		[]common.ObservabilitySignal{common.ProfilesObservabilitySignal},
	)
	require.NoError(t, err)

	// Only profile_statements is populated for a PROFILES-only action.
	assert.Empty(t, cfg.TraceStatements)
	assert.Empty(t, cfg.MetricStatements)
	assert.Empty(t, cfg.LogStatements)
	require.NotEmpty(t, cfg.ProfileStatements)

	assert.Equal(t, profilesCapableContexts, contexts(cfg.ProfileStatements))
	// Each context carries the same set/delete_key statement pair the other signals use.
	for _, s := range cfg.ProfileStatements {
		assert.Equal(t, []string{
			`set(attributes["new.attr"], attributes["old.attr"])`,
			`delete_key(attributes, "old.attr")`,
		}, s.Statements)
	}
}

func TestDeleteAttributeConfig_Profiles(t *testing.T) {
	cfg, err := deleteAttributeConfig(
		[]string{"drop.me"},
		[]common.ObservabilitySignal{common.ProfilesObservabilitySignal},
	)
	require.NoError(t, err)

	assert.Empty(t, cfg.TraceStatements)
	assert.Empty(t, cfg.MetricStatements)
	assert.Empty(t, cfg.LogStatements)
	require.NotEmpty(t, cfg.ProfileStatements)

	assert.Equal(t, profilesCapableContexts, contexts(cfg.ProfileStatements))
	for _, s := range cfg.ProfileStatements {
		assert.Equal(t, []string{`delete_key(attributes, "drop.me")`}, s.Statements)
	}
}

// Profiles combines cleanly with the existing signals without cross-populating the other buckets.
func TestRenameAttributeConfig_ProfilesWithOtherSignals(t *testing.T) {
	cfg, err := renameAttributeConfig(
		map[string]string{"a": "b"},
		[]common.ObservabilitySignal{common.TracesObservabilitySignal, common.ProfilesObservabilitySignal},
	)
	require.NoError(t, err)
	require.NotEmpty(t, cfg.TraceStatements)
	require.NotEmpty(t, cfg.ProfileStatements)
	assert.Equal(t, profilesCapableContexts, contexts(cfg.ProfileStatements))
}
