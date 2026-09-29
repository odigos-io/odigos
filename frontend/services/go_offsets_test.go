package services

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/odigos-io/odigos/frontend/graph/model"
)

func TestParseGoOffsetsContent_jsonString(t *testing.T) {
	inner := `{"timestamp":"2026-07-29T00:16:13.51429777Z","mods":[{"module":"example.com/mod","packages":[{"package":"example.com/mod/pkg","structs":[{"struct":"Foo","fields":[{"field":"Bar","offsets":[{"offset":8,"versions":["1.0.0","1.1.0"]},{"offset":16,"versions":["1.1.0","1.2.0"]}]}]}]}]}]}`
	wrapped, err := json.Marshal(inner)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	parsed, err := parseGoOffsetsContent(string(wrapped))
	if err != nil {
		t.Fatalf("parseGoOffsetsContent: %v", err)
	}
	model := goOffsetsToModel(parsed)
	if len(model.Mods) != 1 || model.Mods[0].Module != "example.com/mod" {
		t.Fatalf("unexpected module: %+v", model.Mods)
	}
	if model.Mods[0].MinVersion != "1.0.0" {
		t.Fatalf("unexpected minVersion: %q", model.Mods[0].MinVersion)
	}
	if model.Mods[0].MaxVersion != "1.2.0" {
		t.Fatalf("unexpected maxVersion: %q", model.Mods[0].MaxVersion)
	}
	wantMinors := []string{"1.2", "1.1", "1.0"}
	if len(model.Mods[0].MinorVersions) != len(wantMinors) {
		t.Fatalf("unexpected minorVersions: %+v", model.Mods[0].MinorVersions)
	}
	for i, want := range wantMinors {
		if model.Mods[0].MinorVersions[i].MinorVersion != want {
			t.Fatalf("unexpected minorVersions[%d]: got %q want %q", i, model.Mods[0].MinorVersions[i].MinorVersion, want)
		}
	}
	if parsed.Timestamp.UTC() != time.Date(2026, 7, 29, 0, 16, 13, 514297770, time.UTC) {
		t.Fatalf("unexpected timestamp: %v", parsed.Timestamp)
	}
}

func TestParseGoOffsetsContent_jsonStringWithSignature(t *testing.T) {
	inner := `{"timestamp":"2026-07-29T00:16:13.51429777Z","mods":[]}---SIGNATURE---abc`
	wrapped, err := json.Marshal(inner)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	parsed, err := parseGoOffsetsContent(string(wrapped))
	if err != nil {
		t.Fatalf("parseGoOffsetsContent: %v", err)
	}
	if len(parsed.Mods) != 0 {
		t.Fatalf("expected empty mods")
	}
}

func TestParseGoOffsetsContent_empty(t *testing.T) {
	parsed, err := parseGoOffsetsContent("  ")
	if err != nil {
		t.Fatalf("parseGoOffsetsContent: %v", err)
	}
	if len(parsed.Mods) != 0 {
		t.Fatalf("expected empty mods")
	}
}

func TestCompareGoOffsets_newVersionsAndModule(t *testing.T) {
	current := mustParseOffsetsFile(t, `{"timestamp":"2026-07-29T00:16:13.51429777Z","mods":[{"module":"example.com/mod","packages":[{"package":"example.com/mod/pkg","structs":[{"struct":"Foo","fields":[{"field":"Bar","offsets":[{"offset":8,"versions":["1.0.0","1.1.0"]}]}]}]}]}]}`)
	proposed := mustParseOffsetsFile(t, `{"timestamp":"2026-07-30T00:00:00Z","mods":[{"module":"example.com/mod","packages":[{"package":"example.com/mod/pkg","structs":[{"struct":"Foo","fields":[{"field":"Bar","offsets":[{"offset":8,"versions":["1.0.0","1.1.0","1.2.0"]}]}]}]}]},{"module":"example.com/new","packages":[{"package":"example.com/new/pkg","structs":[{"struct":"Baz","fields":[{"field":"Q","offsets":[{"offset":0,"versions":["2.0.0"]}]}]}]}]}]}`)

	result := compareGoOffsets(current, proposed)
	if !result.HasUpdates {
		t.Fatalf("expected hasUpdates")
	}
	if len(result.Mods) != 2 {
		t.Fatalf("unexpected mods: %+v", result.Mods)
	}
	if result.Mods[0].Module != "example.com/mod" || result.Mods[0].IsNew {
		t.Fatalf("unexpected first module: %+v", result.Mods[0])
	}
	foundNew := false
	for _, minor := range result.Mods[0].MinorVersions {
		for _, ver := range minor.Versions {
			if ver.Version == "1.2.0" {
				foundNew = true
				if !ver.IsNew {
					t.Fatalf("1.2.0 should be marked new")
				}
			}
			if ver.Version == "1.0.0" && ver.IsNew {
				t.Fatalf("1.0.0 should not be new")
			}
		}
	}
	if !foundNew {
		t.Fatalf("expected 1.2.0 in first module")
	}
	if result.Mods[1].Module != "example.com/new" || !result.Mods[1].IsNew {
		t.Fatalf("unexpected second module: %+v", result.Mods[1])
	}
}

func TestCompareGoOffsets_noUpdates(t *testing.T) {
	current := mustParseOffsetsFile(t, `{"timestamp":"2026-07-29T00:16:13.51429777Z","mods":[{"module":"example.com/mod","packages":[{"package":"example.com/mod/pkg","structs":[{"struct":"Foo","fields":[{"field":"Bar","offsets":[{"offset":8,"versions":["1.0.0"]}]}]}]}]}]}`)
	proposed := mustParseOffsetsFile(t, `{"timestamp":"2026-07-30T00:00:00Z","mods":[{"module":"example.com/mod","packages":[{"package":"example.com/mod/pkg","structs":[{"struct":"Foo","fields":[{"field":"Bar","offsets":[{"offset":8,"versions":["1.0.0"]}]}]}]}]}]}`)

	result := compareGoOffsets(current, proposed)
	if result.HasUpdates {
		t.Fatalf("expected no updates")
	}
	if len(result.Mods) != 1 || result.Mods[0].IsNew {
		t.Fatalf("unexpected mods: %+v", result.Mods)
	}
	if result.Mods[0].MinorVersions[0].Versions[0].IsNew {
		t.Fatalf("existing version should not be new")
	}
	if result.Mods[0].IsRemoved || result.Mods[0].MinorVersions[0].Versions[0].IsRemoved {
		t.Fatalf("unchanged entries should not be removed")
	}
}

// An update overwrites the ConfigMap, so entries the candidate omits are
// dropped — they must be reported even when nothing is added.
func TestCompareGoOffsets_removedVersionsAndModule(t *testing.T) {
	current := mustParseOffsetsFile(t, `{"timestamp":"2026-07-29T00:16:13.51429777Z","mods":[{"module":"example.com/mod","packages":[{"package":"example.com/mod/pkg","structs":[{"struct":"Foo","fields":[{"field":"Bar","offsets":[{"offset":8,"versions":["1.0.0","1.1.0"]}]}]}]}]},{"module":"example.com/gone","packages":[{"package":"example.com/gone/pkg","structs":[{"struct":"Baz","fields":[{"field":"Q","offsets":[{"offset":0,"versions":["2.0.0"]}]}]}]}]}]}`)
	proposed := mustParseOffsetsFile(t, `{"timestamp":"2026-07-30T00:00:00Z","mods":[{"module":"example.com/mod","packages":[{"package":"example.com/mod/pkg","structs":[{"struct":"Foo","fields":[{"field":"Bar","offsets":[{"offset":8,"versions":["1.0.0"]}]}]}]}]}]}`)

	result := compareGoOffsets(current, proposed)
	if !result.HasUpdates {
		t.Fatalf("expected hasUpdates for a removal-only diff")
	}

	byModule := make(map[string]*model.GoOffsetModuleUpdate, len(result.Mods))
	for _, mod := range result.Mods {
		byModule[mod.Module] = mod
	}

	kept := byModule["example.com/mod"]
	if kept == nil || kept.IsRemoved {
		t.Fatalf("example.com/mod should be kept: %+v", kept)
	}
	if kept.MaxVersion != "1.0.0" {
		t.Fatalf("maxVersion should drop to the post-update range, got %q", kept.MaxVersion)
	}
	for _, minor := range kept.MinorVersions {
		for _, ver := range minor.Versions {
			wantRemoved := ver.Version == "1.1.0"
			if ver.IsRemoved != wantRemoved {
				t.Fatalf("%s isRemoved = %v, want %v", ver.Version, ver.IsRemoved, wantRemoved)
			}
			if ver.IsNew {
				t.Fatalf("%s should not be new", ver.Version)
			}
		}
	}

	gone := byModule["example.com/gone"]
	if gone == nil || !gone.IsRemoved {
		t.Fatalf("example.com/gone should be removed: %+v", gone)
	}
	if !gone.MinorVersions[0].IsRemoved || !gone.MinorVersions[0].Versions[0].IsRemoved {
		t.Fatalf("removed module entries should cascade: %+v", gone.MinorVersions[0])
	}
}

func mustParseOffsetsFile(t *testing.T, content string) *versionedModules {
	t.Helper()
	parsed, err := parseGoOffsetsFile(content)
	if err != nil {
		t.Fatalf("parseGoOffsetsFile: %v", err)
	}
	return parsed
}

// A deleted ConfigMap leaves CheckGoOffsetsUpdates with an empty installed set,
// which has to read as "everything is new" so the update can put it back.
func TestCompareGoOffsets_emptyCurrentMarksEverythingNew(t *testing.T) {
	current := &versionedModules{Mods: []*jsonModule{}}
	proposed := mustParseOffsetsFile(t, `{"timestamp":"2026-07-30T00:00:00Z","mods":[{"module":"example.com/mod","packages":[{"package":"example.com/mod/pkg","structs":[{"struct":"Foo","fields":[{"field":"Bar","offsets":[{"offset":8,"versions":["1.0.0","1.1.0"]}]}]}]}]}]}`)

	result := compareGoOffsets(current, proposed)
	if !result.HasUpdates {
		t.Fatalf("expected hasUpdates")
	}
	if len(result.Mods) != 1 || !result.Mods[0].IsNew {
		t.Fatalf("expected the module to be new: %+v", result.Mods)
	}
	if result.Mods[0].IsRemoved {
		t.Fatalf("nothing is removed when there is no installed manifest")
	}
	for _, minor := range result.Mods[0].MinorVersions {
		for _, ver := range minor.Versions {
			if !ver.IsNew {
				t.Fatalf("version %q should be marked new", ver.Version)
			}
		}
	}
}

func TestEncodeGoOffsets(t *testing.T) {
	// Empty input clears the key rather than writing the JSON string `""`, which
	// is what the chart writes for a placeholder ConfigMap.
	encoded, err := encodeGoOffsets(nil)
	if err != nil {
		t.Fatalf("encodeGoOffsets(nil): %v", err)
	}
	if encoded != "" {
		t.Fatalf("expected empty string, got %q", encoded)
	}

	// Non-empty input is stored JSON-encoded, matching `odigos pro update-offsets`.
	encoded, err = encodeGoOffsets([]byte(`{"timestamp":"2026-07-30T00:00:00Z"}`))
	if err != nil {
		t.Fatalf("encodeGoOffsets: %v", err)
	}
	var decoded string
	if err := json.Unmarshal([]byte(encoded), &decoded); err != nil {
		t.Fatalf("stored value is not a JSON string: %v", err)
	}
	if decoded != `{"timestamp":"2026-07-30T00:00:00Z"}` {
		t.Fatalf("round-trip mismatch: %q", decoded)
	}
}
