package actions

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/odigos-io/odigos/common"
)

// BuildDeleteAttributeProcessorConfig and BuildRenameAttributeProcessorConfig moved here from the
// instrumentor in #6162. profiles_statements_test.go covers the PROFILES signal; these tests cover
// the other three arms of the same switch, the signal-validation error paths, and the two places
// where the two builders deliberately differ.

// tfbContextsBySignal is the set of OTTL contexts the transformprocessor accepts for each signal
// in the pinned collector build. Emitting a context outside this set fails collector startup, so
// the values are spelled out rather than read back from the builder under test.
var tfbContextsBySignal = map[common.ObservabilitySignal][]string{
	common.LogsObservabilitySignal:     {"resource", "scope", "log"},
	common.MetricsObservabilitySignal:  {"resource", "scope", "datapoint"},
	common.TracesObservabilitySignal:   {"resource", "scope", "span", "spanevent"},
	common.ProfilesObservabilitySignal: {"resource", "scope", "profile"},
}

// tfbBucket returns the statements the builder put in the bucket belonging to signal.
func tfbBucket(cfg TransformProcessorConfig, signal common.ObservabilitySignal) []OttlStatementConfig {
	switch signal {
	case common.LogsObservabilitySignal:
		return cfg.LogStatements
	case common.MetricsObservabilitySignal:
		return cfg.MetricStatements
	case common.TracesObservabilitySignal:
		return cfg.TraceStatements
	case common.ProfilesObservabilitySignal:
		return cfg.ProfileStatements
	}
	return nil
}

func tfbAllSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{
		common.LogsObservabilitySignal,
		common.MetricsObservabilitySignal,
		common.TracesObservabilitySignal,
		common.ProfilesObservabilitySignal,
	}
}

// tfbPopulatedBuckets names every bucket that carries statements, so a builder writing a signal's
// statements into the wrong bucket is visible even when the contexts themselves look right.
func tfbPopulatedBuckets(cfg TransformProcessorConfig) []string {
	var populated []string
	for _, signal := range tfbAllSignals() {
		if len(tfbBucket(cfg, signal)) > 0 {
			populated = append(populated, string(signal))
		}
	}
	return populated
}

func TestBuildDeleteAttributeProcessorConfig_fillsOnlyTheRequestedSignalsBucket(t *testing.T) {
	for signal, wantContexts := range tfbContextsBySignal {
		t.Run(string(signal), func(t *testing.T) {
			cfg, err := BuildDeleteAttributeProcessorConfig(
				[]string{"secret.token"},
				[]common.ObservabilitySignal{signal},
			)
			require.NoError(t, err)

			assert.Equal(t, []string{string(signal)}, tfbPopulatedBuckets(cfg))

			bucket := tfbBucket(cfg, signal)
			require.Len(t, bucket, len(wantContexts))
			for i, stmt := range bucket {
				assert.Equal(t, wantContexts[i], stmt.Context)
				assert.Equal(t, []string{`delete_key(attributes, "secret.token")`}, stmt.Statements)
			}
		})
	}
}

func TestBuildRenameAttributeProcessorConfig_fillsOnlyTheRequestedSignalsBucket(t *testing.T) {
	for signal, wantContexts := range tfbContextsBySignal {
		t.Run(string(signal), func(t *testing.T) {
			cfg, err := BuildRenameAttributeProcessorConfig(
				map[string]string{"old.attr": "new.attr"},
				[]common.ObservabilitySignal{signal},
			)
			require.NoError(t, err)

			assert.Equal(t, []string{string(signal)}, tfbPopulatedBuckets(cfg))

			bucket := tfbBucket(cfg, signal)
			require.Len(t, bucket, len(wantContexts))
			for i, stmt := range bucket {
				assert.Equal(t, wantContexts[i], stmt.Context)
				assert.Equal(t, []string{
					`set(attributes["new.attr"], attributes["old.attr"])`,
					`delete_key(attributes, "old.attr")`,
				}, stmt.Statements)
			}
		})
	}
}

// An action may select several signals at once; each bucket must then be filled independently
// and with its own context list.
func TestBuildDeleteAttributeProcessorConfig_fillsEveryRequestedSignal(t *testing.T) {
	cfg, err := BuildDeleteAttributeProcessorConfig([]string{"a"}, tfbAllSignals())
	require.NoError(t, err)

	assert.Equal(t, []string{"LOGS", "METRICS", "TRACES", "PROFILES"}, tfbPopulatedBuckets(cfg))
	for signal, wantContexts := range tfbContextsBySignal {
		got := make([]string, 0, len(tfbBucket(cfg, signal)))
		for _, stmt := range tfbBucket(cfg, signal) {
			got = append(got, stmt.Context)
		}
		assert.Equal(t, wantContexts, got, "contexts for %s", signal)
	}
}

func TestBuildRenameAttributeProcessorConfig_fillsEveryRequestedSignal(t *testing.T) {
	cfg, err := BuildRenameAttributeProcessorConfig(map[string]string{"a": "b"}, tfbAllSignals())
	require.NoError(t, err)

	assert.Equal(t, []string{"LOGS", "METRICS", "TRACES", "PROFILES"}, tfbPopulatedBuckets(cfg))
	for signal, wantContexts := range tfbContextsBySignal {
		got := make([]string, 0, len(tfbBucket(cfg, signal)))
		for _, stmt := range tfbBucket(cfg, signal) {
			got = append(got, stmt.Context)
		}
		assert.Equal(t, wantContexts, got, "contexts for %s", signal)
	}
}

// Deleting attributes is security sensitive, so its processor propagates OTTL errors instead of
// swallowing them, while renaming ignores them. The two values are spelled out because flipping
// either one is invisible to every other assertion in this package.
func TestTransformBuildersUseTheirOwnErrorMode(t *testing.T) {
	deleteCfg, err := BuildDeleteAttributeProcessorConfig([]string{"a"}, tfbAllSignals())
	require.NoError(t, err)
	assert.Equal(t, "propagate", deleteCfg.ErrorMode)

	renameCfg, err := BuildRenameAttributeProcessorConfig(map[string]string{"a": "b"}, tfbAllSignals())
	require.NoError(t, err)
	assert.Equal(t, "ignore", renameCfg.ErrorMode)
}

func TestTransformBuildersRejectNilSignals(t *testing.T) {
	deleteCfg, err := BuildDeleteAttributeProcessorConfig([]string{"a"}, nil)
	require.EqualError(t, err, "Signals must be set")
	// The delete builder returns the config it had already started, unlike the rename builder
	// below; callers of either must branch on the error, not on the config being empty.
	assert.Equal(t, TransformProcessorConfig{ErrorMode: "propagate"}, deleteCfg)

	renameCfg, err := BuildRenameAttributeProcessorConfig(map[string]string{"a": "b"}, nil)
	require.EqualError(t, err, "Signals must be set")
	assert.Equal(t, TransformProcessorConfig{}, renameCfg)
}

// An empty signal list is not the same as a nil one: it passes validation and yields a processor
// with no statements at all, so the action is accepted and then does nothing. The Action CRD puts
// no minItems on spec.signals, and an empty JSON array decodes to a non-nil slice, so this is
// reachable from a manifest rather than only from Go.
func TestTransformBuildersAcceptEmptySignalsAndProduceANoOpProcessor(t *testing.T) {
	var decoded []common.ObservabilitySignal
	require.NoError(t, json.Unmarshal([]byte(`[]`), &decoded))
	require.NotNil(t, decoded, "an empty signals array must decode to a non-nil slice for this to be reachable")
	require.Empty(t, decoded)

	deleteCfg, err := BuildDeleteAttributeProcessorConfig([]string{"secret.token"}, decoded)
	require.NoError(t, err)
	assert.Equal(t, TransformProcessorConfig{ErrorMode: "propagate"}, deleteCfg)
	assert.Empty(t, tfbPopulatedBuckets(deleteCfg))

	renameCfg, err := BuildRenameAttributeProcessorConfig(map[string]string{"a": "b"}, decoded)
	require.NoError(t, err)
	assert.Equal(t, TransformProcessorConfig{ErrorMode: "ignore"}, renameCfg)
	assert.Empty(t, tfbPopulatedBuckets(renameCfg))
}

// A signal outside the four known ones is skipped silently rather than rejected, and does not
// disturb the buckets of the signals that are recognised.
func TestTransformBuildersSkipUnknownSignals(t *testing.T) {
	signals := []common.ObservabilitySignal{
		common.ObservabilitySignal("EVENTS"),
		common.TracesObservabilitySignal,
	}

	deleteCfg, err := BuildDeleteAttributeProcessorConfig([]string{"a"}, signals)
	require.NoError(t, err)
	assert.Equal(t, []string{"TRACES"}, tfbPopulatedBuckets(deleteCfg))

	renameCfg, err := BuildRenameAttributeProcessorConfig(map[string]string{"a": "b"}, signals)
	require.NoError(t, err)
	assert.Equal(t, []string{"TRACES"}, tfbPopulatedBuckets(renameCfg))
}

// Attribute names reach OTTL inside a quoted literal, so a name carrying a quote or a backslash
// must come back escaped; emitting it raw would end the literal and break collector startup.
func TestTransformBuildersQuoteAttributeNamesForOttl(t *testing.T) {
	deleteCfg, err := BuildDeleteAttributeProcessorConfig(
		[]string{`weird"name`, `back\slash`},
		[]common.ObservabilitySignal{common.TracesObservabilitySignal},
	)
	require.NoError(t, err)
	require.NotEmpty(t, deleteCfg.TraceStatements)
	assert.Equal(t, []string{
		`delete_key(attributes, "weird\"name")`,
		`delete_key(attributes, "back\\slash")`,
	}, deleteCfg.TraceStatements[0].Statements)

	renameCfg, err := BuildRenameAttributeProcessorConfig(
		map[string]string{`weird"name`: `back\slash`},
		[]common.ObservabilitySignal{common.TracesObservabilitySignal},
	)
	require.NoError(t, err)
	require.NotEmpty(t, renameCfg.TraceStatements)
	assert.Equal(t, []string{
		`set(attributes["back\\slash"], attributes["weird\"name"])`,
		`delete_key(attributes, "weird\"name")`,
	}, renameCfg.TraceStatements[0].Statements)
}

// Every context of a signal shares one statement list, in input order for delete and in sorted
// key order for rename; a builder that emitted only the first attribute would still produce a
// well-formed processor.
func TestBuildDeleteAttributeProcessorConfig_deletesEveryAttributeInEveryContext(t *testing.T) {
	cfg, err := BuildDeleteAttributeProcessorConfig(
		[]string{"c.attr", "a.attr", "b.attr"},
		[]common.ObservabilitySignal{common.TracesObservabilitySignal},
	)
	require.NoError(t, err)
	require.Len(t, cfg.TraceStatements, 4)

	want := []string{
		`delete_key(attributes, "c.attr")`,
		`delete_key(attributes, "a.attr")`,
		`delete_key(attributes, "b.attr")`,
	}
	for _, stmt := range cfg.TraceStatements {
		assert.Equal(t, want, stmt.Statements, "context %s", stmt.Context)
	}
}

// An action with no attributes listed is accepted and produces statement lists that are present
// but empty, which is what the collector sees as "transform nothing".
func TestBuildDeleteAttributeProcessorConfig_emptyAttributeListProducesEmptyStatements(t *testing.T) {
	cfg, err := BuildDeleteAttributeProcessorConfig(
		nil,
		[]common.ObservabilitySignal{common.TracesObservabilitySignal},
	)
	require.NoError(t, err)
	require.Len(t, cfg.TraceStatements, 4)
	for _, stmt := range cfg.TraceStatements {
		assert.Empty(t, stmt.Statements, "context %s", stmt.Context)
	}
}

// The processor config is marshalled into Processor.Spec.ProcessorConfig and read by the
// collector through these exact snake_case keys.
func TestTransformProcessorConfigMarshalsToTheCollectorWireNames(t *testing.T) {
	cfg, err := BuildDeleteAttributeProcessorConfig(
		[]string{"a"},
		[]common.ObservabilitySignal{common.TracesObservabilitySignal},
	)
	require.NoError(t, err)

	raw, err := json.Marshal(cfg)
	require.NoError(t, err)

	var asMap map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &asMap))

	assert.Contains(t, asMap, "error_mode")
	assert.Contains(t, asMap, "trace_statements")
	// The buckets no signal asked for are omitted rather than serialized as null, so the
	// collector does not see an empty transform block for them.
	assert.NotContains(t, asMap, "log_statements")
	assert.NotContains(t, asMap, "metric_statements")
	assert.NotContains(t, asMap, "profile_statements")

	var statements []map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(asMap["trace_statements"], &statements))
	require.NotEmpty(t, statements)
	assert.Contains(t, statements[0], "context")
	assert.Contains(t, statements[0], "statements")
}
