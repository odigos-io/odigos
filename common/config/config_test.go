package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeConfigsUnionOfDisjointDomains(t *testing.T) {
	domains := map[string]Config{
		"first": {
			Receivers:  GenericMap{"otlp": GenericMap{"protocols": "grpc"}},
			Exporters:  GenericMap{"debug/first": GenericMap{}},
			Processors: GenericMap{"batch/first": GenericMap{}},
			Extensions: GenericMap{"pprof": GenericMap{}},
			Connectors: GenericMap{"forward/first": GenericMap{}},
			Service: Service{
				Extensions: []string{"pprof"},
				Pipelines: map[string]Pipeline{
					"traces/first": {Receivers: []string{"otlp"}, Exporters: []string{"debug/first"}},
				},
			},
		},
		"second": {
			Receivers:  GenericMap{"prometheus/self-metrics": GenericMap{}},
			Exporters:  GenericMap{"debug/second": GenericMap{}},
			Processors: GenericMap{"batch/second": GenericMap{}},
			Extensions: GenericMap{"health_check": GenericMap{}},
			Connectors: GenericMap{"forward/second": GenericMap{}},
			Service: Service{
				Extensions: []string{"health_check"},
				Pipelines: map[string]Pipeline{
					"metrics/second": {Receivers: []string{"prometheus/self-metrics"}, Exporters: []string{"debug/second"}},
				},
			},
		},
	}

	merged, err := MergeConfigs(domains)
	require.NoError(t, err)

	// Every section must keep its own components: a merge that writes a domain's section into the
	// wrong section of the result produces a collector config that cannot start.
	assert.Equal(t, GenericMap{
		"otlp":                    GenericMap{"protocols": "grpc"},
		"prometheus/self-metrics": GenericMap{},
	}, merged.Receivers)
	assert.Equal(t, GenericMap{
		"debug/first":  GenericMap{},
		"debug/second": GenericMap{},
	}, merged.Exporters)
	assert.Equal(t, GenericMap{
		"batch/first":  GenericMap{},
		"batch/second": GenericMap{},
	}, merged.Processors)
	assert.Equal(t, GenericMap{
		"pprof":        GenericMap{},
		"health_check": GenericMap{},
	}, merged.Extensions)
	assert.Equal(t, GenericMap{
		"forward/first":  GenericMap{},
		"forward/second": GenericMap{},
	}, merged.Connectors)

	assert.Equal(t, map[string]Pipeline{
		"traces/first":   {Receivers: []string{"otlp"}, Exporters: []string{"debug/first"}},
		"metrics/second": {Receivers: []string{"prometheus/self-metrics"}, Exporters: []string{"debug/second"}},
	}, merged.Service.Pipelines)
	assert.ElementsMatch(t, []string{"pprof", "health_check"}, merged.Service.Extensions)
}

// componentSections enumerates the five component maps that MergeConfigs merges independently.
var componentSections = []struct {
	name string
	set  func(cfg *Config, components GenericMap)
}{
	{"receivers", func(cfg *Config, components GenericMap) { cfg.Receivers = components }},
	{"exporters", func(cfg *Config, components GenericMap) { cfg.Exporters = components }},
	{"processors", func(cfg *Config, components GenericMap) { cfg.Processors = components }},
	{"extensions", func(cfg *Config, components GenericMap) { cfg.Extensions = components }},
	{"connectors", func(cfg *Config, components GenericMap) { cfg.Connectors = components }},
}

func TestMergeConfigsRejectsDuplicateComponentKey(t *testing.T) {
	for _, section := range componentSections {
		t.Run(section.name, func(t *testing.T) {
			first := Config{}
			second := Config{}
			section.set(&first, GenericMap{"shared": GenericMap{"from": "first"}})
			section.set(&second, GenericMap{"shared": GenericMap{"from": "second"}})

			merged, err := MergeConfigs(map[string]Config{"a": first, "b": second})
			require.Error(t, err)
			assert.EqualError(t, err, "duplicate key shared in configs")
			assert.Equal(t, Config{}, merged)
		})
	}
}

func TestMergeConfigsRejectsDuplicatePipelineName(t *testing.T) {
	domains := map[string]Config{
		"a": {Service: Service{Pipelines: map[string]Pipeline{"traces/shared": {Exporters: []string{"debug/a"}}}}},
		"b": {Service: Service{Pipelines: map[string]Pipeline{"traces/shared": {Exporters: []string{"debug/b"}}}}},
	}

	merged, err := MergeConfigs(domains)
	require.Error(t, err)
	assert.EqualError(t, err, "duplicate pipeline traces/shared in configs")
	assert.Equal(t, Config{}, merged)
}

// The domain map is iterated in sorted key order on purpose: Go randomizes map iteration, so
// without the sort the concatenated lists below would be ordered differently on every reconcile
// and the rendered ConfigMap would churn even when nothing changed.
func TestMergeConfigsIsIndependentOfMapIterationOrder(t *testing.T) {
	domainNames := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	domains := make(map[string]Config, len(domainNames))
	wantExtensions := make([]string, 0, len(domainNames))
	wantReaders := make([]GenericMap, 0, len(domainNames))
	for _, name := range domainNames {
		domains[name] = Config{
			Service: Service{
				Extensions: []string{name + "-extension"},
				Telemetry: Telemetry{
					Metrics: MetricsConfig{Readers: []GenericMap{{"reader": name}}},
					Resource: &TelemetryResource{
						Attributes: []ResourceAttribute{{Name: "domain", Value: name}},
					},
				},
			},
		}
		wantExtensions = append(wantExtensions, name+"-extension")
		wantReaders = append(wantReaders, GenericMap{"reader": name})
	}

	wantAttributes := make([]ResourceAttribute, 0, len(domainNames))
	for _, name := range domainNames {
		wantAttributes = append(wantAttributes, ResourceAttribute{Name: "domain", Value: name})
	}

	// Repeat so that a merge relying on map iteration order fails reliably rather than occasionally.
	for range 30 {
		merged, err := MergeConfigs(domains)
		require.NoError(t, err)
		require.Equal(t, wantExtensions, merged.Service.Extensions)
		require.Equal(t, wantReaders, merged.Service.Telemetry.Metrics.Readers)
		require.Equal(t, wantAttributes, merged.Service.Telemetry.Resource.Attributes)
	}
}

func TestMergeConfigsDoesNotAliasInputDomains(t *testing.T) {
	domain := Config{
		Exporters: GenericMap{"debug/only": GenericMap{}},
		Service: Service{
			Pipelines: map[string]Pipeline{"traces/only": {Exporters: []string{"debug/only"}}},
		},
	}
	domains := map[string]Config{"only": domain}

	merged, err := MergeConfigs(domains)
	require.NoError(t, err)

	merged.Exporters["debug/injected"] = GenericMap{}
	merged.Service.Pipelines["traces/injected"] = Pipeline{}

	assert.Equal(t, GenericMap{"debug/only": GenericMap{}}, domains["only"].Exporters)
	assert.Equal(t, map[string]Pipeline{"traces/only": {Exporters: []string{"debug/only"}}},
		domains["only"].Service.Pipelines)
}

func TestMergeConfigsTelemetryMetricsLevel(t *testing.T) {
	tests := []struct {
		name      string
		firstLvl  string
		secondLvl string
		want      string
		wantErr   bool
	}{
		{name: "neither set"},
		{name: "only first set", firstLvl: "detailed", want: "detailed"},
		{name: "only second set", secondLvl: "basic", want: "basic"},
		{name: "both set to the same level", firstLvl: "basic", secondLvl: "basic", want: "basic"},
		{name: "conflicting levels", firstLvl: "basic", secondLvl: "detailed", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domains := map[string]Config{
				"a": {Service: Service{Telemetry: Telemetry{Metrics: MetricsConfig{Level: tt.firstLvl}}}},
				"b": {Service: Service{Telemetry: Telemetry{Metrics: MetricsConfig{Level: tt.secondLvl}}}},
			}

			merged, err := MergeConfigs(domains)
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, "service telemetry metrics level is allowed to be set only once")
				assert.Equal(t, Config{}, merged)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, merged.Service.Telemetry.Metrics.Level)
		})
	}
}

func TestMergeConfigsTelemetryLogsLevel(t *testing.T) {
	tests := []struct {
		name      string
		firstLvl  string
		secondLvl string
		want      string
	}{
		{name: "neither set"},
		{name: "only the first domain sets it", firstLvl: "debug", want: "debug"},
		{name: "only the second domain sets it", secondLvl: "warn", want: "warn"},
		{name: "the later domain wins", firstLvl: "debug", secondLvl: "warn", want: "warn"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domains := map[string]Config{
				"a": {Service: Service{Telemetry: Telemetry{Logs: LogsConfig{Level: tt.firstLvl}}}},
				"b": {Service: Service{Telemetry: Telemetry{Logs: LogsConfig{Level: tt.secondLvl}}}},
			}

			merged, err := MergeConfigs(domains)
			require.NoError(t, err)
			assert.Equal(t, tt.want, merged.Service.Telemetry.Logs.Level)
		})
	}
}

func TestMergeTelemetryResource(t *testing.T) {
	firstAttr := ResourceAttribute{Name: "first", Value: "1"}
	secondAttr := ResourceAttribute{Name: "second", Value: "2"}

	tests := []struct {
		name   string
		first  *TelemetryResource
		second *TelemetryResource
		want   *TelemetryResource
	}{
		{name: "both nil"},
		{
			name:   "first nil",
			second: &TelemetryResource{Attributes: []ResourceAttribute{secondAttr}},
			want:   &TelemetryResource{Attributes: []ResourceAttribute{secondAttr}},
		},
		{
			name:  "second nil",
			first: &TelemetryResource{Attributes: []ResourceAttribute{firstAttr}},
			want:  &TelemetryResource{Attributes: []ResourceAttribute{firstAttr}},
		},
		{
			name:   "first has no attributes",
			first:  &TelemetryResource{},
			second: &TelemetryResource{Attributes: []ResourceAttribute{secondAttr}},
			want:   &TelemetryResource{Attributes: []ResourceAttribute{secondAttr}},
		},
		{
			name:   "second has no attributes",
			first:  &TelemetryResource{Attributes: []ResourceAttribute{firstAttr}},
			second: &TelemetryResource{},
			want:   &TelemetryResource{Attributes: []ResourceAttribute{firstAttr}},
		},
		{
			name:   "both have attributes",
			first:  &TelemetryResource{Attributes: []ResourceAttribute{firstAttr}},
			second: &TelemetryResource{Attributes: []ResourceAttribute{secondAttr}},
			want:   &TelemetryResource{Attributes: []ResourceAttribute{firstAttr, secondAttr}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, mergeTelemetryResource(tt.first, tt.second))
		})
	}
}

func TestMergeTelemetryReaders(t *testing.T) {
	firstReader := GenericMap{"reader": "first"}
	secondReader := GenericMap{"reader": "second"}

	tests := []struct {
		name   string
		first  []GenericMap
		second []GenericMap
		want   []GenericMap
	}{
		{name: "both empty"},
		{name: "only first", first: []GenericMap{firstReader}, want: []GenericMap{firstReader}},
		{name: "only second", second: []GenericMap{secondReader}, want: []GenericMap{secondReader}},
		{
			name:   "both, first then second",
			first:  []GenericMap{firstReader},
			second: []GenericMap{secondReader},
			want:   []GenericMap{firstReader, secondReader},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, mergeTelemetryReaders(tt.first, tt.second))
		})
	}
}

func TestMergeGenericMaps(t *testing.T) {
	merged, err := mergeGenericMaps(
		GenericMap{"a": 1},
		GenericMap{"b": 2},
		GenericMap{"c": 3},
	)
	require.NoError(t, err)
	assert.Equal(t, GenericMap{"a": 1, "b": 2, "c": 3}, merged)

	// A duplicate between non-adjacent maps must be detected too.
	merged, err = mergeGenericMaps(
		GenericMap{"a": 1},
		GenericMap{"b": 2},
		GenericMap{"a": 3},
	)
	require.Error(t, err)
	assert.EqualError(t, err, "duplicate key a in configs")
	assert.Equal(t, GenericMap{}, merged)
}
