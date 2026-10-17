package actions

import (
	"encoding/json"
	"testing"
)

// The exact string k8sutils/pkg/action/configextension.go compares every catalog processor
// against. It lives in a different Go module with no shared constant, so it is spelled out
// here on purpose: deriving it from the same source as the production code would make a
// rename invisible to this test.
const acConfigExtensionMechanism = "odigosConfigExtension"

// acProcessorExpectation is the collector-side consequence of one catalog entry.
type acProcessorExpectation struct {
	processorType  string
	preSpanMetrics bool
}

// The actions whose catalog entry causes a shared collector processor to be synthesized.
// Every other action must declare no processors at all: an action that gains or loses an
// entry here silently changes what runs in the collector pipeline.
var acExpectedConfigExtensionProcessors = map[string][]acProcessorExpectation{
	"DbQueryTemplatization": {{processorType: "odigossqlquery", preSpanMetrics: true}},
	"InferDbAttributes":     {{processorType: "odigossqlquery", preSpanMetrics: true}},
	"PiiMasking":            {{processorType: "odigospiimasking", preSpanMetrics: true}},
}

// k8sutils/pkg/action matches on this literal to decide whether an Action CR is a config
// extension. A catalog entry that misspells it is reported as "not a config extension" and the
// processor is never added to the pipeline, with no error anywhere.
func TestConfigExtensionProcessorsUseTheMechanismNameK8sutilsMatchesOn(t *testing.T) {
	loaded := acLoadCatalog(t)

	declared := 0
	for _, a := range loaded {
		for i, p := range a.Spec.Processors {
			declared++
			if p.ConfigMechanism != acConfigExtensionMechanism {
				t.Errorf("action %q processor %d declares configMechanism %q, want %q",
					a.Metadata.Type, i, p.ConfigMechanism, acConfigExtensionMechanism)
			}
			if p.Type == "" {
				t.Errorf("action %q processor %d has an empty type", a.Metadata.Type, i)
			}
		}
	}
	if declared == 0 {
		t.Fatal("no catalog action declares a processor; this test would pass vacuously")
	}
}

func TestTheCatalogSynthesizesExactlyTheKnownCollectorProcessors(t *testing.T) {
	loaded := acLoadCatalog(t)

	got := map[string][]acProcessorExpectation{}
	for _, a := range loaded {
		for _, p := range a.Spec.Processors {
			got[a.Metadata.Type] = append(got[a.Metadata.Type], acProcessorExpectation{
				processorType:  p.Type,
				preSpanMetrics: p.PreSpanMetrics,
			})
		}
	}

	for actionType, want := range acExpectedConfigExtensionProcessors {
		have, ok := got[actionType]
		if !ok {
			t.Errorf("action %q declares no collector processor; it no longer reaches the collector pipeline", actionType)
			continue
		}
		if len(have) != len(want) {
			t.Errorf("action %q declares %d processors, want %d", actionType, len(have), len(want))
			continue
		}
		for i := range want {
			if have[i] != want[i] {
				t.Errorf("action %q processor %d = %+v, want %+v", actionType, i, have[i], want[i])
			}
		}
	}

	for actionType := range got {
		if _, expected := acExpectedConfigExtensionProcessors[actionType]; !expected {
			t.Errorf("action %q newly declares a collector processor %+v; confirm the processor is wired up and add it here",
				actionType, got[actionType])
		}
	}
}

// Every field below is read straight out of the catalog by
// frontend/services/action_types.go and rendered in the action picker, so an empty one is a
// blank row, a missing docs link or an action nobody can identify.
func TestEveryActionDeclaresTheMetadataTheUiRenders(t *testing.T) {
	loaded := acLoadCatalog(t)

	for _, a := range loaded {
		name := a.Metadata.Type
		if name == "" {
			t.Error("a catalog action has an empty metadata.type; it is keyed under the empty string")
			name = a.Metadata.DisplayName
		}
		for _, field := range []struct{ label, value string }{
			{"metadata.displayName", a.Metadata.DisplayName},
			{"metadata.category", a.Metadata.Category},
			{"spec.subtitle", a.Spec.Subtitle},
			{"spec.description", a.Spec.Description},
			{"spec.docsUrl", a.Spec.DocsURL},
		} {
			if field.value == "" {
				t.Errorf("action %q has an empty %s", name, field.label)
			}
		}
	}
}

// The UI builds the allowed-signal list by flattening these four booleans. An action with
// none of them supported is offered in the picker but can never be applied to any signal.
func TestEveryActionSupportsAtLeastOneSignal(t *testing.T) {
	loaded := acLoadCatalog(t)

	for _, a := range loaded {
		s := a.Spec.Signals
		if !s.Traces.Supported && !s.Metrics.Supported && !s.Logs.Supported && !s.Profiles.Supported {
			t.Errorf("action %q supports no signal at all", a.Metadata.Type)
		}
	}
}

// Both values are spelled out rather than read from the catalog so that changing them in the
// YAML is what fails, not the comparison.
func TestEveryActionDeclaresTheSharedApiVersionAndKind(t *testing.T) {
	loaded := acLoadCatalog(t)

	for _, a := range loaded {
		if a.ApiVersion != "internal.odigos.io/v1beta1" {
			t.Errorf("action %q has apiVersion %q, want %q", a.Metadata.Type, a.ApiVersion, "internal.odigos.io/v1beta1")
		}
		if a.Kind != "Action" {
			t.Errorf("action %q has kind %q, want %q", a.Metadata.Type, a.Kind, "Action")
		}
	}
}

// A field with no name cannot be mapped back onto the Action CR spec, and one with no
// componentType has nothing for the UI to render.
func TestEveryFieldTheUiRendersHasANameAndComponentType(t *testing.T) {
	loaded := acLoadCatalog(t)

	fields := 0
	for _, a := range loaded {
		seen := map[string]bool{}
		for i, f := range a.Spec.Fields {
			fields++
			if f.Name == "" {
				t.Errorf("action %q field %d has an empty name", a.Metadata.Type, i)
			}
			if f.ComponentType == "" {
				t.Errorf("action %q field %q has an empty componentType", a.Metadata.Type, f.Name)
			}
			if seen[f.Name] {
				t.Errorf("action %q declares field %q twice", a.Metadata.Type, f.Name)
			}
			seen[f.Name] = true
		}
	}
	if fields == 0 {
		t.Fatal("no catalog action declares a field; this test would pass vacuously")
	}
}

// initialValue is typed as a string but carries the serialized default for the structured
// components, which the UI parses as JSON before rendering the form. A malformed default
// breaks the form for that action only, and only once a user opens it.
func TestStructuredFieldDefaultsAreValidJson(t *testing.T) {
	loaded := acLoadCatalog(t)

	structured := map[string]bool{"checkboxList": true, "multiTabledInput": true}

	checked := 0
	for _, a := range loaded {
		for _, f := range a.Spec.Fields {
			if !structured[f.ComponentType] || f.InitialValue == "" {
				continue
			}
			checked++
			var parsed any
			if err := json.Unmarshal([]byte(f.InitialValue), &parsed); err != nil {
				t.Errorf("action %q field %q has an initialValue that is not valid JSON: %v",
					a.Metadata.Type, f.Name, err)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no structured field declares an initialValue; this test would pass vacuously")
	}
}

// The checkbox ids are the values written into the Action CR, so a blank or duplicated id
// silently drops or collapses a user's selection.
func TestCheckboxListOptionsHaveUniqueNonEmptyIds(t *testing.T) {
	loaded := acLoadCatalog(t)

	lists := 0
	for _, a := range loaded {
		for _, f := range a.Spec.Fields {
			if f.ComponentType != "checkboxList" {
				continue
			}
			rawOptions, ok := f.ComponentProps["options"].([]any)
			if !ok {
				t.Errorf("action %q checkboxList field %q declares no options list", a.Metadata.Type, f.Name)
				continue
			}
			lists++
			seen := map[string]bool{}
			for i, rawOption := range rawOptions {
				option, ok := rawOption.(map[string]any)
				if !ok {
					t.Errorf("action %q field %q option %d is not a mapping", a.Metadata.Type, f.Name, i)
					continue
				}
				id, _ := option["id"].(string)
				if id == "" {
					t.Errorf("action %q field %q option %d has an empty id", a.Metadata.Type, f.Name, i)
					continue
				}
				if seen[id] {
					t.Errorf("action %q field %q declares option id %q twice", a.Metadata.Type, f.Name, id)
				}
				seen[id] = true
			}
		}
	}
	if lists == 0 {
		t.Fatal("no catalog action declares a checkboxList field; this test would pass vacuously")
	}
}
