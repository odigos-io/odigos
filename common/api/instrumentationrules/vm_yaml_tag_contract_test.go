package instrumentationrules

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

// The structs in this package are shared with the vm-agent, which reads and writes them as yaml.
// yaml does not look at `json` tags: an untagged field is matched against its LOWERCASED Go field
// name, so a field tagged only `json:"odigosLogLevel"` is bound to the key `odigosloglevel` and a
// document written in the CRD's own key names loses the value with no error. #4862 added yaml tags
// to this whole package for that reason, and #6137 had to do the same for
// common/api/actions/pii_masking.go. Nothing enforced it, so #6145 shipped CppCustomProbe and the
// four *WithMetadata types with no yaml tags at all.
//
// This file pins the contract. `common` depends on gopkg.in/yaml.v2 directly and the vm-agent uses
// yaml.v3; both bind untagged fields to the lowercased field name and both nest an embedded struct
// that is not yaml-inlined, so v2 demonstrates the behaviour faithfully.

// vmyKnownGaps are the fields that carry a json tag and no yaml tag today, mapped to why. Every
// entry is checked by the anti-staleness test below, which fails once an entry is fixed or
// removed, so the list cannot rot into an unconditional exemption.
var vmyKnownGaps = map[string]string{
	"AgentDiagnostics.OdigosLogLevel":                   "#4862 missed it",
	"AgentDiagnostics.OpenTelemetryComponentsLogLevel":  "#4862 missed it",
	"CppCustomProbe.Signature":                          "#6145",
	"GolangCustomProbeWithMetadata.GolangCustomProbe":   "#6145",
	"GolangCustomProbeWithMetadata.CustomProbeMetadata": "#6145",
	"JavaCustomProbeWithMetadata.JavaCustomProbe":       "#6145",
	"JavaCustomProbeWithMetadata.CustomProbeMetadata":   "#6145",
	"CppCustomProbeWithMetadata.CppCustomProbe":         "#6145",
	"CppCustomProbeWithMetadata.CustomProbeMetadata":    "#6145",
	"PhpCustomProbeWithMetadata.PhpCustomProbe":         "#6145",
	"PhpCustomProbeWithMetadata.CustomProbeMetadata":    "#6145",
}

// vmyExpectedSilentDrops are the gaps whose yaml key differs from their json key, which is where
// a missing tag stops being cosmetic and starts dropping data. It is spelled out rather than
// derived so that a field being renamed into or out of this set has to be acknowledged: a gap is
// harmless only for as long as its json name stays equal to its own lowercased field name, and
// nothing about the type system preserves that.
var vmyExpectedSilentDrops = []string{
	"AgentDiagnostics.OdigosLogLevel",
	"AgentDiagnostics.OpenTelemetryComponentsLogLevel",
	"CppCustomProbeWithMetadata.CppCustomProbe",
	"CppCustomProbeWithMetadata.CustomProbeMetadata",
	"GolangCustomProbeWithMetadata.CustomProbeMetadata",
	"GolangCustomProbeWithMetadata.GolangCustomProbe",
	"JavaCustomProbeWithMetadata.CustomProbeMetadata",
	"JavaCustomProbeWithMetadata.JavaCustomProbe",
	"PhpCustomProbeWithMetadata.CustomProbeMetadata",
	"PhpCustomProbeWithMetadata.PhpCustomProbe",
}

// vmyField is one struct field of this package as it appears in the source.
type vmyField struct {
	key      string // "<TypeName>.<FieldName>"; an embedded field is keyed by its type name
	jsonTag  string
	yamlTag  string
	hasYAML  bool
	embedded bool
}

// jsonKey is the object key encoding/json uses for the field; "" means the field is promoted.
func (f vmyField) jsonKey() string {
	return strings.Split(f.jsonTag, ",")[0]
}

// yamlKeyWithoutATag is the key yaml falls back to when the field carries no yaml tag.
func (f vmyField) yamlKeyWithoutATag() string {
	return strings.ToLower(strings.TrimPrefix(f.key, strings.Split(f.key, ".")[0]+"."))
}

// dropsDataWithoutAYAMLTag reports whether yaml and json disagree about where the field lives.
// An embedded field is always a disagreement: json promotes it, yaml nests it under the type name.
func (f vmyField) dropsDataWithoutAYAMLTag() bool {
	if f.embedded {
		return true
	}
	return f.jsonKey() != f.yamlKeyWithoutATag()
}

// vmyScanPackage reads every non-generated, non-test source file of this package and returns the
// json-tagged fields of every struct it declares. It parses the source rather than walking types
// with reflect because reflect cannot enumerate a package's types: a reflect-based walk needs a
// hand-maintained list of root types, and a field on a struct nobody remembered to list would be
// exempt from the contract for exactly the reason the contract exists.
func vmyScanPackage(t *testing.T) []vmyField {
	t.Helper()

	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob package sources: %v", err)
	}

	var fields []vmyField
	var scanned []string
	for _, path := range paths {
		base := filepath.Base(path)
		if strings.HasPrefix(base, "zz_generated") || strings.HasSuffix(base, "_test.go") {
			continue
		}
		scanned = append(scanned, base)

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			spec, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			st, ok := spec.Type.(*ast.StructType)
			if !ok {
				return true
			}
			for _, f := range st.Fields.List {
				if f.Tag == nil {
					continue
				}
				tag := reflect.StructTag(strings.Trim(f.Tag.Value, "`"))
				jsonTag, ok := tag.Lookup("json")
				if !ok {
					continue
				}
				name, embedded := vmyFieldName(f)
				yamlTag, hasYAML := tag.Lookup("yaml")
				fields = append(fields, vmyField{
					key:      spec.Name.Name + "." + name,
					jsonTag:  jsonTag,
					yamlTag:  yamlTag,
					hasYAML:  hasYAML,
					embedded: embedded,
				})
			}
			return true
		})
	}

	// Guard against the scan matching nothing (a moved directory, a glob that stops resolving):
	// without this, every contract below would pass over an empty set of fields.
	if len(scanned) < 5 {
		t.Fatalf("scanned only %d source files (%v); the package layout changed", len(scanned), scanned)
	}
	if len(fields) < 40 {
		t.Fatalf("found only %d json-tagged fields in %v; the scan is not reaching the structs", len(fields), scanned)
	}
	return fields
}

// vmyFieldName returns the identifier yaml uses for a field. An embedded field has no name of its
// own, so yaml keys it by the embedded type's name unless the tag inlines it.
func vmyFieldName(f *ast.Field) (string, bool) {
	if len(f.Names) > 0 {
		return f.Names[0].Name, false
	}
	expr := f.Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name, true
	}
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		return sel.Sel.Name, true
	}
	return "<unnamed>", true
}

func vmyFieldsByKey(t *testing.T) map[string]vmyField {
	t.Helper()
	byKey := make(map[string]vmyField)
	for _, f := range vmyScanPackage(t) {
		byKey[f.key] = f
	}
	return byKey
}

func TestEveryJSONTaggedFieldCarriesTheMatchingYAMLTag(t *testing.T) {
	t.Parallel()

	for _, f := range vmyScanPackage(t) {
		if _, known := vmyKnownGaps[f.key]; known {
			continue
		}
		if !f.hasYAML {
			t.Errorf("%s has json:%q but no yaml tag: the vm-agent reads this package as yaml, which "+
				"binds the field to %q instead", f.key, f.jsonTag, f.yamlKeyWithoutATag())
			continue
		}
		if f.yamlTag != f.jsonTag {
			t.Errorf("%s has json:%q but yaml:%q; both wire formats must use the same key and options",
				f.key, f.jsonTag, f.yamlTag)
		}
	}
}

func TestPinnedYAMLTagGapsAreStillReal(t *testing.T) {
	t.Parallel()

	byKey := vmyFieldsByKey(t)
	for key, why := range vmyKnownGaps {
		f, ok := byKey[key]
		if !ok {
			t.Errorf("%s is pinned as a known gap (%s) but no longer exists; delete the entry", key, why)
			continue
		}
		if f.hasYAML {
			t.Errorf("%s is pinned as a known gap (%s) but now has yaml:%q; delete the entry", key, why, f.yamlTag)
		}
	}
}

// Only the gaps where yaml and json disagree about the key actually lose data. Computing that
// subset keeps the finding honest: CppCustomProbe.Signature is missing its tag like the rest, but
// "signature" is already its own lowercased field name, so yaml still binds it today.
func TestKnownYAMLGapsThatSilentlyDropDataAreTheExpectedOnes(t *testing.T) {
	t.Parallel()

	byKey := vmyFieldsByKey(t)
	var dropping []string
	for key := range vmyKnownGaps {
		if f, ok := byKey[key]; ok && f.dropsDataWithoutAYAMLTag() {
			dropping = append(dropping, key)
		}
	}
	sort.Strings(dropping)

	if !reflect.DeepEqual(dropping, vmyExpectedSilentDrops) {
		t.Fatalf("fields silently dropped by yaml =\n  %v\nwant\n  %v", dropping, vmyExpectedSilentDrops)
	}
	if len(dropping) == len(vmyKnownGaps) {
		t.Fatal("every known gap drops data, so this test no longer distinguishes anything; " +
			"it is only meaningful while at least one gap is harmless")
	}
}

// Every embedded field in this package is json-inlined, so none may be pinned for a reason other
// than the missing yaml inline. Without this, the gap list could exempt a deliberately nested embed.
func TestPinnedEmbeddedGapsAreAllJSONInlined(t *testing.T) {
	t.Parallel()

	byKey := vmyFieldsByKey(t)
	embeddedPins := 0
	for key := range vmyKnownGaps {
		f, ok := byKey[key]
		if !ok || !f.embedded {
			continue
		}
		embeddedPins++
		if f.jsonTag != ",inline" {
			t.Errorf("%s is embedded with json:%q, want json:\",inline\"", key, f.jsonTag)
		}
	}
	if embeddedPins == 0 {
		t.Fatal("no embedded field is pinned; this test is not exercising anything")
	}
}

// The tag contract only matters because yaml really does drop these values. Decoding a document
// written in the CRD's own key names proves it: a correctly tagged probe round-trips, while the
// agent diagnostics log levels come back nil and a metadata-carrying probe comes back entirely
// empty, both with a nil error.
func TestYAMLSilentlyDropsFieldsMissingAYAMLTag(t *testing.T) {
	t.Parallel()

	const probeDoc = "className: com.acme.Pricing\nmethodName: calculate\n"
	var java JavaCustomProbe
	if err := yaml.Unmarshal([]byte(probeDoc), &java); err != nil {
		t.Fatalf("decode java probe: %v", err)
	}
	if java.ClassName != "com.acme.Pricing" || java.MethodName != "calculate" {
		t.Fatalf("a yaml-tagged probe must round-trip, got %+v", java)
	}

	var diagnostics AgentDiagnostics
	if err := yaml.Unmarshal([]byte("odigosLogLevel: debug\nopenTelemetryComponentsLogLevel: error\n"), &diagnostics); err != nil {
		t.Fatalf("decode agent diagnostics: %v", err)
	}
	if diagnostics.OdigosLogLevel != nil || diagnostics.OpenTelemetryComponentsLogLevel != nil {
		t.Fatalf("AgentDiagnostics now decodes from yaml (%+v); remove its entries from vmyKnownGaps "+
			"and vmyExpectedSilentDrops, and delete this half of the test", diagnostics)
	}

	var withMetadata JavaCustomProbeWithMetadata
	if err := yaml.Unmarshal([]byte(probeDoc+"notes: hot path\n"), &withMetadata); err != nil {
		t.Fatalf("decode java probe with metadata: %v", err)
	}
	if withMetadata.ClassName != "" || withMetadata.MethodName != "" || withMetadata.Notes != "" {
		t.Fatalf("JavaCustomProbeWithMetadata now decodes inline from yaml (%+v); remove its entries "+
			"from vmyKnownGaps and vmyExpectedSilentDrops, and delete this half of the test", withMetadata)
	}
}

// The encode direction is the half that writes data the vm-agent later has to read back: a
// metadata-carrying probe is emitted as two nested maps rather than one flat probe.
func TestYAMLEncodesProbeMetadataAsNestedMapsInsteadOfInlining(t *testing.T) {
	t.Parallel()

	encoded, err := yaml.Marshal(JavaCustomProbeWithMetadata{
		JavaCustomProbe:     JavaCustomProbe{ClassName: "com.acme.Pricing", MethodName: "calculate"},
		CustomProbeMetadata: CustomProbeMetadata{Notes: "hot path", Category: CustomProbeCategoryPerformance},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var document map[string]interface{}
	if err := yaml.Unmarshal(encoded, &document); err != nil {
		t.Fatalf("re-decode: %v", err)
	}
	var keys []string
	for key := range document {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Spelled out rather than derived from the type names, so renaming an embedded type does not
	// silently rewrite the expectation along with the behaviour.
	want := []string{"customprobemetadata", "javacustomprobe"}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("top-level yaml keys = %v, want %v; if this now flattens to className/methodName/notes "+
			"then the inline tags were added, so drop the JavaCustomProbeWithMetadata entries from "+
			"vmyKnownGaps and vmyExpectedSilentDrops", keys, want)
	}
	if strings.Contains(string(encoded), "\nclassName:") {
		t.Fatalf("probe fields are no longer nested:\n%s", encoded)
	}
}
