package destinations

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
)

// load() writes package globals, so any test that drives it with a synthetic filesystem has to
// put the embedded catalog back for the tests that read it.
func dcLoadFrom(t *testing.T, files map[string]string) error {
	t.Helper()
	t.Cleanup(func() {
		if err := Load(); err != nil {
			t.Fatalf("restoring the embedded catalog: %v", err)
		}
	})
	fsys := fstest.MapFS{}
	for name, body := range files {
		fsys[name] = &fstest.MapFile{Data: []byte(body)}
	}
	return load(fsys)
}

func dcEntry(destType string) string {
	return `apiVersion: internal.odigos.io/v1beta1
kind: Destination
metadata:
  type: ` + destType + `
  displayName: ` + destType + `
  category: managed
spec:
  image: ` + destType + `.svg
  signals:
    traces:
      supported: true
`
}

func dcTypes(t *testing.T) []string {
	t.Helper()
	var types []string
	for _, d := range Get() {
		types = append(types, string(d.Metadata.Type))
	}
	sort.Strings(types)
	return types
}

func TestLoadReadsYamlAndYmlFilesAndSkipsEverythingElse(t *testing.T) {
	err := dcLoadFrom(t, map[string]string{
		"data/alpha.yaml":    dcEntry("alpha"),
		"data/bravo.yml":     dcEntry("bravo"),
		"data/charlie.json":  dcEntry("charlie"),
		"data/delta.yaml.ba": dcEntry("delta"),
		"data/README.md":     "not a destination",
	})
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if got, want := dcTypes(t), []string{"alpha", "bravo"}; !reflect.DeepEqual(got, want) {
		t.Errorf("loaded types = %v, want %v", got, want)
	}
	// the skipped files must not be reachable through the lookup either
	for _, skipped := range []string{"charlie", "delta", "README"} {
		if _, ok := GetDestinationByType(skipped); ok {
			t.Errorf("GetDestinationByType(%q) resolved, but its file is not a .yaml/.yml", skipped)
		}
	}
}

func TestLoadFailsWhenTheDataDirectoryIsMissing(t *testing.T) {
	if err := dcLoadFrom(t, map[string]string{"alpha.yaml": dcEntry("alpha")}); err == nil {
		t.Fatal("load with no data/ directory returned no error")
	}
}

// A failed reload must not take the previously loaded catalog down with it: load() assigns the
// globals only after every file has parsed, so the gateway keeps configuring destinations with
// the last good catalog instead of seeing an empty one.
func TestLoadFailsOnMalformedYamlAndKeepsThePreviousCatalog(t *testing.T) {
	if err := dcLoadFrom(t, map[string]string{"data/alpha.yaml": dcEntry("alpha")}); err != nil {
		t.Fatalf("seeding the catalog: %v", err)
	}

	// the good file sorts first and declares a type the seeded catalog does not have, so a load
	// that published entries as it went would be visible instead of coinciding with the seed
	err := load(fstest.MapFS{
		"data/a-charlie.yaml": &fstest.MapFile{Data: []byte(dcEntry("charlie"))},
		"data/z-broken.yaml": &fstest.MapFile{Data: []byte(`metadata:
  type: broken
   displayName: bad indentation
`)},
	})
	if err == nil {
		t.Fatal("load of a malformed destination file returned no error")
	}

	if got, want := dcTypes(t), []string{"alpha"}; !reflect.DeepEqual(got, want) {
		t.Errorf("after a failed load the catalog is %v, want the previous %v", got, want)
	}
	if _, ok := GetDestinationByType("alpha"); !ok {
		t.Error("after a failed load the previous catalog is no longer resolvable by type")
	}
	for _, destType := range []string{"broken", "charlie"} {
		if _, ok := GetDestinationByType(destType); ok {
			t.Errorf("%q came from a load that failed, but it is resolvable", destType)
		}
	}
}

// load() decides what to parse from the directory entry's name alone and never asks whether it is
// a file, so a directory under data/ whose name ends in .yaml fails the whole catalog load rather
// than being skipped. Pinned as the behaviour today: the assertion that matters either way is
// that a file it cannot read is an error rather than a silently shorter catalog.
func TestLoadFailsWhenADataEntryCannotBeRead(t *testing.T) {
	t.Cleanup(func() {
		if err := Load(); err != nil {
			t.Fatalf("restoring the embedded catalog: %v", err)
		}
	})
	err := load(fstest.MapFS{
		"data/alpha.yaml":            &fstest.MapFile{Data: []byte(dcEntry("alpha"))},
		"data/nested.yaml/inner.txt": &fstest.MapFile{Data: []byte("not a destination")},
	})
	if err == nil {
		t.Fatal("load returned no error for a data/ entry it could not read")
	}
	if _, ok := GetDestinationByType("alpha"); ok {
		t.Error("a destination from a failed load became resolvable")
	}
}

func TestLoadReplacesThePreviousCatalogRatherThanAddingToIt(t *testing.T) {
	if err := dcLoadFrom(t, map[string]string{
		"data/alpha.yaml": dcEntry("alpha"),
		"data/bravo.yaml": dcEntry("bravo"),
	}); err != nil {
		t.Fatalf("seeding the catalog: %v", err)
	}

	if err := load(fstest.MapFS{"data/charlie.yaml": &fstest.MapFile{Data: []byte(dcEntry("charlie"))}}); err != nil {
		t.Fatalf("second load: %v", err)
	}

	if got, want := dcTypes(t), []string{"charlie"}; !reflect.DeepEqual(got, want) {
		t.Errorf("after reloading, the catalog is %v, want %v", got, want)
	}
	if _, ok := GetDestinationByType("alpha"); ok {
		t.Error("a destination from the replaced catalog is still resolvable by type")
	}
}

// Two files declaring the same metadata.type leave both rows in Get() while the lookup keeps only
// one of them, so the UI would list a duplicate that resolves to a single configuration. Pinned
// because it is the behaviour today, not because it is desirable.
func TestADuplicateDestinationTypeCollapsesTheLookupButStaysTwiceInTheList(t *testing.T) {
	first := dcEntry("duplicated")
	second := strings.Replace(dcEntry("duplicated"), "image: duplicated.svg", "image: second.svg", 1)
	if err := dcLoadFrom(t, map[string]string{
		"data/a-duplicated.yaml": first,
		"data/b-duplicated.yaml": second,
	}); err != nil {
		t.Fatalf("load: %v", err)
	}

	if got := len(Get()); got != 2 {
		t.Errorf("Get() returned %d entries, want both duplicates", got)
	}
	resolved, ok := GetDestinationByType("duplicated")
	if !ok {
		t.Fatal("GetDestinationByType did not resolve the duplicated type")
	}
	// data/ is read in sorted order, so the later file wins the lookup
	if resolved.Spec.Image != "second.svg" {
		t.Errorf("the lookup resolved to image %q, want the last file read (second.svg)", resolved.Spec.Image)
	}
}

// Every one of the four support flags is read off an untagged, anonymous struct field whose yaml
// key comes from the lowercased Go field name, so a rename or a cross-wired key silently reports
// a signal as unsupported with no compile error. One fixture per signal, each the only one true.
func TestEachSignalSupportFlagIsParsedOnItsOwn(t *testing.T) {
	signals := []string{"traces", "metrics", "logs", "profiles"}
	for _, only := range signals {
		t.Run(only, func(t *testing.T) {
			body := `apiVersion: internal.odigos.io/v1beta1
kind: Destination
metadata:
  type: probe
spec:
  signals:
`
			for _, s := range signals {
				body += "    " + s + ":\n      supported: " + map[bool]string{true: "true", false: "false"}[s == only] + "\n"
			}
			if err := dcLoadFrom(t, map[string]string{"data/probe.yaml": body}); err != nil {
				t.Fatalf("load: %v", err)
			}
			d, ok := GetDestinationByType("probe")
			if !ok {
				t.Fatal("probe destination not loaded")
			}
			got := map[string]bool{
				"traces":   d.Spec.Signals.Traces.Supported,
				"metrics":  d.Spec.Signals.Metrics.Supported,
				"logs":     d.Spec.Signals.Logs.Supported,
				"profiles": d.Spec.Signals.Profiles.Supported,
			}
			want := map[string]bool{"traces": false, "metrics": false, "logs": false, "profiles": false}
			want[only] = true
			if !reflect.DeepEqual(got, want) {
				t.Errorf("with only %s supported the parsed flags are %v, want %v", only, got, want)
			}
		})
	}
}

// The whole Destination envelope is what the UI renders a destination form from. Parse a file with
// every property populated and require that no field came back zero, so a property added to the
// structs without a yaml tag (or with a typo'd one) cannot go unnoticed.
func TestEveryDestinationPropertyIsParsedFromYaml(t *testing.T) {
	body := `apiVersion: internal.odigos.io/v1beta1
kind: Destination
metadata:
  type: probe
  displayName: Probe
  category: managed
spec:
  image: probe.svg
  signals:
    traces:
      supported: true
    metrics:
      supported: true
    logs:
      supported: true
    profiles:
      supported: true
  testConnectionSupported: true
  fields:
    - name: PROBE_TOKEN
      displayName: Token
      componentType: input
      componentProps:
        type: password
        required: true
      secret: true
      initialValue: seed
      renderCondition: ["PROBE_MODE", "==", "advanced"]
      hideFromReadData: ["PROBE_MODE"]
      customReadDataLabels:
        - condition: PROBE_MODE
          title: Mode
          value: advanced
`
	if err := dcLoadFrom(t, map[string]string{"data/probe.yaml": body}); err != nil {
		t.Fatalf("load: %v", err)
	}
	d, ok := GetDestinationByType("probe")
	if !ok {
		t.Fatal("probe destination not loaded")
	}

	dcRequireNoZeroField(t, "Destination", reflect.ValueOf(d))
	dcRequireNoZeroField(t, "Metadata", reflect.ValueOf(d.Metadata))
	dcRequireNoZeroField(t, "Spec", reflect.ValueOf(d.Spec))
	if len(d.Spec.Fields) != 1 {
		t.Fatalf("parsed %d fields, want 1", len(d.Spec.Fields))
	}
	dcRequireNoZeroField(t, "Field", reflect.ValueOf(d.Spec.Fields[0]))
	if len(d.Spec.Fields[0].CustomReadDataLabels) != 1 {
		t.Fatalf("parsed %d custom read data labels, want 1", len(d.Spec.Fields[0].CustomReadDataLabels))
	}
	dcRequireNoZeroField(t, "CustomReadDataLabel", reflect.ValueOf(*d.Spec.Fields[0].CustomReadDataLabels[0]))

	// spot-check the values themselves, so the reflection sweep above cannot be satisfied by two
	// properties that were parsed into each other's field
	f := d.Spec.Fields[0]
	if f.Name != "PROBE_TOKEN" || f.DisplayName != "Token" || f.ComponentType != "input" || f.InitialValue != "seed" {
		t.Errorf("field parsed as %+v", f)
	}
	if got := f.ComponentProps["type"]; got != "password" {
		t.Errorf("componentProps[type] = %v, want password", got)
	}
	if !reflect.DeepEqual(f.RenderCondition, []string{"PROBE_MODE", "==", "advanced"}) {
		t.Errorf("renderCondition = %v", f.RenderCondition)
	}
	if !reflect.DeepEqual(f.HideFromReadData, []string{"PROBE_MODE"}) {
		t.Errorf("hideFromReadData = %v", f.HideFromReadData)
	}
}

func dcRequireNoZeroField(t *testing.T, name string, v reflect.Value) {
	t.Helper()
	if v.NumField() == 0 {
		t.Fatalf("%s has no fields to check", name)
	}
	for i := 0; i < v.NumField(); i++ {
		if !v.Type().Field(i).IsExported() {
			continue
		}
		if v.Field(i).IsZero() {
			t.Errorf("%s.%s was not parsed from the yaml", name, v.Type().Field(i).Name)
		}
	}
}

func TestTheEmbeddedCatalogLoads(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(Get()) == 0 {
		t.Fatal("the embedded catalog is empty")
	}
}

// Every property here is read by frontend/services.DestinationTypeConfigToCategoryItem to build
// the destination picker, so an entry missing one of them renders as a blank row.
func TestEveryEmbeddedEntryHasTheEnvelopeTheUIReads(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, d := range Get() {
		// spelled out rather than taken from a shared constant: the point is to notice a change
		// to the values the catalog files are written with
		if d.ApiVersion != "internal.odigos.io/v1beta1" {
			t.Errorf("%s: apiVersion = %q", d.Metadata.Type, d.ApiVersion)
		}
		if d.Kind != "Destination" {
			t.Errorf("%s: kind = %q", d.Metadata.Type, d.Kind)
		}
		if d.Metadata.Type == "" {
			t.Errorf("a catalog entry has no metadata.type (displayName %q)", d.Metadata.DisplayName)
		}
		if d.Metadata.DisplayName == "" {
			t.Errorf("%s: no displayName", d.Metadata.Type)
		}
		if d.Metadata.Category != "managed" && d.Metadata.Category != "self hosted" {
			t.Errorf("%s: category %q is neither of the two the UI groups by", d.Metadata.Type, d.Metadata.Category)
		}
		if d.Spec.Image == "" {
			t.Errorf("%s: no image", d.Metadata.Type)
		}
	}
}

func TestNoTwoEmbeddedEntriesShareADestinationType(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	seen := map[string]bool{}
	for _, d := range Get() {
		if seen[string(d.Metadata.Type)] {
			t.Errorf("destination type %q is declared by more than one catalog file", d.Metadata.Type)
		}
		seen[string(d.Metadata.Type)] = true
	}
	if len(seen) != len(Get()) {
		t.Errorf("%d catalog entries resolve to %d distinct types", len(Get()), len(seen))
	}
}

func TestGetDestinationByTypeResolvesEveryEmbeddedEntryToItself(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, d := range Get() {
		got, ok := GetDestinationByType(string(d.Metadata.Type))
		if !ok {
			t.Errorf("%s is in Get() but GetDestinationByType does not resolve it", d.Metadata.Type)
			continue
		}
		if !reflect.DeepEqual(got, d) {
			t.Errorf("%s resolves to a different entry than the one Get() lists", d.Metadata.Type)
		}
	}
}

// GetDestinationByType is an exact map lookup and its callers ignore the bool
// (K8sDestinationToEndpointFormat discards it), so a near-miss type silently produces a
// zero-valued destination type config instead of an error.
func TestGetDestinationByTypeRejectsLookalikeTypes(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, ok := GetDestinationByType("honeycomb"); !ok {
		t.Fatal("honeycomb is not in the catalog; pick another anchor for this test")
	}
	for _, lookalike := range []string{
		"Honeycomb",             // wrong case
		"honeycomb ",            // trailing space
		"honeycombs",            // plural
		"honey-comb",            // hyphenated
		"honey_comb",            // underscored
		"honeycomb.yaml",        // the file name instead of the type
		"googlecloudmonitoring", // a catalog file name whose metadata.type is googlecloud
		"",                      // no type at all
	} {
		if _, ok := GetDestinationByType(lookalike); ok {
			t.Errorf("GetDestinationByType(%q) resolved", lookalike)
		}
	}
}

func TestEveryEmbeddedEntryNamesEachFieldOnce(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	fields := 0
	for _, d := range Get() {
		seen := map[string]bool{}
		for _, f := range d.Spec.Fields {
			if f.Name == "" {
				t.Errorf("%s: a field has no name", d.Metadata.Type)
			}
			if seen[f.Name] {
				t.Errorf("%s: field %q is declared twice, so TransformFieldsToDataAndSecrets cannot tell which one is the secret", d.Metadata.Type, f.Name)
			}
			seen[f.Name] = true
			fields++
		}
	}
	if fields == 0 {
		t.Fatal("no destination field was checked")
	}
}

func TestEveryEmbeddedEntryDeclaresAtLeastOneSupportedSignal(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, d := range Get() {
		s := d.Spec.Signals
		if !s.Traces.Supported && !s.Metrics.Supported && !s.Logs.Supported && !s.Profiles.Supported {
			t.Errorf("%s declares no supported signal, so it can be created but never exports anything", d.Metadata.Type)
		}
	}
}
