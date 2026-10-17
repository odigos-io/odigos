package actions

import (
	"embed"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"gopkg.in/yaml.v3"
)

// acEmbedWithoutDataDir is an embed.FS rooted at this package that deliberately does not
// contain the data/ directory, so load cannot read the catalog from it.
//
//go:embed model.go
var acEmbedWithoutDataDir embed.FS

// acResetCatalog clears the package-level catalog and restores it when the test ends.
// Nothing reads these globals from a goroutine, so a Cleanup restore is safe here.
func acResetCatalog(t *testing.T) {
	t.Helper()
	prevActions, prevByType := loadedActions, actionsByType
	t.Cleanup(func() {
		loadedActions, actionsByType = prevActions, prevByType
	})
	loadedActions, actionsByType = nil, nil
}

// acLoadCatalog loads the real embedded catalog and returns it, so every test in this
// package is independent of the order the others ran in.
func acLoadCatalog(t *testing.T) []Action {
	t.Helper()
	if err := Load(); err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}
	loaded := Get()
	if len(loaded) == 0 {
		t.Fatal("Load() succeeded but the catalog is empty")
	}
	return loaded
}

// acCatalogFileNames returns the names of the catalog files the loader is expected to pick up,
// derived from the embedded directory rather than a hardcoded count, so adding an action does
// not require editing this test.
func acCatalogFileNames(t *testing.T) []string {
	t.Helper()
	entries, err := actionsFS.ReadDir("data")
	if err != nil {
		t.Fatalf("reading the embedded data/ directory: %v", err)
	}
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml") {
			names = append(names, e.Name())
		}
	}
	return names
}

// An embedder that builds the instrumentor or the UI server without calling Load gets a nil
// catalog rather than an error, and k8sutils/pkg/action then reports every config-extension
// action as "not a config extension". That is what made the missing Load call in the
// enterprise instrumentor latent until CORE-1724 moved it into instrumentor.New.
func TestTheCatalogIsEmptyUntilLoadIsCalled(t *testing.T) {
	acResetCatalog(t)

	if got := Get(); got != nil {
		t.Errorf("Get() before Load() = %v, want nil", got)
	}

	// PiiMasking is a config-extension action: resolving it is what decides whether the
	// odigospiimasking processor is ever synthesized for the collector.
	action, ok := GetActionByType("PiiMasking")
	if ok {
		t.Error("GetActionByType(\"PiiMasking\") before Load() reported the action as found")
	}
	if action.Metadata.Type != "" {
		t.Errorf("GetActionByType before Load() returned %+v, want the zero Action", action)
	}
}

func TestLoadPopulatesOneCatalogEntryPerFile(t *testing.T) {
	loaded := acLoadCatalog(t)

	wantFiles := acCatalogFileNames(t)
	if len(loaded) != len(wantFiles) {
		t.Errorf("Load() produced %d actions from %d catalog files (%v); every file must yield exactly one action",
			len(loaded), len(wantFiles), wantFiles)
	}
}

// actionsByType is built by assigning into a map keyed on metadata.type, so two catalog files
// sharing a type would silently drop one of them while Get() still reported both.
func TestCatalogTypesAreUniqueSoNoEntryIsSilentlyOverwritten(t *testing.T) {
	loaded := acLoadCatalog(t)

	seen := map[string]int{}
	for _, a := range loaded {
		seen[a.Metadata.Type]++
	}
	for actionType, count := range seen {
		if count > 1 {
			t.Errorf("action type %q is declared by %d catalog files; the type-keyed lookup keeps only one of them",
				actionType, count)
		}
	}

	if len(actionsByType) != len(loaded) {
		t.Errorf("the type lookup holds %d entries but the catalog holds %d actions; an entry was overwritten",
			len(actionsByType), len(loaded))
	}
}

func TestEveryLoadedActionIsReachableByItsType(t *testing.T) {
	loaded := acLoadCatalog(t)

	for _, want := range loaded {
		got, ok := GetActionByType(want.Metadata.Type)
		if !ok {
			t.Errorf("GetActionByType(%q) did not find an action that Get() returned", want.Metadata.Type)
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("GetActionByType(%q) returned a different value than Get() did", want.Metadata.Type)
		}
	}
}

// CatalogType in k8sutils/pkg/action returns "" for an Action CR with no recognised config
// field, and that empty string is looked up here directly. A catalog entry that parsed with a
// missing metadata.type would be keyed under "" and would then be returned for every such CR.
func TestGetActionByTypeRejectsTheEmptyTypeAndIsCaseSensitive(t *testing.T) {
	acLoadCatalog(t)

	if _, ok := GetActionByType(""); ok {
		t.Error("GetActionByType(\"\") found an action; an unconfigured Action CR would resolve to it")
	}
	// Action types are matched verbatim against the CRD-derived name, so a differently cased
	// catalog entry would silently stop resolving.
	if _, ok := GetActionByType("piimasking"); ok {
		t.Error("GetActionByType(\"piimasking\") matched case-insensitively, want an exact match only")
	}
	if _, ok := GetActionByType("NoSuchActionType"); ok {
		t.Error("GetActionByType reported an unknown action type as found")
	}
}

// Load replaces the catalog rather than appending to it; both the frontend server and the
// instrumentor call it during startup, and a re-entrant embedder may call it more than once.
func TestLoadIsIdempotent(t *testing.T) {
	first := acLoadCatalog(t)
	firstCopy := append([]Action(nil), first...)
	firstLookupSize := len(actionsByType)

	second := acLoadCatalog(t)

	if len(second) != len(firstCopy) {
		t.Fatalf("second Load() produced %d actions, first produced %d", len(second), len(firstCopy))
	}
	if len(actionsByType) != firstLookupSize {
		t.Errorf("second Load() left %d lookup entries, first left %d", len(actionsByType), firstLookupSize)
	}
	if !reflect.DeepEqual(second, firstCopy) {
		t.Error("two consecutive Load() calls produced different catalogs")
	}
}

func TestLoadReportsAnErrorWhenTheCatalogDirectoryIsMissing(t *testing.T) {
	acResetCatalog(t)

	if err := load(acEmbedWithoutDataDir); err == nil {
		t.Fatal("load() on a filesystem without a data/ directory returned no error")
	}

	// A failed load must not leave a half-populated catalog behind.
	if Get() != nil {
		t.Errorf("Get() after a failed load() = %v, want nil", Get())
	}
}

// acCatalogFS builds a synthetic catalog directory; files is a name -> file body mapping.
func acCatalogFS(files map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for name, body := range files {
		fsys["data/"+name] = &fstest.MapFile{Data: []byte(body)}
	}
	return fsys
}

func acActionYaml(actionType string) string {
	return "apiVersion: internal.odigos.io/v1beta1\nkind: Action\nmetadata:\n  type: " + actionType +
		"\n  displayName: " + actionType + "\n  category: enrichment\nspec:\n  subtitle: s\n  description: d\n" +
		"  docsUrl: https://example.invalid\n  signals:\n    traces:\n      supported: true\n"
}

// Both the instrumentor and the UI server treat a load failure as fatal, so one unparsable
// catalog file must stop the whole load rather than quietly shipping a catalog that is missing
// an action - a missing config-extension entry disables the processor with no error.
func TestLoadRefusesTheWholeCatalogWhenOneFileIsMalformed(t *testing.T) {
	acResetCatalog(t)

	fsys := acCatalogFS(map[string]string{
		"good.yaml": acActionYaml("Good"),
		"bad.yaml":  "metadata:\n\ttype: [unbalanced\n",
	})

	if err := load(fsys); err == nil {
		t.Fatal("load() accepted a catalog containing an unparsable file")
	}
	if Get() != nil {
		t.Errorf("a failed load() published %v, want the catalog left untouched", Get())
	}
	if _, ok := GetActionByType("Good"); ok {
		t.Error("a failed load() published the actions it had parsed before the failure")
	}
}

func TestLoadSkipsFilesThatAreNotYaml(t *testing.T) {
	acResetCatalog(t)

	fsys := acCatalogFS(map[string]string{
		"real.yaml":  acActionYaml("Real"),
		"other.yml":  acActionYaml("Other"),
		"README.md":  "# not an action\n",
		"notes.json": "{\"this\": \"is not parsed as an action\"}",
	})

	if err := load(fsys); err != nil {
		t.Fatalf("load() returned an error: %v", err)
	}

	if len(Get()) != 2 {
		t.Errorf("load() produced %d actions, want 2 (only the .yaml and .yml files)", len(Get()))
	}
	for _, actionType := range []string{"Real", "Other"} {
		if _, ok := GetActionByType(actionType); !ok {
			t.Errorf("GetActionByType(%q) did not find the action", actionType)
		}
	}
}

// The loader decides what to parse from the directory entry's name alone, without checking
// whether it is a directory, so a subdirectory named like a catalog file is read as one and
// fails the whole load.
func TestLoadFailsOnADirectoryNamedLikeACatalogFile(t *testing.T) {
	acResetCatalog(t)

	fsys := fstest.MapFS{
		"data/real.yaml":            &fstest.MapFile{Data: []byte(acActionYaml("Real"))},
		"data/nested.yaml/some.txt": &fstest.MapFile{Data: []byte("not an action")},
	}

	if err := load(fsys); err == nil {
		t.Fatal("load() accepted a directory named nested.yaml inside the catalog directory")
	}
	if Get() != nil {
		t.Errorf("a failed load() published %v, want the catalog left untouched", Get())
	}
}

// Two files declaring the same type collapse in the type-keyed lookup while both stay in the
// slice the UI renders, so the picker shows a duplicate row that resolves to one entry.
func TestLoadLetsADuplicateTypeOverwriteTheLookupEntry(t *testing.T) {
	acResetCatalog(t)

	fsys := acCatalogFS(map[string]string{
		"a.yaml": acActionYaml("Same"),
		"b.yaml": acActionYaml("Same"),
	})

	if err := load(fsys); err != nil {
		t.Fatalf("load() returned an error: %v", err)
	}

	if len(Get()) != 2 {
		t.Errorf("Get() returned %d actions, want 2", len(Get()))
	}
	if len(actionsByType) != 1 {
		t.Errorf("the type lookup holds %d entries, want 1 collapsed entry", len(actionsByType))
	}
}

// Reads every catalog file independently of load() and compares the result, so a file being
// skipped, read twice or parsed into the wrong field is visible. The order matters too: it is
// the order the UI lists the available actions in.
func TestEveryCatalogFileContributesItsDeclaredType(t *testing.T) {
	loaded := acLoadCatalog(t)

	var want []string
	for _, name := range acCatalogFileNames(t) {
		raw, err := actionsFS.ReadFile("data/" + name)
		if err != nil {
			t.Fatalf("reading data/%s: %v", name, err)
		}
		var parsed Action
		if err := yaml.Unmarshal(raw, &parsed); err != nil {
			t.Fatalf("parsing data/%s: %v", name, err)
		}
		want = append(want, parsed.Metadata.Type)
	}

	got := make([]string, 0, len(loaded))
	for _, a := range loaded {
		got = append(got, a.Metadata.Type)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("catalog types = %v, want %v (derived from the catalog files)", got, want)
	}
}
