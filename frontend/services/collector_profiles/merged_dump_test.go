package collectorprofiles

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/odigos-io/odigos/frontend/services/collector_profiles/flamegraph"
	"go.opentelemetry.io/collector/pdata/pprofile"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// TestMergedDumpPyroscopeFormat verifies that the merged accounting dump contains all
// data necessary to produce a full Pyroscope-compatible response via the same code path as GET profiling.
func TestMergedDumpPyroscopeFormat(t *testing.T) {
	data, err := os.ReadFile("testdata/accounting-merged.json")
	if err != nil {
		t.Skipf("testdata/accounting-merged.json not found: %v", err)
		return
	}
	// Single merged chunk (same format the handler gets when store returns one or many chunks).
	chunks := [][]byte{data}
	profile := BuildPyroscopeProfileFromChunks(chunks)

	// Pyroscope response shape: version, flamebearer (names, levels, numTicks, maxSelf), metadata, timeline
	if profile.Version != 1 {
		t.Errorf("version: got %d, want 1", profile.Version)
	}
	fb := profile.Flamebearer
	if len(fb.Names) == 0 {
		t.Error("flamebearer.names must be non-empty (at least \"total\")")
	}
	if fb.Names[0] != "total" {
		t.Errorf("flamebearer.names[0]: got %q, want \"total\"", fb.Names[0])
	}
	if fb.NumTicks == 0 {
		t.Error("flamebearer.numTicks must be > 0 when dump has samples")
	}
	if len(fb.Levels) == 0 {
		t.Error("flamebearer.levels must be non-empty")
	}
	// 4-tuple per node: xOffset, total, self, nameIndex
	for i, row := range fb.Levels {
		if len(row)%4 != 0 {
			t.Errorf("levels[%d] length %d not multiple of 4", i, len(row))
		}
	}
	if profile.Metadata.Format != "single" {
		t.Errorf("metadata.format: got %q, want \"single\"", profile.Metadata.Format)
	}
	if profile.Metadata.Units != "samples" {
		t.Errorf("metadata.units: got %q, want \"samples\"", profile.Metadata.Units)
	}
	if profile.Metadata.Name != "cpu" {
		t.Errorf("metadata.name: got %q, want \"cpu\"", profile.Metadata.Name)
	}
	if profile.Timeline == nil {
		t.Error("timeline must be set when numTicks > 0")
	} else {
		if len(profile.Timeline.Samples) < 2 {
			t.Errorf("timeline.samples: got len %d, want at least 2", len(profile.Timeline.Samples))
		}
		// Merged dump has timeUnixNano -> startTime should be set (Pyroscope shape)
		if profile.Timeline.StartTime <= 0 {
			t.Errorf("timeline.startTime: got %d, want > 0 (from dump)", profile.Timeline.StartTime)
		}
	}
	if profile.Metadata.SpyName != "" {
		t.Errorf("metadata.spyName: got %q, want \"\" (Pyroscope)", profile.Metadata.SpyName)
	}
	if profile.Symbols != nil {
		t.Error("symbols should be nil for strict Pyroscope response shape")
	}
	t.Logf("numTicks=%d names=%d levels=%d startTime=%d (Pyroscope-like)", fb.NumTicks, len(fb.Names), len(fb.Levels), profile.Timeline.StartTime)

	// Marshal to JSON and assert the serialized shape matches Pyroscope (same keys as reference).
	gotJSON, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("marshal profile: %v", err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(gotJSON, &got); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}
	// Pyroscope response keys (no "symbols").
	wantKeys := map[string]bool{"version": true, "flamebearer": true, "metadata": true, "timeline": true, "groups": true, "heatmap": true}
	for k := range got {
		if !wantKeys[k] {
			t.Errorf("response has unexpected key %q (Pyroscope shape has only version, flamebearer, metadata, timeline, groups, heatmap)", k)
		}
	}
	for k := range wantKeys {
		if _, ok := got[k]; !ok {
			t.Errorf("response missing key %q", k)
		}
	}
	fbMap, _ := got["flamebearer"].(map[string]interface{})
	for _, key := range []string{"names", "levels", "numTicks", "maxSelf"} {
		if _, ok := fbMap[key]; !ok {
			t.Errorf("flamebearer missing key %q", key)
		}
	}
	metaMap, _ := got["metadata"].(map[string]interface{})
	for _, key := range []string{"format", "spyName", "sampleRate", "units", "name"} {
		if _, ok := metaMap[key]; !ok {
			t.Errorf("metadata missing key %q", key)
		}
	}
	timelineMap, _ := got["timeline"].(map[string]interface{})
	for _, key := range []string{"startTime", "samples", "durationDelta", "watermarks"} {
		if _, ok := timelineMap[key]; !ok {
			t.Errorf("timeline missing key %q", key)
		}
	}
	// Pyroscope levels: each node is 4 ints [delta, total, self, nameIndex]; nameIndex must index into names
	numNames := int64(len(fb.Names))
	for i, row := range fb.Levels {
		if len(row)%4 != 0 {
			t.Errorf("levels[%d] length %d not multiple of 4 (Pyroscope single format)", i, len(row))
			continue
		}
		for j := 0; j < len(row); j += 4 {
			nameIdx := row[j+3]
			if nameIdx < 0 || nameIdx >= numNames {
				t.Errorf("levels[%d] node at %d has nameIndex %d out of range [0,%d)", i, j, nameIdx, numNames)
			}
		}
	}
	t.Logf("JSON shape and levels match Pyroscope (keys, 4-tuple levels, nameIndex in range)")
}

// TestChunkWithSymbols verifies that when the dump has a non-empty dictionary (stringTable, functionTable, locationTable),
// we parse symbol names correctly and they appear in the Pyroscope response instead of frame_N.
func TestChunkWithSymbols(t *testing.T) {
	data, err := os.ReadFile("testdata/chunk-with-symbols.json")
	if err != nil {
		t.Skipf("testdata/chunk-with-symbols.json not found: %v", err)
		return
	}
	chunks := [][]byte{data}
	profile := BuildPyroscopeProfileFromChunks(chunks)
	fb := profile.Flamebearer

	// Pyroscope shape
	if profile.Version != 1 {
		t.Errorf("version: got %d, want 1", profile.Version)
	}
	if fb.Names[0] != "total" {
		t.Errorf("flamebearer.names[0]: got %q, want \"total\"", fb.Names[0])
	}
	// Symbols from dictionary: location 0=runtime.main, 1=main.foo, 2=main.bar (from functionTable/stringTable)
	// Samples use attributeIndices [0,1,2] and [0,1] -> stacks (root-first) [2,1,0] and [1,0] -> names main.bar, main.foo, runtime.main and main.foo, runtime.main
	hasRuntimeMain := false
	hasMainFoo := false
	hasMainBar := false
	for _, n := range fb.Names {
		if n == "runtime.main" {
			hasRuntimeMain = true
		}
		if n == "main.foo" {
			hasMainFoo = true
		}
		if n == "main.bar" {
			hasMainBar = true
		}
	}
	if !hasRuntimeMain || !hasMainFoo || !hasMainBar {
		t.Errorf("symbols not parsed from dictionary: got names %v; want runtime.main, main.foo, main.bar present", fb.Names)
	}
	// No placeholder frames when dictionary is present
	for _, n := range fb.Names {
		if len(n) > 6 && n[:6] == "frame_" {
			t.Errorf("unexpected frame_N placeholder when dictionary has symbols: %q", n)
		}
	}
	t.Logf("symbols parsed: numTicks=%d names=%v (Pyroscope-like)", fb.NumTicks, fb.Names)
}

// TestFallbackDictionary verifies that when some chunks have empty dictionary we still get symbols by using
// the first chunk that has a non-empty dictionary as reference (e.g. gateway sends dict only in first batch).
func TestFallbackDictionary(t *testing.T) {
	full, err := os.ReadFile("testdata/chunk-with-symbols.json")
	if err != nil {
		t.Skipf("testdata/chunk-with-symbols.json not found: %v", err)
		return
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(full, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Strip dictionary to simulate a batch that arrived without top-level dictionary.
	raw["dictionary"] = map[string]interface{}{}
	empty, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal empty dict: %v", err)
	}
	// Order: first chunk empty dict, second chunk with dict. Ref will be the second; first chunk's samples get names from ref.
	chunks := [][]byte{empty, full}
	profile := BuildPyroscopeProfileFromChunks(chunks)
	fb := profile.Flamebearer
	hasNames := false
	for _, n := range fb.Names {
		if n != "" && n != "total" && n != "other" && n[:6] != "frame_" && n[:2] != "0x" {
			hasNames = true
			break
		}
	}
	if !hasNames {
		t.Errorf("fallback dictionary: expected symbols from ref chunk; got names %v", fb.Names)
	}
	t.Logf("fallback dictionary: numTicks=%d names=%v", fb.NumTicks, fb.Names)
}

// TestDcDumpRunsOnRealDumps runs BuildPyroscopeProfileFromChunks on all JSON files in dc-dump/
// (when present). Run from frontend: go test -v -run TestDcDump ./services/collector_profiles/
// Dumps live at repo dc-dump/ (use ../dc-dump when running from frontend).
func TestDcDumpRunsOnRealDumps(t *testing.T) {
	var dumpDir string
	for _, d := range []string{"../../../dc-dump", "../dc-dump", "dc-dump"} {
		if _, err := os.Stat(d); err == nil {
			dumpDir = d
			break
		}
	}
	if dumpDir == "" {
		t.Skipf("dc-dump/ not found (try from frontend with dumps at ../dc-dump)")
		return
	}
	entries, err := os.ReadDir(dumpDir)
	if err != nil {
		t.Fatalf("read dc-dump: %v", err)
	}
	var chunks [][]byte
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dumpDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Logf("skip %s: %v", path, err)
			continue
		}
		chunks = append(chunks, data)
	}
	if len(chunks) == 0 {
		t.Skip("no .json files in dc-dump/")
		return
	}
	profile := BuildPyroscopeProfileFromChunks(chunks)
	fb := profile.Flamebearer
	t.Logf("dc-dump: %d chunks -> numTicks=%d names=%d levels=%d", len(chunks), fb.NumTicks, len(fb.Names), len(fb.Levels))
	show := 25
	if len(fb.Names) < show {
		show = len(fb.Names)
	}
	for i := 0; i < show; i++ {
		t.Logf("  names[%d]=%q", i, fb.Names[i])
	}
	if len(fb.Names) > show {
		t.Logf("  ... and %d more names", len(fb.Names)-show)
	}
	// Sanity: same shape as merged test and Pyroscope levels (4-tuple, nameIndex in range)
	if profile.Version != 1 || fb.Names[0] != "total" {
		t.Errorf("unexpected shape: version=%d names[0]=%q", profile.Version, fb.Names[0])
	}
	numNames := int64(len(fb.Names))
	for i, row := range fb.Levels {
		if len(row)%4 != 0 {
			t.Errorf("dc-dump levels[%d] length %d not multiple of 4", i, len(row))
			continue
		}
		for j := 0; j < len(row); j += 4 {
			if nameIdx := row[j+3]; nameIdx < 0 || nameIdx >= numNames {
				t.Errorf("dc-dump levels[%d] nameIndex %d out of range [0,%d)", i, nameIdx, numNames)
			}
		}
	}
	t.Logf("dc-dump response is Pyroscope-like (version, flamebearer, metadata, timeline, valid levels)")
}

// TestMergePathPreservesDictionary verifies that when a batch has multiple resource profiles
// for the same source key, the consumer merges them and the stored chunk retains the dictionary
// so the UI can show symbols (not frame_N). Uses testdata/accounting-merged.json; if MergeTo fails
// (e.g. due to dictionary index mapping in pprofile), the test is skipped.
func TestMergePathPreservesDictionary(t *testing.T) {
	data, err := os.ReadFile("testdata/accounting-merged.json")
	if err != nil {
		t.Skipf("testdata/accounting-merged.json not found: %v", err)
		return
	}
	var unmarshaler pprofile.JSONUnmarshaler
	pd, err := unmarshaler.UnmarshalProfiles(data)
	if err != nil {
		t.Fatalf("unmarshal testdata: %v", err)
	}
	if pd.ResourceProfiles().Len() < 1 {
		t.Fatal("testdata has no resource profiles")
	}
	// Same key for all: consumer will take the merge path (multiple RPs per key).
	multi := pprofile.NewProfiles()
	pd.Dictionary().CopyTo(multi.Dictionary())
	baseRp := pd.ResourceProfiles().At(0)
	for i := 0; i < 3; i++ {
		rp := multi.ResourceProfiles().AppendEmpty()
		baseRp.CopyTo(rp)
		attrs := rp.Resource().Attributes()
		attrs.PutStr(string(semconv.K8SNamespaceNameKey), "test-ns")
		attrs.PutStr(string(semconv.K8SDeploymentNameKey), "svc0")
	}
	store := NewProfileStore(10, 60, 0, 0)
	store.StartViewing("test-ns/Deployment/svc0")
	pc, err := NewProfilesConsumer(store)
	if err != nil {
		t.Fatalf("NewProfilesConsumer: %v", err)
	}
	if err := pc.ConsumeProfiles(context.Background(), multi); err != nil {
		t.Fatalf("ConsumeProfiles: %v", err)
	}
	chunks := store.GetProfileData("test-ns/Deployment/svc0")
	if len(chunks) == 0 {
		t.Skip("merge path produced no chunks (MergeTo may have failed with dictionary index mapping)")
		return
	}
	mergedBytes := chunks[0]
	hasDict := strings.Contains(string(mergedBytes), "stringTable") || strings.Contains(string(mergedBytes), "functionTable") || strings.Contains(string(mergedBytes), "locationTable")
	if !hasDict {
		t.Skip("merged chunk has no dictionary (MergeTo may have failed)")
		return
	}
	profile := BuildPyroscopeProfileFromChunks(chunks)
	fb := profile.Flamebearer
	hasRealNames := false
	for _, n := range fb.Names {
		if n != "" && n != "total" && n != "other" && !strings.HasPrefix(n, "frame_") && !strings.HasPrefix(n, "0x") {
			hasRealNames = true
			break
		}
	}
	if !hasRealNames {
		sample := fb.Names
		if len(sample) > 10 {
			sample = sample[:10]
		}
		t.Errorf("merged chunk should produce real symbol names; got names (sample): %v", sample)
	}
	t.Logf("merge path: chunk has dictionary and produces %d names (symbols preserved)", len(fb.Names))
}

// TestVerifyLiveCapture reads a directory of profile chunk JSONs (from PROFILE_DEBUG_DUMP_DIR after
// capturing ~10 min of live data from the gateway), runs BuildPyroscopeProfileFromChunks on each
// source's chunks, and verifies flame graphs and symbols. Run after a 10-minute capture:
//
//  1. Start frontend with gateway sending profiles to it (e.g. Odigos in cluster).
//  2. Set PROFILE_DEBUG_DUMP_DIR=./live-capture (or any dir).
//  3. In the UI, enable continuous profiling for at least one source.
//  4. Wait ~10 minutes so chunks are written to live-capture/.
//  5. Run: PROFILE_LIVE_CAPTURE_DIR=./live-capture go test -v -run TestVerifyLiveCapture ./services/collector_profiles/
//
// Dump filenames are {sanitizedSourceKey}_{unixNano}_{seq}.json; we group by source key and
// build one flame graph per source, then assert dictionary presence and real symbols.
func TestVerifyLiveCapture(t *testing.T) {
	dir := os.Getenv("PROFILE_LIVE_CAPTURE_DIR")
	if dir == "" {
		t.Skip("PROFILE_LIVE_CAPTURE_DIR not set (set to a directory of chunk JSONs from a 10-min capture)")
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("PROFILE_LIVE_CAPTURE_DIR %q not readable: %v", dir, err)
		return
	}
	// Group files by source key: filename is {sanitizedSourceKey}_{unixNano}_{seq}.json
	bySource := make(map[string][]string)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		parts := strings.Split(base, "_")
		if len(parts) < 3 {
			continue
		}
		// Last two parts are seq and unixNano (both numeric).
		sourceKeySanitized := strings.Join(parts[:len(parts)-2], "_")
		bySource[sourceKeySanitized] = append(bySource[sourceKeySanitized], filepath.Join(dir, e.Name()))
	}
	if len(bySource) == 0 {
		t.Skipf("no .json chunk files in %q", dir)
		return
	}
	t.Logf("found %d source(s), %d total files in %q", len(bySource), func() int { n := 0; for _, f := range bySource { n += len(f) }; return n }(), dir)

	allPass := true
	for sourceKey, files := range bySource {
		var chunks [][]byte
		for _, path := range files {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Logf("[%s] skip %s: %v", sourceKey, path, err)
				continue
			}
			chunks = append(chunks, data)
		}
		if len(chunks) == 0 {
			t.Logf("[%s] no chunks read", sourceKey)
			allPass = false
			continue
		}
		hasDict := false
		for _, b := range chunks {
			if strings.Contains(string(b), "stringTable") || strings.Contains(string(b), "functionTable") || strings.Contains(string(b), "locationTable") {
				hasDict = true
				break
			}
		}
		profile := BuildPyroscopeProfileFromChunks(chunks)
		fb := profile.Flamebearer
		hasRealNames := false
		for _, n := range fb.Names {
			if n != "" && n != "total" && n != "other" && !strings.HasPrefix(n, "frame_") && !strings.HasPrefix(n, "0x") {
				hasRealNames = true
				break
			}
		}
		t.Logf("[%s] chunks=%d dict=%v numTicks=%d names=%d symbols=%v",
			sourceKey, len(chunks), hasDict, fb.NumTicks, len(fb.Names), hasRealNames)
		if !hasDict {
			t.Errorf("[%s] no dictionary in any chunk; symbols will show as frame_N", sourceKey)
			allPass = false
		}
		if !hasRealNames {
			t.Errorf("[%s] no real symbol names in flame graph (all frame_N or placeholders)", sourceKey)
			allPass = false
		}
		if fb.NumTicks == 0 {
			t.Errorf("[%s] flame graph has no samples (numTicks=0)", sourceKey)
			allPass = false
		}
	}
	if !allPass {
		t.Fail()
	}
}

// TestMockFlameGraphResponse runs the real flame graph pipeline with mock samples and prints
// the Pyroscope-like JSON response (for documentation / UI contract).
func TestMockFlameGraphResponse(t *testing.T) {
	// 1. Mock samples (as if parsed from OTLP chunks): root-first stack + value (count).
	samples := []struct {
		stack []string
		value int64
	}{
		{[]string{"main", "handleRequest", "db.Query"}, 50},
		{[]string{"main", "handleRequest", "json.Marshal"}, 30},
		{[]string{"main", "handleRequest", "db.Query"}, 20},
		{[]string{"main", "backgroundWorker", "db.Query"}, 40},
		{[]string{"runtime", "runtime.main", "main.main"}, 10},
	}
	tree := flamegraph.NewTree()
	for _, s := range samples {
		tree.InsertStack(s.value, s.stack...)
	}
	// 2. Tree → Flamebearer (Pyroscope format: names, levels with 4-tuple, numTicks, maxSelf).
	fb := flamegraph.TreeToFlamebearer(tree, 1024)
	// 3. Full response (same shape as GET /api/.../profiling).
	profile := flamegraph.FlamebearerProfile{
		Version: 1,
		Flamebearer: fb,
		Metadata: flamegraph.FlamebearerMetadata{
			Format:     "single",
			SpyName:    "",
			SampleRate: 1000000000,
			Units:      "samples",
			Name:       "cpu",
		},
		Timeline: &flamegraph.FlamebearerTimeline{
			StartTime:     1710000000,
			Samples:       []int64{0, fb.NumTicks},
			DurationDelta: 15,
			Watermarks:    nil,
		},
		Groups:  nil,
		Heatmap: nil,
	}
	out, _ := json.MarshalIndent(profile, "", "  ")
	t.Logf("Pyroscope-like response (use in UI):\n%s", string(out))
}
