package destinations

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
)

// The catalog file is what the UI offers and what docs/oss/backends/*.mdx is generated from, while
// the common/config configurer is what actually renders the collector pipelines. Nothing ties the
// two together, and the pipeline names a configurer returns are also what CalculateGatewayConfig
// derives the signals advertised back to the agents from, so a disagreement does not degrade one
// destination: it stops the cluster producing that signal at all. Both halves of the disagreement
// have been fixed one destination at a time (#6073 for SignalFx traces, #6102 for Honeycomb
// metrics and logs); these tests drive every catalog entry against its configurer instead.

// dsDest is a Destination CR as the gateway sees it. Destination has no validating webhook, so
// the signal set here can be anything, including a signal the catalog calls unsupported.
type dsDest struct {
	id      string
	t       common.DestinationType
	data    map[string]string
	signals []common.ObservabilitySignal
}

func (d dsDest) GetID() string                            { return d.id }
func (d dsDest) GetType() common.DestinationType          { return d.t }
func (d dsDest) GetConfig() map[string]string             { return d.data }
func (d dsDest) GetSignals() []common.ObservabilitySignal { return d.signals }

func dsEmptyConfig() *config.Config {
	return &config.Config{
		Receivers:  config.GenericMap{},
		Exporters:  config.GenericMap{},
		Processors: config.GenericMap{},
		Extensions: config.GenericMap{},
		Connectors: config.GenericMap{},
		Service:    config.Service{Pipelines: map[string]config.Pipeline{}},
	}
}

var dsSignals = []struct {
	signal   common.ObservabilitySignal
	prefix   string
	declared func(Spec) bool
}{
	{common.TracesObservabilitySignal, "traces/", func(s Spec) bool { return s.Signals.Traces.Supported }},
	{common.MetricsObservabilitySignal, "metrics/", func(s Spec) bool { return s.Signals.Metrics.Supported }},
	{common.LogsObservabilitySignal, "logs/", func(s Spec) bool { return s.Signals.Logs.Supported }},
	{common.ProfilesObservabilitySignal, "profiles/", func(s Spec) bool { return s.Signals.Profiles.Supported }},
}

// Candidate endpoint spellings, tried in order until one gets a destination past its own input
// validation. Several configurers reject a scheme (gRPC endpoints are host:port), several require
// one, and one requires a path, so there is no single value that satisfies all of them.
var dsEndpointDialects = []string{
	"odigos-probe.example.com:4317",
	"https://odigos-probe.example.com:4318",
	"http://odigos-probe.example.com:4318",
	"https://odigos-probe.example.com/api/prom/push",
}

// Values for the handful of fields whose format cannot be guessed from the catalog entry. A new
// destination only needs an entry here if none of the dialects above gets it past validation.
var dsFieldOverrides = map[string]map[string]string{
	"dynamic": {"DYNAMIC_CONFIGURATION_DATA": "endpoint: odigos-probe.example.com:4317\n"},
}

// Destinations whose catalog entry declares a signal that the configurer never renders a pipeline
// for. Each entry is a live gap, not an accepted difference: once the configurer is fixed,
// TestEveryKnownSignalContractGapStillDiverges fails until the entry is deleted.
//
// dynamic: #4831 added "profiles: supported: true" to data/dynamic.yaml and added the profiles
// branch to genericotlp.go, otlphttp.go and pyroscope.go, but not to dynamic.go. Selecting
// profiles on a Dynamic destination returns no error and no pipeline, so the destination reports
// healthy while profiles are never advertised to the agents.
var dsDeclaredButNotRendered = map[string][]common.ObservabilitySignal{
	"dynamic": {common.ProfilesObservabilitySignal},
}

// The mirror image: a configurer that renders a pipeline for a signal its catalog entry calls
// unsupported. The UI will not offer the signal, but a Destination applied straight to the API
// server can select it, and then odigos exports a signal the catalog and the docs say it cannot.
//
// openobserve: data/openobserve.yaml has declared "metrics: supported: false" since #2648, while
// openobserve.go has had an isMetricsEnabled branch since the same PR.
var dsRenderedButNotDeclared = map[string][]common.ObservabilitySignal{
	"openobserve": {common.MetricsObservabilitySignal},
}

// Destinations that emit a connector their own pipelines do not both feed and drain. The
// collector rejects such a configuration, and because every destination shares one gateway
// config, that takes down every other destination in the cluster too.
//
// bonree: bonree.go defines the "servicegraph" connector, exports to it from its traces pipeline
// and receives from it in its metrics pipeline. Selecting only one of the two leaves the
// connector half-wired. Selecting metrics only is the worse half: nothing else in the gateway
// exports to "servicegraph" unless some other destination enables traces.
var dsUnbalancedConnectorGaps = map[string]string{
	"bonree": "the servicegraph connector spans its traces and metrics pipelines",
}

func dsGapped(gaps map[string][]common.ObservabilitySignal, destType common.DestinationType, signal common.ObservabilitySignal) bool {
	for _, s := range gaps[string(destType)] {
		if s == signal {
			return true
		}
	}
	return false
}

func dsFieldValue(f Field, endpoint string) string {
	name := strings.ToUpper(f.Name)
	if strings.Contains(name, "URL") || strings.Contains(name, "ENDPOINT") ||
		strings.Contains(name, "HOST") || strings.Contains(name, "SERVER") {
		return endpoint
	}
	if f.InitialValue != "" {
		return f.InitialValue
	}
	switch f.ComponentType {
	case "dropdown":
		if values, ok := f.ComponentProps["values"].([]interface{}); ok && len(values) > 0 {
			return fmt.Sprintf("%v", values[0])
		}
		return "odigos-probe"
	case "checkbox":
		return "false"
	case "multiInput":
		encoded, err := json.Marshal([]string{"odigos-probe"})
		if err != nil {
			return ""
		}
		return string(encoded)
	case "keyValuePairs":
		// every keyValuePairs field in the catalog is optional, and the two configurers that read
		// one disagree on its encoding, so leave them unset
		return ""
	}
	if strings.Contains(name, "PORT") {
		return "4317"
	}
	return "odigos-probe"
}

func dsData(d Destination, endpoint string) map[string]string {
	data := map[string]string{}
	for _, f := range d.Spec.Fields {
		if value := dsFieldValue(f, endpoint); value != "" {
			data[f.Name] = value
		}
	}
	for name, value := range dsFieldOverrides[string(d.Metadata.Type)] {
		data[name] = value
	}
	return data
}

func dsDeclaredSignals(d Destination) []common.ObservabilitySignal {
	var signals []common.ObservabilitySignal
	for _, s := range dsSignals {
		if s.declared(d.Spec) {
			signals = append(signals, s.signal)
		}
	}
	return signals
}

// dsConfigurableData returns field values that get d past its own input validation with every
// signal it declares selected. Without it a destination that rejects the synthetic data would be
// silently exempt from every contract below.
func dsConfigurableData(t *testing.T, configer config.Configer, d Destination) map[string]string {
	t.Helper()
	declared := dsDeclaredSignals(d)
	var lastErr error
	for _, dialect := range dsEndpointDialects {
		data := dsData(d, dialect)
		_, err := configer.ModifyConfig(dsDest{id: "probe", t: d.Metadata.Type, data: data, signals: declared}, dsEmptyConfig())
		if err == nil {
			return data
		}
		lastErr = err
	}
	t.Errorf("%s: none of the %d endpoint spellings got this destination past its own validation, "+
		"so its signal contract cannot be checked. Add the value it needs to dsFieldOverrides. Last error: %v",
		d.Metadata.Type, len(dsEndpointDialects), lastErr)
	return nil
}

func dsCatalogWithConfigers(t *testing.T) ([]Destination, map[common.DestinationType]config.Configer) {
	t.Helper()
	if err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	configers, err := config.LoadConfigers()
	if err != nil {
		t.Fatalf("LoadConfigers: %v", err)
	}
	dests := Get()
	if len(dests) == 0 {
		t.Fatal("the embedded catalog is empty")
	}
	sort.Slice(dests, func(i, j int) bool { return dests[i].Metadata.Type < dests[j].Metadata.Type })
	return dests, configers
}

// A catalog entry with no configurer is offered by the UI and then fails every gateway reconcile
// with "no configer for <type>".
func TestEveryCatalogEntryHasAConfiger(t *testing.T) {
	dests, configers := dsCatalogWithConfigers(t)
	for _, d := range dests {
		if _, ok := configers[d.Metadata.Type]; !ok {
			t.Errorf("%s is in the catalog but has no entry in common/config's availableConfigers", d.Metadata.Type)
		}
	}
}

// The reverse: a configurer with no catalog entry cannot be reached from the UI at all. debug,
// mock and nop are internal destinations with no catalog entry by design; gcs and sentry are
// complete configurers for real backends that no catalog file offers.
func TestEveryConfigerWithoutACatalogEntryIsOneOfTheKnownOnes(t *testing.T) {
	dests, configers := dsCatalogWithConfigers(t)
	inCatalog := map[common.DestinationType]bool{}
	for _, d := range dests {
		inCatalog[d.Metadata.Type] = true
	}

	known := map[common.DestinationType]bool{"debug": true, "mock": true, "nop": true, "gcs": true, "sentry": true}
	var unreachable []string
	for destType := range configers {
		if !inCatalog[destType] {
			unreachable = append(unreachable, string(destType))
			if !known[destType] {
				t.Errorf("configer %q has no catalog file, so no destination of that type can ever be created", destType)
			}
		}
	}
	sort.Strings(unreachable)
	for destType := range known {
		if inCatalog[destType] {
			t.Errorf("%q now has a catalog file; remove it from this test's known list", destType)
		}
	}
	if len(unreachable) == 0 {
		t.Error("no configer was classified; the catalog/configer comparison did not run")
	}
}

// The central contract. For every non-empty subset of the signals a catalog entry declares, the
// configurer must succeed and must render a destination pipeline for exactly those signals. The
// subsets are what catch a guard that only holds for the full set: Honeycomb rendered all three
// signals together but rejected every subset without traces (#6102).
func TestEveryNonEmptySubsetOfDeclaredSignalsRendersExactlyThosePipelines(t *testing.T) {
	dests, configers := dsCatalogWithConfigers(t)

	subsets := 0
	for _, d := range dests {
		configer, ok := configers[d.Metadata.Type]
		if !ok {
			continue
		}
		data := dsConfigurableData(t, configer, d)
		if data == nil {
			continue
		}
		declared := dsDeclaredSignals(d)

		for mask := 1; mask < 1<<len(declared); mask++ {
			var selected []common.ObservabilitySignal
			want := map[string]bool{}
			for bit, signal := range declared {
				if mask&(1<<bit) == 0 {
					continue
				}
				selected = append(selected, signal)
				if dsGapped(dsDeclaredButNotRendered, d.Metadata.Type, signal) {
					continue
				}
				want[dsPrefixOf(signal)] = true
			}
			subsets++

			cfg := dsEmptyConfig()
			names, err := configer.ModifyConfig(dsDest{id: "probe", t: d.Metadata.Type, data: data, signals: selected}, cfg)
			if err != nil {
				t.Errorf("%s with signals %v: ModifyConfig failed although the catalog declares all of them: %v",
					d.Metadata.Type, selected, err)
				continue
			}
			if _, known := dsUnbalancedConnectorGaps[string(d.Metadata.Type)]; !known {
				for _, side := range dsUnbalancedConnectors(cfg) {
					t.Errorf("%s with signals %v: %s, which the collector rejects and which invalidates the whole gateway config",
						d.Metadata.Type, selected, side)
				}
			}
			got := map[string]bool{}
			for _, name := range names {
				prefix, ok := dsSignalPrefix(name)
				if !ok {
					t.Errorf("%s with signals %v: returned pipeline %q carries none of the four signal prefixes, "+
						"so CalculateGatewayConfig cannot classify it and the signal is never advertised to the agents",
						d.Metadata.Type, selected, name)
					continue
				}
				got[prefix] = true
			}
			for prefix := range want {
				if !got[prefix] {
					t.Errorf("%s with signals %v: no %s pipeline was rendered (returned %v), so that signal is dropped",
						d.Metadata.Type, selected, prefix, names)
				}
			}
			for prefix := range got {
				if !want[prefix] {
					t.Errorf("%s with signals %v: rendered a %s pipeline for a signal that was not selected (returned %v)",
						d.Metadata.Type, selected, prefix, names)
				}
			}
		}
	}

	if subsets < len(dests) {
		t.Errorf("checked %d signal subsets across %d destinations; every destination declares at least one signal, so this is too few", subsets, len(dests))
	}
}

// CalculateGatewayConfig looks each returned name up in Service.Pipelines and writes the result
// back with a forward connector attached. A name that was returned without the pipeline being
// added therefore becomes a pipeline with a receiver and no exporter, and a pipeline that was
// added without being returned keeps its exporter and never gets a receiver. Either one makes the
// whole gateway configuration invalid, which takes down every destination in the cluster, not
// just this one.
func TestEveryRenderedDestinationPipelineIsReturnedAndComplete(t *testing.T) {
	dests, configers := dsCatalogWithConfigers(t)

	pipelines := 0
	for _, d := range dests {
		configer, ok := configers[d.Metadata.Type]
		if !ok {
			continue
		}
		data := dsConfigurableData(t, configer, d)
		if data == nil {
			continue
		}

		for _, signal := range dsDeclaredSignals(d) {
			cfg := dsEmptyConfig()
			names, err := configer.ModifyConfig(dsDest{id: "probe", t: d.Metadata.Type, data: data, signals: []common.ObservabilitySignal{signal}}, cfg)
			if err != nil {
				continue
			}

			returned := map[string]bool{}
			for _, name := range names {
				if returned[name] {
					t.Errorf("%s/%s: pipeline %q was returned twice, so it gets two forward connectors", d.Metadata.Type, signal, name)
				}
				returned[name] = true
			}
			for name := range cfg.Service.Pipelines {
				if !returned[name] {
					t.Errorf("%s/%s: pipeline %q was added to the config but not returned, so it never gets a receiver (returned %v)",
						d.Metadata.Type, signal, name, names)
				}
			}
			for _, name := range names {
				pipeline, ok := cfg.Service.Pipelines[name]
				if !ok {
					t.Errorf("%s/%s: pipeline %q was returned but never added to the config, so it ends up with a receiver and no exporter",
						d.Metadata.Type, signal, name)
					continue
				}
				pipelines++
				if len(pipeline.Exporters) == 0 {
					t.Errorf("%s/%s: pipeline %q has no exporters", d.Metadata.Type, signal, name)
				}
				for _, exporter := range pipeline.Exporters {
					_, isExporter := cfg.Exporters[exporter]
					_, isConnector := cfg.Connectors[exporter]
					if !isExporter && !isConnector {
						t.Errorf("%s/%s: pipeline %q references exporter %q, which the configurer never defined",
							d.Metadata.Type, signal, name, exporter)
					}
				}
			}
		}
	}

	if pipelines < len(dests) {
		t.Errorf("checked %d destination pipelines across %d destinations; this is too few for the assertions above to mean anything", pipelines, len(dests))
	}
}

// Selecting a signal the catalog calls unsupported must not produce a pipeline for it. There is no
// validating webhook on Destination, so a manifest applied straight to the API server can ask for
// one.
func TestNoUndeclaredSignalRendersAPipeline(t *testing.T) {
	dests, configers := dsCatalogWithConfigers(t)

	every := []common.ObservabilitySignal{
		common.TracesObservabilitySignal, common.MetricsObservabilitySignal,
		common.LogsObservabilitySignal, common.ProfilesObservabilitySignal,
	}

	checked := 0
	for _, d := range dests {
		configer, ok := configers[d.Metadata.Type]
		if !ok {
			continue
		}
		data := dsConfigurableData(t, configer, d)
		if data == nil {
			continue
		}
		names, err := configer.ModifyConfig(dsDest{id: "probe", t: d.Metadata.Type, data: data, signals: every}, dsEmptyConfig())
		if err != nil {
			continue
		}
		for _, s := range dsSignals {
			if s.declared(d.Spec) || dsGapped(dsRenderedButNotDeclared, d.Metadata.Type, s.signal) {
				continue
			}
			checked++
			for _, name := range names {
				if strings.HasPrefix(name, s.prefix) {
					t.Errorf("%s: the catalog declares %s unsupported but the configurer rendered %q",
						d.Metadata.Type, s.prefix, name)
				}
			}
		}
	}

	if checked == 0 {
		t.Error("no undeclared signal was checked; every catalog entry would have to declare all four")
	}
}

// Both gap lists above describe production as it is today. This test fails as soon as one of them
// is fixed, so the entry is removed together with the fix instead of outliving it and exempting a
// destination from the contract for good.
func TestEveryKnownSignalContractGapStillDiverges(t *testing.T) {
	dests, configers := dsCatalogWithConfigers(t)
	byType := map[common.DestinationType]Destination{}
	for _, d := range dests {
		byType[d.Metadata.Type] = d
	}

	renders := func(t *testing.T, destType common.DestinationType, signal common.ObservabilitySignal) bool {
		t.Helper()
		d, ok := byType[destType]
		if !ok {
			t.Fatalf("%s is listed as a known gap but is no longer in the catalog; remove the entry", destType)
		}
		configer, ok := configers[destType]
		if !ok {
			t.Fatalf("%s is listed as a known gap but no longer has a configer; remove the entry", destType)
		}
		data := dsConfigurableData(t, configer, d)
		names, err := configer.ModifyConfig(dsDest{id: "probe", t: destType, data: data, signals: []common.ObservabilitySignal{signal}}, dsEmptyConfig())
		if err != nil {
			return false
		}
		for _, name := range names {
			if strings.HasPrefix(name, dsPrefixOf(signal)) {
				return true
			}
		}
		return false
	}

	if len(dsDeclaredButNotRendered)+len(dsRenderedButNotDeclared)+len(dsUnbalancedConnectorGaps) == 0 {
		t.Skip("no known gaps left")
	}

	for destType, reason := range dsUnbalancedConnectorGaps {
		d, ok := byType[common.DestinationType(destType)]
		if !ok {
			t.Errorf("%s is listed in dsUnbalancedConnectorGaps but is no longer in the catalog; remove the entry", destType)
			continue
		}
		configer := configers[d.Metadata.Type]
		data := dsConfigurableData(t, configer, d)
		stillUnbalanced := false
		for _, signal := range dsDeclaredSignals(d) {
			cfg := dsEmptyConfig()
			if _, err := configer.ModifyConfig(dsDest{id: "probe", t: d.Metadata.Type, data: data, signals: []common.ObservabilitySignal{signal}}, cfg); err != nil {
				continue
			}
			if len(dsUnbalancedConnectors(cfg)) > 0 {
				stillUnbalanced = true
			}
		}
		if !stillUnbalanced {
			t.Errorf("%s no longer leaves a connector half-wired (%s); remove it from dsUnbalancedConnectorGaps so the contract covers it", destType, reason)
		}
	}

	for destType, signals := range dsDeclaredButNotRendered {
		for _, signal := range signals {
			d := byType[common.DestinationType(destType)]
			if !dsIsDeclared(d, signal) {
				t.Errorf("%s no longer declares %s supported, so it is not a gap any more; remove it from dsDeclaredButNotRendered", destType, signal)
				continue
			}
			if renders(t, common.DestinationType(destType), signal) {
				t.Errorf("%s now renders a %s pipeline; remove it from dsDeclaredButNotRendered so the contract covers it", destType, signal)
			}
		}
	}
	for destType, signals := range dsRenderedButNotDeclared {
		for _, signal := range signals {
			d := byType[common.DestinationType(destType)]
			if dsIsDeclared(d, signal) {
				t.Errorf("%s now declares %s supported, so it is not a gap any more; remove it from dsRenderedButNotDeclared", destType, signal)
				continue
			}
			if !renders(t, common.DestinationType(destType), signal) {
				t.Errorf("%s no longer renders a %s pipeline; remove it from dsRenderedButNotDeclared so the contract covers it", destType, signal)
			}
		}
	}
}

// dsUnbalancedConnectors reports the connectors in cfg that are used on only one side. A
// collector connector has to appear as an exporter in one pipeline and as a receiver in another;
// the gateway builder attaches the forward connector and the router itself, so anything a
// destination configurer defines has to be wired up by that same configurer.
func dsUnbalancedConnectors(cfg *config.Config) []string {
	asExporter := map[string]bool{}
	asReceiver := map[string]bool{}
	for _, p := range cfg.Service.Pipelines {
		for _, e := range p.Exporters {
			asExporter[e] = true
		}
		for _, r := range p.Receivers {
			asReceiver[r] = true
		}
	}
	var unbalanced []string
	for name := range cfg.Connectors {
		switch {
		case asExporter[name] && !asReceiver[name]:
			unbalanced = append(unbalanced, fmt.Sprintf("connector %q is exported to but never received from", name))
		case asReceiver[name] && !asExporter[name]:
			unbalanced = append(unbalanced, fmt.Sprintf("connector %q is received from but never exported to", name))
		}
	}
	sort.Strings(unbalanced)
	return unbalanced
}

func dsIsDeclared(d Destination, signal common.ObservabilitySignal) bool {
	for _, s := range dsSignals {
		if s.signal == signal {
			return s.declared(d.Spec)
		}
	}
	return false
}

func dsPrefixOf(signal common.ObservabilitySignal) string {
	for _, s := range dsSignals {
		if s.signal == signal {
			return s.prefix
		}
	}
	return ""
}

func dsSignalPrefix(pipelineName string) (string, bool) {
	// spelled out instead of derived from dsSignals, because these are the four prefixes
	// CalculateGatewayConfig switches on when it decides what to advertise to the agents
	for _, prefix := range []string{"traces/", "metrics/", "logs/", "profiles/"} {
		if strings.HasPrefix(pipelineName, prefix) {
			return prefix, true
		}
	}
	return "", false
}
