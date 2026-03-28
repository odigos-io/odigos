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

// TestFlow_SamplesFromOTLPChunk_ChunkWithSymbols asserts Pyroscope-path extraction on testdata (ConvertOtelToGoogle).
func TestFlow_SamplesFromOTLPChunk_ChunkWithSymbols(t *testing.T) {
	data, err := os.ReadFile("testdata/chunk-with-symbols.json")
	if err != nil {
		t.Skipf("testdata/chunk-with-symbols.json not found: %v", err)
		return
	}
	samples, st := flamegraph.SamplesFromOTLPChunk(data)
	if st.Route != flamegraph.RoutePyroscopeOTLP {
		t.Fatalf("route=%s want %s reason=%q", st.Route, flamegraph.RoutePyroscopeOTLP, st.PyroscopeFailReason)
	}
	if len(samples) == 0 {
		t.Fatal("expected at least one sample from chunk-with-symbols")
	}
	var totalValue int64
	nameSet := make(map[string]bool)
	for _, s := range samples {
		totalValue += s.Value
		for _, n := range s.Stack {
			nameSet[n] = true
		}
	}
	if totalValue <= 0 {
		t.Errorf("sum of sample values should be > 0, got %d", totalValue)
	}
	if !nameSet["runtime.main"] || !nameSet["main.foo"] || !nameSet["main.bar"] {
		t.Errorf("expected symbols in stacks; stacks sample names=%v", nameSet)
	}
	t.Logf("SamplesFromOTLPChunk: samples=%d totalValue=%d route=%s", len(samples), totalValue, st.Route)
}

// TestFlow_BuildProfile_FromChunkWithSymbols runs BuildPyroscopeProfileFromChunks on
// chunk-with-symbols.json and asserts full Pyroscope response shape and content.
func TestFlow_BuildProfile_FromChunkWithSymbols(t *testing.T) {
	data, err := os.ReadFile("testdata/chunk-with-symbols.json")
	if err != nil {
		t.Skipf("testdata/chunk-with-symbols.json not found: %v", err)
		return
	}
	chunks := [][]byte{data}
	profile := BuildPyroscopeProfileFromChunks(chunks)
	fb := profile.Flamebearer

	// Response shape
	if profile.Version != 1 {
		t.Errorf("version: got %d, want 1", profile.Version)
	}
	if len(fb.Names) == 0 || fb.Names[0] != "total" {
		t.Errorf("flamebearer.names[0]: got %q, want \"total\"", fb.Names[0])
	}
	if fb.NumTicks <= 0 {
		t.Errorf("numTicks: got %d, want > 0", fb.NumTicks)
	}
	if len(fb.Levels) == 0 {
		t.Error("flamebearer.levels must be non-empty")
	}
	numNames := int64(len(fb.Names))
	for i, row := range fb.Levels {
		if len(row)%4 != 0 {
			t.Errorf("levels[%d]: length %d not multiple of 4", i, len(row))
			continue
		}
		for j := 3; j < len(row); j += 4 {
			idx := row[j]
			if idx < 0 || idx >= numNames {
				t.Errorf("levels[%d] nameIndex %d out of range [0,%d)", i, idx, numNames)
			}
		}
	}
	// Symbols from dictionary present (no frame_N)
	for _, n := range fb.Names {
		if strings.HasPrefix(n, "frame_") {
			t.Errorf("unexpected frame_N when dictionary has symbols: %q", n)
		}
	}
	t.Logf("profile: numTicks=%d names=%v levels=%d", fb.NumTicks, fb.Names, len(fb.Levels))
}

// TestFlow_BuildProfile_FromAccountingMerged runs the full pipeline on accounting-merged.json
// (real OTLP dump) and asserts valid Pyroscope response.
func TestFlow_BuildProfile_FromAccountingMerged(t *testing.T) {
	data, err := os.ReadFile("testdata/accounting-merged.json")
	if err != nil {
		t.Skipf("testdata/accounting-merged.json not found: %v", err)
		return
	}
	chunks := [][]byte{data}
	profile, debug := BuildPyroscopeProfileFromChunksWithDebug(chunks)
	fb := profile.Flamebearer

	if debug.ParseErrors > 0 {
		t.Errorf("parse errors: %d", debug.ParseErrors)
	}
	if profile.Version != 1 {
		t.Errorf("version: got %d, want 1", profile.Version)
	}
	if fb.NumTicks <= 0 && debug.ChunkCount > 0 {
		t.Errorf("numTicks=%d but chunkCount=%d (samples may not have been extracted)", fb.NumTicks, debug.ChunkCount)
	}
	if len(fb.Levels) == 0 && fb.NumTicks > 0 {
		t.Error("levels empty but numTicks > 0")
	}
	numNames := int64(len(fb.Names))
	for i, row := range fb.Levels {
		if len(row)%4 != 0 {
			t.Errorf("levels[%d]: length %d not multiple of 4", i, len(row))
			continue
		}
		for j := 3; j < len(row); j += 4 {
			if idx := row[j]; idx < 0 || idx >= numNames {
				t.Errorf("levels[%d] nameIndex %d out of range [0,%d)", i, idx, numNames)
			}
		}
	}
	if profile.Metadata.Format != "single" || profile.Metadata.Name != "cpu" {
		t.Errorf("metadata: format=%q name=%q", profile.Metadata.Format, profile.Metadata.Name)
	}
	t.Logf("accounting-merged: chunkCount=%d numTicks=%d names=%d levels=%d parseErrors=%d",
		debug.ChunkCount, fb.NumTicks, len(fb.Names), len(fb.Levels), debug.ParseErrors)
}

// TestFlow_SamplesVsBuild_NumTicksConsistency cross-checks: sum of Pyroscope-path sample values equals flame NumTicks.
func TestFlow_SamplesVsBuild_NumTicksConsistency(t *testing.T) {
	data, err := os.ReadFile("testdata/chunk-with-symbols.json")
	if err != nil {
		t.Skipf("testdata/chunk-with-symbols.json not found: %v", err)
		return
	}
	samples, st := flamegraph.SamplesFromOTLPChunk(data)
	if st.Route != flamegraph.RoutePyroscopeOTLP {
		t.Fatalf("route=%s reason=%q", st.Route, st.PyroscopeFailReason)
	}
	var sum int64
	for _, s := range samples {
		sum += s.Value
	}
	profile := BuildPyroscopeProfileFromChunks([][]byte{data})
	if profile.Flamebearer.NumTicks != sum {
		t.Errorf("NumTicks %d != sum of sample values %d", profile.Flamebearer.NumTicks, sum)
	}
	t.Logf("consistency: sample sum=%d NumTicks=%d", sum, profile.Flamebearer.NumTicks)
}

// TestFlow_Consumer_StoreOne_Then_GetProfile runs the full ingest path: pprofile → consumer
// → store (one chunk per RP) → GetProfileData → BuildPyroscopeProfileFromChunks.
// Uses testdata/accounting-merged.json unmarshalled to pprofile, then one RP sent to consumer.
func TestFlow_Consumer_StoreOne_Then_GetProfile(t *testing.T) {
	data, err := os.ReadFile("testdata/accounting-merged.json")
	if err != nil {
		t.Skipf("testdata/accounting-merged.json not found: %v", err)
		return
	}
	var unmarshaler pprofile.JSONUnmarshaler
	pd, err := unmarshaler.UnmarshalProfiles(data)
	if err != nil {
		t.Fatalf("UnmarshalProfiles: %v", err)
	}
	if pd.ResourceProfiles().Len() < 1 {
		t.Fatal("testdata has no resource profiles")
	}
	// Build a batch with a single RP so we hit storeOne path (no merge).
	single := pprofile.NewProfiles()
	pd.Dictionary().CopyTo(single.Dictionary())
	pd.ResourceProfiles().At(0).CopyTo(single.ResourceProfiles().AppendEmpty())
	// Ensure resource has a source key the store will accept.
	attrs := single.ResourceProfiles().At(0).Resource().Attributes()
	attrs.PutStr(string(semconv.K8SNamespaceNameKey), "otel-demo")
	attrs.PutStr(string(semconv.K8SDeploymentNameKey), "accounting")
	key := "otel-demo/Deployment/accounting"

	store := NewProfileStore(10, 60, 5*1024*1024, 0)
	store.StartViewing(key)
	consumer, err := NewProfilesConsumer(store)
	if err != nil {
		t.Fatalf("NewProfilesConsumer: %v", err)
	}
	if err := consumer.ConsumeProfiles(context.Background(), single); err != nil {
		t.Fatalf("ConsumeProfiles: %v", err)
	}
	chunks := store.GetProfileData(key)
	if len(chunks) == 0 {
		t.Fatal("store has no chunks after ConsumeProfiles (source key may not match)")
	}
	profile := BuildPyroscopeProfileFromChunks(chunks)
	fb := profile.Flamebearer
	if profile.Version != 1 {
		t.Errorf("version: got %d, want 1", profile.Version)
	}
	if len(fb.Names) == 0 || fb.Names[0] != "total" {
		t.Errorf("flamebearer.names: got %v", fb.Names)
	}
	if len(fb.Levels) == 0 && fb.NumTicks > 0 {
		t.Error("levels empty but numTicks > 0")
	}
	t.Logf("consumer flow: chunks=%d numTicks=%d names=%d", len(chunks), fb.NumTicks, len(fb.Names))
}

// TestFlow_RealDumps_IfPresent runs the pipeline on real pod dumps when profile-dumps-from-pod
// (or dc-dump) is present, and asserts valid Pyroscope response and optional real symbols.
func TestFlow_RealDumps_IfPresent(t *testing.T) {
	var dumpDir string
	for _, d := range []string{"../../../profile-dumps-from-pod", "../../../dc-dump", "../profile-dumps-from-pod", "profile-dumps-from-pod"} {
		if _, err := os.Stat(d); err == nil {
			dumpDir = d
			break
		}
	}
	if dumpDir == "" {
		t.Skip("profile-dumps-from-pod/ or dc-dump/ not found")
		return
	}
	entries, err := os.ReadDir(dumpDir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	var chunks [][]byte
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
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
		t.Skip("no .json files in dump dir")
		return
	}
	profile, debug := BuildPyroscopeProfileFromChunksWithDebug(chunks)
	fb := profile.Flamebearer

	// Must produce valid Pyroscope shape
	if profile.Version != 1 {
		t.Errorf("version: got %d, want 1", profile.Version)
	}
	if len(fb.Names) == 0 || fb.Names[0] != "total" {
		t.Errorf("flamebearer.names[0]: got %q", fb.Names[0])
	}
	numNames := int64(len(fb.Names))
	for i, row := range fb.Levels {
		if len(row)%4 != 0 {
			t.Errorf("levels[%d] length %d not multiple of 4", i, len(row))
			continue
		}
		for j := 3; j < len(row); j += 4 {
			if idx := row[j]; idx < 0 || idx >= numNames {
				t.Errorf("levels[%d] nameIndex %d out of range", i, idx)
			}
		}
	}
	// Log truncated JSON when we have data (for inspection).
	if fb.NumTicks > 0 {
		out, _ := json.MarshalIndent(profile, "", "  ")
		n := 2000
		if len(out) < n {
			n = len(out)
		}
		t.Logf("real dumps flow: chunks=%d numTicks=%d parseErrors=%d\nSample response (first %d chars):\n%s",
			len(chunks), fb.NumTicks, debug.ParseErrors, n, string(out)[:n])
	}
	t.Logf("real dumps: chunks=%d numTicks=%d names=%d levels=%d parseErrors=%d",
		len(chunks), fb.NumTicks, len(fb.Names), len(fb.Levels), debug.ParseErrors)
}

// TestFlow_GwProfilesDump_IfPresent uses gw-profiles-dump.logs (gateway debug text log).
// It parses the log, builds minimal OTLP JSON from string table + first resource, runs
// BuildPyroscopeProfileFromChunks, and asserts valid Pyroscope shape. Skips if file missing.
func TestFlow_GwProfilesDump_IfPresent(t *testing.T) {
	// Prefer testdata copy; else odigos/gw-profiles-dump.logs (cwd is package dir when running tests).
	candidates := []string{
		"testdata/gw-profiles-dump.logs",
		"../../../../gw-profiles-dump.logs", // odigos/gw-profiles-dump.logs from frontend/services/collector_profiles
		"../../../gw-profiles-dump.logs",    // if run from frontend
	}
	var path string
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			path = c
			break
		}
	}
	if path == "" {
		t.Skip("gw-profiles-dump.logs not found (try testdata/ or repo root)")
		return
	}
	parsed, err := ParseGwProfilesDump(path)
	if err != nil {
		t.Fatalf("ParseGwProfilesDump(%s): %v", path, err)
	}
	if len(parsed.StringTable) == 0 {
		t.Fatal("expected non-empty string table from gw dump")
	}
	if len(parsed.Resources) == 0 {
		t.Fatal("expected at least one ResourceProfiles block in gw dump")
	}
	otlpJSON, err := parsed.ToOTLPJSON(true, 5)
	if err != nil {
		t.Fatalf("ToOTLPJSON: %v", err)
	}
	chunks := [][]byte{otlpJSON}
	profile := BuildPyroscopeProfileFromChunks(chunks)
	fb := profile.Flamebearer

	if profile.Version != 1 {
		t.Errorf("version: got %d, want 1", profile.Version)
	}
	if len(fb.Names) == 0 || fb.Names[0] != "total" {
		t.Errorf("flamebearer.names[0]: got %q", fb.Names[0])
	}
	if fb.NumTicks <= 0 {
		t.Errorf("numTicks: got %d, want > 0", fb.NumTicks)
	}
	// First resource from gw dump should appear in metadata or be preserved in chunk
	t.Logf("gw dump flow: path=%s stringTableLen=%d resources=%d numTicks=%d names=%d",
		path, len(parsed.StringTable), len(parsed.Resources), fb.NumTicks, len(fb.Names))
}
