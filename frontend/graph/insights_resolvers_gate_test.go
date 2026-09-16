package graph

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/odigos-io/odigos/frontend/services/insights"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// insightsResolverEntry is one resolver method that has to go through the
// insights feature gate, bound to a resolver and callable with zero arguments.
type insightsResolverEntry struct {
	name string
	call func() error
}

// insightsResolverEntries enumerates the whole insights GraphQL surface from the
// generated resolver interfaces, which are the schema itself. Driving the
// interfaces rather than a hand-written list means a resolver added later is
// covered by the gate tests below without anyone remembering to list it.
func insightsResolverEntries(t *testing.T, resolver *Resolver) []insightsResolverEntry {
	t.Helper()

	entries := boundInsightsMethods(t, reflect.TypeOf((*InsightsResolver)(nil)).Elem(),
		resolver.Insights(), "Insights.", func(string) bool { return true })
	entries = append(entries, boundInsightsMethods(t, reflect.TypeOf((*MutationResolver)(nil)).Elem(),
		resolver.Mutation(), "Mutation.", func(name string) bool {
			return strings.Contains(name, "Insights")
		})...)
	entries = append(entries, boundInsightsMethods(t, reflect.TypeOf((*QueryResolver)(nil)).Elem(),
		resolver.Query(), "Query.", func(name string) bool { return name == "Insights" })...)
	return entries
}

func boundInsightsMethods(t *testing.T, ifaceType reflect.Type, impl any, prefix string, include func(string) bool) []insightsResolverEntry {
	t.Helper()

	implValue := reflect.ValueOf(impl)
	entries := make([]insightsResolverEntry, 0, ifaceType.NumMethod())
	for i := 0; i < ifaceType.NumMethod(); i++ {
		name := ifaceType.Method(i).Name
		if !include(name) {
			continue
		}
		bound := implValue.MethodByName(name)
		require.True(t, bound.IsValid(), "%s%s is in the schema but not implemented", prefix, name)

		args := make([]reflect.Value, bound.Type().NumIn())
		args[0] = reflect.ValueOf(context.Background())
		for arg := 1; arg < len(args); arg++ {
			args[arg] = reflect.Zero(bound.Type().In(arg))
		}

		entries = append(entries, insightsResolverEntry{
			name: prefix + name,
			call: func() error {
				results := bound.Call(args)
				last := results[len(results)-1]
				if last.IsNil() {
					return nil
				}
				return last.Interface().(error)
			},
		})
	}
	return entries
}

// insightsSurfaceSize is the number of resolvers the insights feature currently
// exposes: 20 fields on the Insights type, 23 mutations and the root query
// field. It exists purely so a reflection bug that finds nothing cannot make
// the gate tests pass vacuously; bump it when the schema grows.
const insightsSurfaceSize = 44

// Insights is an optional enterprise feature and the odigos-insights service it
// talks to only exists in the cluster when the effective configuration turns it
// on. A resolver that skips the gate would dial a service that is not there, so
// this drives every insights resolver against a disabled configuration.
func TestEveryInsightsResolverRefusesToRunWhenInsightsAreDisabled(t *testing.T) {
	stub := newInsightsEngineStub(t)
	resolver := newInsightsDisabledResolver(t, stub)

	entries := insightsResolverEntries(t, resolver)
	require.Len(t, entries, insightsSurfaceSize)

	for _, entry := range entries {
		t.Run(entry.name, func(t *testing.T) {
			err := entry.call()
			require.Error(t, err)
			require.ErrorIs(t, err, insights.ErrNotEnabled)
			requireInsightsErrorCode(t, err, insights.CodeNotEnabled)
		})
	}

	assert.Empty(t, stub.recorded(), "a disabled feature must not reach the insights engine")
}

// The client is built once at server start-up and left nil when the insights
// base URL cannot be parsed, so every resolver has to fail on a nil client
// rather than dereference it.
func TestEveryInsightsResolverFailsWhenTheClientWasNeverInitialized(t *testing.T) {
	resolver := insightsResolverFor(t, "insights:\n  enabled: true\n", nil)

	entries := insightsResolverEntries(t, resolver)
	require.Len(t, entries, insightsSurfaceSize)

	for _, entry := range entries {
		t.Run(entry.name, func(t *testing.T) {
			err := entry.call()
			require.Error(t, err)
			require.ErrorIs(t, err, insights.ErrInternal)
			requireInsightsErrorCode(t, err, insights.CodeInternal)
		})
	}
}

// An unreadable effective config must not be reported as "insights are off":
// the UI hides the whole feature on INSIGHTS_NOT_ENABLED, which would turn a
// transient cluster read failure into a silently missing product area.
func TestEveryInsightsResolverReportsAnUnreadableEffectiveConfigAsAFailure(t *testing.T) {
	stub := newInsightsEngineStub(t)
	resolver := insightsResolverFor(t, "insights: [not-a-map", stub)

	entries := insightsResolverEntries(t, resolver)
	require.Len(t, entries, insightsSurfaceSize)

	for _, entry := range entries {
		t.Run(entry.name, func(t *testing.T) {
			err := entry.call()
			require.Error(t, err)
			requireInsightsErrorCode(t, err, insights.CodeInternal)
			assert.NotErrorIs(t, err, insights.ErrNotEnabled)
		})
	}

	assert.Empty(t, stub.recorded())
}
